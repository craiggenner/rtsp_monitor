package screenshot

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Save writes JPEG data to <baseDir>/<cameraName>/YYYY/MM/DD/HH:MM.jpg
func Save(baseDir, cameraName string, ts time.Time, jpegData []byte) error {
	dir := filepath.Join(
		baseDir,
		cameraName,
		ts.Format("2006"),
		ts.Format("01"),
		ts.Format("02"),
	)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating screenshot directory: %w", err)
	}

	filename := ts.Format("15:04") + ".jpg"
	path := filepath.Join(dir, filename)

	if err := os.WriteFile(path, jpegData, 0o644); err != nil {
		return fmt.Errorf("writing screenshot: %w", err)
	}

	return nil
}
