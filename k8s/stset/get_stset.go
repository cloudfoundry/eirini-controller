package stset

import (
	"context"
	"fmt"

	eirinictrl "code.cloudfoundry.org/eirini-controller"
	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
)

//counterfeiter:generate . StatefulSetByLRPGetter

type StatefulSetByLRPGetter interface {
	GetByLRP(ctx context.Context, lrp *eiriniv1.LRP) ([]appsv1.StatefulSet, error)
}

type getStatefulSetFunc func(ctx context.Context, lrp *eiriniv1.LRP) (*appsv1.StatefulSet, error)

func newGetStatefulSetFunc(stSetGetter StatefulSetByLRPGetter) getStatefulSetFunc {
	return func(ctx context.Context, lrp *eiriniv1.LRP) (*appsv1.StatefulSet, error) {
		statefulSets, err := stSetGetter.GetByLRP(ctx, lrp)
		if err != nil {
			return nil, errors.Wrap(err, "failed to list statefulsets")
		}

		switch len(statefulSets) {
		case 0:
			return nil, eirinictrl.ErrNotFound
		case 1:
			return &statefulSets[0], nil
		default:
			return nil, fmt.Errorf("multiple statefulsets found for LRP {%s}%s", lrp.Namespace, lrp.Name)
		}
	}
}
