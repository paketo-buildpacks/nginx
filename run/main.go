package main

import (
	"fmt"
	"os"

	"github.com/Netflix/go-env"
	"github.com/paketo-buildpacks/nginx"
	"github.com/paketo-buildpacks/packit/v2"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/chronos"
	"github.com/paketo-buildpacks/packit/v2/draft"
	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/postal"
	"github.com/paketo-buildpacks/packit/v2/scribe"
	"github.com/paketo-buildpacks/packit/v2/servicebindings"
)

func main() {
	logger := scribe.NewEmitter(os.Stdout)

	var buildEnv nginx.BuildEnvironment
	_, err := env.UnmarshalFromEnviron(&buildEnv)
	if err != nil {
		fmt.Fprintln(os.Stderr, fmt.Errorf("failed to parse build configuration: %w", err))
		os.Exit(1)
	}

	packit.Run(
		nginx.Detect(buildEnv, nginx.NewParser()),
		nginx.Build(
			buildEnv,
			draft.NewPlanner(),
			postal.NewService(cargo.NewTransport()),
			servicebindings.NewResolver(),
			nginx.NewDefaultConfigGenerator(),
			fs.NewChecksumCalculator(),
			logger,
			chronos.DefaultClock,
		),
	)
}
