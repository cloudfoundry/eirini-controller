package tests

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"code.cloudfoundry.org/eirini-controller/k8s/stset"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/pod-security-admission/api"
)

const DefaultApplicationServiceAccount = "eirini"

func CreateRandomNamespace(clientset kubernetes.Interface) string {
	namespace := fmt.Sprintf("integration-test-%s-%d", GenerateGUID(), GinkgoParallelProcess())
	for namespaceExists(namespace, clientset) {
		namespace = fmt.Sprintf("integration-test-%s-%d", GenerateGUID(), GinkgoParallelProcess())
	}
	createNamespace(namespace, clientset)

	return namespace
}

func namespaceExists(namespace string, clientset kubernetes.Interface) bool {
	_, err := clientset.CoreV1().Namespaces().Get(context.Background(), namespace, metav1.GetOptions{})

	return err == nil
}

func createNamespace(namespace string, clientset kubernetes.Interface) {
	namespaceSpec := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
			Labels: map[string]string{
				api.AuditLevelLabel:   string(api.LevelRestricted),
				api.EnforceLevelLabel: string(api.LevelRestricted),
			},
		},
	}

	_, err := clientset.CoreV1().Namespaces().Create(context.Background(), namespaceSpec, metav1.CreateOptions{})
	Expect(err).NotTo(HaveOccurred())
}

func ConfigureWorkloadsNamespace(namespace, serviceAccountName string, clientset kubernetes.Interface) error {
	_, err := clientset.CoreV1().ServiceAccounts(namespace).Create(context.Background(), &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: namespace,
		},
	}, metav1.CreateOptions{})

	return err
}

func DeleteNamespace(namespace string, clientset kubernetes.Interface) error {
	return clientset.CoreV1().Namespaces().Delete(context.Background(), namespace, metav1.DeleteOptions{})
}

func GetApplicationServiceAccount() string {
	serviceAccountName := os.Getenv("APPLICATION_SERVICE_ACCOUNT")
	if serviceAccountName != "" {
		return serviceAccountName
	}

	return DefaultApplicationServiceAccount
}

func ExposeAsService(clientset kubernetes.Interface, namespace, guid string, appPort int32, pingPath ...string) string {
	service, err := clientset.CoreV1().Services(namespace).Create(context.Background(), &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name: "service-" + guid,
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Port: appPort,
				},
			},
			Selector: map[string]string{
				stset.LabelGUID: guid,
			},
		},
	}, metav1.CreateOptions{})
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	if len(pingPath) > 0 {
		EventuallyWithOffset(1, func() error {
			_, err := RequestServiceFn(namespace, service.Name, appPort, pingPath[0])()

			return err
		}).Should(Succeed())
	}

	return service.Name
}

func RequestServiceFn(namespace, serviceName string, port int32, requestPath string) func() (string, error) {
	client := &http.Client{
		Timeout: time.Second,
		Transport: &http.Transport{
			DisableKeepAlives: true,
		},
	}

	return func() (_ string, err error) {
		defer func() {
			if err != nil {
				fmt.Fprintf(GinkgoWriter, "RequestServiceFn error: %v", err)
			}
		}()

		requestURL := fmt.Sprintf("http://%s.%s:%d/%s", serviceName, namespace, port, requestPath)

		resp, err := client.Get(requestURL)
		if err != nil {
			return "", err
		}
		defer resp.Body.Close()

		content, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return "", err
		}

		if resp.StatusCode != http.StatusOK {
			return string(content), fmt.Errorf("request failed: %s", resp.Status)
		}

		return string(content), nil
	}
}
