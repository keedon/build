// Copyright 2015-2016 Sevki <s@sevki.org>. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package util

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	ini "github.com/vaughan0/go-ini"
)

var (
	file ini.File

	pp = ""
)

func init() {
	wd, _ := os.Getwd()
	pp = GetGitDir(wd)

	var err error
	if file, err = ini.LoadFile(filepath.Join(GetProjectPath(), ".build")); err == nil {
		if err != nil {
			log.Fatalf("error: %v", err)
		}
	}
}

// Getenv returns the envinroment variable. It looks for the envinroment
// variable in the following order. It checks if the current shell session has
// an envinroment variable, checks if it's set in the OS specific section in
// the .build file, and checks it for common in the .build config file.
func Getenv(s string) string {
	if os.Getenv(s) != "" {
		return os.Getenv(s)
	} else if val, exists := file.Get(runtime.GOOS, s); exists {
		return val
	} else if val, exists := file.Get("", s); exists {
		return val
	} else {
		return ""
	}
}
func GetProjectPath() (ProjectPath string) {
	return pp
}
func RelPPath(p string) string {
	rel, _ := filepath.Rel(GetProjectPath(), p)
	return rel
}

// HashFiles will hash files collecetion represented as a string array,
// If the string in the array is directory it will the directory contents to the array
// if the string isn't an absolute path, it will assume that it's a export from a dependency
// and skip that.
func HashFiles(h io.Writer, files []string) {
	fsm := files
RESTART:
	for i, file := range fsm {
		if !filepath.IsAbs(file) {
			continue
		}
		if filepath.Base(file) == BuildOut() {
			continue
		}

		f, err := os.Open(file)

		if err != nil {
			log.Fatalf("hash files: %s\n", err.Error())
		}

		stat, _ := f.Stat()
		if stat.IsDir() {
			fsm = append([]string{}, fsm[i+1:]...)
			fs, _ := f.Readdir(-1)
			for _, x := range fs {
				fsm = append(fsm, (filepath.Join(file, x.Name())))
			}
			f.Close()
			goto RESTART /* to avoid out of bound errors, there may be no files
			in the folder */
		}

		fmt.Fprintf(h, "file %s\n", filepath.Join(pp, file))
		n, _ := io.Copy(h, f)
		fmt.Fprintf(h, "%d bytes\n", n)
		f.Close()
	}
}

func BuildOut() string {
	if Getenv("BUILD_OUT") != "" {
		return Getenv("BUILD_OUT")
	} else {
		return filepath.Join(
			GetProjectPath(),
			"build_out",
		)
	}
}

// HashFilesWithExt will hash files collecetion represented as a string array,
// If the string in the array is directory it will the directory contents to the array
// if the string isn't an absolute path, it will assume that it's a export from a dependency
// and skip that.
func HashFilesWithExt(h io.Writer, files []string, ext string) {
	fsm := files
RESTART:
	for i, file := range fsm {
		if !filepath.IsAbs(file) {
			continue
		}
		if filepath.Base(file) == BuildOut() {
			continue
		}
		f, err := os.Open(file)

		if err != nil {
			log.Fatalf("hash files: %s\n", err.Error())
		}

		stat, _ := f.Stat()
		if stat.IsDir() {
			fsm = append([]string{}, fsm[i+1:]...)
			fs, _ := f.Readdir(-1)
			for _, x := range fs {
				if filepath.Ext(x.Name()) == ext || filepath.Ext(x.Name()) == "" {
					fsm = append(fsm, (filepath.Join(file, x.Name())))
				}

			}
			goto RESTART /* to avoid out of bound errors, there may be no files
			in the folder */
		}
		if filepath.Ext(file) != ext {
			f.Close()
			continue
		}

		fmt.Fprintf(h, "file %s\n", filepath.Join(pp, file))
		n, _ := io.Copy(h, f)
		fmt.Fprintf(h, "%d bytes\n", n)
		f.Close()
	}
}

func HashStrings(h io.Writer, strs []string) {
	for _, str := range strs {
		io.WriteString(h, str)
	}
}
func GetGitDir(p string) string {
	dirs := strings.Split(p, "/")
	for i := len(dirs) - 1; i > 0; i-- {
		try := fmt.Sprintf("/%s/.git", filepath.Join(dirs[0:i+1]...))
		if _, err := os.Lstat(try); os.IsNotExist(err) {
			continue
		} else if err != nil {
			log.Fatal(err)
		}
		pr, _ := filepath.Split(try)
		return pr
	}
	return ""
}
