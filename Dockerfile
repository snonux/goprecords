FROM golang:1.21-alpine AS build
WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/goprecords ./cmd/goprecords

FROM alpine:3.20
RUN apk add --no-cache ca-certificates

WORKDIR /app
COPY --from=build /out/goprecords /usr/local/bin/goprecords

EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/goprecords"]
