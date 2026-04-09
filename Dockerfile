FROM golang:1.26-bookworm

WORKDIR /app
COPY go.mod ./
COPY xorcipher/ ./xorcipher/

CMD ["bash", "-c", "GOEXPERIMENT=simd go test -bench=. -benchmem -count=5 -timeout=10m ./xorcipher/"]