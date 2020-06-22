package integration

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/paketo-buildpacks/occam"
	"github.com/paketo-buildpacks/packit/pexec"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var (
	nginxBuildpack        string
	offlineNginxBuildpack string
	version               string
	buildpackInfo struct {
		Buildpack struct {
			ID string
			Name string
		}
	}
)

func TestIntegration(t *testing.T) {
	var Expect = NewWithT(t).Expect

	root, err := filepath.Abs("./..")
	Expect(err).ToNot(HaveOccurred())

	file, err := os.Open("../buildpack.toml")
	Expect(err).NotTo(HaveOccurred())
	defer file.Close()

	_, err = toml.DecodeReader(file, &buildpackInfo)
	Expect(err).NotTo(HaveOccurred())

	buildpackStore := occam.NewBuildpackStore()

	version, err = GetGitVersion()
	Expect(err).NotTo(HaveOccurred())

	nginxBuildpack, err = buildpackStore.Get.
		WithVersion(version).
		Execute(root)
	Expect(err).NotTo(HaveOccurred())

	offlineNginxBuildpack, err = buildpackStore.Get.
		WithOfflineDependencies().
		WithVersion(version).
		Execute(root)
	Expect(err).NotTo(HaveOccurred())

	SetDefaultEventuallyTimeout(5 * time.Second)

	suite := spec.New("Integration", spec.Report(report.Terminal{}), spec.Parallel())
	suite("Caching", testCaching)
	suite("Logging", testLogging)
	suite("NoConfApp", testNoConfApp)
	suite("Offline", testOffline)
	suite("SimpleApp", testSimpleApp)
	suite.Run(t)
}

func GetGitVersion() (string, error) {
	gitExec := pexec.NewExecutable("git")
	revListOut := bytes.NewBuffer(nil)

	err := gitExec.Execute(pexec.Execution{
		Args:   []string{"rev-list", "--tags", "--max-count=1"},
		Stdout: revListOut,
	})
	if err != nil {
		return "", err
	}

	stdout := bytes.NewBuffer(nil)
	err = gitExec.Execute(pexec.Execution{
		Args:   []string{"describe", "--tags", strings.TrimSpace(revListOut.String())},
		Stdout: stdout,
	})
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(strings.TrimPrefix(stdout.String(), "v")), nil
}
