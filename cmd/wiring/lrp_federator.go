package wiring

import (
	eirinictrl "code.cloudfoundry.org/eirini-controller"
	"code.cloudfoundry.org/eirini-controller/k8s/reconciler"
	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	"code.cloudfoundry.org/lager"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

func LRPFederator(logger lager.Logger, manager manager.Manager, config eirinictrl.ControllerConfig) error {
	logger = logger.Session("lrp-federator")

	lrpFederator, err := createLRPFederator(logger, manager.GetClient(), config, manager.GetScheme(), manager.GetConfig())
	if err != nil {
		return errors.Wrap(err, "Failed to create LRP reconciler")
	}

	err = builder.
		ControllerManagedBy(manager).
		For(&eiriniv1.LRP{}).
		Complete(lrpFederator)

	return errors.Wrapf(err, "Failed to build LRP reconciler")
}

func createLRPFederator(
	logger lager.Logger,
	controllerClient client.Client,
	cfg eirinictrl.ControllerConfig,
	scheme *runtime.Scheme,
	kubeConfig *rest.Config,
) (*reconciler.LRPFederator, error) {

	logger = logger.Session("lrp-federator")
	dynamicClient, err := dynamic.NewForConfig(kubeConfig)
	if err != nil {
		return nil, err
	}
	return reconciler.NewLRPFederator(logger, controllerClient, dynamicClient), nil
}
