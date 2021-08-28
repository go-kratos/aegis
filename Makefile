# sra Makefile

TOOLS_SH = "./hack/tools.sh"
PKG_DIR?=''

.PHONY: test
test:
	${TOOLS_SH} test ${PKG_DIR}

.PHONY: deps
deps:
	${TOOLS_SH} deps ${PKG_DIR}

.PHONY: lint
lint:
	curl -L https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s v1.42.0
	${TOOLS_SH} lint ${PKG_DIR}

.PHONY: build
build:
	${TOOLS_SH} build ${PKG_DIR}	