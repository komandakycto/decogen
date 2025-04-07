V := @

test:
	go clean -testcache
	go test -race -coverprofile=coverage.out ./...

vendor:
	$(V)go mod tidy
	$(V)go mod vendor
	$(V)git add vendor