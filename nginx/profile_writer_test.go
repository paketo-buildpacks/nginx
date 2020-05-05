package nginx_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/nginx/nginx"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testProfileWriter(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect        = NewWithT(t).Expect
		profileWriter nginx.ProfileWriter

		layerDir string
	)

	it.Before(func() {
		var err error
		profileWriter = nginx.NewProfileWriter()
		layerDir, err = ioutil.TempDir("", "layer")
		Expect(err).NotTo(HaveOccurred())
	})

	it.After(func() {
		err := os.RemoveAll(layerDir)
		Expect(err).NotTo(HaveOccurred())
	})
	when("writing a script into the profile.d directory", func() {
		it("writes an executable file", func() {

			Expect(profileWriter.Write(layerDir, "script-name", "some-contents")).To(Succeed())

			scriptPath := filepath.Join(layerDir, "profile.d", "script-name")
			Expect(scriptPath).To(BeAnExistingFile())

			fstat, err := os.Stat(scriptPath)
			Expect(err).NotTo(HaveOccurred())

			Expect(fstat.Mode()).To(Equal(os.FileMode(0644)))

			contents, err := ioutil.ReadFile(scriptPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(Equal("some-contents"))
		})
	})

	when("failure cases", func() {
		when("unable to create the profile.d dir", func() {
			it.Before(func() {
				Expect(os.Chmod(layerDir, 0000))
			})

			it.After(func() {
				Expect(os.Chmod(layerDir, os.ModePerm))
			})
			it("returns an error", func() {
				err := profileWriter.Write(layerDir, "script-name", "some-contents")
				Expect(err).To(MatchError(ContainSubstring(
					fmt.Sprintf("failed to create dir %s:", filepath.Join(layerDir, "profile.d")),
				)))
			})
		})
		when("unable to write script file", func() {
			var profileDir string
			it.Before(func() {
				profileDir = filepath.Join(layerDir, "profile.d")
				Expect(os.MkdirAll(profileDir, 0000)).To(Succeed())
			})
			it.After(func() {
				Expect(os.Chmod(profileDir, os.ModePerm)).To(Succeed())
			})
			it("returns an error", func() {
				err := profileWriter.Write(layerDir, "script-name", "some-contents")
				Expect(err).To(MatchError(ContainSubstring("failed to write profile script:")))
			})
		})
	})

}
