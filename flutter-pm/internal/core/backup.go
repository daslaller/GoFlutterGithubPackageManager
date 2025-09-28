package core

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

// BackupPubspec creates a timestamped backup of pubspec.yaml next to it and returns the backup path.
func BackupPubspec(root string) (string, error) {
	proj, err := NearestPubspec(root)
	if err != nil {
		return "", err
	}
	src := proj.PubspecPath
	stamp := time.Now().Format("20060102T150405")
	bak := filepath.Join(proj.Path, fmt.Sprintf(".pubspec.backup.%s.yaml", stamp))
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer in.Close()
	// Use a temp file then rename for Windows safety
	tmp := bak + ".tmp"
	out, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		os.Remove(tmp)
		return "", err
	}
	if err := out.Close(); err != nil {
		os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, bak); err != nil {
		// Fallback: copy to final path directly
		_ = os.Remove(bak)
		if err2 := copyFile(tmp, bak); err2 != nil {
			_ = os.Remove(tmp)
			return "", err
		}
		_ = os.Remove(tmp)
	}
	return bak, nil
}

func copyFile(src, dst string) error {
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()
	d, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer d.Close()
	_, err = io.Copy(d, s)
	return err
}
