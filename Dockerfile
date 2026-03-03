FROM golang:1.24-alpine AS builder

WORKDIR /app

RUN apk add --no-cache ca-certificates git

COPY . .

RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/api ./cmd/api

FROM gcr.io/distroless/base-debian12 AS runtime

ENV APP_ENV=production

WORKDIR /

COPY --from=builder /bin/api /bin/api

EXPOSE 8080

USER nonroot:nonroot

ENTRYPOINT ["/bin/api"]

