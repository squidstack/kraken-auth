# kraken-auth/Dockerfile
FROM golang:1.24-alpine AS build
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY . .
RUN go mod tidy
RUN CGO_ENABLED=0 go build -a -ldflags '-extldflags "-static"' -o app main.go

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /app/app .
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["./app"]