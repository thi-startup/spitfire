
#===========================================================================================================#
# BUILD
#===========================================================================================================#
version = $(shell cat ./VERSION)
current_time = $(shell date --iso-8601=seconds)
git_description = $(shell git describe --always --dirty --tags --long)
linker_flags = '-s -X main.buildTime=${current_time} -X main.hash=${git_description} -X main.version=${version}'

## build/api: build the cmd/api application
.PHONY: build
build:
	@echo 'Building cmd/api...'
	go build -ldflags=${linker_flags} -o=./bin/spitfire .

test:
	@go test $(shell go list ./...) -cover
