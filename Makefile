# sra Makefile

TOOLS_SH = "./hack/tools.sh"
PKG_DIR?=''

LINTER := bin/golangci-lint
$(LINTER): 
	curl -SL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s latest

.PHONY: test
test:
	${TOOLS_SH} test ${PKG_DIR}

.PHONY: deps
deps:
	${TOOLS_SH} deps ${PKG_DIR}

.PHONY: lint
lint: $(LINTER)
	${TOOLS_SH} lint ${PKG_DIR}

.PHONY: build
build:
	${TOOLS_SH} build ${PKG_DIR}	
