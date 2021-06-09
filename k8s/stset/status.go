package stset

import (
	"context"

	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	"code.cloudfoundry.org/lager"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
)

type StatefulsetGetter interface {
	GetByLRP(ctx context.Context, lrp *eiriniv1.LRP) ([]appsv1.StatefulSet, error)
}

type StatusGetter struct {
	logger         lager.Logger
	getStatefulSet getStatefulSetFunc
}

func NewStatusGetter(logger lager.Logger, statefulsetGetter StatefulsetGetter) StatusGetter {
	return StatusGetter{
		logger:         logger,
		getStatefulSet: newGetStatefulSetFunc(statefulsetGetter),
	}
}

func (g StatusGetter) GetStatus(ctx context.Context, lrp *eiriniv1.LRP) (eiriniv1.LRPStatus, error) {
	statefulSet, err := g.getStatefulSet(ctx, lrp)
	if err != nil {
		return eiriniv1.LRPStatus{}, errors.Wrap(err, "failed to get statefulset for LRP")
	}

	return eiriniv1.LRPStatus{
		Replicas: statefulSet.Status.ReadyReplicas,
	}, nil
}
