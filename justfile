binary := "aws-login"

# Show available recipes
default:
    @just --list

# Install binary to GOBIN/GOPATH
install:
    go install .

# Build binary into bin/
build:
    mkdir -p bin
    go build -o bin/{{ binary }} .

# Run tests
test:
    go test ./...
