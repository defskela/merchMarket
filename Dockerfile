FROM golang:1.23

WORKDIR /app

COPY . /app

RUN go mod download

RUN go build -o merchmarket ./cmd/app

CMD ["./merchmarket"]