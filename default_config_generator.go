package nginx

import (
	"fmt"

	"github.com/paketo-buildpacks/packit/v2/fs"
)

type DefaultConfigGenerator struct {
}

func NewDefaultConfigGenerator() DefaultConfigGenerator {
	return DefaultConfigGenerator{}
}

func (g DefaultConfigGenerator) Generate(templateSource, destination string) error {
	err := fs.Copy(templateSource, destination)
	if err != nil {
		return fmt.Errorf("failed to generate default nginx.conf: %w", err)
	}
	return nil
}
