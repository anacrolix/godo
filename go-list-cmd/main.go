package main

import (
	"fmt"
	"go/build"
	"log"
	"os"
	"path/filepath"
	"strings"
)

func walkCmds(walkRoot string, f func(cmdPath string)) {
	filepath.Walk(walkRoot, func(p string, fi os.FileInfo, err error) error {
		if err != nil || !fi.IsDir() {
			return nil
		}
		_, file := filepath.Split(p)
		if strings.HasPrefix(file, "_") || file == "testdata" {
			return filepath.SkipDir
		}
		if file != "." && file != ".." && strings.HasPrefix(file, ".") {
			return filepath.SkipDir
		}
		pkg, err := build.Default.ImportDir(p, 0)
		if err != nil {
			if _, noGo := err.(*build.NoGoError); noGo {
				return nil
			}
		} else if !pkg.IsCommand() {
			return nil
		}
		f(p)
		return nil
	})
}

func printCmds(walkRoot, srcDir, prefix string) {
	walkCmds(walkRoot, func(p string) {
		rel, _ := filepath.Rel(srcDir, p)
		var ps string
		if rel == "." {
			ps = prefix
		} else if prefix == "" {
			ps = rel
		} else {
			if strings.HasSuffix(prefix, "/") {
				ps = prefix + rel
			} else {
				ps = prefix + "/" + rel
			}
		}
		fmt.Printf("%s\n", ps)
	})
}

func main() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	s := ""
	if len(os.Args) > 1 {
		s = os.Args[1]
	}
	i := strings.LastIndexByte(s, '/')
	if i == -1 {
		s = ""
	} else {
		s = s[:i]
	}
	if build.IsLocalImport(s) {
		printCmds(s, s, s)
		return
	}
	for _, sd := range build.Default.SrcDirs() {
		printCmds(filepath.Join(sd, filepath.FromSlash(s)), sd, "")
	}
}
