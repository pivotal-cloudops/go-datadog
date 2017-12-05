default: test

test: deps
	go test ./...

deps: 
	go get -t ./...
