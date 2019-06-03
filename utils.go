package desc

import (
	"os"
	"path/filepath"
	"strings"
)

func absDir(dir string) (string, error) {
	if dir[0] != "/"[0] {
		var prefix string
		if dir[0] == "$"[0] {
			spl := strings.SplitN(dir, "/", 2)
			prefix = os.Getenv(spl[0][1:])
			dir = spl[1]
		} else {
			var err error
			prefix, err = os.Getwd()
			if isErr(err) {
				return "", err
			}
		}
		dir = filepath.Join(prefix, dir)
	}
	return dir, nil
}
