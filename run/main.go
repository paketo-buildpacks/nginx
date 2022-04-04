package main

import (
	"os"

	"github.com/paketo-buildpacks/nginx"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/draft"
	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/scribe"
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
