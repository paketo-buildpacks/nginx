package nginx

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"text/template"
)

type DefaultConfigGenerator struct {
}

func NewDefaultConfigGenerator() DefaultConfigGenerator {
	return DefaultConfigGenerator{}
}

func (g DefaultConfigGenerator) Generate(templateSource, destination string, env BuildEnvironment) error {
	if _, err := os.Stat(templateSource); err != nil {
		return fmt.Errorf("failed to locate nginx.conf template: %w", err)
	}
	t := template.Must(template.New("template.conf").Delims("$((", "))").ParseFiles(templateSource))

	if env.WebServerRoot == "" {
		env.WebServerRoot = `./public`
	}

	if !filepath.IsAbs(env.WebServerRoot) {
		env.WebServerRoot = filepath.Join(`{{ env "APP_ROOT" }}`, env.WebServerRoot)
	}

	var b bytes.Buffer
	err := t.Execute(&b, env)
	if err != nil {
		// not tested
		return err
	}

	f, err := os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create %s: %w", destination, err)
	}
	defer f.Close()

	_, err = io.Copy(f, &b)
	if err != nil {
		// not tested
		return err
	}
	return nil
}
