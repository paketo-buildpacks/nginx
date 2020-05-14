package nginx

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func CopyBinFile(dstPath, srcPath string) error {
	var err error
	err = os.MkdirAll(filepath.Dir(dstPath), os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create destination parent directory: %w", err)
	}
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("failed to open source file for reading: %w", err)
	}
	defer src.Close()

	dst, err := os.OpenFile(dstPath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("failed to open destination file for writing: %w", err)
	}
	defer dst.Close()

	_, err = io.Copy(dst, src)
	if err != nil {
		return fmt.Errorf("copy failed: %w", err)
	}

	return nil
}
