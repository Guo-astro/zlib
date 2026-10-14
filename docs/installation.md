# Installation

[README](../README.md) · [Quick start](../README.md#quick-start)

Choose one installation method. Prebuilt binaries are available for macOS, Linux
and Windows on x86-64 and ARM64; they do not require Go.

## Package managers

macOS / Linux:

```bash
brew install heartleo/tap/zlib
```

Windows (PowerShell):

```powershell
winget install --id heartleo.zlib --exact
```

## Install script or binary download

On macOS / Linux:

```bash
curl -fsSL https://raw.githubusercontent.com/heartleo/zlib/main/install.sh | sh
```

For a manual install, download the archive for your OS and architecture from
[Releases](https://github.com/heartleo/zlib/releases/latest). Extract `zlib`
(`zlib.exe` on Windows) into a directory on your `PATH`.

## Go install

Requires Go 1.26 or later:

```sh
go install github.com/heartleo/zlib/cmd/zlib@latest
```

Add Go's binary directory to your `PATH`: `$GOBIN` if set, otherwise the `bin`
directory under `go env GOPATH`.

## Build from source

Requires the Go version declared in [go.mod](../go.mod).

```sh
git clone https://github.com/heartleo/zlib
cd zlib
```

macOS / Linux:

```bash
go build -o zlib ./cmd/zlib
./zlib --help
```

Windows (PowerShell):

```powershell
go build -o zlib.exe ./cmd/zlib
.\zlib.exe --help
```

The binary stays in the repository directory. Put it on `PATH` to run it as `zlib`
from other directories or from an agent. To check code changes, run `go test ./...`
and `go vet ./...`.

## Verify and upgrade

```sh
zlib version
zlib --help
```

Upgrade using the method you installed with: `brew upgrade heartleo/tap/zlib`,
`winget upgrade --id heartleo.zlib --exact`, or rerun the install script / Go install
command. For a manual install, replace the executable with a newer release.

If a command in these docs is missing, check your installed version. Docs on a
development branch may describe changes that have not reached a release yet;
build that branch from source to try them.

Continue with the [quick start](../README.md#quick-start).
