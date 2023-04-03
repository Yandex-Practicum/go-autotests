SHELL=/bin/bash

AUTOTESTS = gophermarttest metricstest devopstest shortenertest shortenertestbeta
UTILS = random statictest shortenerstress

all: prep autotests utils perm

prep:
	go mod tidy

autotests:
	$(foreach TARGET,$(AUTOTESTS),GOOS=linux GOARCH=amd64 go test -c -o=bin/$(TARGET)-linux-amd64 -o=bin/$(TARGET) ./cmd/$(TARGET)/... ;)
	$(foreach TARGET,$(AUTOTESTS),GOOS=windows GOARCH=amd64 go test -c -o=bin/$(TARGET)-windows-amd64.exe ./cmd/$(TARGET)/... ;)
	$(foreach TARGET,$(AUTOTESTS),GOOS=darwin GOARCH=amd64 go test -c -o=bin/$(TARGET)-darwin-amd64 ./cmd/$(TARGET)/... ;)
	$(foreach TARGET,$(AUTOTESTS),GOOS=darwin GOARCH=arm64 go test -c -o=bin/$(TARGET)-darwin-arm64 ./cmd/$(TARGET)/... ;)

utils:
	$(foreach TARGET,$(UTILS),GOOS=linux GOARCH=amd64 go build -buildvcs=false -o=bin/$(TARGET)-linux-amd64 -o=bin/$(TARGET) ./cmd/$(TARGET)/... ;)
	$(foreach TARGET,$(UTILS),GOOS=windows GOARCH=amd64 go build -buildvcs=false -o=bin/$(TARGET)-windows-amd64.exe ./cmd/$(TARGET)/... ;)
	$(foreach TARGET,$(UTILS),GOOS=darwin GOARCH=amd64 go build -buildvcs=false -o=bin/$(TARGET)-darwin-amd64 ./cmd/$(TARGET)/... ;)
	$(foreach TARGET,$(UTILS),GOOS=darwin GOARCH=arm64 go build -buildvcs=false -o=bin/$(TARGET)-darwin-arm64 ./cmd/$(TARGET)/... ;)

perm:
	chmod -R +x bin
