package nginx

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/template"

	"github.com/paketo-buildpacks/packit/v2/scribe"
)

//go:embed assets/default.conf
var DefaultConfigTemplate string

type DefaultConfigGenerator struct {
	logs scribe.Emitter
}

func NewDefaultConfigGenerator(logs scribe.Emitter) DefaultConfigGenerator {
	return DefaultConfigGenerator{logs: logs}
}

func (g DefaultConfigGenerator) Generate(config Configuration) error {
	g.logs.Process("Generating %s", config.NGINXConfLocation)
	t := template.Must(template.New("template.conf").Delims("$((", "))").Parse(DefaultConfigTemplate))

	if !filepath.IsAbs(config.WebServerRoot) {
		config.WebServerRoot = filepath.Join(`{{ env "APP_ROOT" }}`, config.WebServerRoot)
	}

	g.logs.Subprocess("Setting server root directory to '%s'", config.WebServerRoot)

	if config.WebServerLocationPath == "" {
		config.WebServerLocationPath = "/"
	}

	g.logs.Subprocess("Setting server location path to '%s'", config.WebServerLocationPath)

	if config.WebServerEnablePushState {
		g.logs.Subprocess("Enabling push state routing")
	}

	if config.WebServerForceHTTPS {
		g.logs.Subprocess("Setting server to redirect HTTP requests to HTTPS")
	}

	if config.BasicAuthFile != "" {
		g.logs.Subprocess("Enabling basic authentication with .htpasswd credentials")
	}

	if config.NGINXStubStatusPort != "" {
		g.logs.Subprocess("Enabling basic status information with stub_status module")
	}

	if config.WebServerIncludeFilePath != "" {
		g.logs.Subprocess("Enabling including custom config")
	}

	g.logs.Break()

	var b bytes.Buffer
	err := t.Execute(&b, config)
	if err != nil {
		// not tested
		return err
	}

	f, err := os.OpenFile(config.NGINXConfLocation, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", config.NGINXConfLocation, err)
	}
	defer f.Close()

	_, err = io.Copy(f, &b)
	if err != nil {
		// not tested
		return err
	}
	return nil
}
