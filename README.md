Order SERVICE

Установка и запуск


* make dRun - установить все
Чтобы в кафку отправить данные, необходимо ввести - make producer дальше зайти в model.json и скопировать его -> вставить в консоль

Интерфейс сервиса будет доступен по адресу: http://localhost:8080
Swagger документация: http://localhost:8080/swagger/index.html#/

Технический стек

Язык программирования: Go 1.25.1
Web-фреймворк: Gin
База данных: PostgreSQL
Миграции: Migrate
Валидация: go-playground/validator
Логгирование: slog
Документация API: Swagger
Кэш: Redis 
Контейнеризация: Docker, Docker Compose
