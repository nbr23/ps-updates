FROM golang:alpine as builder

WORKDIR /build

RUN apk add gcc musl-dev
COPY go* main.go .

RUN go build -trimpath -o /build/ps-updates

FROM alpine

COPY --from=builder /build/ps-updates /usr/bin/ps-updates

CMD ps-updates