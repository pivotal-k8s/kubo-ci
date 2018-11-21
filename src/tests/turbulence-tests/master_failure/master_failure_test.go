package master_failure_test

import (
	. "tests/test_helpers"

	boshdir "github.com/cloudfoundry/bosh-cli/director"
	"github.com/cppforlife/turbulence/incident"
	"github.com/cppforlife/turbulence/incident/selector"
	"github.com/cppforlife/turbulence/tasks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
)

var (
	deployment                    boshdir.Deployment
	kubectl                       *KubectlRunner
	nginxSpec                     = PathFromRoot("specs/nginx.yml")
	countRunningApiServerOnMaster func() int
	numberOfMasters               int
	director                      boshdir.Director
)

var _ = MasterFailureDescribe("A single master and etcd failure", func() {

	BeforeEach(func() {
		var err error
		director = NewDirector(testconfig.Bosh)
		deployment, err = director.FindDeployment(testconfig.Bosh.Deployment)
		Expect(err).NotTo(HaveOccurred())
		numberOfMasters = len(DeploymentVmsOfType(deployment, MasterVMType, ""))
		countRunningApiServerOnMaster = CountProcessesOnVmsOfType(deployment, MasterVMType, "kube-apiserver", VMRunningState)

		Expect(countRunningApiServerOnMaster()).To(Equal(numberOfMasters), "Precondition is to have all masters running before the test")

		kubectl = NewKubectlRunner()
		kubectl.Setup()
	})

	AfterEach(func() {
		kubectl.RunKubectlCommand("delete", "-f", nginxSpec).Wait("60s")
		director.EnableResurrection(true)
		kubectl.Teardown()
	})

	Specify("The cluster is healthy after master is resurrected", func() {
		killOneMaster := incident.Request{
			Selector: selector.Request{
				Deployment: &selector.NameRequest{
					Name: testconfig.Bosh.Deployment,
				},
				Group: &selector.NameRequest{
					Name: MasterVMType,
				},
				ID: &selector.IDRequest{
					Limit: selector.MustNewLimitFromString("1"),
				},
			},
			Tasks: tasks.OptionsSlice{
				tasks.KillOptions{},
			},
		}

		createTurbulenceIncident(killOneMaster, true, "Killing master")
	})

	Specify("The cluster is healthy after master is rebooted and bosh resurrector is off", func() {
		By("Turning off the resurrector")
		director.EnableResurrection(false)

		rebootOneMaster := incident.Request{
			Selector: selector.Request{
				Deployment: &selector.NameRequest{
					Name: testconfig.Bosh.Deployment,
				},
				Group: &selector.NameRequest{
					Name: MasterVMType,
				},
				ID: &selector.IDRequest{
					Limit: selector.MustNewLimitFromString("1"),
				},
			},
			Tasks: tasks.OptionsSlice{
				tasks.ShutdownOptions{
					Reboot: true,
				},
			},
		}

		createTurbulenceIncident(rebootOneMaster, false, "Rebooting master")
	})
})

func createTurbulenceIncident(request incident.Request, waitForIncident bool, msg string) {
	By("Deploying a workload on the k8s cluster")
	Eventually(kubectl.RunKubectlCommand("create", "-f", nginxSpec), "30s", "5s").Should(gexec.Exit(0))
	Eventually(kubectl.RunKubectlCommand("rollout", "status", "deployment/nginx", "-w"), "120s").Should(gexec.Exit(0))

	By("Creating Turbulence Incident")
	hellRaiser := TurbulenceClient(testconfig.Turbulence)
	incident := hellRaiser.CreateIncident(request)
	if waitForIncident {
		incident.Wait()
	}

	Eventually(countRunningApiServerOnMaster, "10m", "2s").Should(Equal(numberOfMasters-1), msg+" FAILED")

	By("Waiting for resurrection")
	Eventually(func() bool { return AllComponentsAreHealthy(kubectl) }, "600s", "20s").Should(BeTrue())

	By("Checking that all master jobs are running")
	Eventually(func() []boshdir.VMInfo { return DeploymentVmsOfType(deployment, MasterVMType, VMRunningState) }, "60s", "2s").Should(HaveLen(numberOfMasters))

	By("Checking for the workload on the k8s cluster")
	session := kubectl.RunKubectlCommand("get", "deployment", "nginx")
	Eventually(session, "120s").Should(gexec.Exit(0))

	By("Checking that master is back consistently")
	Consistently(func() []boshdir.VMInfo { return DeploymentVmsOfType(deployment, MasterVMType, VMRunningState) }, "60s", "2s").Should(HaveLen(numberOfMasters))
}
