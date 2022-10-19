package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/joshuatcasey/libdependency/retrieve"
	"github.com/joshuatcasey/libdependency/versionology"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/vacation"
	"golang.org/x/crypto/openpgp"
)

type GithubTagReponse struct {
	Name string `json:"name"`
}

type NginxMetadata struct {
	SemverVersion *semver.Version
}

func (nginxMetadata NginxMetadata) Version() *semver.Version {
	return nginxMetadata.SemverVersion
}

func main() {
	retrieve.NewMetadata("nginx", getNginxVersions, generateMetadata)
}

func getNginxVersions() (versionology.VersionFetcherArray, error) {
	body, err := httpGet("https://api.github.com/repos/nginx/nginx/tags")
	if err != nil {
		return nil, err
	}

	var tags []GithubTagReponse
	err = json.Unmarshal(body, &tags)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshall tags: %w", err)
	}

	var versions []versionology.VersionFetcher
	for _, tag := range tags {
		versions = append(versions, NginxMetadata{
			semver.MustParse(strings.TrimPrefix(tag.Name, "release-")),
		})
	}

	return versions, nil
}

func generateMetadata(hasVersion versionology.VersionFetcher) ([]versionology.Dependency, error) {
	nginxVersion := hasVersion.Version().String()
	nginxURL := fmt.Sprintf("https://nginx.org/download/nginx-%s.tar.gz", nginxVersion)

	sourceSHA, err := getDependencySHA(nginxVersion)
	if err != nil {
		return nil, fmt.Errorf("could get sha: %w", err)
	}

	dep := cargo.ConfigMetadataDependency{
		Version:         nginxVersion,
		ID:              "nginx",
		Name:            "Nginx Server",
		Source:          nginxURL,
		SourceChecksum:  fmt.Sprintf("sha256:%s", sourceSHA),
		DeprecationDate: nil,
		Licenses:        retrieve.LookupLicenses(nginxURL, decompress),
		PURL:            retrieve.GeneratePURL("nginx", nginxVersion, sourceSHA, nginxURL),
		CPE:             fmt.Sprintf("cpe:2.3:a:nginx:nginx:%s:*:*:*:*:*:*:*", nginxVersion),
		Stacks:          []string{"io.buildpacks.stacks.bionic"},
	}

	bionicDependency, err := versionology.NewDependency(dep, "bionic")
	if err != nil {
		return nil, fmt.Errorf("could get sha: %w", err)
	}

	dep.Stacks = []string{"io.buildpacks.stacks.jammy"}

	jammyDependency, err := versionology.NewDependency(dep, "jammy")
	if err != nil {
		return nil, fmt.Errorf("could get sha: %w", err)
	}

	return []versionology.Dependency{bionicDependency, jammyDependency}, nil
}

func httpGet(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("could not make get request: %w", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("could not read response: %w", err)
	}

	return body, nil
}

func downloadFromURL(url, path string) error {
	content, err := httpGet(url)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, bytes.NewReader(content))
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
	}

	return nil
}

func getDependencySHA(version string) (string, error) {
	url := fmt.Sprintf("https://nginx.org/download/nginx-%s.tar.gz", version)

	dependencyOutputDir, err := os.MkdirTemp("", "nginx")
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}
	dependencyOutputPath := filepath.Join(dependencyOutputDir, filepath.Base(url))

	err = downloadFromURL(url, dependencyOutputPath)
	if err != nil {
		return "", fmt.Errorf("failed to download dependency: %w", err)
	}

	err = verifyASC(version, dependencyOutputPath)
	if err != nil {
		return "", fmt.Errorf("dependency signature verification failed: %w", err)
	}

	return fs.NewChecksumCalculator().Sum(dependencyOutputPath)
}

func getGPGKeys() ([]string, error) {
	var nginxGPGKeys []string
	for _, keyURL := range []string{
		// Key URLs from https://nginx.org/en/pgp_keys.html
		"http://nginx.org/keys/mdounin.key",
		"http://nginx.org/keys/maxim.key",
		"http://nginx.org/keys/sb.key",
		"http://nginx.org/keys/thresh.key",
	} {
		body, err := httpGet(keyURL)
		if err != nil {
			return []string{}, err
		}

		nginxGPGKeys = append(nginxGPGKeys, string(body))
	}

	return nginxGPGKeys, nil
}

func getDependencySignature(version string) (string, error) {
	body, err := httpGet(fmt.Sprintf("http://nginx.org/download/nginx-%s.tar.gz.asc", version))
	if err != nil {
		return "", err
	}

	return string(body), nil
}

func verifyASC(version, path string) error {
	gpgKeys, err := getGPGKeys()
	if err != nil {
		return fmt.Errorf("could not get GPG keys: %w", err)
	}

	if len(gpgKeys) == 0 {
		return errors.New("no pgp keys provided")
	}

	asc, err := getDependencySignature(version)
	if err != nil {
		return fmt.Errorf("could not get dependency signature: %w", err)
	}

	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("could not open file: %w", err)
	}
	defer file.Close()

	for _, gpgKey := range gpgKeys {
		keyring, err := openpgp.ReadArmoredKeyRing(strings.NewReader(gpgKey))
		if err != nil {
			return fmt.Errorf("could not read armored key ring: %w", err)
		}

		_, err = openpgp.CheckArmoredDetachedSignature(keyring, file, strings.NewReader(asc))
		if err != nil {
			log.Printf("failed to check signature: %s", err.Error())
			continue
		}
		log.Printf("found valid pgp key")
		return nil
	}

	return errors.New("no valid pgp keys provided")
}

func decompress(artifact io.Reader, destination string) error {
	archive := vacation.NewArchive(artifact)

	err := archive.StripComponents(1).Decompress(destination)
	if err != nil {
		return fmt.Errorf("failed to decompress source file: %w", err)
	}

	return nil
}
