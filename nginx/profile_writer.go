package nginx

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
)

type ProfileWriter struct{}

func NewProfileWriter() ProfileWriter {
	return ProfileWriter{}
}

func (p ProfileWriter) Write(layerDir, scriptName, scriptContents string) error {
	profileDir := filepath.Join(layerDir, "profile.d")
	err := os.MkdirAll(profileDir, 0755)
	if err != nil {
		return fmt.Errorf("failed to create dir %s: %w", profileDir, err)
	}
	scriptFilePath := filepath.Join(profileDir, scriptName)

	err = ioutil.WriteFile(scriptFilePath, []byte(scriptContents), 0644)
	if err != nil {
		return fmt.Errorf("failed to write profile script: %w", err)
	}
	return nil
}
