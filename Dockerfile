# Базовый образ для сборки
FROM golang:latest

# Копируем файлы приложения в рабочую директорию образа
COPY . /go/src/app
WORKDIR /go/src/app

# Устанавливаем необходимые зависимости
RUN go get -d -v ./...

# Собираем приложение
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o app .

# Добавляем сертификаты в образ
COPY cert /home/rino/.postgresql/

# Определяем точку входа в контейнер
ENTRYPOINT ["./app"]

# Запускаем контейнер с указанным портом
EXPOSE 8080
CMD ["app"]

