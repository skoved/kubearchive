// Copyright KubeArchive Authors
// SPDX-License-Identifier: Apache-2.0

package files

import (
	"errors"
	"fmt"
	"os"
	"strings"
)

// Determines if path exists in the file system. Returns true if the path exists. Returns false if the path does not
// exist. Returns error if a file system error other than os.ErrNotExist occurs
func PathExists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	return false, err
}

// Returns map representing all non directory files in path. The key is the file name and value is the path to the file.
// It does not search for files in sub directories
func DirectoryFiles(path string) (map[string]string, error) {
	dirFiles := make(map[string]string)
	dir, err := os.ReadDir(path)
	if err != nil {
		return dirFiles, err
	}
	for _, file := range dir {
		if file.IsDir() || strings.HasPrefix(file.Name(), "..") {
			continue
		}
		dirFiles[file.Name()] = fmt.Sprintf("%s/%s", path, file.Name())
	}
	return dirFiles, nil
}
