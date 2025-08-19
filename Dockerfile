FROM golang:1.25-alpine as builder
WORKDIR /src/app
COPY go.mod go.sum ./
RUN go mod download
RUN go mod tidy
COPY . ./
RUN go build -ldflags "-s -w" GOEXPERIMENT=greenteagc -o go-exchange-api cmd/go-exchange-api/main.go
RUN go build -ldflags "-s -w" -o healthcheck cmd/healthcheck/main.go

FROM gcr.io/distroless/base-debian12:nonroot
ENV GIN_MODE release
WORKDIR /app
COPY --from=builder /src/app/go-exchange-api ./go-exchange-api
COPY --from=builder /src/app/healthcheck ./healthcheck
HEALTHCHECK --interval=30s --timeout=10s --retries=3 CMD ["./healthcheck"]
ENTRYPOINT ["./go-exchange-api"]
