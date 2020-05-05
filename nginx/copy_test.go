package nginx_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/nginx/nginx"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testCopy(t *testing.T, when spec.G, it spec.S) {
	var (
		Expect     = NewWithT(t).Expect
		tmpSrcDir  string
		tmpDestDir string
	)

	it.Before(func() {
		var err error
		tmpSrcDir, err = ioutil.TempDir("", "copy")
		Expect(err).NotTo(HaveOccurred())
		tmpDestDir, err = ioutil.TempDir("", "copy")
		Expect(err).NotTo(HaveOccurred())
		Expect(ioutil.WriteFile(filepath.Join(tmpSrcDir, "afile"), []byte("some-contents"), 0644)).To(Succeed())
	})

	it.After(func() {
		Expect(os.RemoveAll(tmpSrcDir)).NotTo(HaveOccurred())
		Expect(os.RemoveAll(tmpDestDir)).NotTo(HaveOccurred())
	})

	when("copying a file", func() {
		it("succeeds", func() {
			destFile := filepath.Join(tmpDestDir, "newdir", "newfile")
			Expect(nginx.CopyBinFile(destFile, filepath.Join(tmpSrcDir, "afile"))).To(Succeed())
			Expect(destFile).To(BeAnExistingFile())
			fstat, err := os.Stat(destFile)
			Expect(err).NotTo(HaveOccurred())

			Expect(fstat.Mode()).To(Equal(os.FileMode(0755)))

			contents, err := ioutil.ReadFile(destFile)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(contents)).To(Equal("some-contents"))
		})
	})

	when("failure cases", func() {
		when("unable to create destination parent dir", func() {
			it.Before(func() {
				Expect(os.Chmod(tmpDestDir, 0000))
			})

			it.After(func() {
				Expect(os.Chmod(tmpDestDir, os.ModePerm))
			})
			it("returns an error", func() {
				err := nginx.CopyBinFile(filepath.Join(tmpDestDir, "new-dir", "a-new-file"), filepath.Join(tmpSrcDir, "non-existent"))
				Expect(err).To(MatchError(ContainSubstring("failed to create destination parent directory:")))
			})
		})
		when("source file doesn't exist", func() {
			it("returns an error", func() {
				err := nginx.CopyBinFile(filepath.Join(tmpDestDir, "a-new-file"), filepath.Join(tmpSrcDir, "non-existent"))
				Expect(err).To(MatchError(ContainSubstring("failed to open source file for reading:")))
			})
		})
		when("unable to create the destination file", func() {
			it.Before(func() {
				Expect(os.Chmod(tmpDestDir, 0000))
			})

			it.After(func() {
				Expect(os.Chmod(tmpDestDir, os.ModePerm))
			})
			it("returns an error", func() {
				err := nginx.CopyBinFile(filepath.Join(tmpDestDir, "a-new-file"), filepath.Join(tmpSrcDir, "afile"))
				Expect(err).To(MatchError(ContainSubstring("failed to open destination file for writing:")))
			})
		})

	})

}
