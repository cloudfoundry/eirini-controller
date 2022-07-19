package prometheus

import (
	"context"
	"errors"

	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	prometheusapi "github.com/prometheus/client_golang/prometheus"
	"k8s.io/utils/clock"
)

const (
	LRPCreations             = "eirini_lrp_creations"
	LRPCreationsHelp         = "The total number of created lrps"
	LRPCreationDurations     = "eirini_lrp_creation_durations"
	LRPCreationDurationsHelp = "The duration of lrp creations"
)

//counterfeiter:generate . LRPDesirer

type LRPDesirer interface {
	Desire(ctx context.Context, lrp *eiriniv1.LRP) error
}

type LRPDesirerDecorator struct {
	LRPDesirer
	creations         prometheusapi.Counter
	creationDurations prometheusapi.Histogram
	clock             clock.PassiveClock
}

func NewLRPDesirerDecorator(
	desirer LRPDesirer,
	registry prometheusapi.Registerer,
	clck clock.PassiveClock,
) (*LRPDesirerDecorator, error) {
	creations, err := registerCounter(registry, LRPCreations, "The total number of created lrps")
	if err != nil {
		return nil, err
	}

	creationDurations, err := registerHistogram(registry, LRPCreationDurations, LRPCreationDurationsHelp)
	if err != nil {
		return nil, err
	}

	return &LRPDesirerDecorator{
		LRPDesirer:        desirer,
		creations:         creations,
		creationDurations: creationDurations,
		clock:             clck,
	}, nil
}

func (d *LRPDesirerDecorator) Desire(ctx context.Context, lrp *eiriniv1.LRP) error {
	start := d.clock.Now()

	err := d.LRPDesirer.Desire(ctx, lrp)
	if err == nil {
		d.creations.Inc()
		d.creationDurations.Observe(float64(d.clock.Since(start).Milliseconds()))
	}

	return err
}

func registerCounter(registry prometheusapi.Registerer, name, help string) (prometheusapi.Counter, error) {
	c := prometheusapi.NewCounter(prometheusapi.CounterOpts{
		Name: name,
		Help: help,
	})

	err := registry.Register(c)
	if err == nil {
		return c, nil
	}

	var are prometheusapi.AlreadyRegisteredError
	if errors.As(err, &are) {
		return are.ExistingCollector.(prometheusapi.Counter), nil //nolint:forcetypeassert
	}

	return nil, err
}

func registerHistogram(registry prometheusapi.Registerer, name, help string) (prometheusapi.Histogram, error) {
	h := prometheusapi.NewHistogram(prometheusapi.HistogramOpts{
		Name: name,
		Help: help,
	})

	err := registry.Register(h)
	if err == nil {
		return h, nil
	}

	var are prometheusapi.AlreadyRegisteredError
	if errors.As(err, &are) {
		return are.ExistingCollector.(prometheusapi.Histogram), nil //nolint:forcetypeassert
	}

	return nil, err
}
