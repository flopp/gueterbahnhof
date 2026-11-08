.PHONY: all clean run test

all: 
	go build -o bin/generator ./cmd/generator

clean:
	rm -rf bin

run:
	@go run ./cmd/generator --verbose --output ./.output

test:
	go test ./...