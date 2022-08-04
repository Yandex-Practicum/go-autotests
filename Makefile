all: prep gophermarttest devopstest shortenertest statictest random _race perm
default: prep gophermarttest devopstest shortenertest statictest random perm
race: prep _race perm

prep:
	go mod tidy

gophermarttest:
	GOOS=linux GOARCH=amd64 go test -c -o=bin/gophermarttest ./cmd/gophermarttest/...

devopstest:
	GOOS=linux GOARCH=amd64 go test -c -o=bin/devopstest ./cmd/devopstest/...

shortenertest:
	GOOS=linux GOARCH=amd64 go test -c -o=bin/shortenertest ./cmd/shortenertest/...

statictest:
	GOOS=linux GOARCH=amd64 go build -o=bin/statictest ./cmd/statictest/...
	GOOS=windows GOARCH=amd64 go build -o=bin/statictest_windows-amd64 ./cmd/statictest/...
	GOOS=darwin GOARCH=amd64 go build -o=bin/statictest_darwin-amd64 ./cmd/statictest/...

random:
	GOOS=linux GOARCH=amd64 go build -o=bin/random ./cmd/random/...

_race:
	GOOS=linux GOARCH=amd64 go test -race -c -o=bin/shortenertest-race ./cmd/shortenertest/...

perm:
	chmod -R +x bin