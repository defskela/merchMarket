# Используем официальный образ Go для сборки
FROM golang:1.21 AS builder
WORKDIR /app

# Копируем файлы проекта и загружаем зависимости
COPY go.mod go.sum ./
RUN go mod download

# Копируем исходники
COPY . .

# Запускаем тесты перед сборкой
RUN go test -v ./... 

# Собираем бинарник
RUN go build -o merchmarket ./cmd/app

# Минимальный образ для запуска
FROM alpine:latest
WORKDIR /root/

# Копируем скомпилированный бинарник
COPY --from=builder /app/merchmarket .

# Запускаем приложение
CMD ["./merchmarket"]
