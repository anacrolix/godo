package main

import (
	"fmt"
	"go/build"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
)

const (
	debug = false
	// Whether to redirect the go cmd's stdout and stderr to tty.
	goTTY             = false
	execWithPidSuffix = false
)

// args should not include the executed file path common to argv[0]. goFlags
// are flags passed to the command used to build the command. pkgSpec is the
// package to build/execute. pkgArgs are the final command's arguments.
func processArgs(args []string) (goFlags []string, pkgSpec string, pkgArgs []string) {
	pkgSpec = "."
	for i, arg := range args {
		if arg == "--" {
			goFlags = args[:i]
			pkgArgs = args[i+1:]
			return
		}
		if !strings.HasPrefix(arg, "-") {
			pkgSpec = arg
			goFlags = args[:i]
			pkgArgs = args[i+1:]
			return
		}
	}
	pkgArgs = args
	return
}

// Return includes the ".". Stolen from `cmd/go`. Another way to do this is
// parse the output of `go env GOEXE`. Possibly this doesn't take into account
// targeting another OS, but then godo is intended to immediately invoke it,
// so maybe that isn't a concern.
func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

func installEnv(GOBIN string) (ret []string) {
	env := os.Environ()
	ret = make([]string, 0, len(env)+1)
	for _, p := range env {
		if strings.HasPrefix(p, "GOBIN=") {
			continue
		}
		if strings.HasPrefix(p, "GODEBUG=") {
			continue
		}
		ret = append(ret, p)
	}
	ret = append(ret, "GOBIN="+GOBIN)
	return
}

func buildEnv() (ret []string) {
	env := os.Environ()
	ret = make([]string, 0, len(env))
	for _, p := range env {
		if strings.HasPrefix(p, "GODEBUG=") {
			continue
		}
		ret = append(ret, p)
	}
	return
}

func copyFile(src, dst string) (err error) {
	srcFile, err := os.Open(src)
	if err != nil {
		return
	}
	defer srcFile.Close()
	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY, 0755)
	if err != nil {
		return
	}
	defer dstFile.Close()
	_, err = io.Copy(dstFile, srcFile)
	return
}

func getPackage(spec string, flags []string) {
	args := []string{"get"}
	args = append(args, flags...)
	args = append(args, "-d", spec)
	cmd := exec.Command("go", args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stderr
	cmd.Env = installEnv("/dev/null")
	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error getting package: %s\n", err)
		os.Exit(1)
	}
}

func fixAbsPkgSpec(s string) string {
	if !filepath.IsAbs(s) {
		return s
	}
	wd, err := os.Getwd()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	s, err = filepath.Rel(wd, s)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return s
}

func withWorkDir(tmpDir string, f func()) (err error) {
	origDir, err := os.Getwd()
	if err != nil {
		return
	}
	err = os.Chdir(tmpDir)
	if err != nil {
		return
	}
	defer func() {
		err = os.Chdir(origDir)
	}()
	f()
	return
}

func main() {
	err := mainErr()
	if err != nil {
		log.Fatal(err)
	}
}

func mainErr() error {
	if len(os.Args[1:]) == 1 {
		switch os.Args[1] {
		case "-h", "--help":
			fmt.Fprintf(os.Stderr, "%s", "godo is an alternative to `go run`.\n\nUsage:\n  godo [go build flags] <package spec> [binary arguments]\n  godo -h | --help\n")
			return nil
		default:
		}
	}
	goFlags, pkgSpec, pkgArgs := processArgs(os.Args[1:])
	if debug {
		log.Println(goFlags, pkgSpec, pkgArgs)
	}
	var execFilePath *string
	if err := withWorkDir(pkgSpec, func() {
		pkg, err := build.Import(".", ".", 0)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error locating package: %s\n", err)
			os.Exit(2)
		}
		if !pkg.IsCommand() {
			fmt.Fprintln(os.Stderr, "package is not a command")
			os.Exit(2)
		}
		godoDir := filepath.Join(build.Default.GOPATH, "godo")
		pkgBase := pkg.ImportPath
		if pkgBase == "." {
			wd, err := os.Getwd()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			pkgBase = wd
		}
		pkgBase = filepath.Base(pkgBase)
		stageExeName := pkgBase + exeSuffix()
		execExeName := func() string {
			if execWithPidSuffix {
				return pkgBase + "." + fmt.Sprintf("%d", os.Getpid()) + exeSuffix()
			} else {
				return stageExeName
			}
		}()
		execFilePath = func() *string {
			s := filepath.Join(godoDir, execExeName)
			return &s
		}()
		buildArgs := []string{"build"}
		buildArgs = append(buildArgs, goFlags...)
		buildArgs = append(buildArgs, "-o", godoDir, ".")
		cmd := exec.Command("go", buildArgs...)
		cmd.Env = buildEnv()
		cmd.Stdout = os.Stderr
		cmd.Stderr = os.Stderr
		if goTTY {
			tty, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
			if err == nil {
				defer tty.Close()
				cmd.Stdout = tty
				cmd.Stderr = tty
			}
		}
		err = cmd.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "error building command: %s\n", err)
			os.Exit(1)
		}
		if execWithPidSuffix {
			err = copyFile(filepath.Join(godoDir, stageExeName), *execFilePath)
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
		}
	}); err != nil {
		return err
	}
	execArgv := append([]string{*execFilePath}, pkgArgs...)
	// fmt.Fprintf(os.Stderr, "exec %q\n", execArgv)
	err := syscall.Exec(*execFilePath, execArgv, os.Environ())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error execing command: %s\n", err)
		os.Exit(1)
	}
	return nil
}
