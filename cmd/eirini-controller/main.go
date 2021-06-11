package main

import (
	"context"
	"fmt"
	"os"

	eirinictrl "code.cloudfoundry.org/eirini-controller"
	cmdcommons "code.cloudfoundry.org/eirini-controller/cmd"
	"code.cloudfoundry.org/eirini-controller/k8s"
	eirinievent "code.cloudfoundry.org/eirini-controller/k8s/event"
	"code.cloudfoundry.org/eirini-controller/k8s/jobs"
	"code.cloudfoundry.org/eirini-controller/k8s/pdb"
	"code.cloudfoundry.org/eirini-controller/k8s/reconciler"
	"code.cloudfoundry.org/eirini-controller/k8s/stset"
	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	eirinischeme "code.cloudfoundry.org/eirini-controller/pkg/generated/clientset/versioned/scheme"
	"code.cloudfoundry.org/eirini-controller/prometheus"
	"code.cloudfoundry.org/eirini-controller/util"
	"code.cloudfoundry.org/lager"
	"github.com/jessevdk/go-flags"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/clock"
	kscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/clientcmd"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type options struct {
	ConfigFile string `short:"c" long:"config" description:"Config for running eirini-controller"`
}

func main() {
	if err := kscheme.AddToScheme(eirinischeme.Scheme); err != nil {
		cmdcommons.Exitf("failed to add the k8s scheme to the LRP CRD scheme: %v", err)
	}

	var opts options
	_, err := flags.ParseArgs(&opts, os.Args)
	cmdcommons.ExitfIfError(err, "Failed to parse args")

	var cfg eirinictrl.ControllerConfig
	err = cmdcommons.ReadConfigFile(opts.ConfigFile, &cfg)
	cmdcommons.ExitfIfError(err, "Failed to read config file")

	kubeConfig, err := clientcmd.BuildConfigFromFlags("", cfg.ConfigPath)
	cmdcommons.ExitfIfError(err, "Failed to build kubeconfig")

	cmdcommons.ExitfIfError(err, "Failed to create k8s runtime client")

	logger := lager.NewLogger("eirini-controller")
	logger.RegisterSink(lager.NewPrettySink(os.Stdout, lager.DEBUG))

	managerOptions := manager.Options{
		MetricsBindAddress: "0",
		Scheme:             eirinischeme.Scheme,
		Namespace:          cfg.WorkloadsNamespace,
		Logger:             util.NewLagerLogr(logger),
		LeaderElection:     true,
		LeaderElectionID:   "eirini-controller-leader",
	}

	if cfg.PrometheusPort > 0 {
		managerOptions.MetricsBindAddress = fmt.Sprintf(":%d", cfg.PrometheusPort)
	}

	if cfg.LeaderElectionID != "" {
		managerOptions.LeaderElectionNamespace = cfg.LeaderElectionNamespace
		managerOptions.LeaderElectionID = cfg.LeaderElectionID
	}

	mgr, err := manager.New(kubeConfig, managerOptions)
	cmdcommons.ExitfIfError(err, "Failed to create k8s controller runtime manager")

	lrpReconciler, err := createLRPReconciler(logger, mgr.GetClient(), cfg, mgr.GetScheme())
	cmdcommons.ExitfIfError(err, "Failed to create LRP reconciler")

	taskReconciler := createTaskReconciler(logger, mgr.GetClient(), cfg, mgr.GetScheme())
	podCrashReconciler := createPodCrashReconciler(logger, cfg.WorkloadsNamespace, mgr.GetClient())

	err = builder.
		ControllerManagedBy(mgr).
		For(&eiriniv1.LRP{}).
		Owns(&appsv1.StatefulSet{}).
		Complete(lrpReconciler)
	cmdcommons.ExitfIfError(err, "Failed to build LRP reconciler")

	err = builder.
		ControllerManagedBy(mgr).
		For(&eiriniv1.Task{}).
		Owns(&batchv1.Job{}).
		Complete(taskReconciler)
	cmdcommons.ExitfIfError(err, "Failed to build Task reconciler")

	err = mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Event{}, reconciler.IndexEventInvolvedObjectName, getEventInvolvedObjectName())
	cmdcommons.ExitfIfError(err, fmt.Sprintf("Failed to create index %q", reconciler.IndexEventInvolvedObjectName))

	err = mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Event{}, reconciler.IndexEventInvolvedObjectKind, getEventInvolvedObjectKind())
	cmdcommons.ExitfIfError(err, fmt.Sprintf("Failed to create index %q", reconciler.IndexEventInvolvedObjectKind))

	err = mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Event{}, reconciler.IndexEventReason, getEventReason())
	cmdcommons.ExitfIfError(err, fmt.Sprintf("Failed to create index %q", reconciler.IndexEventReason))

	predicates := []predicate.Predicate{reconciler.NewSourceTypeUpdatePredicate(stset.AppSourceType)}
	err = builder.
		ControllerManagedBy(mgr).
		For(&corev1.Pod{}, builder.WithPredicates(predicates...)).
		Complete(podCrashReconciler)
	cmdcommons.ExitfIfError(err, "Failed to build Pod Crash reconciler")

	err = mgr.Start(ctrl.SetupSignalHandler())
	cmdcommons.ExitfIfError(err, "Failed to start manager")
}

func getEventInvolvedObjectName() client.IndexerFunc {
	return createEventOwnerIndexerFunc(func(event *corev1.Event) string {
		return event.InvolvedObject.Name
	})
}

func getEventInvolvedObjectKind() client.IndexerFunc {
	return createEventOwnerIndexerFunc(func(event *corev1.Event) string {
		return event.InvolvedObject.Kind
	})
}

func getEventReason() client.IndexerFunc {
	return createEventOwnerIndexerFunc(func(event *corev1.Event) string {
		return event.Reason
	})
}

func createEventOwnerIndexerFunc(getEventAttribute func(*corev1.Event) string) client.IndexerFunc {
	return func(rawObj client.Object) []string {
		event, _ := rawObj.(*corev1.Event)

		return []string{getEventAttribute(event)}
	}
}

func createLRPReconciler(
	logger lager.Logger,
	controllerClient client.Client,
	cfg eirinictrl.ControllerConfig,
	scheme *runtime.Scheme,
) (*reconciler.LRP, error) {
	logger = logger.Session("lrp-reconciler")
	lrpToStatefulSetConverter := stset.NewLRPToStatefulSetConverter(
		cfg.ApplicationServiceAccount,
		cfg.RegistrySecretName,
		cfg.UnsafeAllowAutomountServiceAccountToken,
		cfg.AllowRunImageAsRoot,
		k8s.CreateLivenessProbe,
		k8s.CreateReadinessProbe,
	)

	pdbUpdater := pdb.NewUpdater(controllerClient)
	desirer := stset.NewDesirer(logger, lrpToStatefulSetConverter, pdbUpdater, controllerClient, scheme)
	updater := stset.NewUpdater(logger, controllerClient, pdbUpdater)

	decoratedDesirer, err := prometheus.NewLRPDesirerDecorator(desirer, metrics.Registry, clock.RealClock{})
	if err != nil {
		return nil, err
	}

	return reconciler.NewLRP(
		logger,
		controllerClient,
		decoratedDesirer,
		updater,
	), nil
}

func createTaskReconciler(
	logger lager.Logger,
	controllerClient client.Client,
	cfg eirinictrl.ControllerConfig,
	scheme *runtime.Scheme,
) *reconciler.Task {
	taskToJobConverter := jobs.NewTaskToJobConverter(
		cfg.ApplicationServiceAccount,
		cfg.RegistrySecretName,
		cfg.UnsafeAllowAutomountServiceAccountToken,
	)

	desirer := jobs.NewDesirer(logger, taskToJobConverter, controllerClient, scheme)
	statusGetter := jobs.NewStatusGetter(logger)

	return reconciler.NewTask(logger, controllerClient, desirer, statusGetter, cfg.TaskTTLSeconds)
}

func createPodCrashReconciler(logger lager.Logger, workloadsNamespace string, controllerClient client.Client) *reconciler.PodCrash {
	crashEventGenerator := eirinievent.NewDefaultCrashEventGenerator(controllerClient)

	return reconciler.NewPodCrash(logger, controllerClient, crashEventGenerator)
}
