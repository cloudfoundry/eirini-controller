package integration_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.cloudfoundry.org/eirini-controller/k8s/stset"
	eiriniv1 "code.cloudfoundry.org/eirini-controller/pkg/apis/eirini/v1"
	"code.cloudfoundry.org/eirini-controller/tests"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	policy_v1beta1_types "k8s.io/client-go/kubernetes/typed/policy/v1beta1"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func TestEiriniK8sClient(t *testing.T) {
	SetDefaultEventuallyTimeout(4 * time.Minute)
	RegisterFailHandler(Fail)
	RunSpecs(t, "EiriniK8sClient Suite")
}

var (
	fixture *tests.Fixture
	ctx     context.Context
)

var _ = BeforeSuite(func() {
	fixture = tests.NewFixture(GinkgoWriter)
})

var _ = BeforeEach(func() {
	fixture.SetUp()
	ctx = context.Background()
})

var _ = AfterEach(func() {
	fixture.TearDown()
})

var _ = AfterSuite(func() {
	fixture.Destroy()
})

func labelSelector(lrp *eiriniv1.LRP) string {
	return fmt.Sprintf(
		"%s=%s,%s=%s",
		stset.LabelGUID, lrp.Spec.GUID,
		stset.LabelVersion, lrp.Spec.Version,
	)
}

func listStatefulSets(lrp *eiriniv1.LRP) []appsv1.StatefulSet {
	list, err := fixture.Clientset.AppsV1().StatefulSets(fixture.Namespace).List(context.Background(), metav1.ListOptions{
		LabelSelector: fmt.Sprintf(
			"%s=%s,%s=%s",
			stset.LabelGUID, lrp.Spec.GUID,
			stset.LabelVersion, lrp.Spec.Version,
		),
	})
	Expect(err).NotTo(HaveOccurred())

	return list.Items
}

func cleanupStatefulSet(lrp *eiriniv1.LRP) {
	backgroundPropagation := metav1.DeletePropagationBackground
	deleteOptions := metav1.DeleteOptions{PropagationPolicy: &backgroundPropagation}
	listOptions := metav1.ListOptions{LabelSelector: labelSelector(lrp)}
	err := fixture.Clientset.AppsV1().StatefulSets(fixture.Namespace).DeleteCollection(context.Background(), deleteOptions, listOptions)
	Expect(err).ToNot(HaveOccurred())
}

func listPodsByLabel(labelSelector string) []corev1.Pod {
	pods, err := fixture.Clientset.CoreV1().Pods(fixture.Namespace).List(context.Background(), metav1.ListOptions{LabelSelector: labelSelector})
	Expect(err).NotTo(HaveOccurred())

	return pods.Items
}

func listPods(lrp *eiriniv1.LRP) []corev1.Pod {
	return listPodsByLabel(labelSelector(lrp))
}

func podDisruptionBudgets() policy_v1beta1_types.PodDisruptionBudgetInterface {
	return fixture.Clientset.PolicyV1beta1().PodDisruptionBudgets(fixture.Namespace)
}

func podNamesFromPods(pods []corev1.Pod) []string {
	names := []string{}
	for _, p := range pods {
		names = append(names, p.Name)
	}

	return names
}

func nodeNamesFromPods(pods []corev1.Pod) []string {
	names := []string{}

	for _, p := range pods {
		nodeName := p.Spec.NodeName
		if nodeName != "" {
			names = append(names, nodeName)
		}
	}

	return names
}

func getNodeCount() int {
	nodeList, err := fixture.Clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
	Expect(err).ToNot(HaveOccurred())

	return len(nodeList.Items)
}

func getSecret(ns, name string) (*corev1.Secret, error) {
	return fixture.Clientset.CoreV1().Secrets(ns).Get(context.Background(), name, metav1.GetOptions{})
}
