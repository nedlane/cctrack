package tailnet

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// shellQuote wraps s in single quotes for safe interpolation into a remote
// `sh -c` command, escaping any embedded single quotes.
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// joinClean joins and cleans a path, guarding against absolute remoteDir
// escaping the extraction root.
func joinClean(root, rel string) string {
	return filepath.Clean(filepath.Join(root, rel))
}

// mergeTree copies every regular file under src into dst, preserving relative
// paths. Existing files are overwritten. Used by the tar fallback to relocate
// the extracted leaf into the host's mirror dir. Missing src is not an error.
func mergeTree(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("expected directory at %s", src)
	}
	return filepath.Walk(src, func(path string, fi os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if fi.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		if !fi.Mode().IsRegular() {
			return nil
		}
		return copyFile(path, target)
	})
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
