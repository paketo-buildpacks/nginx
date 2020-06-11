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

func (p Parser) ParseVersion(workingDir, cnbPath string) (string, string, error) {

	bpTOML, err := os.Open(filepath.Join(cnbPath, "buildpack.toml"))
	if err != nil {
		return "", "", err
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

	_, err = toml.DecodeReader(bpTOML, &buildpackTOML)
	if err != nil {
		return "", "", err
	}

	bpYML, err := os.Open(filepath.Join(workingDir, BuildpackYMLSource))
	if err != nil {
		if os.IsNotExist(err) {
			return buildpackTOML.Metadata.DefaultVersions[NGINX], "buildpack.toml", nil
		}
		return "", "", err
	}

	var buildpackYML struct {
		Config struct {
			Version string `yaml:"version"`
		} `yaml:"nginx"`
	}

	var version string
	var versionSource = "buildpack.yml"
	err = yaml.NewDecoder(bpYML).Decode(&buildpackYML)
	if err != nil {
		return "", "", err
	}

	version = buildpackYML.Config.Version

	if version == "mainline" {
		version = buildpackTOML.Metadata.VersionLines.Mainline
	}

	if version == "stable" {
		version = buildpackTOML.Metadata.VersionLines.Stable
	}

	if version == "" {
		versionSource = "buildpack.toml"
		version = buildpackTOML.Metadata.DefaultVersions[NGINX]
	}

	return version, versionSource, nil
}
