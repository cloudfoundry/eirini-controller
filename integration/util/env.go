package util

import (
	"os"
	"path/filepath"

	"github.com/onsi/ginkgo"
)

func GetKubeconfig() string {
	kubeconf := os.Getenv("INTEGRATION_KUBECONFIG")
	if kubeconf == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			ginkgo.Fail("INTEGRATION_KUBECONFIG not provided, failed to use default: " + err.Error())
		}
		return filepath.Join(homeDir, ".kube", "config")
	}
	return kubeconf
}

func GetEiriniDockerHubPassword() string {
	password := os.Getenv("EIRINIUSER_PASSWORD")
	if password == "" {
		ginkgo.Skip("eiriniuser password not provided. Please export EIRINIUSER_PASSWORD")
	}
	return password
}