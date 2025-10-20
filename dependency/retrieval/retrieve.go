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
	"time"

	"github.com/Masterminds/semver/v3"
	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/joshuatcasey/collections"
	"github.com/paketo-buildpacks/libdependency/retrieve"
	"github.com/paketo-buildpacks/libdependency/versionology"
	"github.com/paketo-buildpacks/packit/v2/cargo"
	"github.com/paketo-buildpacks/packit/v2/fs"
	"github.com/paketo-buildpacks/packit/v2/vacation"
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

type StackAndTargetPair struct {
	stacks []string
	target string
}

var supportedStacks = []StackAndTargetPair{
	{stacks: []string{"io.buildpacks.stacks.jammy"}, target: "jammy"},
}

var supportedPlatforms = map[string][]string{
	"linux": {"amd64", "arm64"},
}

type PlatformStackTarget struct {
	stacks []string
	target string
	os     string
	arch   string
}

func getSuportedPlatformStackTargets() []PlatformStackTarget {
	var platformStackTargets []PlatformStackTarget

	for os, architectures := range supportedPlatforms {
		for _, arch := range architectures {
			for _, pair := range supportedStacks {
				platformStackTargets = append(platformStackTargets, PlatformStackTarget{
					stacks: pair.stacks,
					target: pair.target,
					os:     os,
					arch:   arch,
				})
			}
		}
	}

	return platformStackTargets
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
	nginxSourceURL := fmt.Sprintf("https://nginx.org/download/nginx-%s.tar.gz", nginxVersion)

	sourceSHA, err := getDependencySHA(nginxVersion)
	if err != nil {
		return nil, fmt.Errorf("could get sha: %w", err)
	}

	eolDate, err := getEOL(hasVersion.Version())
	if err != nil {
		return nil, err
	}

	cpe := fmt.Sprintf("cpe:2.3:a:nginx:nginx:%s:*:*:*:*:*:*:*", nginxVersion)
	purl := retrieve.GeneratePURL("nginx", nginxVersion, sourceSHA, nginxSourceURL)

	return collections.TransformFuncWithError(getSuportedPlatformStackTargets(), func(platformTarget PlatformStackTarget) (versionology.Dependency, error) {
		fmt.Printf("Generating metadata for %s %s %s %s\n", platformTarget.os, platformTarget.arch, platformTarget.target, nginxVersion)
		configMetadataDependency := cargo.ConfigMetadataDependency{
			CPE:             cpe,
			ID:              "nginx",
			Licenses:        []interface{}{"BSD-2-Clause", "BSD-2-Clause-NetBSD"},
			Name:            "Nginx Server",
			PURL:            purl,
			Source:          nginxSourceURL,
			SourceChecksum:  fmt.Sprintf("sha256:%s", sourceSHA),
			Version:         nginxVersion,
			DeprecationDate: eolDate,
			Stacks:          platformTarget.stacks,
			OS:              platformTarget.os,
			Arch:            platformTarget.arch,
		}

		return versionology.NewDependency(configMetadataDependency, platformTarget.target)
	})
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
		"http://nginx.org/keys/maxim.key",          // Maxim Konovalov’s PGP public key
		"http://nginx.org/keys/arut.key",           // Roman Arutyunyan's PGP public key
		"https://nginx.org/keys/pluknet.key",       // Sergey Kandaurov’s PGP public key
		"http://nginx.org/keys/sb.key",             // Sergey Budnevitch’s PGP public key
		"http://nginx.org/keys/thresh.key",         // Konstantin Pavlov’s PGP public key
		"https://nginx.org/keys/nginx_signing.key", // nginx public key
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

		// Reset file pointer to beginning for each key attempt
		_, err = file.Seek(0, 0)
		if err != nil {
			return fmt.Errorf("could not reset file position: %w", err)
		}

		_, err = openpgp.CheckArmoredDetachedSignature(keyring, file, strings.NewReader(asc), nil)
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

func getEOL(version *semver.Version) (*time.Time, error) {
	minorVersion := fmt.Sprintf("%d.%d", version.Major(), version.Minor())
	endpoint := fmt.Sprintf("https://endoflife.date/api/nginx/%s.json", minorVersion)
	resp, err := http.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to query url %q: %w", endpoint, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to query url %q with: status code %d", endpoint, resp.StatusCode)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	type eolData struct {
		EolString string `json:"eol"`
	}

	d := eolData{}

	err = json.Unmarshal(body, &d)
	if err != nil {
		return nil, fmt.Errorf("could not unmarshal eol metadata: %w", err)
	}

	eol, err := time.Parse(time.DateOnly, d.EolString)
	if err != nil {
		return nil, fmt.Errorf("could not parse eol %q: %w", d.EolString, err)
	}

	return &eol, nil
}
