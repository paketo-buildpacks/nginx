package main

import (
	"os"

	"github.com/paketo-buildpacks/nginx"
	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/cargo"
	"github.com/paketo-buildpacks/packit/chronos"
	"github.com/paketo-buildpacks/packit/fs"
	"github.com/paketo-buildpacks/packit/postal"
)

func main() {
	parser := nginx.NewParser()
	transport := cargo.NewTransport()
	dependencyService := postal.NewService(transport)
	entryResolver := nginx.NewPlanEntryResolver()
	logger := nginx.NewLogEmitter(os.Stdout)
	profileWriter := nginx.NewProfileWriter(logger)
	calculator := fs.NewChecksumCalculator()

	packit.Run(
		nginx.Detect(parser),
		nginx.Build(
			entryResolver,
			dependencyService,
			profileWriter,
			calculator,
			logger,
			chronos.DefaultClock,
		),
	)
}
