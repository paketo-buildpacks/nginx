package nginx

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/paketo-buildpacks/packit/scribe"
)

type ProfileWriter struct {
	logger scribe.Emitter
}

func NewProfileWriter(logger scribe.Emitter) ProfileWriter {
	return ProfileWriter{
		logger: logger,
	}
}

func (p ProfileWriter) Write(layerDir, scriptName, scriptContents string) error {
	profileDir := filepath.Join(layerDir, "profile.d")
	err := os.MkdirAll(profileDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create dir %s: %w", profileDir, err)
	}
	scriptFilePath := filepath.Join(profileDir, scriptName)

	p.logger.Process("Writing profile.d/configure.sh")
	p.logger.Subprocess("Calls executable that parses templates in nginx conf")
	err = ioutil.WriteFile(scriptFilePath, []byte(scriptContents), 0644)
	if err != nil {
		return fmt.Errorf("failed to write profile script: %w", err)
	}

	return nil
}
