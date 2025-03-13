# image-prompt

[![Build Status](https://github.com/vearutop/image-prompt/workflows/test-unit/badge.svg)](https://github.com/vearutop/image-prompt/actions?query=branch%3Amaster+workflow%3Atest-unit)
[![Coverage Status](https://codecov.io/gh/vearutop/image-prompt/branch/master/graph/badge.svg)](https://codecov.io/gh/vearutop/image-prompt)
[![GoDevDoc](https://img.shields.io/badge/dev-doc-00ADD8?logo=go)](https://pkg.go.dev/github.com/vearutop/image-prompt)
[![Time Tracker](https://wakatime.com/badge/github/vearutop/image-prompt.svg)](https://wakatime.com/badge/github/vearutop/image-prompt)
![Code lines](https://sloc.xyz/github/vearutop/image-prompt/?category=code)
![Comments](https://sloc.xyz/github/vearutop/image-prompt/?category=comments)

<!--- TODO Update README.md -->

Project template with GitHub actions for Go.

## Install

```
go install github.com/vearutop/image-prompt@latest
$(go env GOPATH)/bin/image-prompt --help
```

Or download binary from [releases](https://github.com/vearutop/image-prompt/releases).

### Linux AMD64

```
wget https://github.com/vearutop/image-prompt/releases/latest/download/linux_amd64.tar.gz && tar xf linux_amd64.tar.gz && rm linux_amd64.tar.gz
./image-prompt -version
```

### Macos Intel

```
wget https://github.com/vearutop/image-prompt/releases/latest/download/darwin_amd64.tar.gz && tar xf darwin_amd64.tar.gz && rm darwin_amd64.tar.gz
codesign -s - ./image-prompt
./image-prompt -version
```

### Macos Apple Silicon (M1, etc...)

```
wget https://github.com/vearutop/image-prompt/releases/latest/download/darwin_arm64.tar.gz && tar xf darwin_arm64.tar.gz && rm darwin_arm64.tar.gz
codesign -s - ./image-prompt
./image-prompt -version
```


## Usage

Create a new repository from this template, check out it and run `./run_me.sh` to replace template name with name of
your repository.
