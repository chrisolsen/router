test:
	go test
.PHONY: test

test-coverage:
	go test -v -coverprofile cover.out .
	go tool cover -html=cover.out -o cover.html
.PHONY: test-coverage
