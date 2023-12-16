FROM golang:alpine as builder
WORKDIR /src/app
COPY go.mod go.sum ./
RUN go mod download
COPY . ./
RUN go build -ldflags "-s -w" -o go-exchange-api

FROM alpine
ENV GIN_MODE release
WORKDIR /app
COPY --from=builder /src/app/go-exchange-api ./go-exchange-api
RUN addgroup -S appgroup && \
    adduser -S appuser -G appgroup
USER appuser
ENTRYPOINT ["./go-exchange-api"]
