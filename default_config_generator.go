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

func (g DefaultConfigGenerator) Generate(templateSource, destination, rootDir string) error {
	if _, err := os.Stat(templateSource); err != nil {
		return fmt.Errorf("failed to locate nginx.conf template: %w", err)
	}
	t := template.Must(template.New("template.conf").Delims("$((", "))").ParseFiles(templateSource))
	data := nginxConfig{
		Root: `{{ env "APP_ROOT" }}/public`,
	}

	if rootDir != "" {
		if filepath.IsAbs(rootDir) {
			data.Root = rootDir
		} else {
			data.Root = filepath.Join(`{{ env "APP_ROOT" }}`, rootDir)
		}
	}

	var b bytes.Buffer
	err := t.Execute(&b, data)
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

type nginxConfig struct {
	Root string
}
