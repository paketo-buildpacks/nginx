package nginx_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/nginx"
	"github.com/paketo-buildpacks/nginx/fakes"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/sbom"

	//nolint Ignore SA1019, informed usage of deprecated package
	"github.com/paketo-buildpacks/packit/v2/paketosbom"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testBuild(t *testing.T, context spec.G, it spec.S) {

	var (
		Expect = NewWithT(t).Expect

		layersDir    string
		cnbPath      string
		workspaceDir string

		dependencyService *fakes.DependencyService
		configGenerator   *fakes.ConfigGenerator
		calculator        *fakes.Calculator
		sbomGenerator     *fakes.SBOMGenerator

		buffer *bytes.Buffer

		buildContext packit.BuildContext
		build        packit.BuildFunc
	)

	it.Before(func() {
		layersDir = t.TempDir()
		cnbPath = t.TempDir()
		workspaceDir = t.TempDir()

		buffer = bytes.NewBuffer(nil)

		dependencyService = &fakes.DependencyService{}
		dependencyService.ResolveCall.Returns.Dependency = postal.Dependency{
			ID:             "nginx",
			Checksum:       "sha256:some-sha",
			Source:         "some-source",
			SourceChecksum: "sha256:some-source-sha",
			Stacks:         []string{"some-stack"},
			URI:            "some-uri",
			Version:        "1.19.8",
		}
		dependencyService.GenerateBillOfMaterialsCall.Returns.BOMEntrySlice = []packit.BOMEntry{
			{
				Name: "nginx",
				Metadata: paketosbom.BOMMetadata{
					Version: "nginx-dependency-version",
					Checksum: paketosbom.BOMChecksum{
						Algorithm: paketosbom.SHA256,
						Hash:      "nginx-dependency-sha",
					},
					URI: "nginx-dependency-uri",
				},
			},
		}

		configGenerator = &fakes.ConfigGenerator{}
		calculator = &fakes.Calculator{}
		calculator.SumCall.Returns.String = "some-bin-sha"

		sbomGenerator = &fakes.SBOMGenerator{}
		sbomGenerator.GenerateFromDependencyCall.Returns.SBOM = sbom.SBOM{}

		Expect(os.Mkdir(filepath.Join(cnbPath, "bin"), os.ModePerm)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(cnbPath, "bin", "configure"), []byte("binary-contents"), 0600)).To(Succeed())

		Expect(os.WriteFile(filepath.Join(workspaceDir, "nginx.conf"), []byte("worker_processes 2;"), 0600)).To(Succeed())

		buildContext = packit.BuildContext{
			BuildpackInfo: packit.BuildpackInfo{
				Name:        "Some Buildpack",
				SBOMFormats: []string{sbom.CycloneDXFormat, sbom.SPDXFormat},
				Version:     "1.2.3",
			},
			CNBPath:    cnbPath,
			WorkingDir: workspaceDir,
			Stack:      "some-stack",
			Platform:   packit.Platform{Path: "platform"},
			Plan: packit.BuildpackPlan{
				Entries: []packit.BuildpackPlanEntry{
					{
						Name: "nginx",
						Metadata: map[string]interface{}{
							"version-source": "BP_NGINX_VERSION",
							"version":        "1.19.*",
							"launch":         true,
						},
					},
				},
			},
			Layers: packit.Layers{Path: layersDir},
		}

		build = nginx.Build(
			nginx.Configuration{
				NGINXConfLocation: "./nginx.conf",
				WebServerRoot:     "./public",
			},
			dependencyService,
			configGenerator,
			calculator,
			sbomGenerator,
			scribe.NewEmitter(buffer),
			chronos.DefaultClock,
		)
	})

	it("does a build", func() {
		result, err := build(buildContext)
		Expect(err).NotTo(HaveOccurred())

		Expect(len(result.Layers)).To(Equal(1))
		layer := result.Layers[0]

		Expect(layer.Name).To(Equal("nginx"))
		Expect(layer.Path).To(Equal(filepath.Join(layersDir, "nginx")))
		Expect(layer.Build).To(BeFalse())
		Expect(layer.Launch).To(BeTrue())
		Expect(layer.Cache).To(BeFalse())
		Expect(layer.SharedEnv).To(Equal(packit.Environment{
			"PATH.append": filepath.Join(layersDir, "nginx", "sbin"),
			"PATH.delim":  ":",
		}))
		Expect(layer.LaunchEnv).To(Equal(packit.Environment{
			"EXECD_CONF.default": filepath.Join(workspaceDir, nginx.ConfFile),
		}))
		Expect(layer.Metadata).To(Equal(map[string]interface{}{
			nginx.DepKey:          "sha256:some-sha",
			nginx.ConfigureBinKey: "some-bin-sha",
		}))
		Expect(layer.ExecD).To(Equal([]string{filepath.Join(cnbPath, "bin", "configure")}))

		Expect(result.Launch.BOM).To(Equal([]packit.BOMEntry{
			{
				Name: "nginx",
				Metadata: paketosbom.BOMMetadata{
					Version: "nginx-dependency-version",
					Checksum: paketosbom.BOMChecksum{
						Algorithm: paketosbom.SHA256,
						Hash:      "nginx-dependency-sha",
					},
					URI: "nginx-dependency-uri",
				},
			},
		}))

		Expect(result.Launch.Processes).To(Equal([]packit.Process{
			{
				Type:    "web",
				Command: "nginx",
				Args: []string{
					"-p", workspaceDir,
					"-c", filepath.Join(workspaceDir, nginx.ConfFile),
					"-g", "pid /tmp/nginx.pid;",
				},
				Direct:  true,
				Default: true,
			},
		}))

		Expect(layer.SBOM.Formats()).To(HaveLen(2))
		cdx := layer.SBOM.Formats()[0]
		spdx := layer.SBOM.Formats()[1]

		Expect(cdx.Extension).To(Equal("cdx.json"))
		content, err := io.ReadAll(cdx.Content)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(MatchJSON(`{
			"bomFormat": "CycloneDX",
			"components": [],
			"metadata": {
				"tools": [
					{
						"name": "syft",
						"vendor": "anchore",
						"version": "[not provided]"
					}
				]
			},
			"specVersion": "1.3",
			"version": 1
		}`))

		Expect(spdx.Extension).To(Equal("spdx.json"))
		content, err = io.ReadAll(spdx.Content)
		Expect(err).NotTo(HaveOccurred())
		Expect(string(content)).To(MatchJSON(`{
			"SPDXID": "SPDXRef-DOCUMENT",
			"creationInfo": {
				"created": "0001-01-01T00:00:00Z",
				"creators": [
					"Organization: Anchore, Inc",
					"Tool: syft-"
				],
				"licenseListVersion": "3.16"
			},
			"dataLicense": "CC0-1.0",
			"documentNamespace": "https://paketo.io/packit/unknown-source-type/unknown-88cfa225-65e0-5755-895f-c1c8f10fde76",
			"name": "unknown",
			"relationships": [
				{
					"relatedSpdxElement": "SPDXRef-DOCUMENT",
					"relationshipType": "DESCRIBES",
					"spdxElementId": "SPDXRef-DOCUMENT"
				}
			],
			"spdxVersion": "SPDX-2.2"
		}`))

		Expect(filepath.Join(layersDir, "nginx")).To(BeADirectory())

		Expect(dependencyService.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbPath, "buildpack.toml")))
		Expect(dependencyService.ResolveCall.Receives.Name).To(Equal("nginx"))
		Expect(dependencyService.ResolveCall.Receives.Version).To(Equal("1.19.*"))
		Expect(dependencyService.ResolveCall.Receives.Stack).To(Equal("some-stack"))

		Expect(dependencyService.DeliverCall.Receives.Dependency).To(Equal(
			postal.Dependency{
				ID:             "nginx",
				Checksum:       "sha256:some-sha",
				Source:         "some-source",
				SourceChecksum: "sha256:some-source-sha",
				Stacks:         []string{"some-stack"},
				URI:            "some-uri",
				Version:        "1.19.8",
			},
		))
		Expect(dependencyService.DeliverCall.Receives.CnbPath).To(Equal(cnbPath))
		Expect(dependencyService.DeliverCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "nginx")))
		Expect(dependencyService.DeliverCall.Receives.PlatformPath).To(Equal("platform"))
		Expect(calculator.SumCall.CallCount).To(Equal(1))

		Expect(sbomGenerator.GenerateFromDependencyCall.Receives.Dependency).To(Equal(postal.Dependency{
			ID:             "nginx",
			Checksum:       "sha256:some-sha",
			Source:         "some-source",
			SourceChecksum: "sha256:some-source-sha",
			Stacks:         []string{"some-stack"},
			URI:            "some-uri",
			Version:        "1.19.8",
		}))
		Expect(sbomGenerator.GenerateFromDependencyCall.Receives.Dir).To(Equal(filepath.Join(layersDir, "nginx")))
	})

	context("when live reload is enabled", func() {
		it.Before(func() {
			build = nginx.Build(
				nginx.Configuration{
					NGINXConfLocation: "./nginx.conf",
					WebServerRoot:     "./public",
					LiveReloadEnabled: true,
				},
				dependencyService,
				configGenerator,
				calculator,
				sbomGenerator,
				scribe.NewEmitter(buffer),
				chronos.DefaultClock,
			)
		})

		it("uses watchexec to set the start command", func() {
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			Expect(result.Launch.Processes).To(Equal([]packit.Process{
				{
					Type:    "web",
					Command: "watchexec",
					Args: []string{
						"--restart",
						"--watch", workspaceDir,
						"--shell", "none",
						"--",
						"nginx",
						"-p", workspaceDir,
						"-c", filepath.Join(workspaceDir, nginx.ConfFile),
						"-g", "pid /tmp/nginx.pid;",
					},
					Direct:  true,
					Default: true,
				},
				{
					Type:    "no-reload",
					Command: "nginx",
					Args: []string{
						"-p", workspaceDir,
						"-c", filepath.Join(workspaceDir, nginx.ConfFile),
						"-g", "pid /tmp/nginx.pid;",
					},
					Direct: true,
				},
			}))
		})
	})

	context("when version source is buildpack.yml", func() {
		it.Before(func() {
			dependencyService.ResolveCall.Returns.Dependency = postal.Dependency{
				ID:             "nginx",
				Checksum:       "sha256:some-sha",
				Source:         "some-source",
				SourceChecksum: "sha256:some-source-sha",
				Stacks:         []string{"some-stack"},
				URI:            "some-uri",
				Version:        "some-bp-yml-version",
			}

			buildContext.Plan.Entries[0].Metadata = map[string]interface{}{
				"version-source": "buildpack.yml",
				"version":        "some-bp-yml-version",
				"launch":         true,
			}
		})

		it("does a build", func() {
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			Expect(len(result.Layers)).To(Equal(1))
			layer := result.Layers[0]

			Expect(layer.Name).To(Equal("nginx"))
			Expect(layer.Path).To(Equal(filepath.Join(layersDir, "nginx")))
			Expect(layer.Build).To(BeFalse())
			Expect(layer.Launch).To(BeTrue())
			Expect(layer.Cache).To(BeFalse())
			Expect(layer.SharedEnv).To(Equal(packit.Environment{
				"PATH.append": filepath.Join(layersDir, "nginx", "sbin"),
				"PATH.delim":  ":",
			}))
			Expect(layer.Metadata).To(Equal(map[string]interface{}{
				nginx.DepKey:          "sha256:some-sha",
				nginx.ConfigureBinKey: "some-bin-sha",
			}))

			Expect(result.Launch.BOM).To(Equal([]packit.BOMEntry{
				{
					Name: "nginx",
					Metadata: paketosbom.BOMMetadata{
						Version: "nginx-dependency-version",
						Checksum: paketosbom.BOMChecksum{
							Algorithm: paketosbom.SHA256,
							Hash:      "nginx-dependency-sha",
						},
						URI: "nginx-dependency-uri",
					},
				},
			}))

			Expect(result.Launch.Processes).To(Equal([]packit.Process{
				{
					Type:    "web",
					Command: "nginx",
					Args: []string{
						"-p", workspaceDir,
						"-c", filepath.Join(workspaceDir, nginx.ConfFile),
						"-g", "pid /tmp/nginx.pid;",
					},
					Direct:  true,
					Default: true,
				},
			}))

			Expect(filepath.Join(layersDir, "nginx")).To(BeADirectory())

			Expect(dependencyService.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbPath, "buildpack.toml")))
			Expect(dependencyService.ResolveCall.Receives.Name).To(Equal("nginx"))
			Expect(dependencyService.ResolveCall.Receives.Version).To(Equal("some-bp-yml-version"))
			Expect(dependencyService.ResolveCall.Receives.Stack).To(Equal("some-stack"))

			Expect(dependencyService.DeliverCall.Receives.Dependency).To(Equal(
				postal.Dependency{
					ID:             "nginx",
					Checksum:       "sha256:some-sha",
					Source:         "some-source",
					SourceChecksum: "sha256:some-source-sha",
					Stacks:         []string{"some-stack"},
					URI:            "some-uri",
					Version:        "some-bp-yml-version",
				},
			))
			Expect(dependencyService.DeliverCall.Receives.CnbPath).To(Equal(cnbPath))
			Expect(dependencyService.DeliverCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "nginx")))
			Expect(dependencyService.DeliverCall.Receives.PlatformPath).To(Equal("platform"))
			Expect(calculator.SumCall.CallCount).To(Equal(1))

			Expect(buffer.String()).To(ContainSubstring("WARNING: Setting the server version through buildpack.yml will be deprecated soon in Nginx Server Buildpack v2.0.0"))
			Expect(buffer.String()).To(ContainSubstring("Please specify the version through the $BP_NGINX_VERSION environment variable instead. See docs for more information."))
		})
	})

	context("when reusing a layer", func() {
		it.Before(func() {
			err := os.WriteFile(filepath.Join(layersDir, "nginx.toml"), []byte(`[metadata]
			dependency-sha = "some-sha"
			configure-bin-sha = "some-bin-sha"
			`), 0600)
			Expect(err).NotTo(HaveOccurred())

			buildContext.Plan.Entries[0].Metadata["version"] = "1.17.*"
		})

		it.After(func() {
			Expect(os.RemoveAll(filepath.Join(layersDir, "nginx.toml"))).To(Succeed())
		})

		it("does not re-build the nginx layer", func() {
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			Expect(len(result.Layers)).To(Equal(1))
			layer := result.Layers[0]

			Expect(layer.Name).To(Equal("nginx"))
			Expect(layer.Path).To(Equal(filepath.Join(layersDir, "nginx")))
			Expect(layer.Build).To(BeFalse())
			Expect(layer.Launch).To(BeTrue())
			Expect(layer.Cache).To(BeFalse())
			Expect(layer.Metadata).To(Equal(map[string]interface{}{
				nginx.DepKey:          "some-sha",
				nginx.ConfigureBinKey: "some-bin-sha",
			}))

			Expect(result.Launch.BOM).To(Equal([]packit.BOMEntry{
				{
					Name: "nginx",
					Metadata: paketosbom.BOMMetadata{
						Version: "nginx-dependency-version",
						Checksum: paketosbom.BOMChecksum{
							Algorithm: paketosbom.SHA256,
							Hash:      "nginx-dependency-sha",
						},
						URI: "nginx-dependency-uri",
					},
				},
			}))

			Expect(result.Launch.Processes).To(Equal([]packit.Process{
				{
					Type:    "web",
					Command: "nginx",
					Args: []string{
						"-p", workspaceDir,
						"-c", filepath.Join(workspaceDir, nginx.ConfFile),
						"-g", "pid /tmp/nginx.pid;",
					},
					Direct:  true,
					Default: true,
				},
			}))

			Expect(dependencyService.DeliverCall.CallCount).To(Equal(0))
		})
	})

	context("when BP_NGINX_CONF_LOCATION is set to a relative path", func() {
		it.Before(func() {
			Expect(os.Mkdir(filepath.Join(workspaceDir, "some-relative-path"), os.ModePerm)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(workspaceDir, "some-relative-path", "nginx.conf"), []byte("worker_processes 2;"), 0600)).To(Succeed())

			build = nginx.Build(
				nginx.Configuration{NGINXConfLocation: "some-relative-path/nginx.conf"},
				dependencyService,
				configGenerator,
				calculator,
				sbomGenerator,
				scribe.NewEmitter(buffer),
				chronos.DefaultClock,
			)
		})

		it("assumes path is relative to /workspace", func() {
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Launch.Processes[0].Args).To(Equal([]string{
				"-p", workspaceDir,
				"-c", filepath.Join(workspaceDir, "some-relative-path", "nginx.conf"),
				"-g", "pid /tmp/nginx.pid;",
			}))
			Expect(result.Layers[0].LaunchEnv).To(Equal(packit.Environment{
				"EXECD_CONF.default": filepath.Join(workspaceDir, "some-relative-path/nginx.conf"),
			}))
		})
	})

	context("when BP_NGINX_CONF_LOCATION is set to an absolute path", func() {
		it.Before(func() {
			Expect(os.Mkdir(filepath.Join(workspaceDir, "some-absolute-path"), os.ModePerm)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(workspaceDir, "some-absolute-path", "nginx.conf"), []byte("worker_processes 2;"), 0600)).To(Succeed())

			build = nginx.Build(
				nginx.Configuration{NGINXConfLocation: filepath.Join(workspaceDir, "some-absolute-path", "nginx.conf")},
				dependencyService,
				configGenerator,
				calculator,
				sbomGenerator,
				scribe.NewEmitter(buffer),
				chronos.DefaultClock,
			)
		})

		it("uses the location as-is", func() {
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Launch.Processes[0].Args).To(Equal([]string{
				"-p", workspaceDir,
				"-c", filepath.Join(workspaceDir, "some-absolute-path", "nginx.conf"),
				"-g", "pid /tmp/nginx.pid;",
			}))
			Expect(result.Layers[0].LaunchEnv).To(Equal(packit.Environment{
				"EXECD_CONF.default": filepath.Join(workspaceDir, "some-absolute-path", "nginx.conf"),
			}))
		})
	})

	context("when BP_WEB_SERVER=nginx in the build env", func() {
		it.Before(func() {
			build = nginx.Build(
				nginx.Configuration{
					NGINXConfLocation: "./nginx.conf",
					WebServer:         "nginx",
					WebServerRoot:     "custom",
				},
				dependencyService,
				configGenerator,
				calculator,
				sbomGenerator,
				scribe.NewEmitter(buffer),
				chronos.DefaultClock,
			)
		})

		it("generates a basic nginx.conf and passes env var configuration into template generator", func() {
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())
			Expect(configGenerator.GenerateCall.Receives.Config).To(Equal(nginx.Configuration{
				NGINXConfLocation: filepath.Join(workspaceDir, "nginx.conf"),
				WebServer:         "nginx",
				WebServerRoot:     "custom",
			}))

			Expect(result.Layers[0].LaunchEnv).To(Equal(packit.Environment{
				"APP_ROOT.default":   workspaceDir, // generated nginx conf relies on this env var
				"EXECD_CONF.default": filepath.Join(workspaceDir, "nginx.conf"),
				"PORT.default":       "8080",
			}))
		})

		context("and nginx layer is being reused", func() {
			it.Before(func() {
				err := os.WriteFile(filepath.Join(layersDir, "nginx.toml"), []byte(`[metadata]
			dependency-sha = "some-sha"
			configure-bin-sha = "some-bin-sha"
			`), 0600)
				Expect(err).NotTo(HaveOccurred())
			})

			it.After(func() {
				Expect(os.RemoveAll(filepath.Join(layersDir, "nginx.toml"))).To(Succeed())
			})

			it("still generates the nginx.conf file", func() {
				_, err := build(buildContext)
				Expect(err).NotTo(HaveOccurred())
				Expect(configGenerator.GenerateCall.Receives.Config).To(Equal(nginx.Configuration{
					NGINXConfLocation: filepath.Join(workspaceDir, "nginx.conf"),
					WebServer:         "nginx",
					WebServerRoot:     "custom",
				}))
			})
		})
	})

	context("when the nginx.conf and included files need their permissions set", func() {
		it.Before(func() {
			Expect(os.WriteFile(filepath.Join(workspaceDir, "nginx.conf"), []byte("include custom.conf;"), 0600)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(workspaceDir, "custom.conf"), []byte("worker_processes 2;"), 0600)).To(Succeed())
		})

		it("modifies their permissions to be group read-writable", func() {
			_, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			info, err := os.Stat(filepath.Join(workspaceDir, "nginx.conf"))
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode().String()).To(Equal("-rw-rw----"))

			info, err = os.Stat(filepath.Join(workspaceDir, "custom.conf"))
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode().String()).To(Equal("-rw-rw----"))
		})
	})

	context("when there is no configuration file", func() {
		it.Before(func() {
			Expect(os.Remove(filepath.Join(workspaceDir, "nginx.conf"))).To(Succeed())
		})

		it("does not attempt to set permissions or processes", func() {
			result, err := build(buildContext)
			Expect(err).NotTo(HaveOccurred())

			Expect(len(result.Layers)).To(Equal(1))
			Expect(result.Launch.Processes).To(BeEmpty())
		})
	})

	context("failure cases", func() {
		context("when the dependency cannot be resolved", func() {
			it.Before(func() {
				dependencyService.ResolveCall.Returns.Error = errors.New("failed to resolve dependency")
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError("failed to resolve dependency"))
			})
		})

		context("unable to generate nginx.conf", func() {
			it.Before(func() {
				build = nginx.Build(
					nginx.Configuration{WebServer: "nginx"},
					dependencyService,
					configGenerator,
					calculator,
					sbomGenerator,
					scribe.NewEmitter(buffer),
					chronos.DefaultClock,
				)
				configGenerator.GenerateCall.Returns.Error = errors.New("some config error")
			})

			it("fails with descriptive error", func() {
				_, err := build(buildContext)

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("failed to generate nginx.conf : some config error")))
			})
		})

		context("when the layer cannot be retrieved", func() {
			it.Before(func() {
				err := os.WriteFile(filepath.Join(layersDir, "nginx.toml"), nil, 0000)
				Expect(err).NotTo(HaveOccurred())
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("failed to parse layer content metadata")))
			})
		})

		context("configure bin checksum fails", func() {
			it.Before(func() {
				calculator.SumCall.Returns.Error = errors.New("some-error")
			})

			it("fails with descriptive error", func() {
				_, err := build(buildContext)

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("checksum failed for file")))
			})
		})

		context("when the layer cannot be reset", func() {
			it.Before(func() {
				Expect(os.MkdirAll(filepath.Join(layersDir, "nginx", "something"), os.ModePerm)).To(Succeed())
				Expect(os.Chmod(filepath.Join(layersDir, "nginx"), 0500)).To(Succeed())
			})

			it.After(func() {
				Expect(os.Chmod(filepath.Join(layersDir, "nginx"), os.ModePerm)).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("could not remove file")))
			})
		})

		context("when the dependency cannot be installed", func() {
			it.Before(func() {
				dependencyService.DeliverCall.Returns.Error = errors.New("failed to deliver dependency")
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError("failed to deliver dependency"))
			})
		})

		context("when generating the SBOM returns an error", func() {
			it.Before(func() {
				sbomGenerator.GenerateFromDependencyCall.Returns.Error = errors.New("failed to generate SBOM")
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError(ContainSubstring("failed to generate SBOM")))
			})
		})

		context("when formatting the SBOM returns an error", func() {
			it.Before(func() {
				buildContext.BuildpackInfo.SBOMFormats = []string{"random-format"}
			})

			it("returns an error", func() {
				_, err := build(buildContext)
				Expect(err).To(MatchError("unsupported SBOM format: 'random-format'"))
			})
		})
	})
}
