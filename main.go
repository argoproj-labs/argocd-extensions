package main

import (
	"flag"
	"os"

	"sigs.k8s.io/controller-runtime/pkg/client/config"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	extensionv1 "github.com/argoproj/argocd-extensions/api/v1"
	"github.com/argoproj/argocd-extensions/controllers"
	//+kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(extensionv1.AddToScheme(scheme))
	//+kubebuilder:scaffold:scheme
}

func main() {
	var enableLeaderElection bool
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if explicitPath := flag.Lookup("kubeconfig").Value.String(); explicitPath != "" {
		loadingRules = &clientcmd.ClientConfigLoadingRules{ExplicitPath: explicitPath}
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, &clientcmd.ConfigOverrides{})
	namespace, _, err := clientConfig.Namespace()
	if err != nil {
		setupLog.Error(err, "unable to get namespace")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(config.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Port:                   9443,
		HealthProbeBindAddress: "0",
		LeaderElection:         enableLeaderElection,
		MetricsBindAddress:     "0",
		LeaderElectionID:       "632aad60.argoproj.io",
		Namespace:              namespace,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.ArgoCDExtensionReconciler{
		Client:         mgr.GetClient(),
		Scheme:         mgr.GetScheme(),
		ExtensionsPath: "/tmp/extensions",
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "ArgoCDExtension")
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

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
