binary := "aws-login"

# Show available recipes
default:
    @just --list

# Install binary to GOBIN/GOPATH
install: build
    go install .

# Build binary into bin/
build:
    mkdir -p bin
    go build -ldflags "-X github.com/ericcecchi/aws-login/internal/awslogin.version=$(git describe --tags --abbrev=0 2>/dev/null || echo dev)" -o bin/{{ binary }} .

# Run tests
test:
    go test ./...
