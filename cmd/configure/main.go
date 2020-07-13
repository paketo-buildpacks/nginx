package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"text/template"
)

func main() {
	log.SetFlags(0)

	tmpl, err := template.New("configure").
		Option("missingkey=zero").
		Funcs(template.FuncMap{
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
		}).
		ParseFiles(os.Args[1])
	if err != nil {
		log.Fatalf("failed to parse template: %s", err)
	}

	file, err := os.Create(os.Args[1])
	if err != nil {
		log.Fatalf("failed to create nginx.conf: %s", err)
	}
	defer file.Close()

	if err := tmpl.ExecuteTemplate(file, filepath.Base(os.Args[1]), nil); err != nil {
		log.Fatalf("failed to execute template: %s", err)
	}
}
