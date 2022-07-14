package nginx

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v2"
)

type Parser struct{}

func NewParser() Parser {
	return Parser{}
}

func (p Parser) ParseYml(workingDir string) (string, bool, error) {
	bpYML, err := os.Open(filepath.Join(workingDir, BuildpackYMLSource))
	if err != nil {
		return "", false, err
	}

	var buildpackYML struct {
		Config struct {
			Version string `yaml:"version"`
		} `yaml:"nginx"`
	}

	err = yaml.NewDecoder(bpYML).Decode(&buildpackYML)
	if err != nil {
		return "", false, err
	}

	return buildpackYML.Config.Version, true, nil
}

func (p Parser) ResolveVersion(cnbPath, version string) (string, error) {
	bpTOML, err := os.Open(filepath.Join(cnbPath, "buildpack.toml"))
	if err != nil {
		return "", err
	}

	var buildpackTOML struct {
		Metadata struct {
			DefaultVersions map[string]string `toml:"default-versions"`
			VersionLines    struct {
				Mainline string `toml:"mainline"`
				Stable   string `toml:"stable"`
			} `toml:"version-lines"`
		} `toml:"metadata"`
	}

	_, err = toml.NewDecoder(bpTOML).Decode(&buildpackTOML)
	if err != nil {
		return "", err
	}

	if version == "mainline" {
		version = buildpackTOML.Metadata.VersionLines.Mainline
	}

	if version == "stable" {
		version = buildpackTOML.Metadata.VersionLines.Stable
	}

	if version == "" {
		version = buildpackTOML.Metadata.DefaultVersions[NGINX]
	}

	return version, nil
}
