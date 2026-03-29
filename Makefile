GOPATH ?= $(GOPATH)

.PHONY: test
test:
	go clean -testcache
	go test -count=1 -race -covermode=atomic -coverprofile ./.project/cover.out `go list ./... | grep -v /mocks/ | grep -v /mocks | grep -v mocks`
	go run github.com/nikolaydubina/go-cover-treemap@latest -coverprofile ./.project/cover.out > ./.project/icons/coverage.svg


.PHONY: race-test 
race-test: 
	go test ./... -race -v -cover -covermode=atomic 