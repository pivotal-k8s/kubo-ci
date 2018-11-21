package external_traffic_policy_test

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"regexp"
	"strings"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func getSourceIPFromEchoserver(appURL string) (string, error) {

	httpClient := http.Client{
		Timeout: time.Duration(5 * time.Second),
	}

	result, err := httpClient.Get(appURL)
	if err != nil {
		fmt.Fprintf(GinkgoWriter, "Failed to get response from %s: %v\n", appURL, err)
		return "", err
	}
	if result != nil && result.StatusCode != 200 {
		return "", fmt.Errorf("Failed to get response from %s: StatusCode %v\n", appURL, result.StatusCode)
	}

	defer result.Body.Close()
	body, err := ioutil.ReadAll(result.Body)
	if err != nil {
		return "", err
	}
	re := regexp.MustCompile("client_address=(.*)")

	return re.FindAllStringSubmatch(string(body), -1)[0][0], nil
}

var _ = Describe("When deploying a loadbalancer", func() {
	var loadbalancerAddress string

	Context("with externalTrafficPolicy to local", func() {
		It("shows a different source client IPs", func() {
			if iaas != "gce" && iaas != "azure" {
				Skip("Test only valid for GCE and Azure")
			}

			deployEchoserver := runner.RunKubectlCommand("create", "-f", echoserverLBSpec)
			Eventually(deployEchoserver, "120s").Should(gexec.Exit(0))
			rolloutWatch := runner.RunKubectlCommand("rollout", "status", "deployment/echoserver", "-w")
			Eventually(rolloutWatch, "120s").Should(gexec.Exit(0))

			loadbalancerAddress = ""
			Eventually(func() string {
				loadbalancerAddress = runner.GetLBAddress("echoserver", iaas)
				return loadbalancerAddress
			}, "240s", "60s").Should(Not(Equal("")))

			appURL := fmt.Sprintf("http://%s", loadbalancerAddress)
			var ipAddress string
			Eventually(func() error {
				var err error
				ipAddress, err = getSourceIPFromEchoserver(appURL)
				return err
			}, "90s", "15s").Should(Succeed())
			segments := strings.Split(ipAddress, ".")

			runner.RunKubectlCommandWithTimeout("patch", "svc/echoserver", "-p", "{\"spec\":{\"externalTrafficPolicy\":\"Local\"}}")
			prefix := segments[0] + "." + segments[1] + "."

			loadbalancerAddress = runner.GetLBAddress("echoserver", iaas)
			appURL = fmt.Sprintf("http://%s", loadbalancerAddress)
			// reset cache
			runner.RunKubectlCommand("delete", "pods", "--all")

			Eventually(func() string {
				newPrefix, err := getSourceIPFromEchoserver(appURL)
				if err != nil {
					GinkgoWriter.Write([]byte(err.Error()))
				}
				return newPrefix
			}, "600s", "60s").Should(And(Not(BeEmpty()), Not(HavePrefix(prefix))))
		})
	})

	AfterEach(func() {
		if iaas == "gce" || iaas == "azure" {
			runner.RunKubectlCommand("delete", "-f", echoserverLBSpec).Wait("60s")
		}
	})
})

var _ = Describe("When using a NodePort service", func() {
	Context("with externalTrafficPolicy to local", func() {
		It("shows a different source client IPs", func() {
			if iaas != "vsphere" {
				Skip("Test only valid for vSphere")
			}
			deployEchoserver := runner.RunKubectlCommand("create", "-f", echoserverNodePortSpec)
			Eventually(deployEchoserver, "120s").Should(gexec.Exit(0))
			rolloutWatch := runner.RunKubectlCommand("rollout", "status", "daemonset/echoserver", "-w")
			Eventually(rolloutWatch, "120s").Should(gexec.Exit(0))

			appURL := fmt.Sprintf("http://%s", runner.GetAppAddress("svc/echoserver"))
			var sourceIP string
			Eventually(func() error {
				var err error
				sourceIP, err = getSourceIPFromEchoserver(appURL)
				return err
			}, "90s", "15s").Should(Succeed())
			segments := strings.Split(sourceIP, ".")

			runner.RunKubectlCommandWithTimeout("patch", "svc/echoserver", "-p", "{\"spec\":{\"externalTrafficPolicy\":\"Local\"}}")
			prefix := segments[0] + "." + segments[1] + "."

			// reset cache
			runner.RunKubectlCommand("delete", "pods", "--all")

			Eventually(func() string {
				newSourceIP, err := getSourceIPFromEchoserver(appURL)
				if err != nil {
					GinkgoWriter.Write([]byte(err.Error()))
				}
				return newSourceIP
			}, "600s", "60s").Should(And(Not(BeEmpty()), Not(HavePrefix(prefix))))
		})
	})
	AfterEach(func() {
		if iaas == "vsphere" {
			runner.RunKubectlCommand("delete", "-f", echoserverNodePortSpec).Wait("60s")
		}
	})
})
