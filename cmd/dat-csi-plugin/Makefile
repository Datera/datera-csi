VERSION ?= $(shell cat ../../VERSION)
GITHASH ?= $(shell git describe --match nEvErMatch --always --abbrev=10 --dirty)
GOSDK_V ?= $(shell go mod graph | grep "github.com/Datera/datera-csi github.com/Datera/go-sdk" | awk -F '@' '{print $$2}')
NAME=dat-csi-plugin

compile:
	@echo "==> Building the Datera CSI Driver Version ${VERSION}"
	@echo "==> datera-csi version ${VERSION}-${GITHASH}"
	@echo "==> go-sdk version ${GOSDK_V}"
	@env go get -d ./...
	@env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags 'osusergo netgo static_build' -o ${NAME} -ldflags "-X 'github.com/Datera/datera-csi/pkg/driver.Version=${VERSION}' -X 'github.com/Datera/datera-csi/pkg/driver.SdkVersion=${GOSDK_V}' -X 'github.com/Datera/datera-csi/pkg/driver.Githash=${GITHASH}'" github.com/Datera/datera-csi/cmd/dat-csi-plugin
	@env go vet ./...

iscsi-rpc:
	@echo "==> Building iscsi-rpc"
	@env go get -d ./...
	@env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags 'osusergo netgo static_build' -o iscsi-send github.com/Datera/datera-csi/cmd/iscsi-send
	@env go vet ./...
	@env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags 'osusergo netgo static_build' -o iscsi-recv github.com/Datera/datera-csi/cmd/iscsi-recv
	@env go vet ./...


local:
	@echo "==> datera-csi version ${VERSION}-${GITHASH}"
	@echo "==> go-sdk version ${GOSDK_V}"
	@echo "==> Building the Datera CSI Driver Version ${VERSION} For Local System"
	@env CGO_ENABLED=0 GOARCH=amd64 go build -tags 'osusergo netgo static_build' -o ${NAME} -ldflags "-X 'github.com/Datera/datera-csi/pkg/driver.Version=${VERSION}' -X 'github.com/Datera/datera-csi/pkg/driver.SdkVersion=${GOSDK_V}' -X 'github.com/Datera/datera-csi/pkg/driver.Githash=${GITHASH}'" github.com/Datera/datera-csi/cmd/dat-csi-plugin
	@env go vet ./...

iscsi-rpc-local:
	@echo "==> Building iscsi-rpc"
	@env go get -d ./...
	@env CGO_ENABLED=0 GOARCH=amd64 go build -tags 'osusergo netgo static_build' -o iscsi-send github.com/Datera/datera-csi/cmd/iscsi-send
	@env go vet ./...
	@env CGO_ENABLED=0 GOARCH=amd64 go build -tags 'osusergo netgo static_build' -o iscsi-recv github.com/Datera/datera-csi/cmd/iscsi-recv
	@env go vet ./...

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

build: compile iscsi-rpc
	@echo "==> Building Docker Image Latest"
	@docker build -t "dateraiodev/dat-csi-plugin:latest" ../.. -f Dockerfile
	@docker build -t "dateraiodev/iscsi:latest" ../.. -f ../../deploy/kubernetes/iscsi/Dockerfile

build-release: compile iscsi-rpc
	@echo "==> Building Docker Image $(VERSION)"
	@docker build -t "dateraiodev/dat-csi-plugin:$(VERSION)" ../.. -f Dockerfile

build-release-patch: compile iscsi-rpc
	@echo "==> Building Docker Image $(VERSION).p1"
	@docker build -t "dateraiodev/dat-csi-plugin:$(VERSION).p1" ../.. -f Dockerfile
