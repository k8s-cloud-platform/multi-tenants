/*
Copyright 2022 The KCP Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"crypto/x509"
	"errors"
	"net"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/tools/clientcmd"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/klog/v2"
	"k8s.io/utils/pointer"

	"github.com/k8s-cloud-platform/multi-tenants/pkg/apis/tenancy/v1alpha1"
	"github.com/k8s-cloud-platform/multi-tenants/pkg/kubeconfig"
	"github.com/k8s-cloud-platform/multi-tenants/pkg/secret"
)

func (c *TenantController) reconcilePhase(tenant *v1alpha1.Tenant) {
	if tenant.Status.Phase == "" {
		tenant.Status.SetPhase(v1alpha1.TenantPhasePending)
	}

	if meta.IsStatusConditionFalse(tenant.Status.Conditions, v1alpha1.TenantConditionProvisioned) {
		tenant.Status.SetPhase(v1alpha1.TenantPhaseProvisioning)
	}

	if meta.IsStatusConditionTrue(tenant.Status.Conditions, v1alpha1.TenantConditionProvisioned) {
		tenant.Status.SetPhase(v1alpha1.TenantPhaseProvisioned)
	}

	if meta.IsStatusConditionFalse(tenant.Status.Conditions, v1alpha1.TenantConditionReady) {
		tenant.Status.SetPhase(v1alpha1.TenantPhaseFailed)
	}

	if meta.IsStatusConditionTrue(tenant.Status.Conditions, v1alpha1.TenantConditionReady) {
		tenant.Status.SetPhase(v1alpha1.TenantPhaseReady)
	}

	if !tenant.DeletionTimestamp.IsZero() {
		tenant.Status.SetPhase(v1alpha1.TenantPhaseTerminating)
	}
}

func (c *TenantController) reconcileSecret(ctx context.Context, tenant *v1alpha1.Tenant) error {
	// check server cert
	_, err := c.ClientSet.CoreV1().Secrets(tenant.Name).Get(ctx, "server-cert", metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			klog.Error(err, "unable to get secret object")
			return err
		}
	} else {
		klog.V(2).Info("secret[server-cert] already exists, skip kubeconfig phase")
		return nil
	}

	// server ca
	serverCA, serverCAKey, err := secret.NewCA(nil)
	if err != nil {
		klog.ErrorS(err, "unable to new ca for server")
		return err
	}
	// apiserver
	serverCert, serverKey, err := secret.NewCertAndKey(serverCA, serverCAKey, &certutil.Config{
		CommonName: "kube-apiserver",
		AltNames: certutil.AltNames{
			DNSNames: []string{
				"kube-apiserver." + tenant.Name + ".svc",
				"localhost",
			},
			IPs: []net.IP{
				net.ParseIP("127.0.0.1"),
			},
		},
		Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	if err != nil {
		klog.ErrorS(err, "unable to cert secret for kube-apiserver")
		return err
	}
	// apiserver-kubelet-client
	kubeletCert, kubeletKey, err := secret.NewCertAndKey(serverCA, serverCAKey, &certutil.Config{
		CommonName:   "kube-apiserver-kubelet-client",
		Organization: []string{"system:masters"},
		Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	if err != nil {
		klog.ErrorS(err, "unable to cert secret for apiserver-kubelet-client")
		return err
	}

	// front proxy ca
	frontCA, frontCAKey, err := secret.NewCA(&certutil.Config{
		AltNames: certutil.AltNames{
			DNSNames: []string{"front-proxy-ca"},
		},
	})
	if err != nil {
		klog.ErrorS(err, "unable to new ca for front proxy")
		return err
	}
	// front-proxy-client
	frontCert, frontKey, err := secret.NewCertAndKey(frontCA, frontCAKey, &certutil.Config{
		CommonName: "front-proxy-client",
		Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	if err != nil {
		klog.ErrorS(err, "unable to cert secret for front-proxy-client")
		return err
	}

	// sa.pub
	pub, pubKey, err := secret.NewPubAndKey()
	if err != nil {
		klog.ErrorS(err, "unable to new pub and key for sa")
		return err
	}
	encodedPub, err := secret.EncodePublicKeyPEM(pub)
	if err != nil {
		klog.Error(err, "unable to encode public key pem for sa")
		return err
	}

	obj := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "server-cert",
			Namespace: tenant.Name,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion: tenant.APIVersion,
					Kind:       tenant.Kind,
					Name:       tenant.Name,
					UID:        tenant.UID,
				},
			},
		},
		Type: corev1.SecretType("kcp/kube-secret"),
		Data: func() map[string][]byte {
			// update addon secret num if modified
			result := make(map[string][]byte, len(c.EtcdSecret)+12)
			for k, v := range c.EtcdSecret {
				result[k] = v
			}
			result["ca.crt"] = secret.EncodeCertPEM(serverCA)
			result["ca.key"] = secret.EncodePrivateKeyPEM(serverCAKey)
			result["apiserver.crt"] = secret.EncodeCertPEM(serverCert)
			result["apiserver.key"] = secret.EncodePrivateKeyPEM(serverKey)
			result["apiserver-kubelet-client.crt"] = secret.EncodeCertPEM(kubeletCert)
			result["apiserver-kubelet-client.key"] = secret.EncodePrivateKeyPEM(kubeletKey)
			result["front-proxy-ca.crt"] = secret.EncodeCertPEM(frontCA)
			result["front-proxy-ca.key"] = secret.EncodePrivateKeyPEM(frontCAKey)
			result["front-proxy-client.crt"] = secret.EncodeCertPEM(frontCert)
			result["front-proxy-client.key"] = secret.EncodePrivateKeyPEM(frontKey)
			result["sa.pub"] = encodedPub
			result["sa.key"] = secret.EncodePrivateKeyPEM(pubKey)
			return result
		}(),
	}
	if _, err := c.ClientSet.CoreV1().Secrets(obj.Namespace).Create(ctx, obj, metav1.CreateOptions{}); err != nil {
		klog.Error(err, "unable to create secret")
		return err
	}

	return nil
}

func (c *TenantController) reconcileKubeConfig(ctx context.Context, tenant *v1alpha1.Tenant) error {
	generateAdmin := true
	if _, err := c.ClientSet.CoreV1().Secrets(tenant.Name).Get(ctx, "kubeconfig-admin", metav1.GetOptions{}); err == nil {
		klog.V(2).Info("secret[kubeconfig-admin] already exists, skip kubeconfig[kubeconfig-admin] phase")
		generateAdmin = false
	} else if !apierrors.IsNotFound(err) {
		klog.Error(err, "unable to get secret object")
		return err
	}

	generateCM := true
	if _, err := c.ClientSet.CoreV1().Secrets(tenant.Name).Get(ctx, "kubeconfig-controller-manager", metav1.GetOptions{}); err == nil {
		klog.V(2).Info("secret[server-cert] already exists, skip kubeconfig[kubeconfig-controller-manager] phase")
		generateCM = false
	} else if !apierrors.IsNotFound(err) {
		klog.Error(err, "unable to get secret object")
		return err
	}

	// direct return if already created
	if !generateAdmin && !generateCM {
		return nil
	}

	serverCert, err := c.ClientSet.CoreV1().Secrets(tenant.Name).Get(ctx, "server-cert", metav1.GetOptions{})
	if err != nil {
		klog.ErrorS(err, "unable to get secret for server-cert")
		return err
	}

	ca, ok := serverCert.Data["ca.crt"]
	if !ok {
		klog.Error("ca.crt is empty in server-cert secret")
		return errors.New("empty ca.crt")
	}
	caCert, err := secret.DecodeCertPEM(ca)
	if err != nil {
		klog.ErrorS(err, "unable to decode cert pem")
		return err
	}

	key, ok := serverCert.Data["ca.key"]
	if !ok {
		klog.Error("ca.key is empty in server-cert secret")
		return errors.New("empty ca.key")
	}
	caKey, err := secret.DecodePrivateKeyPEM(key)
	if err != nil {
		klog.ErrorS(err, "unable to decode key pem")
		return err
	}

	// admin kubeconfig
	if generateAdmin {
		config, err := kubeconfig.NewWithSecret(
			tenant.Name,
			"https://kube-apiserver."+tenant.Name+".svc:6443",
			caCert,
			caKey,
			&certutil.Config{
				CommonName:   "kubernetes-admin",
				Organization: []string{"system:masters"},
				Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			},
		)
		if err != nil {
			klog.ErrorS(err, "unable to generate secret for admin.conf")
			return err
		}
		adminConfig, err := clientcmd.Write(*config)
		if err != nil {
			klog.ErrorS(err, "unable to decode to kubeconfig for admin.conf")
			return err
		}

		obj := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubeconfig-admin",
				Namespace: tenant.Name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: tenant.APIVersion,
						Kind:       tenant.Kind,
						Name:       tenant.Name,
						UID:        tenant.UID,
					},
				},
			},
			Type: corev1.SecretType("kcp/kubeconfig"),
			Data: map[string][]byte{
				"admin.conf": adminConfig,
			},
		}
		if _, err := c.ClientSet.CoreV1().Secrets(obj.Namespace).Create(ctx, obj, metav1.CreateOptions{}); err != nil {
			klog.Error(err, "unable to create secret")
			return err
		}
	}

	// controller-manager kubeconfig
	if generateCM {
		// controller-manager.conf
		config, err := kubeconfig.NewWithSecret(
			tenant.Name,
			"https://kube-apiserver."+tenant.Name+".svc:6443",
			caCert,
			caKey,
			&certutil.Config{
				CommonName: "system:kube-controller-manager",
				Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			},
		)
		if err != nil {
			klog.ErrorS(err, "unable to generate secret for admin.conf")
			return err
		}
		adminConfig, err := clientcmd.Write(*config)
		if err != nil {
			klog.ErrorS(err, "unable to decode to kubeconfig for admin.conf")
			return err
		}

		obj := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kubeconfig-controller-manager",
				Namespace: tenant.Name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: tenant.APIVersion,
						Kind:       tenant.Kind,
						Name:       tenant.Name,
						UID:        tenant.UID,
					},
				},
			},
			Type: corev1.SecretType("kcp/kubeconfig"),
			Data: map[string][]byte{
				"controller-manager.conf": adminConfig,
			},
		}
		if _, err := c.ClientSet.CoreV1().Secrets(obj.Namespace).Create(ctx, obj, metav1.CreateOptions{}); err != nil {
			klog.Error(err, "unable to create secret")
			return err
		}
	}

	return nil
}

func (c *TenantController) reconcileAPIServer(ctx context.Context, tenant *v1alpha1.Tenant) error {
	generateDeploy := true
	if _, err := c.ClientSet.AppsV1().Deployments(tenant.Name).Get(ctx, "kube-apiserver", metav1.GetOptions{}); err == nil {
		// already exists
		klog.V(2).Info("deployment[kube-apiserver] already exists for tenant, skip")
		generateDeploy = false
	} else if !apierrors.IsNotFound(err) {
		klog.ErrorS(err, "unable to get deployment for apiserver")
		return err
	}

	generateSvc := true
	if _, err := c.ClientSet.CoreV1().Services(tenant.Name).Get(ctx, "kube-apiserver", metav1.GetOptions{}); err == nil {
		// already exists
		klog.V(2).Info("service[kube-apiserver] already exists for tenant, skip")
		generateSvc = false
	} else if !apierrors.IsNotFound(err) {
		klog.ErrorS(err, "unable to get service for apiserver")
		return err
	}

	// apiserver deployment
	if generateDeploy {
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kube-apiserver",
				Namespace: tenant.Name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: tenant.APIVersion,
						Kind:       tenant.Kind,
						Name:       tenant.Name,
						UID:        tenant.UID,
					},
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: pointer.Int32(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app":    "kube-apiserver",
						"tenant": tenant.Name,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app":    "kube-apiserver",
							"tenant": tenant.Name,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:            "apiserver",
								Image:           "k8s.gcr.io/kube-apiserver:v1.23.4",
								ImagePullPolicy: corev1.PullIfNotPresent,
								Command: []string{
									"kube-apiserver",
									"--advertise-address=0.0.0.0",
									"--allow-privileged=true",
									"--authorization-mode=Node,RBAC",
									"--client-ca-file=/etc/kubernetes/pki/ca.crt",
									"--enable-admission-plugins=NodeRestriction",
									"--enable-bootstrap-token-auth=true",
									"--etcd-cafile=/etc/kubernetes/pki/etcd-ca.crt",
									"--etcd-certfile=/etc/kubernetes/pki/apiserver-etcd-client.crt",
									"--etcd-keyfile=/etc/kubernetes/pki/apiserver-etcd-client.key",
									"--etcd-servers=" + c.EtcdServers,
									"--etcd-prefix=/" + tenant.Name + "/registry",
									"--insecure-port=0",
									"--kubelet-client-certificate=/etc/kubernetes/pki/apiserver-kubelet-client.crt",
									"--kubelet-client-key=/etc/kubernetes/pki/apiserver-kubelet-client.key",
									"--kubelet-preferred-address-types=InternalIP,ExternalIP,Hostname",
									"--proxy-client-cert-file=/etc/kubernetes/pki/front-proxy-client.crt",
									"--proxy-client-key-file=/etc/kubernetes/pki/front-proxy-client.key",
									"--requestheader-allowed-names=front-proxy-client",
									"--requestheader-client-ca-file=/etc/kubernetes/pki/front-proxy-ca.crt",
									"--requestheader-extra-headers-prefix=X-Remote-Extra-",
									"--requestheader-group-headers=X-Remote-Group",
									"--requestheader-username-headers=X-Remote-User",
									"--secure-port=6443",
									"--service-account-issuer=https://kubernetes.default.svc.cluster.local",
									"--service-account-key-file=/etc/kubernetes/pki/sa.pub",
									"--service-account-signing-key-file=/etc/kubernetes/pki/sa.key",
									"--service-cluster-ip-range=10.101.0.0/16",
									"--tls-cert-file=/etc/kubernetes/pki/apiserver.crt",
									"--tls-private-key-file=/etc/kubernetes/pki/apiserver.key",
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "server-cert",
										MountPath: "/etc/kubernetes/pki",
										ReadOnly:  true,
									},
								},
								LivenessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Host:   "127.0.0.1",
											Path:   "/livez",
											Port:   intstr.FromInt(6443),
											Scheme: corev1.URISchemeHTTPS,
										},
									},
									InitialDelaySeconds: 10,
									PeriodSeconds:       10,
									TimeoutSeconds:      15,
									SuccessThreshold:    1,
									FailureThreshold:    8,
								},
								ReadinessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Host:   "127.0.0.1",
											Path:   "/readyz",
											Port:   intstr.FromInt(6443),
											Scheme: corev1.URISchemeHTTPS,
										},
									},
									PeriodSeconds:    1,
									TimeoutSeconds:   15,
									SuccessThreshold: 1,
									FailureThreshold: 3,
								},
								StartupProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Host:   "127.0.0.1",
											Path:   "/livez",
											Port:   intstr.FromInt(6443),
											Scheme: corev1.URISchemeHTTPS,
										},
									},
									InitialDelaySeconds: 10,
									PeriodSeconds:       10,
									TimeoutSeconds:      15,
									SuccessThreshold:    1,
									FailureThreshold:    24,
								},
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: "server-cert",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										SecretName: "server-cert",
									},
								},
							},
						},
					},
				},
			},
		}
		if _, err := c.ClientSet.AppsV1().Deployments(deployment.Namespace).Create(ctx, deployment, metav1.CreateOptions{}); err != nil {
			klog.ErrorS(err, "unable to create deployment for apiserver")
			return err
		}
	}

	// apiserver service
	if generateSvc {
		service := &corev1.Service{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kube-apiserver",
				Namespace: tenant.Name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: tenant.APIVersion,
						Kind:       tenant.Kind,
						Name:       tenant.Name,
						UID:        tenant.UID,
					},
				},
			},
			Spec: corev1.ServiceSpec{
				Selector: map[string]string{
					"app":    "kube-apiserver",
					"tenant": tenant.Name,
				},
				Ports: []corev1.ServicePort{
					{
						Name:       "https",
						Protocol:   corev1.ProtocolTCP,
						Port:       6443,
						TargetPort: intstr.FromInt(6443),
					},
				},
			},
		}
		if _, err := c.ClientSet.CoreV1().Services(service.Namespace).Create(ctx, service, metav1.CreateOptions{}); err != nil {
			klog.ErrorS(err, "unable to create service for apiserver")
			return err
		}
	}

	return nil
}

func (c *TenantController) reconcileControllerManager(ctx context.Context, tenant *v1alpha1.Tenant) error {
	generateDeploy := true
	if _, err := c.ClientSet.AppsV1().Deployments(tenant.Name).Get(ctx, "kube-controller-manager", metav1.GetOptions{}); err == nil {
		// already exists
		klog.V(2).Info("deployment[kube-controller-manager] already exists for tenant, skip")
		generateDeploy = false
	} else if !apierrors.IsNotFound(err) {
		klog.ErrorS(err, "unable to get deployment for controller-manager")
		return err
	}

	if generateDeploy {
		deployment := &appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "kube-controller-manager",
				Namespace: tenant.Name,
				OwnerReferences: []metav1.OwnerReference{
					{
						APIVersion: tenant.APIVersion,
						Kind:       tenant.Kind,
						Name:       tenant.Name,
						UID:        tenant.UID,
					},
				},
			},
			Spec: appsv1.DeploymentSpec{
				Replicas: pointer.Int32(1),
				Selector: &metav1.LabelSelector{
					MatchLabels: map[string]string{
						"app":    "kube-controller-manager",
						"tenant": tenant.Name,
					},
				},
				Template: corev1.PodTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"app":    "kube-controller-manager",
							"tenant": tenant.Name,
						},
					},
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name:            "controller-manager",
								Image:           "k8s.gcr.io/kube-controller-manager:v1.23.4",
								ImagePullPolicy: corev1.PullIfNotPresent,
								Command: []string{
									"kube-controller-manager",
									"--allocate-node-cidrs=true",
									"--authentication-kubeconfig=/etc/kubernetes/kubeconfig/controller-manager.conf",
									"--authorization-kubeconfig=/etc/kubernetes/kubeconfig/controller-manager.conf",
									"--bind-address=0.0.0.0",
									"--client-ca-file=/etc/kubernetes/pki/ca.crt",
									"--cluster-cidr=10.100.0.0/16",
									"--cluster-signing-cert-file=/etc/kubernetes/pki/ca.crt",
									"--cluster-signing-key-file=/etc/kubernetes/pki/ca.key",
									"--controllers=*,bootstrapsigner,tokencleaner",
									"--kubeconfig=/etc/kubernetes/kubeconfig/controller-manager.conf",
									"--leader-elect=true",
									"--node-cidr-mask-size=24",
									"--requestheader-client-ca-file=/etc/kubernetes/pki/front-proxy-ca.crt",
									"--root-ca-file=/etc/kubernetes/pki/ca.crt",
									"--service-account-private-key-file=/etc/kubernetes/pki/sa.key",
									"--use-service-account-credentials=true",
								},
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      "server-cert",
										MountPath: "/etc/kubernetes/pki",
										ReadOnly:  true,
									},
									{
										Name:      "kubeconfig",
										MountPath: "/etc/kubernetes/kubeconfig",
										ReadOnly:  true,
									},
								},
								LivenessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Host:   "127.0.0.1",
											Path:   "/livez",
											Port:   intstr.FromInt(6443),
											Scheme: corev1.URISchemeHTTPS,
										},
									},
									InitialDelaySeconds: 10,
									PeriodSeconds:       10,
									TimeoutSeconds:      15,
									SuccessThreshold:    1,
									FailureThreshold:    8,
								},
								ReadinessProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Host:   "127.0.0.1",
											Path:   "/readyz",
											Port:   intstr.FromInt(6443),
											Scheme: corev1.URISchemeHTTPS,
										},
									},
									PeriodSeconds:    1,
									TimeoutSeconds:   15,
									SuccessThreshold: 1,
									FailureThreshold: 3,
								},
								StartupProbe: &corev1.Probe{
									ProbeHandler: corev1.ProbeHandler{
										HTTPGet: &corev1.HTTPGetAction{
											Host:   "127.0.0.1",
											Path:   "/livez",
											Port:   intstr.FromInt(6443),
											Scheme: corev1.URISchemeHTTPS,
										},
									},
									InitialDelaySeconds: 10,
									PeriodSeconds:       10,
									TimeoutSeconds:      15,
									SuccessThreshold:    1,
									FailureThreshold:    24,
								},
							},
						},
						Volumes: []corev1.Volume{
							{
								Name: "server-cert",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										SecretName: "server-cert",
									},
								},
							},
							{
								Name: "kubeconfig",
								VolumeSource: corev1.VolumeSource{
									Secret: &corev1.SecretVolumeSource{
										SecretName: "kubeconfig-controller-manager",
									},
								},
							},
						},
					},
				},
			},
		}
		if _, err := c.ClientSet.AppsV1().Deployments(deployment.Namespace).Create(ctx, deployment, metav1.CreateOptions{}); err != nil {
			klog.ErrorS(err, "unable to create deployment for controller-manager")
			return err
		}
	}

	return nil
}
