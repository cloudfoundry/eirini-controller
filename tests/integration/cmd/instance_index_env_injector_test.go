package cmd_test

import (
	"fmt"
	"net"
	"os"

	eirinictrl "code.cloudfoundry.org/eirini-controller"
	"code.cloudfoundry.org/eirini-controller/tests"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("InstanceIndexEnvInjector", func() {
	var (
		config         *eirinictrl.InstanceIndexEnvInjectorConfig
		configFilePath string
		session        *gexec.Session
		certDir        string
	)

	BeforeEach(func() {
		config = &eirinictrl.InstanceIndexEnvInjectorConfig{
			KubeConfig: eirinictrl.KubeConfig{
				ConfigPath: fixture.KubeConfigPath,
			},
			Port: int32(8080 + GinkgoParallelNode()),
		}
		certDir, _ = tests.GenerateKeyPairDir("tls", "my-domain")

		env := fmt.Sprintf("%s=%s", eirinictrl.EnvInstanceEnvInjectorCertDir, certDir)
		session, configFilePath = eiriniBins.InstanceIndexEnvInjector.Run(config, env)
	})

	AfterEach(func() {
		if configFilePath != "" {
			Expect(os.Remove(configFilePath)).To(Succeed())
		}

		if session != nil {
			Eventually(session.Kill()).Should(gexec.Exit())
		}

		Expect(os.RemoveAll(certDir)).To(Succeed())
	})

	It("runs the webhook service and registers it", func() {
		Eventually(func() error {
			_, err := net.Dial("tcp", fmt.Sprintf(":%d", config.Port))

			return err
		}).Should(Succeed())

		Consistently(session).ShouldNot(gexec.Exit())
	})

	When("the config file doesn't exist", func() {
		It("exits reporting missing config file", func() {
			session = eiriniBins.InstanceIndexEnvInjector.Restart("/does/not/exist", session)
			Eventually(session).Should(gexec.Exit())
			Expect(session.ExitCode).ToNot(BeZero())
			Expect(session.Err).To(gbytes.Say("Failed to read config file: failed to read file"))
		})
	})
})
