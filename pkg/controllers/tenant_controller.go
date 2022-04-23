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

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/k8s-cloud-platform/multi-tenants/pkg/apis/tenancy/v1alpha1"
	"github.com/k8s-cloud-platform/multi-tenants/pkg/patcher"
)

const (
	tenantFinalizer = "tenancy.kcp.io/tenants"
)

type TenantController struct {
	client.Client
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
		runtimeObj := &v1alpha1.Tenant{}
		_, err := patcher.Patch(ctx, c.Client, runtimeObj, func() error {
			runtimeObj.ObjectMeta.Finalizers = tenant.ObjectMeta.Finalizers
			return nil
		})
		if err != nil {
			klog.ErrorS(err, "unable to create or patch Tenant", "name", tenant.Name)
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

	controllerutil.RemoveFinalizer(tenant, tenantFinalizer)
	return reconcile.Result{}, nil
}

func (c *TenantController) reconcileNormal(ctx context.Context, tenant *v1alpha1.Tenant) (reconcile.Result, error) {
	klog.V(1).InfoS("reconcile for Tenant normal", "name", tenant.Name)

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
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}

func (c *TenantController) reconcileSecret(ctx context.Context, tenant *v1alpha1.Tenant) error {
	// check server cert
	exists, err := checkSecret(ctx, c.Client, tenant.Name, "server-cert")
	if err != nil {
		klog.Error(err, "unable to check secret object")
		return err
	}
	if exists {
		klog.Info("secret[server-cert] already exists, skip kubeconfig phase")
		return nil
	}

	// server ca
	// apiserver
	// apiserver-kubelet-client

	// front proxy ca
	// front-proxy-client

	return nil
}

func (c *TenantController) reconcileKubeConfig(ctx context.Context, tenant *v1alpha1.Tenant) error {
	// check admin kubeconfig
	exists, err := checkSecret(ctx, c.Client, tenant.Name, "kubeconfig-admin")
	if err != nil {
		klog.Error(err, "unable to check secret object")
		return err
	}
	if exists {
		klog.Info("secret[kubeconfig-admin] already exists, skip kubeconfig phase")
		return nil
	}

	// admin.conf

	// check controller-manager kubeconfig
	exists, err = checkSecret(ctx, c.Client, tenant.Name, "kubeconfig-controller-manager")
	if err != nil {
		klog.Error(err, "unable to check secret object")
		return err
	}
	if exists {
		klog.Info("secret[kubeconfig-controller-manager] already exists, skip kubeconfig phase")
		return nil
	}

	// controller-manager.conf

	return nil
}

func (c *TenantController) reconcileAPIServer(ctx context.Context, tenant *v1alpha1.Tenant) error {
	return nil
}

func (c *TenantController) reconcileControllerManager(ctx context.Context, tenant *v1alpha1.Tenant) error {
	return nil
}

// checkSecret checks if secret exists
// returns err if error happens, and true if exists, false if not exists
func checkSecret(ctx context.Context, cli client.Client, namespace, name string) (bool, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	if err := cli.Get(ctx, client.ObjectKeyFromObject(secret), secret); err != nil {
		if !apierrors.IsNotFound(err) {
			klog.Error(err, "unable to get secret object")
			return false, err
		}
	} else {
		// already exists
		klog.Info("secret already exists, skip secret phase")
		return true, nil
	}
	return false, nil
}
