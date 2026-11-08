.PHONY: all clean

all: 
	go build -o bin/generator ./cmd/generator

clean:
	rm -rf bin

run:
	@go run ./cmd/generator --verbose --output ./.output