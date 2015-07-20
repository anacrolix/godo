# godo

`godo` is an alternative to `go run`. It's useful when you want to avoid the `go build`/`go install`, `/path/to/somebin` cycle. There's also a bash completion script, which makes it easy to quickly invoke a Go package from your `GOPATH`.

`godo` differs from `go run` in the following respects:

* The package to execute is specified by import path, rather than some constituent source files.
 * `godo github.com/anacrolix/missinggo/cmd/nop` vs.
 * `go run $GOPATH/src/github.com/anacrolix/missinggo/cmd/nop/*.go`
* The generated binary is stored at `$TMPDIR/godo/$pkgname.$$`. This means that:
 * It's easy to locate the binary for a process running through godo.
 * The proctitle is `$pkgname.$$`.
 * There is a bounded history of binaries due to the OS PID reuse policy.
* The main package isn't rebuilt unnecessarily, avoiding a link step for successive `godo` calls to the same package. For example a `go run` invocation for a large application on my system has a mandatory ~0.8s delay even if the files haven't changed. `godo` has this at 0.04s.

## Usage

```sh
$ godo -h
godo is an alternative to `go run`.

Usage:
  godo [go build flags] <package spec> [binary arguments]
  godo -h | --help
```

### Example
```
# first run
$ time godo github.com/anacrolix/missinggo/cmd/nop

real	0m0.244s
user	0m0.177s
sys	0m0.071s

$ time godo github.com/anacrolix/missinggo/cmd/nop

real	0m0.046s
user	0m0.012s
sys	0m0.022s

# historical binaries
$ ls -tr $TMPDIR/godo/ | grep nop.
nop.40586
nop.40590
```

## Installation

    go get github.com/anacrolix/godo

Bash completion:

    go install github.com/anacrolix/godo/go-list-cmd
    . "$GOPATH/src/github.com/anacrolix/godo/complete.sh"

## Godo-ception

```
$ godo github.com/anacrolix/godo cmd/go run "$GOPATH/src/github.com/anacrolix/godo/"*.go cmd/go list github.com/anacrolix/...
<all my herptastic Go code>
```
