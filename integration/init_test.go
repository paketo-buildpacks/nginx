package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/BurntSushi/toml"
	"github.com/onsi/gomega/format"
	"github.com/paketo-buildpacks/occam"
	"github.com/paketo-buildpacks/occam/packagers"
	"github.com/sclevine/spec"
	"github.com/sclevine/spec/report"

	. "github.com/onsi/gomega"
)

var settings struct {
	Buildpacks struct {
		NGINX struct {
			Online  string
			Offline string
		}
		Watchexec struct {
			Online string
		}
		BuildPlan struct {
			Online string
		}
	}

	Buildpack struct {
		ID   string
		Name string
	}

	Config struct {
		Watchexec string `json:"watchexec"`
		BuildPlan string `json:"build-plan"`
	}
}

func TestIntegration(t *testing.T) {
	format.MaxLength = 0
	var Expect = NewWithT(t).Expect

	root, err := filepath.Abs("./..")
	Expect(err).ToNot(HaveOccurred())

	file, err := os.Open("../buildpack.toml")
	Expect(err).NotTo(HaveOccurred())
	defer file.Close()

	_, err = toml.NewDecoder(file).Decode(&settings)
	Expect(err).NotTo(HaveOccurred())

	file, err = os.Open("../integration.json")
	Expect(err).NotTo(HaveOccurred())
	defer file.Close()

	Expect(json.NewDecoder(file).Decode(&settings.Config)).To(Succeed())

	buildpackStore := occam.NewBuildpackStore()
	libpakBuildpackStore := occam.NewBuildpackStore().WithPackager(packagers.NewLibpak())

	settings.Buildpacks.NGINX.Online, err = buildpackStore.Get.
		WithVersion("1.2.3").
		Execute(root)
	Expect(err).NotTo(HaveOccurred())

	settings.Buildpacks.NGINX.Offline, err = buildpackStore.Get.
		WithOfflineDependencies().
		WithVersion("1.2.3").
		Execute(root)
	Expect(err).NotTo(HaveOccurred())

	settings.Buildpacks.Watchexec.Online, err = libpakBuildpackStore.Get.
		Execute(settings.Config.Watchexec)
	Expect(err).ToNot(HaveOccurred())

	settings.Buildpacks.BuildPlan.Online, err = libpakBuildpackStore.Get.
		Execute(settings.Config.BuildPlan)
	Expect(err).ToNot(HaveOccurred())

	SetDefaultEventuallyTimeout(5 * time.Second)

	suite := spec.New("Integration", spec.Report(report.Terminal{}), spec.Parallel())
	suite("Require", testRequire)
	suite("Caching", testCaching)
	suite("CustomConfApp", testCustomConfApp)
	suite("Logging", testLogging)
	suite("NoConfApp", testNoConfApp)
	suite("Offline", testOffline)
	suite("SimpleApp", testSimpleApp)
	suite.Run(t)
}
