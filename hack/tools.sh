#!/bin/bash -e

GO111MODULE=on
HOME=$(
	cd "$(dirname "$0")"
	cd ..
	pwd
)

LINTER=${HOME}/bin/golangci-lint
LINTER_CONF=${HOME}/.golangci.yml

function test() {
	cd "$1"
	echo "test $(sed -n 1p go.mod | cut -d ' ' -f2)"
	go test -v -race ./...
}

function deps() {
	cd "$1"
	echo "download  $(sed -n 1p go.mod | cut -d ' ' -f2)"
	go get -v -d ./...
}

function lint() {
	cd "$1"
	echo "lint $(sed -n 1p go.mod | cut -d ' ' -f2)"
	eval '${LINTER} run --timeout=5m --config=${LINTER_CONF} --exclude-use-default=false'
}

function build() {
	cd "$1"
	echo "build $(sed -n 1p go.mod | cut -d ' ' -f2)"
	go build -v ./...
}

case $1 in
deps)
	shift
	deps "$@"
	;;
test)
	shift
	test "$@"
	;;
lint)
	shift
	lint "$@"
	;;
build)
	shift
	build "$@"
	;;
esac
