package nginx_test

import (
	"errors"
	"testing"

	"github.com/paketo-buildpacks/nginx"
	"github.com/paketo-buildpacks/nginx/fakes"
	"github.com/paketo-buildpacks/packit/v2/servicebindings"
	"github.com/sclevine/spec"

	. "github.com/onsi/gomega"
)

func testConfiguration(t *testing.T, context spec.G, it spec.S) {
	var Expect = NewWithT(t).Expect

	context("LoadConfiguration", func() {
		var bindingsResolver *fakes.BindingsResolver

		it.Before(func() {
			bindingsResolver = &fakes.BindingsResolver{}
			bindingsResolver.ResolveOneCall.Returns.Binding = servicebindings.Binding{
				Name: "first",
				Type: "htpasswd",
				Path: "/path/to/binding/",
				Entries: map[string]*servicebindings.Entry{
					".htpasswd": servicebindings.NewEntry("/path/to/binding/.htpasswd"),
				},
			}
		})

		it("loads the buildpack configuration", func() {
			config, err := nginx.LoadConfiguration([]string{
				"BP_NGINX_CONF_LOCATION=some-conf-location",
				"BP_NGINX_VERSION=some-nginx-version",
				"BP_LIVE_RELOAD_ENABLED=true",
				"BP_WEB_SERVER=some-web-server",
				"BP_WEB_SERVER_FORCE_HTTPS=true",
				"BP_WEB_SERVER_ENABLE_PUSH_STATE=true",
				"BP_WEB_SERVER_ROOT=some-root",
				"BP_WEB_SERVER_LOCATION_PATH=some-location-path",
				"BP_WEB_SERVER_INCLUDE_FILE_PATH=some-location-include",
				"BP_NGINX_STUB_STATUS_PORT=8083",
			}, bindingsResolver, "some-platform-path")
			Expect(err).NotTo(HaveOccurred())
			Expect(config).To(Equal(nginx.Configuration{
				NGINXConfLocation:        "some-conf-location",
				NGINXVersion:             "some-nginx-version",
				LiveReloadEnabled:        true,
				WebServer:                "some-web-server",
				WebServerForceHTTPS:      true,
				WebServerEnablePushState: true,
				WebServerRoot:            "some-root",
				WebServerLocationPath:    "some-location-path",
				WebServerIncludeFilePath: "some-location-include",
				NGINXStubStatusPort:      "8083",
			}))
		})

		context("when no BP_NGINX_CONF_LOCATION is set", func() {
			it("assigns a default", func() {
				config, err := nginx.LoadConfiguration(nil, bindingsResolver, "some-platform-path")
				Expect(err).NotTo(HaveOccurred())
				Expect(config.NGINXConfLocation).To(Equal("./nginx.conf"))
			})
		})

		context("when no BP_WEB_SERVER_ROOT is set", func() {
			it("assigns a default", func() {
				config, err := nginx.LoadConfiguration(nil, bindingsResolver, "some-platform-path")
				Expect(err).NotTo(HaveOccurred())
				Expect(config.WebServerRoot).To(Equal("./public"))
			})
		})

		context("when BP_WEB_SERVER=nginx", func() {
			context("when a .htpasswd service binding is provided", func() {
				it("loads the binding path", func() {
					config, err := nginx.LoadConfiguration([]string{"BP_WEB_SERVER=nginx"}, bindingsResolver, "some-platform-path")
					Expect(err).NotTo(HaveOccurred())
					Expect(config.BasicAuthFile).To(Equal("/path/to/binding/.htpasswd"))
				})
			})

			context("when a .htpasswd service binding is NOT provided", func() {
				it.Before(func() {
					bindingsResolver.ResolveOneCall.Returns.Error = errors.New("expected exactly 1")
				})

				it("does not load the binding path", func() {
					config, err := nginx.LoadConfiguration([]string{"BP_WEB_SERVER=nginx"}, bindingsResolver, "some-platform-path")
					Expect(err).NotTo(HaveOccurred())
					Expect(config.BasicAuthFile).To(Equal(""))
				})
			})
		})

		context("failure cases", func() {
			context("when the environment cannot be parsed", func() {
				it("returns an error", func() {
					_, err := nginx.LoadConfiguration([]string{
						"this is not a parseable environment variable",
					}, bindingsResolver, "some-platform-path")
					Expect(err).To(MatchError("failed to parse environment variables: items in environ must have format key=value"))
				})
			})

			context("when resolving the .htpasswd service binding fails", func() {
				it.Before(func() {
					bindingsResolver.ResolveOneCall.Returns.Error = errors.New("some bindings error")
				})

				it("returns an error", func() {
					_, err := nginx.LoadConfiguration([]string{"BP_WEB_SERVER=nginx"}, bindingsResolver, "some-platform-path")
					Expect(err).To(MatchError(ContainSubstring("some bindings error")))
				})
			})

			context("when the .htpasswd service binding is malformed", func() {
				it.Before(func() {
					bindingsResolver.ResolveOneCall.Returns.Binding = servicebindings.Binding{
						Name: "first",
						Type: "htpasswd",
						Path: "/path/to/binding/",
						Entries: map[string]*servicebindings.Entry{
							"some-irrelevant-file": servicebindings.NewEntry("some-irrelevant-path"),
						},
					}
				})

				it("returns an error", func() {
					_, err := nginx.LoadConfiguration([]string{"BP_WEB_SERVER=nginx"}, bindingsResolver, "some-platform-path")
					Expect(err).To(MatchError("binding of type 'htpasswd' does not contain required entry '.htpasswd'"))
				})
			})
		})
	})
}
