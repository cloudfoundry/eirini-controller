package integration_test

import (
	"context"
	"fmt"

	"code.cloudfoundry.org/eirini-controller/k8s"
	"code.cloudfoundry.org/eirini-controller/k8s/client"
	"code.cloudfoundry.org/eirini-controller/k8s/pdb"
	"code.cloudfoundry.org/eirini-controller/k8s/stset"
	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	eirinischeme "code.cloudfoundry.org/eirini-controller/pkg/generated/clientset/versioned/scheme"
	"code.cloudfoundry.org/eirini-controller/tests"
	"code.cloudfoundry.org/lager/lagertest"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("LRPClient", func() {
	var (
		allowRunImageAsRoot bool
		lrpClient           *k8s.LRPClient
		lrp                 *eiriniv1.LRP
	)

	BeforeEach(func() {
		allowRunImageAsRoot = false
		lrp = createLRP(fixture.Namespace, "odin")
	})

	AfterEach(func() {
		cleanupStatefulSet(lrp)
		Eventually(func() []appsv1.StatefulSet {
			return listStatefulSets(lrp)
		}).Should(BeEmpty())
	})

	JustBeforeEach(func() {
		lrpClient = createLrpClient(fixture.Namespace, allowRunImageAsRoot)
	})

	Describe("Desire", func() {
		var desireErr error

		JustBeforeEach(func() {
			desireErr = lrpClient.Desire(ctx, lrp)
		})

		It("succeeds", func() {
			Expect(desireErr).NotTo(HaveOccurred())
		})

		// join all tests in a single with By()
		It("should create a StatefulSet object", func() {
			statefulset := getStatefulSetForLRP(lrp)
			Expect(statefulset.Name).To(ContainSubstring("odin-space-foo"))
			Expect(statefulset.Namespace).To(Equal(fixture.Namespace))
			Expect(statefulset.Spec.Template.Spec.ImagePullSecrets).To(ConsistOf(corev1.LocalObjectReference{Name: "registry-secret"}))
			Expect(statefulset.Labels).To(SatisfyAll(
				HaveKeyWithValue(stset.LabelGUID, lrp.Spec.GUID),
				HaveKeyWithValue(stset.LabelVersion, lrp.Spec.Version),
				HaveKeyWithValue(stset.LabelSourceType, "APP"),
				HaveKeyWithValue(stset.LabelAppGUID, "the-app-guid"),
			))

			Expect(statefulset.Spec.Replicas).To(Equal(int32ptr(lrp.Spec.Instances)))
			Expect(statefulset.Spec.Template.Spec.SecurityContext.RunAsNonRoot).To(PointTo(BeTrue()))
			Expect(statefulset.Spec.Template.Spec.Containers[0].Command).To(Equal(lrp.Spec.Command))
			Expect(statefulset.Spec.Template.Spec.Containers[0].Image).To(Equal(lrp.Spec.Image))
			Expect(statefulset.Spec.Template.Spec.Containers[0].Env).To(ContainElement(corev1.EnvVar{Name: "FOO", Value: "BAR"}))
		})

		It("should create all associated pods", func() {
			var podNames []string

			Eventually(func() []string {
				podNames = podNamesFromPods(listPods(lrp))

				return podNames
			}).Should(HaveLen(lrp.Spec.Instances))

			for i := 0; i < lrp.Spec.Instances; i++ {
				podIndex := i
				Expect(podNames[podIndex]).To(ContainSubstring("odin-space-foo"))

				Eventually(func() string {
					return getPodPhase(podIndex, lrp)
				}).Should(Equal("Ready"))
			}

			Eventually(func() int32 {
				return getStatefulSetForLRP(lrp).Status.ReadyReplicas
			}).Should(Equal(int32(2)))
		})

		It("should create a pod disruption budget for the lrp", func() {
			statefulset := getStatefulSetForLRP(lrp)
			pdb, err := podDisruptionBudgets().Get(context.Background(), statefulset.Name, metav1.GetOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(pdb).NotTo(BeNil())
			Expect(pdb.Spec.MinAvailable).To(PointTo(Equal(intstr.FromString("50%"))))
			Expect(pdb.Spec.MaxUnavailable).To(BeNil())
		})

		When("the lrp has 1 instance", func() {
			BeforeEach(func() {
				lrp.Spec.Instances = 1
			})

			It("should not create a pod disruption budget for the lrp", func() {
				_, err := podDisruptionBudgets().Get(context.Background(), "ödin", metav1.GetOptions{})
				Expect(err).To(MatchError(ContainSubstring("not found")))
			})
		})

		When("additional app info is provided", func() {
			BeforeEach(func() {
				lrp.Spec.OrgName = "odin-org"
				lrp.Spec.OrgGUID = "odin-org-guid"
				lrp.Spec.SpaceName = "odin-space"
				lrp.Spec.SpaceGUID = "odin-space-guid"
			})

			DescribeTable("sets appropriate annotations to statefulset", func(key, value string) {
				statefulset := getStatefulSetForLRP(lrp)
				Expect(statefulset.Annotations).To(HaveKeyWithValue(key, value))
			},
				Entry("SpaceName", stset.AnnotationSpaceName, "odin-space"),
				Entry("SpaceGUID", stset.AnnotationSpaceGUID, "odin-space-guid"),
				Entry("OrgName", stset.AnnotationOrgName, "odin-org"),
				Entry("OrgGUID", stset.AnnotationOrgGUID, "odin-org-guid"),
			)

			It("sets appropriate labels to statefulset", func() {
				statefulset := getStatefulSetForLRP(lrp)
				Expect(statefulset.Labels).To(HaveKeyWithValue(stset.LabelGUID, lrp.Spec.GUID))
				Expect(statefulset.Labels).To(HaveKeyWithValue(stset.LabelVersion, lrp.Spec.Version))
				Expect(statefulset.Labels).To(HaveKeyWithValue(stset.LabelSourceType, "APP"))
			})
		})

		When("the app has more than one instances", func() {
			BeforeEach(func() {
				lrp.Spec.Instances = 2
			})

			It("should schedule app pods on different nodes", func() {
				if getNodeCount() == 1 {
					Skip("target cluster has only one node")
				}

				Eventually(func() []corev1.Pod {
					return listPods(lrp)
				}).Should(HaveLen(2))

				var nodeNames []string
				Eventually(func() []string {
					nodeNames = nodeNamesFromPods(listPods(lrp))

					return nodeNames
				}).Should(HaveLen(2))
				Expect(nodeNames[0]).ToNot(Equal(nodeNames[1]))
			})
		})

		When("private docker registry credentials are provided", func() {
			BeforeEach(func() {
				lrp.Spec.Image = "eiriniuser/notdora:latest"
				lrp.Spec.Command = nil
				lrp.Spec.PrivateRegistry = &eiriniv1.PrivateRegistry{
					Username: "eiriniuser",
					Password: tests.GetEiriniDockerHubPassword(),
				}
			})

			It("creates a private registry secret", func() {
				Expect(desireErr).NotTo(HaveOccurred())
				statefulset := getStatefulSetForLRP(lrp)
				Expect(statefulset.Spec.Template.Spec.ImagePullSecrets).To(HaveLen(2))
				privateRegistrySecretName := statefulset.Spec.Template.Spec.ImagePullSecrets[1].Name
				secret, err := getSecret(fixture.Namespace, privateRegistrySecretName)
				Expect(err).NotTo(HaveOccurred())
				Expect(secret).NotTo(BeNil())
			})

			It("sets the ImagePullSecret correctly in the pod template", func() {
				Eventually(func() []corev1.Pod {
					return listPods(lrp)
				}).Should(HaveLen(lrp.Spec.Instances))

				for i := 0; i < lrp.Spec.Instances; i++ {
					podIndex := i
					Eventually(func() string {
						return getPodPhase(podIndex, lrp)
					}).Should(Equal("Ready"))
				}
			})
		})

		When("we create the same StatefulSet again", func() {
			It("should not error", func() {
				err := lrpClient.Desire(ctx, lrp)
				Expect(err).NotTo(HaveOccurred())
			})
		})

		When("using a docker image that needs root access", func() {
			BeforeEach(func() {
				allowRunImageAsRoot = true

				lrp.Spec.Image = "eirini/nginx-integration"
				lrp.Spec.Command = nil
				lrp.Spec.Health.Type = "http"
				lrp.Spec.Health.Port = 8080
			})

			It("should start all the pods", func() {
				var podNames []string

				Eventually(func() []string {
					podNames = podNamesFromPods(listPods(lrp))

					return podNames
				}).Should(HaveLen(lrp.Spec.Instances))

				for i := 0; i < lrp.Spec.Instances; i++ {
					podIndex := i
					Eventually(func() string {
						return getPodPhase(podIndex, lrp)
					}).Should(Equal("Ready"))
				}

				Eventually(func() int32 {
					return getStatefulSetForLRP(lrp).Status.ReadyReplicas
				}).Should(BeNumerically("==", lrp.Spec.Instances))
			})
		})

		When("the LRP has 0 target instances", func() {
			BeforeEach(func() {
				lrp.Spec.Instances = 0
			})

			It("still creates a statefulset, with 0 replicas", func() {
				statefulset := getStatefulSetForLRP(lrp)
				Expect(statefulset.Name).To(ContainSubstring("odin-space-foo"))
				Expect(statefulset.Spec.Replicas).To(Equal(int32ptr(0)))
			})
		})

		When("the the app has sidecars", func() {
			assertEqualValues := func(actual, expected *resource.Quantity) {
				Expect(actual.Value()).To(Equal(expected.Value()))
			}

			BeforeEach(func() {
				lrp.Spec.Image = "eirini/busybox"
				lrp.Spec.Command = []string{"/bin/sh", "-c", "echo Hello from app; sleep 3600"}
				lrp.Spec.Sidecars = []eiriniv1.Sidecar{
					{
						Name:     "the-sidecar",
						Command:  []string{"/bin/sh", "-c", "echo Hello from sidecar; sleep 3600"},
						MemoryMB: 101,
					},
				}
			})

			It("deploys the app with the sidcar container", func() {
				statefulset := getStatefulSetForLRP(lrp)
				Expect(statefulset.Spec.Template.Spec.Containers).To(HaveLen(2))
			})

			It("sets resource limits on the sidecar container", func() {
				statefulset := getStatefulSetForLRP(lrp)
				containers := statefulset.Spec.Template.Spec.Containers
				for _, container := range containers {
					if container.Name == "the-sidecar" {
						limits := container.Resources.Limits
						requests := container.Resources.Requests

						expectedMemory := resource.NewScaledQuantity(101, resource.Mega)
						expectedDisk := resource.NewScaledQuantity(lrp.Spec.DiskMB, resource.Mega)
						expectedCPU := resource.NewScaledQuantity(int64(lrp.Spec.CPUWeight*10), resource.Milli)

						assertEqualValues(limits.Memory(), expectedMemory)
						assertEqualValues(limits.StorageEphemeral(), expectedDisk)
						assertEqualValues(requests.Memory(), expectedMemory)
						assertEqualValues(requests.Cpu(), expectedCPU)
					}
				}
			})
		})

		When("the app has user defined annotations", func() {
			BeforeEach(func() {
				lrp.Spec.UserDefinedAnnotations = map[string]string{
					"prometheus.io/scrape": "yes, please",
				}
			})

			It("sets them on the pod template", func() {
				statefulset := getStatefulSetForLRP(lrp)
				Expect(statefulset.Spec.Template.Annotations).To(HaveKeyWithValue("prometheus.io/scrape", "yes, please"))
			})
		})
	})

	Describe("Update", func() {
		Describe("scaling", func() {
			var (
				instancesBefore int
				instancesAfter  int
				statefulset     *appsv1.StatefulSet
			)

			BeforeEach(func() {
				instancesBefore = 1
				instancesAfter = 2
			})

			JustBeforeEach(func() {
				lrp.Spec.Instances = instancesBefore
				Expect(lrpClient.Desire(ctx, lrp)).To(Succeed())

				lrp.Spec.Instances = instancesAfter
				Expect(lrpClient.Update(ctx, lrp)).To(Succeed())
				statefulset = getStatefulSetForLRP(lrp)
			})

			It("updates instance count", func() {
				Expect(statefulset.Spec.Replicas).To(PointTo(Equal(int32(2))))
			})

			When("scaling up from 1 to 2 instances", func() {
				It("should create a pod disruption budget for the lrp", func() {
					pdb, err := podDisruptionBudgets().Get(context.Background(), statefulset.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(pdb).NotTo(BeNil())
				})
			})

			When("scaling up from 2 to 3 instances", func() {
				BeforeEach(func() {
					instancesBefore = 2
					instancesAfter = 3
				})

				It("should keep the existing pod disruption budget for the lrp", func() {
					pdb, err := podDisruptionBudgets().Get(context.Background(), statefulset.Name, metav1.GetOptions{})
					Expect(err).NotTo(HaveOccurred())
					Expect(pdb).NotTo(BeNil())
				})
			})

			When("scaling down from 2 to 1 instances", func() {
				BeforeEach(func() {
					instancesBefore = 2
					instancesAfter = 1
				})

				It("should delete the pod disruption budget for the lrp", func() {
					_, err := podDisruptionBudgets().Get(context.Background(), statefulset.Name, metav1.GetOptions{})
					Expect(err).To(MatchError(ContainSubstring("not found")))
				})
			})

			When("scaling down from 1 to 0 instances", func() {
				BeforeEach(func() {
					instancesBefore = 1
					instancesAfter = 0
				})

				It("should keep the lrp without a pod disruption budget", func() {
					_, err := podDisruptionBudgets().Get(context.Background(), statefulset.Name, metav1.GetOptions{})
					Expect(err).To(MatchError(ContainSubstring("not found")))
				})
			})
		})

		Describe("updating image", func() {
			var (
				imageBefore string
				imageAfter  string
				statefulset *appsv1.StatefulSet
			)

			BeforeEach(func() {
				imageBefore = "eirini/dorini"
				imageAfter = "eirini/notdora"
			})

			JustBeforeEach(func() {
				lrp.Spec.Image = imageBefore
				Expect(lrpClient.Desire(ctx, lrp)).To(Succeed())

				lrp.Spec.Image = imageAfter
				Expect(lrpClient.Update(ctx, lrp)).To(Succeed())
				statefulset = getStatefulSetForLRP(lrp)
			})

			It("updates the image", func() {
				Expect(statefulset.Spec.Template.Spec.Containers[0].Image).To(Equal(imageAfter))
			})
		})
	})
})

func createLRP(namespace, name string) *eiriniv1.LRP {
	lrp := &eiriniv1.LRP{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: eiriniv1.LRPSpec{
			GUID:    tests.GenerateGUID(),
			Version: tests.GenerateGUID(),
			Command: []string{
				"/bin/sh",
				"-c",
				"while true; do echo hello; sleep 10;done",
			},
			AppName:   name,
			AppGUID:   "the-app-guid",
			SpaceName: "space-foo",
			Instances: 2,
			Image:     "eirini/busybox",
			DiskMB:    2047,
			Env: map[string]string{
				"FOO": "BAR",
			},
		},
	}

	lrp, err := fixture.EiriniClientset.EiriniV1().LRPs(fixture.Namespace).Create(context.Background(), lrp, metav1.CreateOptions{})
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	return lrp
}

func createLrpClient(workloadsNamespace string, allowRunImageAsRoot bool) *k8s.LRPClient {
	logger := lagertest.NewTestLogger("test-" + workloadsNamespace)

	lrpToStatefulSetConverter := stset.NewLRPToStatefulSetConverter(
		tests.GetApplicationServiceAccount(),
		"registry-secret",
		false,
		allowRunImageAsRoot,
		k8s.CreateLivenessProbe,
		k8s.CreateReadinessProbe,
	)

	return k8s.NewLRPClient(
		logger,
		client.NewSecret(fixture.Clientset),
		client.NewStatefulSet(fixture.Clientset, workloadsNamespace),
		client.NewPod(fixture.Clientset, workloadsNamespace),
		pdb.NewUpdater(client.NewPodDisruptionBudget(fixture.Clientset)),
		client.NewEvent(fixture.Clientset),
		lrpToStatefulSetConverter,
		eirinischeme.Scheme,
	)
}

func int32ptr(i int) *int32 {
	i32 := int32(i)

	return &i32
}

func getPodPhase(index int, lrp *eiriniv1.LRP) string {
	pod := listPods(lrp)[index]
	status := pod.Status

	if status.Phase != corev1.PodRunning {
		return fmt.Sprintf("Pod - %s", status.Phase)
	}

	if len(status.ContainerStatuses) == 0 {
		return "Containers status unknown"
	}

	for _, containerStatus := range status.ContainerStatuses {
		if containerStatus.State.Running == nil {
			return fmt.Sprintf("Container %s - %v", containerStatus.Name, containerStatus.State)
		}

		if !containerStatus.Ready {
			return fmt.Sprintf("Container %s is not Ready", containerStatus.Name)
		}
	}

	return "Ready"
}

func getStatefulSetForLRP(lrp *eiriniv1.LRP) *appsv1.StatefulSet {
	ss, getErr := fixture.Clientset.AppsV1().StatefulSets(fixture.Namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: labelSelector(lrp),
	})
	Expect(getErr).NotTo(HaveOccurred())
	Expect(ss.Items).To(HaveLen(1))

	return &ss.Items[0]
}
