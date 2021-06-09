package stset_test

import (
	"code.cloudfoundry.org/eirini-controller/k8s/stset"
	"code.cloudfoundry.org/eirini-controller/k8s/stset/stsetfakes"
	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	"code.cloudfoundry.org/lager"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var _ = Describe("Update", func() {
	var (
		logger             lager.Logger
		statefulSetGetter  *stsetfakes.FakeStatefulSetByLRPGetter
		statefulSetUpdater *stsetfakes.FakeStatefulSetUpdater
		pdbUpdater         *stsetfakes.FakePodDisruptionBudgetUpdater

		updatedLRP *eiriniv1.LRP
		err        error
	)

	BeforeEach(func() {
		logger = lagertest.NewTestLogger("handler-test")

		statefulSetGetter = new(stsetfakes.FakeStatefulSetByLRPGetter)
		statefulSetUpdater = new(stsetfakes.FakeStatefulSetUpdater)
		pdbUpdater = new(stsetfakes.FakePodDisruptionBudgetUpdater)

		updatedLRP = &eiriniv1.LRP{
			Spec: eiriniv1.LRPSpec{
				GUID:      "guid_1234",
				Version:   "version_1234",
				AppName:   "baldur",
				SpaceName: "space-foo",
				Instances: 5,
				Image:     "new/image",
			},
		}

		replicas := int32(3)

		st := []appsv1.StatefulSet{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "baldur",
					Namespace: "the-namespace",
					Annotations: map[string]string{
						stset.AnnotationProcessGUID: "Baldur-guid",
					},
				},
				Spec: appsv1.StatefulSetSpec{
					Replicas: &replicas,
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{Name: "another-container", Image: "another/image"},
								{Name: stset.ApplicationContainerName, Image: "old/image"},
							},
						},
					},
				},
			},
		}

		statefulSetGetter.GetByLRPReturns(st, nil)
	})

	JustBeforeEach(func() {
		updater := stset.NewUpdater(logger, statefulSetGetter, statefulSetUpdater, pdbUpdater)
		err = updater.Update(ctx, updatedLRP)
	})

	It("succeeds", func() {
		Expect(err).NotTo(HaveOccurred())
	})

	It("updates the statefulset", func() {
		Expect(statefulSetUpdater.UpdateCallCount()).To(Equal(1))

		_, namespace, st := statefulSetUpdater.UpdateArgsForCall(0)
		Expect(namespace).To(Equal("the-namespace"))
		Expect(st.GetAnnotations()).NotTo(HaveKey("another"))
		Expect(*st.Spec.Replicas).To(Equal(int32(5)))
		Expect(st.Spec.Template.Spec.Containers[0].Image).To(Equal("another/image"))
		Expect(st.Spec.Template.Spec.Containers[1].Image).To(Equal("new/image"))
	})

	It("updates the pod disruption budget", func() {
		Expect(pdbUpdater.UpdateCallCount()).To(Equal(1))
		_, actualStatefulSet, actualLRP := pdbUpdater.UpdateArgsForCall(0)
		Expect(actualStatefulSet.Namespace).To(Equal("the-namespace"))
		Expect(actualStatefulSet.Name).To(Equal("baldur"))
		Expect(actualLRP).To(Equal(updatedLRP))
	})

	When("updating the pod disruption budget fails", func() {
		BeforeEach(func() {
			pdbUpdater.UpdateReturns(errors.New("update-error"))
		})

		It("returns an error", func() {
			Expect(err).To(MatchError(ContainSubstring("update-error")))
		})
	})

	When("the image is missing", func() {
		BeforeEach(func() {
			updatedLRP.Spec.Image = ""
		})

		It("succeeds", func() {
			Expect(err).NotTo(HaveOccurred())
		})

		It("doesn't reset the image", func() {
			Expect(statefulSetUpdater.UpdateCallCount()).To(Equal(1))

			_, _, st := statefulSetUpdater.UpdateArgsForCall(0)
			Expect(st.Spec.Template.Spec.Containers[1].Image).To(Equal("old/image"))
		})
	})

	When("update fails", func() {
		BeforeEach(func() {
			statefulSetUpdater.UpdateReturns(nil, errors.New("boom"))
		})

		It("should return a meaningful message", func() {
			Expect(err).To(MatchError(ContainSubstring("failed to update statefulset")))
		})
	})

	When("update fails because of a conflict", func() {
		BeforeEach(func() {
			statefulSetUpdater.UpdateReturnsOnCall(0, nil, k8serrors.NewConflict(schema.GroupResource{}, "foo", errors.New("boom")))
			statefulSetUpdater.UpdateReturnsOnCall(1, &appsv1.StatefulSet{}, nil)
		})

		It("should retry", func() {
			Expect(statefulSetUpdater.UpdateCallCount()).To(Equal(2))
		})
	})

	When("the app does not exist", func() {
		BeforeEach(func() {
			statefulSetGetter.GetByLRPReturns(nil, errors.New("sorry"))
		})

		It("should return an error", func() {
			Expect(err).To(MatchError(ContainSubstring("failed to list statefulsets")))
		})

		It("should not create the app", func() {
			Expect(statefulSetUpdater.UpdateCallCount()).To(Equal(0))
		})
	})
})
