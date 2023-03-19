package nginx

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Netflix/go-env"
	"github.com/paketo-buildpacks/packit/v2/servicebindings"
)

//go:generate faux --interface BindingsResolver --output fakes/bindings_resolver.go
type BindingsResolver interface {
	ResolveOne(typ, provider, platformDir string) (servicebindings.Binding, error)
}

type Configuration struct {
	NGINXConfLocation        string `env:"BP_NGINX_CONF_LOCATION"`
	NGINXVersion             string `env:"BP_NGINX_VERSION"`
	LiveReloadEnabled        bool   `env:"BP_LIVE_RELOAD_ENABLED"`
	WebServer                string `env:"BP_WEB_SERVER"`
	WebServerForceHTTPS      bool   `env:"BP_WEB_SERVER_FORCE_HTTPS"`
	WebServerEnablePushState bool   `env:"BP_WEB_SERVER_ENABLE_PUSH_STATE"`
	WebServerRoot            string `env:"BP_WEB_SERVER_ROOT"`
	WebServerLocationPath    string `env:"BP_WEB_SERVER_LOCATION_PATH"`
	NGINXStubStatusPort      string `env:"BP_NGINX_STUB_STATUS_PORT"`

	BasicAuthFile string
}

func LoadConfiguration(environ []string, bindingsResolver BindingsResolver, platformPath string) (Configuration, error) {
	es, err := env.EnvironToEnvSet(environ)
	if err != nil {
		return Configuration{}, fmt.Errorf("failed to parse environment variables: %w", err)
	}

	configuration := Configuration{
		NGINXConfLocation: "./nginx.conf",
		WebServerRoot:     "./public",
	}

	err = env.Unmarshal(es, &configuration)
	if err != nil {
		return Configuration{}, err
	}

	if configuration.WebServer == "nginx" {
		binding, err := bindingsResolver.ResolveOne("htpasswd", "", platformPath)
		if err != nil && !strings.Contains(err.Error(), "expected exactly 1") {
			return Configuration{}, err
		}

		if err == nil {
			if _, ok := binding.Entries[".htpasswd"]; !ok {
				return Configuration{}, errors.New("binding of type 'htpasswd' does not contain required entry '.htpasswd'")
			}

			configuration.BasicAuthFile = filepath.Join(binding.Path, ".htpasswd")
		}
	}

	return configuration, nil
}
