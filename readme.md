<p align="center"><img src="https://uploads.photo/images/Ed7f.png" width="200"/></p>

<p align="center" style="font-size:1.8em;">Мой блог. Бот для чата</p>

##  Список технологий

- Язык программирования: [Go](https://golang.org/doc/)
- [Telegram Bot API](https://github.com/go-telegram-bot-api/telegram-bot-api)
- [RabbitMQ Client Library](https://github.com/streadway/amqp)
- [Proxy](https://github.com/elazarl/goproxy)

## Разворачивание проекта для разработки

1. Скопировать файл окружения
    ```bash
    cp ./.env.dist ./.env
    ```
    
2. Заменить переменные окружения в созданном файле

3. Скомпилировать
    ```bash
    go build .
    ```

4. Запустить
    ```bash
    make run-dev
    ```

## Разворачивание проекта для работы

1. Скопировать файл окружения
    ```bash
    cp ./.env.dist ./.env
    ```
    
2. Заменить переменные окружения в созданном файле

3. Скомпилировать
    ```bash
    make build
    ```

4. Запустить
    ```bash
    make run
    ```
