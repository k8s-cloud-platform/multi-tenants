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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/k8s-cloud-platform/multi-tenants/pkg/apis/tenancy/v1alpha1"
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
		runtimeObject := tenant.DeepCopy()
		_, err := controllerutil.CreateOrPatch(ctx, c.Client, runtimeObject, func() error {
			runtimeObject.ObjectMeta.Finalizers = tenant.ObjectMeta.Finalizers
			runtimeObject.ObjectMeta.DeletionGracePeriodSeconds = tenant.ObjectMeta.DeletionGracePeriodSeconds
			runtimeObject.ObjectMeta.DeletionTimestamp = tenant.ObjectMeta.DeletionTimestamp
			return nil
		})
		if err != nil {
			klog.ErrorS(err, "unable to create or patch Tenant", "name", tenant.Name)
			reterr = kerrors.NewAggregate([]error{reterr, err})
		}
	}()

	// Add finalizer first if not exist to avoid the race condition between init and delete
	if !controllerutil.ContainsFinalizer(tenant, "tenancy.kcp.io/tenants") {
		controllerutil.AddFinalizer(tenant, "tenancy.kcp.io/tenants")
		return ctrl.Result{}, nil
	}

	if !tenant.ObjectMeta.DeletionTimestamp.IsZero() {
		return c.reconcileDelete(ctx, tenant)
	}

	return c.reconcileNormal(ctx, tenant)
}

func (c *TenantController) reconcileDelete(ctx context.Context, tenant *v1alpha1.Tenant) (reconcile.Result, error) {
	klog.V(1).InfoS("reconcile for Tenant delete", "name", tenant.Name)

	if tenant.Name == "default" {
		klog.Error("default tenant should not delete")
		tenant.ObjectMeta.DeletionTimestamp = nil
		tenant.ObjectMeta.DeletionGracePeriodSeconds = nil
		return reconcile.Result{}, nil
	}

	controllerutil.RemoveFinalizer(tenant, "tenancy.kcp.io/tenants")
	return reconcile.Result{}, nil
}

func (c *TenantController) reconcileNormal(ctx context.Context, tenant *v1alpha1.Tenant) (reconcile.Result, error) {
	klog.V(1).InfoS("reconcile for Tenant normal", "name", tenant.Name)
	return reconcile.Result{}, nil
}
