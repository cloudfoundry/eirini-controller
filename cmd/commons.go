package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	// Kubernetes has a tricky way to add authentication
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func ReadConfigFile(path string, conf interface{}) error {
	if path == "" {
		return nil
	}

	fileBytes, err := ioutil.ReadFile(filepath.Clean(path))
	if err != nil {
		return errors.Wrap(err, "failed to read file")
	}

	return errors.Wrap(yaml.Unmarshal(fileBytes, conf), "failed to unmarshal yaml")
}

func ExitIfError(err error) {
	ExitfIfError(err, "an unexpected error occurred")
}

func ExitfIfError(err error, message string) {
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("%s: %w", message, err))
		os.Exit(1)
	}
}

func Exitf(messageFormat string, args ...interface{}) {
	ExitIfError(fmt.Errorf(messageFormat, args...))
}

func GetOrDefault(actualValue, defaultValue string) string {
	if actualValue != "" {
		return actualValue
	}

	return defaultValue
}

func GetEnvOrDefault(envVar, defaultValue string) string {
	return GetOrDefault(os.Getenv(envVar), defaultValue)
}
