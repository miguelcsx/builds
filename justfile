# justfile

build:
    go build -o bin/builds ./cmd/builds

run *args:
    ./bin/builds {{args}}

test:
    go test ./...

clean:
    rm -rf bin/ build-*/

# Example usage:
# just run -output ./build-output clang -O2 -g -fopenmp --offload-arch=native main.c -foffload-lto -Rpass=kernel-info
