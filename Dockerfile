FROM golang:alpine as builder
WORKDIR /src/app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN go build -ldflags "-s -w" -o go-exchange-api cmd/go-exchange-api/main.go
RUN go build -ldflags "-s -w" -o healthcheck cmd/healthcheck/main.go

FROM alpine
ENV GIN_MODE release
WORKDIR /app
COPY --from=builder /src/app/go-exchange-api ./go-exchange-api
COPY --from=builder /src/app/healthcheck ./healthcheck
RUN addgroup -S appgroup && \
    adduser -S appuser -G appgroup
USER appuser
HEALTHCHECK --interval=30s --timeout=10s --retries=3 CMD ["./healthcheck"]
ENTRYPOINT ["./go-exchange-api"]
