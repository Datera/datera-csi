VERSION ?= $(shell cat ../../VERSION)
GITHASH ?= $(shell git describe --match nEvErMatch --always --abbrev=10 --dirty)
GOSDK_V ?= $(shell go mod graph | grep "github.com/Datera/datera-csi github.com/Datera/go-sdk" | awk -F '@' '{print $$2}')
NAME=dat-csi-plugin

# This builds the CSI plugin for linux
compile:
	@echo "==> Building the Datera CSI Driver Version ${VERSION}"
	@echo "==> datera-csi version ${VERSION}-${GITHASH}"
	@echo "==> go-sdk version ${GOSDK_V}"
	@env go get -d ./...
	@env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags 'osusergo netgo static_build' -o ${NAME} -ldflags "-X 'github.com/Datera/datera-csi/pkg/driver.Version=${VERSION}' -X 'github.com/Datera/datera-csi/pkg/driver.SdkVersion=${GOSDK_V}' -X 'github.com/Datera/datera-csi/pkg/driver.Githash=${GITHASH}'" github.com/Datera/datera-csi/cmd/dat-csi-plugin
	@env go vet ./...

# This builds just the iscsi-send and iscsi-recv binaries for linux
iscsi-rpc:
	@echo "==> Building iscsi-rpc"
	@env go get -d ./...
	@env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags 'osusergo netgo static_build' -o iscsi-send github.com/Datera/datera-csi/cmd/iscsi-send
	@env go vet ./...
	@env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags 'osusergo netgo static_build' -o iscsi-recv github.com/Datera/datera-csi/cmd/iscsi-recv
	@env go vet ./...

# This builds the CSI plugin for the local system
local:
	@echo "==> datera-csi version ${VERSION}-${GITHASH}"
	@echo "==> go-sdk version ${GOSDK_V}"
	@echo "==> Building the Datera CSI Driver Version ${VERSION} For Local System"
	@env CGO_ENABLED=0 GOARCH=amd64 go build -tags 'osusergo netgo static_build' -o ${NAME} -ldflags "-X 'github.com/Datera/datera-csi/pkg/driver.Version=${VERSION}' -X 'github.com/Datera/datera-csi/pkg/driver.SdkVersion=${GOSDK_V}' -X 'github.com/Datera/datera-csi/pkg/driver.Githash=${GITHASH}'" github.com/Datera/datera-csi/cmd/dat-csi-plugin
	@env go vet ./...

# This builds just the iscsi-send and iscsi-recv binaries for the local system
iscsi-rpc-local:
	@echo "==> Building iscsi-rpc"
	@env go get -d ./...
	@env CGO_ENABLED=0 GOARCH=amd64 go build -tags 'osusergo netgo static_build' -o iscsi-send github.com/Datera/datera-csi/cmd/iscsi-send
	@env go vet ./...
	@env CGO_ENABLED=0 GOARCH=amd64 go build -tags 'osusergo netgo static_build' -o iscsi-recv github.com/Datera/datera-csi/cmd/iscsi-recv
	@env go vet ./...

# This builds the "latest" image
build: compile iscsi-rpc
	@echo "==> Building Docker Image Latest"
	@docker build -t "dateraiodev/dat-csi-plugin:latest" ../.. -f Dockerfile
	@docker build -t "dateraiodev/iscsi:latest" ../.. -f ../../deploy/kubernetes/iscsi/Dockerfile

# This builds the image for the version stored in the VERSION file
build-release: compile iscsi-rpc
	@echo "==> Building Docker Image $(VERSION)"
	@docker build -t "dateraiodev/dat-csi-plugin:$(VERSION)" ../.. -f Dockerfile

# This bulids a patch image for the current version in the VERSION file.  Useful
# if a hotfix needs to be made
build-release-patch: compile iscsi-rpc
	@echo "==> Building Docker Image $(VERSION).p1"
	@docker build -t "dateraiodev/dat-csi-plugin:$(VERSION).p1" ../.. -f Dockerfile

# These are just various testing makefile calls.  They're not super useful
# because make buffers the output and doesn't display anything until it's
# finished.  It's usually better just to run these commands directly
test:
	@echo "==> Testing the Datera CSI Driver"
	@sudo env "PATH=${PATH}" go test ../../pkg/client ../../pkg/driver

testv:
	@echo "==> Testing the Datera CSI Driver (verbose)"
	@sudo env "PATH=${PATH}" go test -v ../../pkg/client ../../pkg/driver

testcc:
	@echo "==> Testing the Datera CSI Driver Client (compile only)"
	@env go test -c ../../pkg/client

testcd:
	@echo "==> Testing the Datera CSI Driver (compile only)"
	@env go test -c ../../pkg/driver

clean:
	@echo "==> Cleaning artifacts"
	@GOOS=linux go clean -i -x ./...
	rm -f iscsi-recv iscsi-send
