FROM golang:1.26-alpine AS build
RUN apk add --no-cache gcc musl-dev
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags='-s -w' -o /compose-port-registry ./cmd/compose-port-registry

FROM gcr.io/distroless/static:nonroot
COPY --from=build /compose-port-registry /compose-port-registry
USER nonroot:nonroot
ENTRYPOINT ["/compose-port-registry"]
