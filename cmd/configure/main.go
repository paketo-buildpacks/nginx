package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"text/template"
)

func main() {
	log.SetFlags(0)

	mainConf := os.Args[1]
	body, err := ioutil.ReadFile(mainConf)
	if err != nil {
		log.Fatalf("Could not read config file (%s): %s", mainConf, err)
	}

	confs, err := getIncludedConfs(string(body), filepath.Dir(mainConf))
	if err != nil {
		log.Fatalf(err.Error())
	}
	confs = append(confs, mainConf)
	templFuncs := template.FuncMap{
		"env": os.Getenv,
		"port": func() string {
			return os.Getenv("PORT")
		},
		"module": func(name string) string {
			module := filepath.Join(os.Args[2], name+".so")

			_, err := os.Stat(module)
			if err != nil {
				if !errors.Is(err, os.ErrNotExist) {
					log.Fatalf("failed to execute module function: %s", err)
				}

				module = filepath.Join(os.Args[3], name+".so")
			}

			return fmt.Sprintf("load_module %s;", module)
		},
	}

	for _, conf := range confs {
		tmpl, err := template.New("configure").
			Option("missingkey=zero").
			Funcs(templFuncs).
			ParseFiles(conf)
		if err != nil {
			log.Fatalf("failed to parse template: %s", err)
		}

		file, err := os.Create(conf)
		if err != nil {
			log.Fatalf("failed to create %s: %s", filepath.Base(conf), err)
		}
		defer file.Close()

		if err := tmpl.ExecuteTemplate(file, filepath.Base(conf), nil); err != nil {
			log.Fatalf("failed to execute template: %s", err)
		}
	}
}

func getIncludedConfs(confText string, confDir string) ([]string, error) {
	includeFiles := []string{}
	includeRe := regexp.MustCompile(`include\s+(\S*.conf);`)
	matches := includeRe.FindAllStringSubmatch(confText, -1)
	for _, v := range matches {
		if len(v) == 2 {
			conf := v[1]
			if !filepath.IsAbs(conf) {
				conf = filepath.Join(confDir, conf)
			}
			matchFiles, err := filepath.Glob(conf)
			if err != nil {
				return nil, fmt.Errorf("failed to get 'include' files for %s: %w", conf, err)
			}

			if len(matchFiles) == 0 {
				return nil, fmt.Errorf("failed to get 'include' files for %s: no matching files exist", conf)
			}

			includeFiles = append(includeFiles, matchFiles...)
		}
	}

	return includeFiles, nil
}
