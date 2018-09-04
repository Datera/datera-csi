VERSION ?= v0.1.0
NAME=dat-csi-plugin

compile:
	@echo "==> Building the Datera CSI Driver"
	@env go get -d ./...
	@env CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o ${NAME}
	@env go vet ./...

local:
	@echo "==> Building the Datera CSI Driver"
	@env CGO_ENABLED=0 GOARCH=amd64 go build -o ${NAME}
	@env go vet ./...

test:
	@echo "==> Testing the Datera CSI Driver"
	@sudo env "PATH=${PATH}" go test ../../pkg/client ../../pkg/driver

testv:
	@echo "==> Testing the Datera CSI Driver"
	@sudo env "PATH=${PATH}" go test -v ../../pkg/client ../../pkg/driver

clean:
	@echo "==> Cleaning artifacts"
	@GOOS=linux go clean -i -x ./...
