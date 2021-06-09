package reconciler_test

import (
	"context"

	"code.cloudfoundry.org/eirini-controller/k8s/reconciler"
	"code.cloudfoundry.org/eirini-controller/k8s/reconciler/reconcilerfakes"
	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/pkg/errors"
	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var _ = Describe("reconciler.LRP", func() {
	var (
		logger            *lagertest.TestLogger
		lrpsCrClient      *reconcilerfakes.FakeLRPsCrClient
		workloadClient    *reconcilerfakes.FakeLRPWorkloadCLient
		statefulsetGetter *reconcilerfakes.FakeStatefulSetGetter
		lrpreconciler     *reconciler.LRP
		resultErr         error
	)

	BeforeEach(func() {
		lrpsCrClient = new(reconcilerfakes.FakeLRPsCrClient)
		workloadClient = new(reconcilerfakes.FakeLRPWorkloadCLient)
		statefulsetGetter = new(reconcilerfakes.FakeStatefulSetGetter)
		logger = lagertest.NewTestLogger("lrp-reconciler")
		lrpreconciler = reconciler.NewLRP(logger, lrpsCrClient, workloadClient, statefulsetGetter)

		lrpsCrClient.GetLRPReturns(&eiriniv1.LRP{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "some-lrp",
				Namespace: "some-ns",
			},
			Spec: eiriniv1.LRPSpec{
				GUID:        "the-lrp-guid",
				Version:     "the-lrp-version",
				Command:     []string{"ls", "-la"},
				Instances:   10,
				ProcessType: "web",
				AppName:     "the-app",
				AppGUID:     "the-app-guid",
				OrgName:     "the-org",
				OrgGUID:     "the-org-guid",
				SpaceName:   "the-space",
				SpaceGUID:   "the-space-guid",
				Image:       "eirini/dorini",
				Env: map[string]string{
					"FOO": "BAR",
				},
				Ports:     []int32{8080, 9090},
				MemoryMB:  1024,
				DiskMB:    512,
				CPUWeight: 128,
				Sidecars: []eiriniv1.Sidecar{
					{
						Name:     "hello-sidecar",
						Command:  []string{"sh", "-c", "echo hello"},
						MemoryMB: 8,
						Env: map[string]string{
							"SIDE": "BUS",
						},
					},
					{
						Name:     "bye-sidecar",
						Command:  []string{"sh", "-c", "echo bye"},
						MemoryMB: 16,
						Env: map[string]string{
							"SIDE": "CAR",
						},
					},
				},
				VolumeMounts: []eiriniv1.VolumeMount{
					{
						MountPath: "/path/to/mount",
						ClaimName: "claim-q1",
					},
					{
						MountPath: "/path/in/the/other/direction",
						ClaimName: "claim-c2",
					},
				},
				Health: eiriniv1.Healthcheck{
					Type:      "http",
					Port:      9090,
					Endpoint:  "/heath",
					TimeoutMs: 80,
				},
				UserDefinedAnnotations: map[string]string{
					"user-annotaions.io": "yes",
				},
			},
		}, nil)

		statefulsetGetter.GetReturns(nil, apierrors.NewNotFound(schema.GroupResource{}, "foo"))
	})

	JustBeforeEach(func() {
		_, resultErr = lrpreconciler.Reconcile(context.Background(), reconcile.Request{
			NamespacedName: types.NamespacedName{
				Namespace: "some-ns",
				Name:      "app",
			},
		})
	})

	It("creates a statefulset for each CRD", func() {
		Expect(resultErr).NotTo(HaveOccurred())

		Expect(workloadClient.UpdateCallCount()).To(Equal(0))
		Expect(workloadClient.DesireCallCount()).To(Equal(1))

		_, lrp := workloadClient.DesireArgsForCall(0)
		Expect(lrp.Namespace).To(Equal("some-ns"))
		Expect(lrp.Spec.GUID).To(Equal("the-lrp-guid"))
		Expect(lrp.Spec.Version).To(Equal("the-lrp-version"))
		Expect(lrp.Spec.Command).To(ConsistOf("ls", "-la"))
		Expect(lrp.Spec.Instances).To(Equal(10))
		Expect(lrp.Spec.PrivateRegistry).To(BeNil())
		Expect(lrp.Spec.ProcessType).To(Equal("web"))
		Expect(lrp.Spec.AppName).To(Equal("the-app"))
		Expect(lrp.Spec.AppGUID).To(Equal("the-app-guid"))
		Expect(lrp.Spec.OrgName).To(Equal("the-org"))
		Expect(lrp.Spec.OrgGUID).To(Equal("the-org-guid"))
		Expect(lrp.Spec.SpaceName).To(Equal("the-space"))
		Expect(lrp.Spec.SpaceGUID).To(Equal("the-space-guid"))
		Expect(lrp.Spec.Image).To(Equal("eirini/dorini"))
		Expect(lrp.Spec.Env).To(Equal(map[string]string{
			"FOO": "BAR",
		}))
		Expect(lrp.Spec.Ports).To(Equal([]int32{8080, 9090}))
		Expect(lrp.Spec.MemoryMB).To(Equal(int64(1024)))
		Expect(lrp.Spec.DiskMB).To(Equal(int64(512)))
		Expect(lrp.Spec.CPUWeight).To(Equal(uint8(128)))
		Expect(lrp.Spec.Sidecars).To(Equal([]eiriniv1.Sidecar{
			{
				Name:     "hello-sidecar",
				Command:  []string{"sh", "-c", "echo hello"},
				MemoryMB: 8,
				Env: map[string]string{
					"SIDE": "BUS",
				},
			},
			{
				Name:     "bye-sidecar",
				Command:  []string{"sh", "-c", "echo bye"},
				MemoryMB: 16,
				Env: map[string]string{
					"SIDE": "CAR",
				},
			},
		}))
		Expect(lrp.Spec.VolumeMounts).To(Equal([]eiriniv1.VolumeMount{
			{
				MountPath: "/path/to/mount",
				ClaimName: "claim-q1",
			},
			{
				MountPath: "/path/in/the/other/direction",
				ClaimName: "claim-c2",
			},
		}))
		Expect(lrp.Spec.Health).To(Equal(eiriniv1.Healthcheck{
			Type:      "http",
			Port:      9090,
			Endpoint:  "/heath",
			TimeoutMs: 80,
		}))
		Expect(lrp.Spec.UserDefinedAnnotations).To(Equal(map[string]string{
			"user-annotaions.io": "yes",
		}))
	})

	It("does not update the LRP CR", func() {
		Expect(lrpsCrClient.UpdateLRPStatusCallCount()).To(BeZero())
	})

	When("the statefulset for the LRP already exists", func() {
		BeforeEach(func() {
			statefulsetGetter.GetReturns(&appsv1.StatefulSet{}, nil)

			workloadClient.GetStatusReturns(eiriniv1.LRPStatus{
				Replicas: 9,
			}, nil)
		})

		It("updates the CR status accordingly", func() {
			Expect(resultErr).NotTo(HaveOccurred())

			Expect(lrpsCrClient.UpdateLRPStatusCallCount()).To(Equal(1))
			_, actualLrp, actualLrpStatus := lrpsCrClient.UpdateLRPStatusArgsForCall(0)
			Expect(actualLrp.Name).To(Equal("some-lrp"))
			Expect(actualLrpStatus.Replicas).To(Equal(int32(9)))
		})

		When("gettting the LRP status fails", func() {
			BeforeEach(func() {
				workloadClient.GetStatusReturns(eiriniv1.LRPStatus{}, errors.New("boom"))
			})

			It("does not update the statefulset status", func() {
				Expect(resultErr).To(MatchError(ContainSubstring("boom")))
				Expect(workloadClient.UpdateCallCount()).To(Equal(1))
				Expect(lrpsCrClient.UpdateLRPStatusCallCount()).To(BeZero())
			})
		})
	})

	When("private registry credentials are specified in the LRP CRD", func() {
		BeforeEach(func() {
			lrpsCrClient.GetLRPReturns(&eiriniv1.LRP{
				Spec: eiriniv1.LRPSpec{
					Image: "private-registry.com:5000/repo/app-image:latest",
					PrivateRegistry: &eiriniv1.PrivateRegistry{
						Username: "docker-user",
						Password: "docker-password",
					},
				},
			}, nil)
		})

		It("configures a private registry", func() {
			_, lrp := workloadClient.DesireArgsForCall(0)
			privateRegistry := lrp.Spec.PrivateRegistry
			Expect(privateRegistry).NotTo(BeNil())
			Expect(privateRegistry.Username).To(Equal("docker-user"))
			Expect(privateRegistry.Password).To(Equal("docker-password"))
		})
	})

	When("the LRP doesn't exist", func() {
		BeforeEach(func() {
			lrpsCrClient.GetLRPReturns(nil, apierrors.NewNotFound(schema.GroupResource{}, "my-lrp"))
		})

		It("does not return an error", func() {
			Expect(resultErr).NotTo(HaveOccurred())
		})
	})

	When("the controller client fails to get the CRD", func() {
		BeforeEach(func() {
			lrpsCrClient.GetLRPReturns(nil, errors.New("boom"))
		})

		It("returns an error", func() {
			Expect(resultErr).To(MatchError(ContainSubstring("boom")))
		})
	})

	When("the getting the statefulset fails", func() {
		BeforeEach(func() {
			statefulsetGetter.GetReturns(nil, errors.New("boom"))
		})

		It("returns an error", func() {
			Expect(resultErr).To(MatchError("failed to get statefulSet: boom"))
		})
	})

	When("the workload client fails to desire the app", func() {
		BeforeEach(func() {
			workloadClient.DesireReturns(errors.New("boom"))
		})

		It("returns an error", func() {
			Expect(resultErr).To(MatchError("failed to desire lrp: boom"))
		})
	})

	When("the workload client fails to update the app", func() {
		BeforeEach(func() {
			statefulsetGetter.GetReturns(&appsv1.StatefulSet{}, nil)
			workloadClient.UpdateReturns(errors.New("boom"))
		})

		It("returns an error", func() {
			Expect(resultErr).To(MatchError(ContainSubstring("boom")))
		})
	})
})
