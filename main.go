/*
  This file is part of Astarte.

  Copyright 2020-23 SECO Mind Srl

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

package main

import (
	"flag"
	"os"
	"time"

	"github.com/go-logr/logr"
	zaplogfmt "github.com/sykesm/zap-logfmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	zapcr "sigs.k8s.io/controller-runtime/pkg/log/zap"

	apiv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha1"
	apiv1alpha2 "github.com/astarte-platform/astarte-kubernetes-operator/apis/api/v1alpha2"
	ingressv1alpha1 "github.com/astarte-platform/astarte-kubernetes-operator/apis/ingress/v1alpha1"
	controllersapi "github.com/astarte-platform/astarte-kubernetes-operator/controllers/api"
	ingresscontrollers "github.com/astarte-platform/astarte-kubernetes-operator/controllers/ingress"
	voyagercrd "github.com/astarte-platform/astarte-kubernetes-operator/external/voyager/v1beta1"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	// Setup Scheme for other CRDs and the CRD themselves
	utilruntime.Must(voyagercrd.AddToScheme(scheme))
	utilruntime.Must(apiextensionsv1.AddToScheme(scheme))

	utilruntime.Must(apiv1alpha1.AddToScheme(scheme))
	utilruntime.Must(apiv1alpha2.AddToScheme(scheme))
	utilruntime.Must(ingressv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	flag.StringVar(&metricsAddr, "metrics-addr", ":8080", "The address the metric endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "enable-leader-election", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	flag.Parse()

	// setup logger
	log := setupLogger()

	// Set the controller logger to log, which will be propagated through the whole operator, generating
	// uniform and structured logs.
	logf.SetLogger(log)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:             scheme,
		MetricsBindAddress: metricsAddr,
		Port:               9443,
		LeaderElection:     enableLeaderElection,
		LeaderElectionID:   "30c4295b.astarte-platform.org",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllersapi.AstarteReconciler{
		Client:   mgr.GetClient(),
		Log:      ctrl.Log.WithName("controllers").WithName("Astarte"),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("astarte-controller"),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Astarte")
		os.Exit(1)
	}
	if err = (&controllersapi.AstarteVoyagerIngressReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("AstarteVoyagerIngress"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AstarteVoyagerIngress")
		os.Exit(1)
	}
	if err = (&controllersapi.FlowReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("Flow"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Flow")
		os.Exit(1)
	}
	if err = (&ingresscontrollers.AstarteDefaultIngressReconciler{
		Client: mgr.GetClient(),
		Log:    ctrl.Log.WithName("controllers").WithName("ingress").WithName("AstarteDefaultIngress"),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "AstarteDefaultIngress")
		os.Exit(1)
	}
	// When testing the operator locally, we may want not to run webhooks. Thus, put them behind
	// an environment variable
	if os.Getenv("ENABLE_WEBHOOKS") != "false" {
		if err = (&apiv1alpha2.Astarte{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Astarte")
			os.Exit(1)
		}
		if err = (&apiv1alpha2.AstarteVoyagerIngress{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "AstarteVoyagerIngress")
			os.Exit(1)
		}
		if err = (&apiv1alpha2.Flow{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "Flow")
			os.Exit(1)
		}
		if err = (&ingressv1alpha1.AstarteDefaultIngress{}).SetupWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "AstarteDefaultIngress")
			os.Exit(1)
		}
	}
	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func setupLogger() logr.Logger {
	configLog := zap.NewProductionEncoderConfig()

	configLog.EncodeTime = func(ts time.Time, encoder zapcore.PrimitiveArrayEncoder) {
		encoder.AppendString(ts.UTC().Format(time.RFC3339))
	}

	logfmtEncoder := zaplogfmt.NewEncoder(configLog)

	// Construct a new logr.logger.
	return zapcr.New(zapcr.UseDevMode(true), zapcr.WriteTo(os.Stdout), zapcr.Encoder(logfmtEncoder))
}
