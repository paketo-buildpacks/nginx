package internal

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"text/template"
)

func Run(mainConf, localModulePath, globalModulePath string) error {
	log.SetFlags(0)

	if _, err := os.Stat(mainConf); err != nil {
		return nil
	}

	confs, err := getIncludedConfs(mainConf)
	if err != nil {
		return err
	}

	confs = append(confs, mainConf)
	templFuncs := template.FuncMap{
		"env": os.Getenv,
		"tempDir": func() string {
			return os.TempDir()
		},
		"port": func() string {
			return os.Getenv("PORT")
		},
		"module": func(name string) (string, error) {
			module := filepath.Join(localModulePath, name+".so")

			_, err := os.Stat(module)
			if err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					return "", fmt.Errorf("failed to execute module function: %s", err)
				}

				module = filepath.Join(globalModulePath, name+".so")
			}

			return fmt.Sprintf("load_module %s;", module), nil
		},
	}

	for _, conf := range confs {
		content, err := os.ReadFile(conf)
		if err != nil {
			return fmt.Errorf("failed to read config file: %s", err)
		}

		tmpl, err := template.New("configure").Option("missingkey=zero").Funcs(templFuncs).Parse(string(content))
		if err != nil {
			return fmt.Errorf("failed to parse template: %s", err)
		}

		buffer := bytes.NewBuffer(nil)
		err = tmpl.Execute(buffer, nil)
		if err != nil {
			return fmt.Errorf("failed to execute template: %w", err)
		}

		if hex.EncodeToString(sha256.New().Sum(content)) != hex.EncodeToString(sha256.New().Sum(buffer.Bytes())) {
			err = os.WriteFile(conf, buffer.Bytes(), 0600)
			if err != nil {
				return fmt.Errorf("failed to overwrite template: %w", err)
			}
		}
	}

	return nil
}

var IncludeConfRegexp = regexp.MustCompile(`include\s+(\S*.conf);`)

func getIncludedConfs(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("could not read config file (%s): %w", path, err)
	}

	var files []string
	for _, match := range IncludeConfRegexp.FindAllStringSubmatch(string(content), -1) {
		if len(match) == 2 {
			glob := match[1]
			if !filepath.IsAbs(glob) {
				glob = filepath.Join(filepath.Dir(path), glob)
			}

			matches, err := filepath.Glob(glob)
			if err != nil {
				return nil, fmt.Errorf("failed to get 'include' files for %s: %w", glob, err)
			}

			files = append(files, matches...)
		}
	}

	return files, nil
}
