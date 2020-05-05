package main

import (
	"github.com/cloudfoundry/packit"
	"github.com/paketo-buildpacks/nginx/nginx"
)

func main() {
	parser := nginx.NewParser()
	packit.Detect(nginx.Detect(parser))
}
