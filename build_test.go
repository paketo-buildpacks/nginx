package nginx_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/paketo-buildpacks/nginx"
	"github.com/paketo-buildpacks/nginx/fakes"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/servicebindings"

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

		entryResolver     *fakes.EntryResolver
		dependencyService *fakes.DependencyService
		config            *fakes.ConfigGenerator
		calculator        *fakes.Calculator
		bindings          *fakes.Bindings

		buffer *bytes.Buffer

		build packit.BuildFunc
	)

	it.Before(func() {
		var err error
		layersDir, err = os.MkdirTemp("", "layers")
		Expect(err).NotTo(HaveOccurred())

		cnbPath, err = os.MkdirTemp("", "cnb")
		Expect(err).NotTo(HaveOccurred())

		workspaceDir, err = os.MkdirTemp("", "workspace")
		Expect(err).NotTo(HaveOccurred())

		buffer = bytes.NewBuffer(nil)

		entryResolver = &fakes.EntryResolver{}
		entryResolver.ResolveCall.Returns.BuildpackPlanEntry = packit.BuildpackPlanEntry{
			Name: "nginx",
			Metadata: map[string]interface{}{
				"version-source": "BP_NGINX_VERSION",
				"version":        "1.19.*",
				"launch":         true,
			},
		}
		entryResolver.MergeLayerTypesCall.Returns.Launch = true

		dependencyService = &fakes.DependencyService{}
		dependencyService.ResolveCall.Returns.Dependency = postal.Dependency{
			ID:           "nginx",
			SHA256:       "some-sha",
			Source:       "some-source",
			SourceSHA256: "some-source-sha",
			Stacks:       []string{"some-stack"},
			URI:          "some-uri",
			Version:      "1.19.8",
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

		bindings = &fakes.Bindings{}
		config = &fakes.ConfigGenerator{}
		calculator = &fakes.Calculator{}
		calculator.SumCall.Returns.String = "some-bin-sha"

		// create fake configure binary
		Expect(os.Mkdir(filepath.Join(cnbPath, "bin"), os.ModePerm)).To(Succeed())
		Expect(os.WriteFile(filepath.Join(cnbPath, "bin", "configure"), []byte("binary-contents"), 0600)).To(Succeed())

		Expect(os.WriteFile(filepath.Join(workspaceDir, "nginx.conf"), []byte("worker_processes 2;"), 0600)).To(Succeed())

		build = nginx.Build(nginx.BuildEnvironment{}, entryResolver, dependencyService, bindings, config, calculator, scribe.NewEmitter(buffer), chronos.DefaultClock)
	})

	it("does a build", func() {
		result, err := build(packit.BuildContext{
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
		})
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
			"EXECD_CONF.override": filepath.Join(workspaceDir, nginx.ConfFile),
		}))
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

		Expect(filepath.Join(layersDir, "nginx")).To(BeADirectory())
		Expect(filepath.Join(workspaceDir, "logs")).To(BeADirectory())

		Expect(entryResolver.ResolveCall.Receives.BuildpackPlanEntrySlice).To(Equal([]packit.BuildpackPlanEntry{
			{
				Name: "nginx",
				Metadata: map[string]interface{}{
					"version-source": "BP_NGINX_VERSION",
					"version":        "1.19.*",
					"launch":         true,
				},
			},
		}))

		Expect(dependencyService.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbPath, "buildpack.toml")))
		Expect(dependencyService.ResolveCall.Receives.Name).To(Equal("nginx"))
		Expect(dependencyService.ResolveCall.Receives.Version).To(Equal("1.19.*"))
		Expect(dependencyService.ResolveCall.Receives.Stack).To(Equal("some-stack"))

		Expect(dependencyService.DeliverCall.Receives.Dependency).To(Equal(
			postal.Dependency{
				ID:           "nginx",
				SHA256:       "some-sha",
				Source:       "some-source",
				SourceSHA256: "some-source-sha",
				Stacks:       []string{"some-stack"},
				URI:          "some-uri",
				Version:      "1.19.8",
			},
		))
		Expect(dependencyService.DeliverCall.Receives.CnbPath).To(Equal(cnbPath))
		Expect(dependencyService.DeliverCall.Receives.LayerPath).To(Equal(filepath.Join(layersDir, "nginx")))
		Expect(dependencyService.DeliverCall.Receives.PlatformPath).To(Equal("platform"))
		Expect(calculator.SumCall.CallCount).To(Equal(1))
	})

	context("when live reload is enabled", func() {
		it.Before(func() {
			build = nginx.Build(nginx.BuildEnvironment{Reload: true}, entryResolver, dependencyService, bindings, config, calculator, scribe.NewEmitter(buffer), chronos.DefaultClock)
		})

		it("uses watchexec to set the start command", func() {
			result, err := build(packit.BuildContext{
				CNBPath:    cnbPath,
				WorkingDir: workspaceDir,
				Stack:      "some-stack",
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
			})
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
				ID:           "nginx",
				SHA256:       "some-sha",
				Source:       "some-source",
				SourceSHA256: "some-source-sha",
				Stacks:       []string{"some-stack"},
				URI:          "some-uri",
				Version:      "some-bp-yml-version",
			}
			entryResolver.ResolveCall.Returns.BuildpackPlanEntry = packit.BuildpackPlanEntry{
				Name: "nginx",
				Metadata: map[string]interface{}{
					"version-source": "buildpack.yml",
					"version":        "some-bp-yml-version",
					"launch":         true,
				},
			}
		})

		it("does a build", func() {
			result, err := build(packit.BuildContext{
				BuildpackInfo: packit.BuildpackInfo{
					Name:    "Some Buildpack",
					Version: "1.2.3",
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
								"version-source": "buildpack.yml",
								"version":        "some-bp-yml-version",
								"launch":         true,
							},
						},
					},
				},
				Layers: packit.Layers{Path: layersDir},
			})
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

			Expect(filepath.Join(layersDir, "nginx")).To(BeADirectory())
			Expect(filepath.Join(workspaceDir, "logs")).To(BeADirectory())

			Expect(entryResolver.ResolveCall.Receives.BuildpackPlanEntrySlice).To(Equal([]packit.BuildpackPlanEntry{
				{
					Name: "nginx",
					Metadata: map[string]interface{}{
						"version-source": "buildpack.yml",
						"version":        "some-bp-yml-version",
						"launch":         true,
					},
				},
			}))

			Expect(dependencyService.ResolveCall.Receives.Path).To(Equal(filepath.Join(cnbPath, "buildpack.toml")))
			Expect(dependencyService.ResolveCall.Receives.Name).To(Equal("nginx"))
			Expect(dependencyService.ResolveCall.Receives.Version).To(Equal("some-bp-yml-version"))
			Expect(dependencyService.ResolveCall.Receives.Stack).To(Equal("some-stack"))

			Expect(dependencyService.DeliverCall.Receives.Dependency).To(Equal(
				postal.Dependency{
					ID:           "nginx",
					SHA256:       "some-sha",
					Source:       "some-source",
					SourceSHA256: "some-source-sha",
					Stacks:       []string{"some-stack"},
					URI:          "some-uri",
					Version:      "some-bp-yml-version",
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

			entryResolver.MergeLayerTypesCall.Returns.Launch = true
		})
		it.After(func() {
			Expect(os.RemoveAll(filepath.Join(layersDir, "nginx.toml"))).To(Succeed())
		})

		it("does not re-build the nginx layer", func() {
			result, err := build(packit.BuildContext{
				CNBPath:    cnbPath,
				WorkingDir: workspaceDir,
				Stack:      "some-stack",
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{
						{
							Name: "nginx",
							Metadata: map[string]interface{}{
								"version-source": "BP_NGINX_VERSION",
								"version":        "1.17.*",
								"launch":         true,
							},
						},
					},
				},
				Layers: packit.Layers{Path: layersDir},
			})

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
			buildEnv := nginx.BuildEnvironment{
				ConfLocation: "some-relative-path/nginx.conf",
			}
			build = nginx.Build(buildEnv, entryResolver, dependencyService, bindings, config, calculator, scribe.NewEmitter(buffer), chronos.DefaultClock)

			Expect(os.Mkdir(filepath.Join(workspaceDir, "some-relative-path"), os.ModePerm)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(workspaceDir, "some-relative-path", "nginx.conf"), []byte("worker_processes 2;"), 0600)).To(Succeed())
		})

		it("assumes path is relative to /workspace", func() {
			result, err := build(packit.BuildContext{
				CNBPath:    cnbPath,
				WorkingDir: workspaceDir,
				Stack:      "some-stack",
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
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Launch.Processes[0].Args).To(Equal([]string{
				"-p", workspaceDir,
				"-c", filepath.Join(workspaceDir, "some-relative-path", "nginx.conf"),
				"-g", "pid /tmp/nginx.pid;",
			}))
			Expect(result.Layers[0].LaunchEnv).To(Equal(packit.Environment{
				"EXECD_CONF.override": filepath.Join(workspaceDir, "some-relative-path/nginx.conf"),
			}))
		})
	})

	context("when BP_NGINX_CONF_LOCATION is set to an absolute path", func() {
		it.Before(func() {
			buildEnv := nginx.BuildEnvironment{
				ConfLocation: filepath.Join(workspaceDir, "some-absolute-path", "nginx.conf"),
			}
			build = nginx.Build(buildEnv, entryResolver, dependencyService, bindings, config, calculator, scribe.NewEmitter(buffer), chronos.DefaultClock)

			Expect(os.Mkdir(filepath.Join(workspaceDir, "some-absolute-path"), os.ModePerm)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(workspaceDir, "some-absolute-path", "nginx.conf"), []byte("worker_processes 2;"), 0600)).To(Succeed())
		})

		it("uses the location as-is", func() {
			result, err := build(packit.BuildContext{
				CNBPath:    cnbPath,
				WorkingDir: workspaceDir,
				Stack:      "some-stack",
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
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Launch.Processes[0].Args).To(Equal([]string{
				"-p", workspaceDir,
				"-c", filepath.Join(workspaceDir, "some-absolute-path", "nginx.conf"),
				"-g", "pid /tmp/nginx.pid;",
			}))
			Expect(result.Layers[0].LaunchEnv).To(Equal(packit.Environment{
				"EXECD_CONF.override": filepath.Join(workspaceDir, "some-absolute-path", "nginx.conf"),
			}))
		})
	})

	context("when BP_WEB_SERVER=nginx in the build env", func() {
		it.Before(func() {
			buildEnv := nginx.BuildEnvironment{
				WebServer:     "nginx",
				WebServerRoot: "custom",
			}
			build = nginx.Build(buildEnv, entryResolver, dependencyService, bindings, config, calculator, scribe.NewEmitter(buffer), chronos.DefaultClock)
		})

		it("generates a basic nginx.conf and passes env var configuration into template generator", func() {
			result, err := build(packit.BuildContext{
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
			})
			Expect(err).NotTo(HaveOccurred())
			Expect(config.GenerateCall.Receives.Env).To(Equal(nginx.BuildEnvironment{
				ConfLocation:  filepath.Join(workspaceDir, "nginx.conf"),
				WebServer:     "nginx",
				WebServerRoot: "custom",
			}))

			Expect(result.Layers[0].LaunchEnv).To(Equal(packit.Environment{
				"APP_ROOT.override":   workspaceDir, // generated nginx conf relies on this env var
				"EXECD_CONF.override": filepath.Join(workspaceDir, "nginx.conf"),
			}))
		})

		context("and a well-formed htpasswd service binding is provided", func() {
			it.Before(func() {
				buildEnv := nginx.BuildEnvironment{
					WebServer: "nginx",
				}
				bindings.ResolveCall.Returns.BindingSlice = []servicebindings.Binding{
					{
						Name: "first",
						Type: "htpasswd",
						Path: "/path/to/binding/",
						Entries: map[string]*servicebindings.Entry{
							".htpasswd": servicebindings.NewEntry("/path/to/binding/.htpasswd"),
						},
					},
				}
				build = nginx.Build(buildEnv, entryResolver, dependencyService, bindings, config, calculator, scribe.NewEmitter(buffer), chronos.DefaultClock)
			})
			it("passes the binding path into the conf generator", func() {
				_, err := build(packit.BuildContext{
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
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(config.GenerateCall.Receives.Env).To(Equal(nginx.BuildEnvironment{
					ConfLocation:  filepath.Join(workspaceDir, nginx.ConfFile),
					WebServer:     "nginx",
					BasicAuthFile: "/path/to/binding/.htpasswd",
				}))
			})
		})

		context("and nginx layer is being reused", func() {
			it.Before(func() {
				err := os.WriteFile(filepath.Join(layersDir, "nginx.toml"), []byte(`[metadata]
			dependency-sha = "some-sha"
			configure-bin-sha = "some-bin-sha"
			`), 0600)
				Expect(err).NotTo(HaveOccurred())

				entryResolver.MergeLayerTypesCall.Returns.Launch = true
			})

			it.After(func() {
				Expect(os.RemoveAll(filepath.Join(layersDir, "nginx.toml"))).To(Succeed())
			})

			it("still generates the nginx.conf file", func() {
				_, err := build(packit.BuildContext{
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
				})
				Expect(err).NotTo(HaveOccurred())
				Expect(config.GenerateCall.Receives.Env).To(Equal(nginx.BuildEnvironment{
					ConfLocation:  filepath.Join(workspaceDir, "nginx.conf"),
					WebServer:     "nginx",
					WebServerRoot: "custom",
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
			_, err := build(packit.BuildContext{
				CNBPath:    cnbPath,
				WorkingDir: workspaceDir,
				Stack:      "some-stack",
				Plan: packit.BuildpackPlan{
					Entries: []packit.BuildpackPlanEntry{{Name: "nginx"}},
				},
				Layers: packit.Layers{Path: layersDir},
			})
			Expect(err).NotTo(HaveOccurred())

			info, err := os.Stat(filepath.Join(workspaceDir, "nginx.conf"))
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode().String()).To(Equal("-rw-rw----"))

			info, err = os.Stat(filepath.Join(workspaceDir, "custom.conf"))
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode().String()).To(Equal("-rw-rw----"))
		})
	})

	context("failure cases", func() {
		context("when the dependency cannot be resolved", func() {
			it.Before(func() {
				dependencyService.ResolveCall.Returns.Error = errors.New("failed to resolve dependency")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath: cnbPath,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "nginx"},
						},
					},
					Layers: packit.Layers{Path: layersDir},
					Stack:  "some-stack",
				})
				Expect(err).To(MatchError("failed to resolve dependency"))
			})
		})

		context("unable to create log directory", func() {
			it.Before(func() {
				Expect(os.Chmod(workspaceDir, 0000))
			})

			it.After(func() {
				Expect(os.Chmod(workspaceDir, os.ModePerm))
			})

			it("fails with descriptive error", func() {
				_, err := build(packit.BuildContext{
					CNBPath:    cnbPath,
					WorkingDir: workspaceDir,
					Stack:      "some-stack",
				})

				Expect(err).To(HaveOccurred())
				logsDir := filepath.Join(workspaceDir, "logs")
				Expect(err).To(MatchError(ContainSubstring(fmt.Sprintf("failed to create logs dir : mkdir %s", logsDir))))
			})
		})

		context("unable to resolve .htpasswd service binding", func() {
			it.Before(func() {
				build = nginx.Build(nginx.BuildEnvironment{WebServer: "nginx"}, entryResolver, dependencyService, bindings, config, calculator, scribe.NewEmitter(buffer), chronos.DefaultClock)
				bindings.ResolveCall.Returns.Error = errors.New("some bindings error")
			})

			it("fails with descriptive error", func() {
				_, err := build(packit.BuildContext{
					CNBPath:    cnbPath,
					WorkingDir: workspaceDir,
					Stack:      "some-stack",
				})

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("some bindings error")))
			})
		})

		context("there's more than one htpasswd service binding", func() {
			it.Before(func() {
				bindings.ResolveCall.Returns.BindingSlice = []servicebindings.Binding{
					{
						Name: "first",
						Type: "htpasswd",
						Path: "/path/to/binding/",
						Entries: map[string]*servicebindings.Entry{
							".htpasswd": servicebindings.NewEntry("/path/to/binding/.htpasswd"),
						},
					},
					{
						Name: "second",
						Type: "htpasswd",
						Path: "/path/to/binding/",
						Entries: map[string]*servicebindings.Entry{
							".htpasswd": servicebindings.NewEntry("/path/to/binding/.htpasswd"),
						},
					},
				}
				build = nginx.Build(nginx.BuildEnvironment{WebServer: "nginx"}, entryResolver, dependencyService, bindings, config, calculator, scribe.NewEmitter(buffer), chronos.DefaultClock)
			})

			it("fails with descriptive error", func() {
				_, err := build(packit.BuildContext{
					CNBPath:    cnbPath,
					WorkingDir: workspaceDir,
					Stack:      "some-stack",
				})

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("binding resolver found more than one binding of type 'htpasswd'")))
			})
		})

		context("the htpasswd service binding is malformed", func() {
			it.Before(func() {
				bindings.ResolveCall.Returns.BindingSlice = []servicebindings.Binding{
					{
						Name: "first",
						Type: "htpasswd",
						Path: "/path/to/binding/",
						Entries: map[string]*servicebindings.Entry{
							"some-irrelevant-file": servicebindings.NewEntry("some-irrelevant-path"),
						},
					},
				}
				build = nginx.Build(nginx.BuildEnvironment{WebServer: "nginx"}, entryResolver, dependencyService, bindings, config, calculator, scribe.NewEmitter(buffer), chronos.DefaultClock)
			})

			it("fails with descriptive error", func() {
				_, err := build(packit.BuildContext{
					CNBPath:    cnbPath,
					WorkingDir: workspaceDir,
					Stack:      "some-stack",
				})

				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ContainSubstring("binding of type 'htpasswd' does not contain required entry '.htpasswd'")))
			})
		})

		context("unable to generate nginx.conf", func() {
			it.Before(func() {
				build = nginx.Build(nginx.BuildEnvironment{WebServer: "nginx"}, entryResolver, dependencyService, bindings, config, calculator, scribe.NewEmitter(buffer), chronos.DefaultClock)
				config.GenerateCall.Returns.Error = errors.New("some config error")
			})

			it("fails with descriptive error", func() {
				_, err := build(packit.BuildContext{
					CNBPath:    cnbPath,
					WorkingDir: workspaceDir,
					Stack:      "some-stack",
				})

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
				_, err := build(packit.BuildContext{
					CNBPath:    cnbPath,
					WorkingDir: workspaceDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "nginx"},
						},
					},
					Layers: packit.Layers{Path: layersDir},
					Stack:  "some-stack",
				})
				Expect(err).To(MatchError(ContainSubstring("failed to parse layer content metadata")))
			})
		})

		context("configure bin checksum fails", func() {
			it.Before(func() {
				calculator.SumCall.Returns.Error = errors.New("some-error")
			})

			it("fails with descriptive error", func() {
				_, err := build(packit.BuildContext{
					CNBPath:    cnbPath,
					WorkingDir: workspaceDir,
					Stack:      "some-stack",
					Layers:     packit.Layers{Path: layersDir},
				})

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
				_, err := build(packit.BuildContext{
					CNBPath:    cnbPath,
					WorkingDir: workspaceDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "nginx"},
						},
					},
					Layers: packit.Layers{Path: layersDir},
					Stack:  "some-stack",
				})
				Expect(err).To(MatchError(ContainSubstring("could not remove file")))
			})
		})

		context("when the dependency cannot be installed", func() {
			it.Before(func() {
				dependencyService.DeliverCall.Returns.Error = errors.New("failed to deliver dependency")
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath:    cnbPath,
					WorkingDir: workspaceDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "nginx"},
						},
					},
					Layers: packit.Layers{Path: layersDir},
					Stack:  "some-stack",
				})
				Expect(err).To(MatchError("failed to deliver dependency"))
			})
		})

		context("when the exec.d binary cannot be copied", func() {
			it.Before(func() {
				Expect(os.Remove(filepath.Join(cnbPath, "bin", "configure"))).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath:    cnbPath,
					WorkingDir: workspaceDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "nginx"},
						},
					},
					Layers: packit.Layers{Path: layersDir},
					Stack:  "some-stack",
				})
				Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
			})
		})

		context("when the included confs cannot be fetched", func() {
			it.Before(func() {
				Expect(os.Remove(filepath.Join(workspaceDir, "nginx.conf"))).To(Succeed())
			})

			it("returns an error", func() {
				_, err := build(packit.BuildContext{
					CNBPath:    cnbPath,
					WorkingDir: workspaceDir,
					Plan: packit.BuildpackPlan{
						Entries: []packit.BuildpackPlanEntry{
							{Name: "nginx"},
						},
					},
					Layers: packit.Layers{Path: layersDir},
					Stack:  "some-stack",
				})
				Expect(err).To(MatchError(ContainSubstring("failed to find configuration files")))
				Expect(err).To(MatchError(ContainSubstring("no such file or directory")))
			})
		})
	})
}
