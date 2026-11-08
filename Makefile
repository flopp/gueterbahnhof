.PHONY: all clean run test

all: 
	go build -o bin/generator ./cmd/generator

clean:
	rm -rf bin

run:
	@go run ./cmd/generator --verbose --output ./.output
	@python3 -m http.server 8000 --bind 127.0.0.1 --directory ./.output

test:
	go test ./...