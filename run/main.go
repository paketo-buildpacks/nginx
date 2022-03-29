package main

import (
	"os"

	"github.com/paketo-buildpacks/nginx"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/cargo"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/draft"
	"github.com/paketo-buildpacks/packit/fs"
	"github.com/paketo-buildpacks/packit/postal"
	"github.com/paketo-buildpacks/packit/scribe"
)

func main() {
	logger := scribe.NewEmitter(os.Stdout)

	packit.Run(
		nginx.Detect(nginx.NewParser()),
		nginx.Build(
			draft.NewPlanner(),
			postal.NewService(cargo.NewTransport()),
			fs.NewChecksumCalculator(),
			logger,
			chronos.DefaultClock,
		),
	)
}
