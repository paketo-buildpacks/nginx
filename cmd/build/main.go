package main

import (
	"os"
	"time"

	"github.com/cloudfoundry/packit"
	"github.com/cloudfoundry/packit/cargo"
	"github.com/cloudfoundry/packit/fs"
	"github.com/cloudfoundry/packit/postal"
	"github.com/paketo-buildpacks/nginx/nginx"
)

func main() {
	transport := cargo.NewTransport()
	dependencyService := postal.NewService(transport)
	entryResolver := nginx.NewPlanEntryResolver()
	profileWriter := nginx.NewProfileWriter()
	logger := nginx.NewLogEmitter(os.Stdout)
	clock := nginx.NewClock(time.Now)

	calculator := fs.NewChecksumCalculator()
	packit.Build(nginx.Build(entryResolver, dependencyService, profileWriter, calculator, logger, clock))
}
