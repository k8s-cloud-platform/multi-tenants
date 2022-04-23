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

package app

import (
	"context"
	"flag"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/k8s-cloud-platform/multi-tenants/cmd/controller-manager/app/options"
	"github.com/k8s-cloud-platform/multi-tenants/pkg/apis/tenancy/v1alpha1"
	"github.com/k8s-cloud-platform/multi-tenants/pkg/controllers"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
}

// NewControllerManagerCommand creates a *cobra.Command object with default parameters
func NewControllerManagerCommand() *cobra.Command {
	opts := options.NewOptions()

	cmd := &cobra.Command{
		Use:  "controller-manager",
		Long: `KCP controller manager.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := opts.Log.ValidateAndApply(); err != nil {
				return err
			}

			cliflag.PrintFlags(cmd.Flags())
			buildInfo, ok := debug.ReadBuildInfo()
			if ok {
				klog.Infof("build info: \n%s", buildInfo)
			}

			if errs := opts.Validate(); len(errs) != 0 {
				return errs.ToAggregate()
			}

			return run(ctrl.SetupSignalHandler(), opts)
		},
	}

	fs := cmd.Flags()
	opts.AddFlags(fs)
	fs.AddGoFlagSet(flag.CommandLine)

	return cmd
}

func run(ctx context.Context, opts *options.Options) error {
	config, err := ctrl.GetConfig()
	if err != nil {
		klog.ErrorS(err, "unable to load kubeconfig")
		return err
	}

	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:                        scheme,
		LeaderElection:                opts.LeaderElection.LeaderElect,
		LeaderElectionReleaseOnCancel: true,
		LeaderElectionResourceLock:    opts.LeaderElection.ResourceLock,
		LeaderElectionNamespace:       opts.LeaderElection.ResourceNamespace,
		LeaderElectionID:              opts.LeaderElection.ResourceName,
		LeaseDuration:                 &opts.LeaderElection.LeaseDuration.Duration,
		RenewDeadline:                 &opts.LeaderElection.RenewDeadline.Duration,
		RetryPeriod:                   &opts.LeaderElection.RetryPeriod.Duration,
		//ClientDisableCacheFor: []client.Object{&corev1.Secret{}},
	})
	if err != nil {
		klog.ErrorS(err, "unable to start controller-manager")
		return err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.ErrorS(err, "unable to new clientset for kubeconfig")
		return err
	}

	namespace, name, err := cache.SplitMetaNamespaceKey(opts.EtcdSecret)
	if err != nil {
		klog.ErrorS(err, "unable to split etcd-secret")
		return err
	}
	if namespace == "" {
		namespace = "default"
	}
	etcdSecret, err := clientset.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		klog.ErrorS(err, "unable to get secret for etcd-secret")
		return err
	}

	if err = (&controllers.TenantController{
		EtcdSecret:  etcdSecret.Data,
		EtcdServers: opts.EtcdServers,
		Client:      mgr.GetClient(),
		Reader:      mgr.GetAPIReader(),
		ClientSet:   clientset,
	}).SetupWithManager(mgr, controller.Options{
		MaxConcurrentReconciles: opts.ConcurrencyTenantSync,
	}); err != nil {
		klog.ErrorS(err, "unable to create tenant controller")
		return err
	}

	if err := preStart(ctx, mgr, opts); err != nil {
		klog.ErrorS(err, "unable to handle pre start")
		return err
	}

	klog.Info("starting controller-manager")
	if err := mgr.Start(ctx); err != nil {
		klog.ErrorS(err, "unable to run controller-manager")
		return err
	}

	// never reach here
	return nil
}

func preStart(ctx context.Context, mgr manager.Manager, opts *options.Options) error {
	// handle for default tenants
	for _, tenant := range strings.Split(opts.DefaultTenants, ",") {
		klog.InfoS("init default tenant", "name", tenant)
		tenantObj := &v1alpha1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Name: tenant,
			},
		}
		if err := mgr.GetAPIReader().Get(ctx, client.ObjectKeyFromObject(tenantObj), tenantObj); err != nil {
			if !apierrors.IsNotFound(err) {
				return err
			}
		} else {
			klog.InfoS("tenant already exists", "name", tenant)
			continue
		}
		if err := mgr.GetClient().Create(ctx, tenantObj); err != nil {
			return err
		}
		klog.InfoS("success to create tenant", "name", tenant)
	}
	return nil
}
