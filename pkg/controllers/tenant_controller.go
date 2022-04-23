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
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/k8s-cloud-platform/multi-tenants/pkg/apis/tenancy/v1alpha1"
	"github.com/k8s-cloud-platform/multi-tenants/pkg/conditions"
	"github.com/k8s-cloud-platform/multi-tenants/pkg/patcher"
)

const (
	tenantFinalizer = "tenancy.kcp.io/tenants"
)

type TenantController struct {
	EtcdSecret  map[string][]byte
	EtcdServers string
	Client      client.Client
	Reader      client.Reader
	ClientSet   *kubernetes.Clientset
}

var _ reconcile.Reconciler = &TenantController{}

// SetupWithManager sets up the controller with the Manager.
func (c *TenantController) SetupWithManager(mgr ctrl.Manager, options controller.Options) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Tenant{}).
		WithOptions(options).
		Complete(c)
}

func (c *TenantController) Reconcile(ctx context.Context, req reconcile.Request) (_ reconcile.Result, reterr error) {
	klog.V(1).InfoS("reconcile for Tenant", "name", req.Name)

	tenant := &v1alpha1.Tenant{}
	if err := c.Client.Get(ctx, req.NamespacedName, tenant); err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, err
	}

	defer func() {
		c.reconcilePhase(tenant)
		runtimeObj := tenant.DeepCopy()
		_, err := patcher.Patch(ctx, c.Client, runtimeObj, func() error {
			runtimeObj.ObjectMeta.Finalizers = tenant.ObjectMeta.Finalizers
			runtimeObj.ObjectMeta.OwnerReferences = tenant.ObjectMeta.OwnerReferences
			runtimeObj.Status.Phase = tenant.Status.Phase
			runtimeObj.Status.Conditions = tenant.Status.Conditions
			return nil
		})
		if err != nil {
			klog.ErrorS(err, "unable to patch Tenant", "name", tenant.Name)
			reterr = kerrors.NewAggregate([]error{reterr, err})
		}
	}()

	// Add finalizer first if not exist to avoid the race condition between init and delete
	if !controllerutil.ContainsFinalizer(tenant, tenantFinalizer) {
		controllerutil.AddFinalizer(tenant, tenantFinalizer)
		return ctrl.Result{}, nil
	}

	if !tenant.ObjectMeta.DeletionTimestamp.IsZero() {
		return c.reconcileDelete(ctx, tenant)
	}
	return c.reconcileNormal(ctx, tenant)
}

func (c *TenantController) reconcileDelete(ctx context.Context, tenant *v1alpha1.Tenant) (reconcile.Result, error) {
	klog.V(1).InfoS("reconcile for Tenant delete", "name", tenant.Name)

	if !tenant.Status.IsPhase(v1alpha1.TenantPhaseTerminating) {
		// wait for phase to be terminating
		return reconcile.Result{}, nil
	}

	// secret、deployment、service delete by GC, OwnerReference
	controllerutil.RemoveFinalizer(tenant, tenantFinalizer)
	return reconcile.Result{}, nil
}

func (c *TenantController) reconcileNormal(ctx context.Context, tenant *v1alpha1.Tenant) (reconcile.Result, error) {
	klog.V(1).InfoS("reconcile for Tenant normal", "name", tenant.Name)

	// ensure namespace
	if _, err := c.ClientSet.CoreV1().Namespaces().Get(ctx, tenant.Name, metav1.GetOptions{}); err != nil {
		if !apierrors.IsNotFound(err) {
			klog.ErrorS(err, "unable to get namespace for tenant")
			return reconcile.Result{}, err
		}
		if _, err := c.ClientSet.CoreV1().Namespaces().Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: tenant.Name,
			}}, metav1.CreateOptions{}); err != nil {
			klog.ErrorS(err, "unable to create namespace for tenant")
			return reconcile.Result{}, err
		}
	}

	if !conditions.Has(tenant, v1alpha1.TenantConditionProvisioned) ||
		conditions.IsFalse(tenant, v1alpha1.TenantConditionProvisioned) {
		// handle for provisioning
		phases := map[string]func(context.Context, *v1alpha1.Tenant) error{
			"secret":            c.reconcileSecret,
			"kubeconfig":        c.reconcileKubeConfig,
			"apiserver":         c.reconcileAPIServer,
			"controllermanager": c.reconcileControllerManager,
		}

		for phase, fun := range phases {
			err := fun(ctx, tenant)
			if err != nil {
				klog.ErrorS(err, "unable to handle for phase", "phase", phase)
				conditions.MarkFalse(tenant, v1alpha1.TenantConditionProvisioned, phase+"Failed", "failed to handle "+phase)
				return reconcile.Result{}, err
			}
		}

		conditions.MarkTrue(tenant, v1alpha1.TenantConditionProvisioned, "Success", "Success to provision")
	}

	// check if ready
	checkDeploy := func(namespace, name string) (reconcile.Result, error) {
		deploy, err := c.ClientSet.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			klog.ErrorS(err, "unable to get deployment", "namespace", namespace, "name", name)
			return reconcile.Result{}, err
		}
		if deploy.Status.Replicas != deploy.Status.ReadyReplicas {
			klog.Warningf("deployment[%s] is not ready", name)
			return reconcile.Result{Requeue: true, RequeueAfter: 10 * time.Second}, nil
		}
		return reconcile.Result{}, nil
	}

	if result, err := checkDeploy(tenant.Name, "kube-apiserver"); err != nil {
		return reconcile.Result{}, err
	} else if result.Requeue {
		return result, nil
	}
	if result, err := checkDeploy(tenant.Name, "kube-controller-manager"); err != nil {
		return reconcile.Result{}, err
	} else if result.Requeue {
		return result, nil
	}

	conditions.MarkTrue(tenant, v1alpha1.TenantConditionReady, "Success", "Ready")
	return reconcile.Result{}, nil
}
