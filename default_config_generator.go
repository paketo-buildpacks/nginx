package nginx

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/template"

	"github.com/paketo-buildpacks/packit/v2/scribe"
)

type DefaultConfigGenerator struct {
	logs scribe.Emitter
}

func NewDefaultConfigGenerator(logs scribe.Emitter) DefaultConfigGenerator {
	return DefaultConfigGenerator{
		logs: logs,
	}
}

func (g DefaultConfigGenerator) Generate(env BuildEnvironment) error {
	g.logs.Process("Generating %s", env.ConfLocation)
	t := template.Must(template.New("template.conf").Delims("$((", "))").Parse(defaultConf))

	if env.WebServerRoot == "" {
		env.WebServerRoot = `./public`
	}

	if !filepath.IsAbs(env.WebServerRoot) {
		env.WebServerRoot = filepath.Join(`{{ env "APP_ROOT" }}`, env.WebServerRoot)
	}

	g.logs.Subprocess("Setting server root directory to '%s'", env.WebServerRoot)

	if env.WebServerPushStateEnabled {
		g.logs.Subprocess("Enabling push state routing")
	}

	if env.WebServerForceHTTPS {
		g.logs.Subprocess("Setting server to redirect HTTP requests to HTTPS")
	}

	if env.BasicAuthFile != "" {
		g.logs.Subprocess("Enabling basic authentication with .htpasswd credentials")
	}

	g.logs.Break()

	var b bytes.Buffer
	err := t.Execute(&b, env)
	if err != nil {
		// not tested
		return err
	}

	f, err := os.OpenFile(env.ConfLocation, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", env.ConfLocation, err)
	}
	defer f.Close()

	_, err = io.Copy(f, &b)
	if err != nil {
		// not tested
		return err
	}
	return nil
}
