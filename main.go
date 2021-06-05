/*
SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/gardener/gardenlogin-controller-manager/controllers"
	"github.com/gardener/gardenlogin-controller-manager/internal/util"
	"github.com/gardener/gardenlogin-controller-manager/webhooks"

	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	//+kubebuilder:scaffold:scheme
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(gardencorev1alpha1.AddToScheme(scheme))
	utilruntime.Must(gardencorev1beta1.AddToScheme(scheme))
}

func main() {
	var (
		metricsAddr          string
		enableLeaderElection bool
		probeAddr            string
		certDir              string
		configFile           string
	)

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.StringVar(&certDir, "cert-dir", "/tmp/k8s-webhook-server/serving-certs", "CertDir is the directory that contains the server key and certificate.")
	flag.StringVar(&configFile, "config-file", "/etc/gardenlogin-controller-manager/config.yaml", "The path to the configuration file.")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	cmConfig, err := util.ReadControllerManagerConfiguration(configFile)
	if err != nil {
		fmt.Printf("error reading config: %s", err.Error()) // Logger not yet set up
		os.Exit(1)
	}

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		MetricsBindAddress:     metricsAddr,
		Port:                   9443,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "40ca8637.gardener.cloud",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	config := mgr.GetConfig()

	kube, err := kubernetes.NewForConfig(config)
	if err != nil {
		setupLog.Error(err, "could not create kubernetes client")
		os.Exit(1)
	}

	recorder := CreateRecorder(kube)

	if err = (&controllers.ShootReconciler{
		ClientSet:                   controllers.NewClientSet(config, mgr.GetClient(), kube),
		Log:                         ctrl.Log.WithName("controllers").WithName("Terminal"),
		Scheme:                      mgr.GetScheme(),
		Recorder:                    recorder,
		Config:                      cmConfig,
		ReconcilerCountPerNamespace: map[string]int{},
	}).SetupWithManager(mgr, cmConfig.Controllers.Shoot); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Shoot")
		os.Exit(1)
	}

	//+kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	// Setup webhooks
	setupLog.Info("setting up webhook server")

	hookServer := &webhook.Server{
		CertDir: certDir,
	}
	if err := mgr.Add(hookServer); err != nil {
		setupLog.Error(err, "unable register webhook server with manager")
		os.Exit(1)
	}

	setupLog.Info("registering webhooks to the webhook server")
	hookServer.Register("/validate-configmap", &webhook.Admission{Handler: &webhooks.ConfigmapValidator{
		Log:    ctrl.Log.WithName("webhooks").WithName("ConfigmapValidation"),
		Config: cmConfig,
	}})

	setupLog.Info("starting manager")

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func CreateRecorder(kubeClient kubernetes.Interface) record.EventRecorder {
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(&v1.EventSinkImpl{Interface: v1.New(kubeClient.CoreV1().RESTClient()).Events("")})

	return eventBroadcaster.NewRecorder(scheme, corev1.EventSource{Component: "Shoot"})
}
