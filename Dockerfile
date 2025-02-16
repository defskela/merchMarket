FROM golang:1.23

WORKDIR /app

COPY . /app

RUN go mod download

# Запуск тестов с предупреждением, но без остановки сборки
RUN go test -v ./... || echo "Tests failed, but continuing build..."

RUN go build -o merchmarket ./cmd/app

CMD ["./merchmarket"]