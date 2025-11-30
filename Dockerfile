FROM golang:1.25.1 AS builder

WORKDIR /app

# Install deps
COPY go.mod go.sum ./
RUN go mod download

# copy source
COPY . .

# Build Api binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/api ./cmd/api

# Build Seed binary
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/seed ./cmd/seed

# ---------- Runtime stage ----------
FROM alpine:3.20

WORKDIR /app

# minimal deps
RUN apk add --no-cache ca-certificates

# copy binaries
COPY --from=builder /out/api /app/api
COPY --from=builder /out/seed /app/seed

# default command = api (docker-compose ile override edeceğiz)
CMD ["/app/api"]
