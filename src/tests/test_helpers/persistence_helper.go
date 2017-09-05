package test_helpers

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

func UndeployGuestBook(kubectl *KubectlRunner) {
	guestBookSpec := PathFromRoot("specs/pv-guestbook.yml")
	Eventually(kubectl.RunKubectlCommand("delete", "-f", guestBookSpec), "120s").Should(gexec.Exit(0))
}

func DeployGuestBook(kubectl *KubectlRunner) {

	guestBookSpec := PathFromRoot("specs/pv-guestbook.yml")
	Eventually(kubectl.RunKubectlCommand("apply", "-f", guestBookSpec), "120s").Should(gexec.Exit(0))
	Eventually(kubectl.RunKubectlCommand("rollout", "status", "deployment/frontend", "-w"), "120s").Should(gexec.Exit(0))
	Eventually(kubectl.RunKubectlCommand("rollout", "status", "deployment/redis-master", "-w"), "120s").Should(gexec.Exit(0))

}

func PostToGuestBook(address string, testValue string) {

	url := fmt.Sprintf("http://%s/guestbook.php?cmd=set&key=messages&value=%s", address, testValue)
	_, err := http.Get(url)
	Expect(err).ToNot(HaveOccurred())

}

func GetValueFromGuestBook(address string) string {

	httpClient := http.Client{
		Timeout: time.Duration(5 * time.Second),
	}
	url := fmt.Sprintf("http://%s/guestbook.php?cmd=get&key=messages", address)
	response, err := httpClient.Get(url)
	if err != nil {
		return fmt.Sprintf("error occured : %s", err.Error())
	}

	bodyBytes, err := ioutil.ReadAll(response.Body)
	Expect(err).ToNot(HaveOccurred())
	return string(bodyBytes)

}
