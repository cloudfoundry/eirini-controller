package tests

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"

	eiriniclient "code.cloudfoundry.org/eirini-controller/pkg/generated/clientset/versioned"
	eirinischeme "code.cloudfoundry.org/eirini-controller/pkg/generated/clientset/versioned/scheme"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	kscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	runtimeclient "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	basePortNumber = 20000
	portRange      = 1000
)

type Fixture struct {
	Clientset         kubernetes.Interface
	EiriniClientset   eiriniclient.Interface
	RuntimeClient     runtimeclient.Client
	Namespace         string
	KubeConfigPath    string
	Writer            io.Writer
	nextAvailablePort int
	portMux           *sync.Mutex
	extraNamespaces   []string
}

func makeKubeConfigCopy() string {
	kubeConfig := GetKubeconfig()
	if kubeConfig == "" {
		return ""
	}

	tmpKubeConfig, err := ioutil.TempFile("", "kube.cfg")
	Expect(err).NotTo(HaveOccurred())

	defer tmpKubeConfig.Close()

	kubeConfigContents, err := os.Open(kubeConfig)
	Expect(err).NotTo(HaveOccurred())

	defer kubeConfigContents.Close()

	_, err = io.Copy(tmpKubeConfig, kubeConfigContents)
	Expect(err).NotTo(HaveOccurred())

	return tmpKubeConfig.Name()
}

func NewFixture(writer io.Writer) *Fixture {
	kubeConfigPath := makeKubeConfigCopy()
	config, err := clientcmd.BuildConfigFromFlags("", kubeConfigPath)
	Expect(err).NotTo(HaveOccurred(), "failed to build config from flags")

	clientset, err := kubernetes.NewForConfig(config)
	Expect(err).NotTo(HaveOccurred(), "failed to create clientset")

	lrpclientset, err := eiriniclient.NewForConfig(config)
	Expect(err).NotTo(HaveOccurred(), "failed to create clientset")

	err = kscheme.AddToScheme(eirinischeme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	runtimeClient, err := runtimeclient.New(config, runtimeclient.Options{Scheme: eirinischeme.Scheme})
	Expect(err).NotTo(HaveOccurred(), "failed to create runtime client")

	return &Fixture{
		KubeConfigPath:    kubeConfigPath,
		Clientset:         clientset,
		EiriniClientset:   lrpclientset,
		RuntimeClient:     runtimeClient,
		Writer:            writer,
		nextAvailablePort: basePortNumber + portRange*GinkgoParallelProcess(),
		portMux:           &sync.Mutex{},
	}
}

func (f *Fixture) SetUp() {
	f.Namespace = f.CreateExtraNamespace()
}

func (f *Fixture) NextAvailablePort() int {
	f.portMux.Lock()
	defer f.portMux.Unlock()

	if f.nextAvailablePort > f.maxPortNumber() {
		Fail("Ginkgo node %d is not allowed to allocate more than %d ports", GinkgoParallelProcess(), portRange)
	}

	port := f.nextAvailablePort
	f.nextAvailablePort++

	return port
}

func (f Fixture) maxPortNumber() int {
	return basePortNumber + portRange*GinkgoParallelProcess() + portRange
}

func (f *Fixture) TearDown() {
	f.printDebugInfo()

	for _, ns := range f.extraNamespaces {
		_ = f.deleteNamespace(ns)
	}
}

func (f *Fixture) Destroy() {
	Expect(os.RemoveAll(f.KubeConfigPath)).To(Succeed())
}

func (f *Fixture) CreateExtraNamespace() string {
	name := f.configureNewNamespace()
	fmt.Fprintf(GinkgoWriter, "Created namespace %q\n", name)
	f.extraNamespaces = append(f.extraNamespaces, name)

	return name
}

func (f *Fixture) configureNewNamespace() string {
	namespace := CreateRandomNamespace(f.Clientset)
	Expect(ConfigureWorkloadsNamespace(namespace, GetApplicationServiceAccount(), f.Clientset)).To(Succeed(), "failed to configure workloads namespace")

	return namespace
}

func (f *Fixture) deleteNamespace(namespace string) error {
	return DeleteNamespace(namespace, f.Clientset)
}

func (f *Fixture) printDebugInfo() {
	fmt.Fprintln(f.Writer, "Jobs:")

	jobs, _ := f.Clientset.BatchV1().Jobs(f.Namespace).List(context.Background(), metav1.ListOptions{})

	for _, job := range jobs.Items {
		fmt.Fprintf(f.Writer, "Job: %s status is: %#v\n", job.Name, job.Status)
		fmt.Fprintln(f.Writer, "-----------")
	}

	statefulsets, _ := f.Clientset.AppsV1().StatefulSets(f.Namespace).List(context.Background(), metav1.ListOptions{})

	fmt.Fprintf(f.Writer, "StatefulSets:")

	for _, s := range statefulsets.Items {
		fmt.Fprintf(f.Writer, "StatefulSet: %s status is: %#v\n", s.Name, s.Status)
		fmt.Fprintln(f.Writer, "-----------")
	}

	pods, _ := f.Clientset.CoreV1().Pods(f.Namespace).List(context.Background(), metav1.ListOptions{})

	fmt.Fprintf(f.Writer, "Pods:")

	for _, p := range pods.Items {
		fmt.Fprintf(f.Writer, "Pod: %s status is: %#v\n", p.Name, p.Status)
		fmt.Fprintln(f.Writer, "-----------")

		fmt.Fprintf(f.Writer, "Pod: %s logs are: \n", p.Name)
		logsReq := f.Clientset.CoreV1().Pods(f.Namespace).GetLogs(p.Name, &corev1.PodLogOptions{})

		if err := consumeRequest(logsReq, f.Writer); err != nil {
			fmt.Fprintf(f.Writer, "Failed to get logs for Pod: %s becase: %v \n", p.Name, err)
		}
	}
}

func consumeRequest(request rest.ResponseWrapper, out io.Writer) error {
	readCloser, err := request.Stream(context.Background())
	if err != nil {
		return err
	}
	defer readCloser.Close()

	r := bufio.NewReader(readCloser)

	for {
		bytes, err := r.ReadBytes('\n')
		if _, writeErr := out.Write(bytes); writeErr != nil {
			return writeErr
		}

		if err != nil {
			if !errors.Is(err, io.EOF) {
				return err
			}

			return nil
		}
	}
}
