package main

import (
	"errors"
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

	"github.com/davecgh/go-spew/spew"
	"golang.org/x/tools/go/packages"
)

const (
	debug = false
	// Whether to redirect the go cmd's stdout and stderr to tty.
	goTTY             = false
	execWithPidSuffix = false
)

const exitCodeUsage = 2

type exitError struct {
	error
	code int
}

func (me exitError) ExitCode() int {
	return me.code
}

// args should not include the executed file path common to argv[0]. goFlags
// are flags passed to the command used to build the command. pkgSpec is the
// package to build/execute. pkgArgs are the final command's arguments.
func processArgs(args []string) (goFlags []string, pkgSpec string, pkgArgs []string, err error) {
	for i, arg := range args {
		if arg == "--" && len(args[i+1:]) > 0 {
			pkgSpec = args[i+1]
			goFlags = args[:i]
			pkgArgs = args[i+2:]
			return
		}
	}
	err = exitError{
		errors.New(`expected "--" then a command package directory path`),
		exitCodeUsage}
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
		code := 1
		var exiter exitError
		if errors.As(err, &exiter) {
			code = exiter.ExitCode()
		}
		fmt.Fprintf(os.Stderr, "godo: %v\n", err)
		os.Exit(code)
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
	goFlags, pkgSpec, pkgArgs, err := processArgs(os.Args[1:])
	if err != nil {
		err = fmt.Errorf("processing args: %w", err)
		return err
	}
	if debug {
		log.Println(goFlags, pkgSpec, pkgArgs)
	}
	var execFilePath, cmdName *string
	if wdErr := withWorkDir(pkgSpec, func() {
		var pkgs []*packages.Package
		pkgs, err = packages.Load(&packages.Config{
			Mode: 0,
		})
		//pkg, err = build.Import(".", ".", 0)
		if err != nil {
			err = fmt.Errorf("error locating package: %w", err)
			return
		}
		if len(pkgs) != 1 {
			err = fmt.Errorf("more than one package loaded")
			return
		}
		if false {
			spew.Dump(pkgs)
		}
		pkg := pkgs[0]
		for _, pkgErr := range pkg.Errors {
			err = errors.Join(err, pkgErr)
		}
		if err != nil {
			err = fmt.Errorf("package errors: %w", err)
			return
		}
		if false {
			spewConfig := spew.ConfigState{
				DisableMethods: true,
			}
			spewConfig.Dump(*pkg)
		}
		if pkg.Name != "main" {
			err = exitError{fmt.Errorf("package %q is not a command", pkg.PkgPath), exitCodeUsage}
			return
		}
		godoDir := filepath.Join(build.Default.GOPATH, "godo")
		pkgBase := pkgs[0].PkgPath
		if pkgBase == "." {
			wd, err := os.Getwd()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(1)
			}
			pkgBase = wd
		}
		pkgBase = filepath.Base(pkgBase)
		cmdName = &pkgBase
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
		// By adding a directory separator, we ensure that go always interprets the target as a
		// directory to put the binary in. Otherwise, if the directory doesn't exist, it creates
		// "$GOPATH/godo" as the binary itself.
		buildArgs = append(buildArgs, "-o", godoDir+string(filepath.Separator), ".")
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
	}); wdErr != nil {
		return wdErr
	}
	if err != nil {
		return err
	}
	execArgv := append([]string{*execFilePath}, pkgArgs...)
	//fmt.Fprintf(os.Stderr, "exec %q\n", execArgv)

	// There's no feedback when godo execs, so sometimes if the command doesn't output anything, you
	// can't tell if it's stuck building still. This might be an argument to handle build flags
	// manually, so we can add godo specific ones, and drop requiring the -- separator again.
	fmt.Fprintf(os.Stderr, "godo: starting %v\n", *cmdName)
	err = syscall.Exec(*execFilePath, execArgv, os.Environ())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error execing command [arv0=%q, argv=%q, environ=%q]: %s\n",
			*execFilePath, execArgv, os.Environ(), err)
		os.Exit(1)
	}
	return nil
}
