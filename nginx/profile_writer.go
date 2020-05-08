package nginx

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type ProfileWriter struct {
	logger LogEmitter
}

func NewProfileWriter(logger LogEmitter) ProfileWriter {
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

	p.logger.Subprocess("    Writing profile.d/configure.sh")
	p.logger.Action("Calls executable that parses templates in nginx conf")
	err = ioutil.WriteFile(scriptFilePath, []byte(scriptContents), 0644)
	if err != nil {
		return fmt.Errorf("failed to write profile script: %w", err)
	}

	return nil
}
