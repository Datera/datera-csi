VERSION ?= $(shell cat ../../VERSION)
GITHASH ?= $(shell git describe --match nEvErMatch --always --abbrev=10 --dirty)
NAME=dat-csi-plugin

compile:
	@echo "==> Building the Datera CSI Driver Version ${VERSION}"
	@env go get -d ./...
	@env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags 'osusergo netgo static_build' -o ${NAME} -ldflags "-X 'github.com/Datera/datera-csi/pkg/driver.Version=${VERSION}' -X 'github.com/Datera/datera-csi/pkg/driver.Githash=${GITHASH}'"
	@env go vet ./...

iscsi-rpc:
	@echo "==> Building iscsi-rpc"
	@env go get -d ./...
	@cd ../iscsi-send
	@env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags 'osusergo netgo static_build' -o iscsi-send
	@env go vet ./...
	@cd ../iscsi-recv
	@env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -tags 'osusergo netgo static_build' -o iscsi-recv
	@env go vet ./...
	@cd ../dat-csi-plugin/


local:
	@echo "==> Building the Datera CSI Driver Version ${VERSION} For Local System"
	@env CGO_ENABLED=0 GOARCH=amd64 go build -tags 'osusergo netgo static_build' -o ${NAME} -ldflags "-X 'github.com/Datera/datera-csi/pkg/driver.Version=${VERSION}' -X 'github.com/Datera/datera-csi/pkg/driver.Githash=${GITHASH}'"
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

build: compile iscsi-rpc
	@echo "==> Building Docker Image Latest"
	@docker build -t "dateraiodev/dat-csi-plugin:latest" ../.. -f Dockerfile
	@docker build -t "dateraiodev/iscsi:latest" ../.. -f ../../deploy/kubernetes/iscsi/Dockerfile

build-release: compile iscsi-rpc
	@echo "==> Building Docker Image $(VERSION)"
	@docker build -t "dateraiodev/dat-csi-plugin:$(VERSION)" ../.. -f Dockerfile
