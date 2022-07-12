package internal

import (
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
		tmpl, err := template.New("configure").Option("missingkey=zero").Funcs(templFuncs).ParseFiles(conf)
		if err != nil {
			return fmt.Errorf("failed to parse template: %s", err)
		}

		file, err := os.Create(conf)
		if err != nil {
			return fmt.Errorf("failed to create %s: %s", filepath.Base(conf), err)
		}
		defer file.Close()

		if err := tmpl.ExecuteTemplate(file, filepath.Base(conf), nil); err != nil {
			return fmt.Errorf("failed to execute template: %s", err)
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
