MAKEGO := make/go
MAKEGO_REMOTE := git@github.com:lekkodev/makego.git
PROJECT := go-sdk
GO_MODULE := github.com/lekkodev/go-sdk
DOCKER_ORG := lekko
DOCKER_PROJECT := example
FILE_IGNORES := $(FILE_IGNORES) .vscode/

include make/example/all.mk
