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
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	cliflag "k8s.io/component-base/cli/flag"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller"

	"github.com/k8s-cloud-platform/multi-tenants/cmd/controller-manager/app/options"
	"github.com/k8s-cloud-platform/multi-tenants/pkg/apis/tenancy/v1alpha1"
	"github.com/k8s-cloud-platform/multi-tenants/pkg/controllers"
	// +kubebuilder:scaffold:imports
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	// +kubebuilder:scaffold:scheme
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

			ctx := ctrl.SetupSignalHandler()
			return run(ctx, opts)
		},
	}

	fs := cmd.Flags()
	opts.AddFlags(fs)
	fs.AddGoFlagSet(flag.CommandLine)

	return cmd
}

func run(ctx context.Context, opts *options.Options) error {
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:         scheme,
		LeaderElection: false,
	})
	if err != nil {
		klog.ErrorS(err, "unable to start controller-manager")
		os.Exit(1)
	}

	if err = (&controllers.TenantController{
		Client: mgr.GetClient(),
	}).SetupWithManager(mgr, controller.Options{
		MaxConcurrentReconciles: 1,
	}); err != nil {
		klog.ErrorS(err, "unable to create tenant controller")
		os.Exit(1)
	}

	// +kubebuilder:scaffold:builder

	klog.Info("starting controller-manager")
	if err := mgr.Start(ctx); err != nil {
		klog.ErrorS(err, "unable to run controller-manager")
		os.Exit(1)
	}

	// never reach here
	return nil
}
