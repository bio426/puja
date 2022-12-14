# syntax=docker/dockerfile:1

FROM golang:1.18.5-alpine

WORKDIR /app

COPY go.mod ./
COPY go.sum ./

RUN go mod download

COPY *.go ./

RUN go build -o /out

EXPOSE 1323

CMD ["/out"]
