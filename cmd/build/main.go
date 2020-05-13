package main

import (
	"os"
	"time"

	"github.com/paketo-buildpacks/packit"
	"github.com/paketo-buildpacks/packit/cargo"
	"github.com/paketo-buildpacks/packit/fs"
	"github.com/paketo-buildpacks/packit/postal"
	"github.com/paketo-buildpacks/nginx/nginx"
)

func main() {
	transport := cargo.NewTransport()
	dependencyService := postal.NewService(transport)
	entryResolver := nginx.NewPlanEntryResolver()
	logger := nginx.NewLogEmitter(os.Stdout)
	profileWriter := nginx.NewProfileWriter(logger)
	clock := nginx.NewClock(time.Now)

	calculator := fs.NewChecksumCalculator()
	packit.Build(nginx.Build(entryResolver, dependencyService, profileWriter, calculator, logger, clock))
}
