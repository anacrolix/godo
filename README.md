# godo

godo is an executable that builds and invokes the command given by a Go package spec.

```sh
$ godo github.com/anacrolix/dms
22:22:16 main.go:211: added 1148 items from cache
22:22:16 dms.go:892: HTTP srv on [::]:1338
22:22:16 dms.go:194: started SSDP on lo0
22:22:16 dms.go:194: started SSDP on en0
...
```

Arguments given *before* the package spec are passed to `go build`. For example `-v` and `-race`.

```
anacrolix@Matts-MacBook-Pro:~$ godo -v github.com/anacrolix/dms
github.com/anacrolix/dms/soap
golang.org/x/net/internal/iana
github.com/anacrolix/dms/transcode
github.com/anacrolix/dms/rrcache
golang.org/x/net/ipv4
github.com/anacrolix/dms/ssdp
github.com/anacrolix/dms/dlna/dms
github.com/anacrolix/dms
<program begins running>
```

Arguments passed *after* the package spec, are passed to the invoked executable.

```
$ godo github.com/motemen/gore -h
Usage of /var/folders/j8/n6cvt4453nzcp5cn9xpbcn9r0000gn/T/godo/gore.12943:
  -autoimport=false: formats and adjusts imports automatically
  -context="": import packages, functions, variables and constants from external golang source files
  -pkg="": specify a package where the session will be run inside
```

Binaries are built and invoked at `$TMPDIR/godo/`. Binaries are stored with the PID of the godo instance that invoked them as a suffix. This makes it easy to locate the binary that belongs to any process invoked by godo, populates the proctitle with a unique executable name, and results in a conveniently bounded history of invoked binaries.

```
$ echo $TMPDIR
/var/folders/j8/n6cvt4453nzcp5cn9xpbcn9r0000gn/T/
$ godo github.com/motemen/gore -h
Usage of /var/folders/j8/n6cvt4453nzcp5cn9xpbcn9r0000gn/T/godo/gore.12943:
<snip>
$ godo github.com/motemen/gore -h
Usage of /var/folders/j8/n6cvt4453nzcp5cn9xpbcn9r0000gn/T/godo/gore.13055:
<snip>
$ ls "$TMPDIR/godo/gore.*" -t
gore.12943  gore.13055
```

### Godoception

```
$ godo github.com/anacrolix/godo cmd/go run "$GOPATH/src/github.com/anacrolix/godo/"*.go cmd/go list github.com/anacrolix/...
<all my herptastic Go code>
```

## Installation

    go get github.com/anacrolix/godo
