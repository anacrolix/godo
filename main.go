package main

import (
	"fmt"
	"go/build"
	"log"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strings"
	"syscall"
)

const debug = false

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
	goFlags = args
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

func main() {
	goFlags, pkgSpec, pkgArgs := processArgs(os.Args[1:])
	if debug {
		log.Println(goFlags, pkgSpec, pkgArgs)
	}
	pkg, err := build.Import(pkgSpec, ".", 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error locating package: %s\n", err)
		os.Exit(2)
	}
	if !pkg.IsCommand() {
		fmt.Fprintln(os.Stderr, "package is not a command")
		os.Exit(2)
	}
	exeName := path.Base(pkg.ImportPath) + "." + fmt.Sprintf("%d", os.Getpid()) + exeSuffix()
	godoDir := path.Join(os.TempDir(), "godo")
	buildOutputPath := path.Join(godoDir, exeName)
	fmt.Fprintf(os.Stderr, "building command at %q\n", buildOutputPath)
	buildArgs := []string{"build", "-i", "-o", buildOutputPath, "-i"}
	buildArgs = append(buildArgs, goFlags...)
	buildArgs = append(buildArgs, pkgSpec)
	tty, err := os.OpenFile("/dev/tty", os.O_WRONLY, 0)
	if err != nil {
		tty = os.Stderr
	} else {
		defer tty.Close()
	}
	cmd := exec.Command("go", buildArgs...)
	cmd.Stderr = tty
	cmd.Stdout = tty
	err = cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error building command: %s\n", err)
		os.Exit(1)
	}
	execArgv := append([]string{buildOutputPath}, pkgArgs...)
	fmt.Fprintf(os.Stderr, "exec %q\n", execArgv)
	err = syscall.Exec(buildOutputPath, execArgv, os.Environ())
	if err != nil {
		fmt.Fprintf(os.Stderr, "error execing command: %s\n", err)
		os.Exit(1)
	}
}
