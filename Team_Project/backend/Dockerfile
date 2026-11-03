# Stage 1: Build. temporary environment used only to compile the code. It will be completely discarded in the final image.
FROM golang:1.25-alpine AS build

WORKDIR /src

# Copy dependency manifests before source so Docker can cache downloads.
COPY go.mod go.sum ./
# Download Go module dependencies.
RUN go mod download

COPY . .
# Build the entry point from ./cmd/server as a standalone Linux binary.
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-s -w" \
    -o /out/book-metaverse-api \
    ./cmd/server

# Stage 2: Start from a small Linux image for the final runtime.
FROM alpine:3.22

RUN apk add --no-cache ca-certificates \
    && addgroup -S app \
    && adduser -S app -G app

# /usr/local/bin/book-metaverse-api is a fully compiled, standalone binary executable
COPY --from=build /out/book-metaverse-api /usr/local/bin/book-metaverse-api

USER app
# only for documentation, indicating application inside the container is designed to listen on port 8080.
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/book-metaverse-api"]
