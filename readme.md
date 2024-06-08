Для запуска приложения понадобится:
1) Установить docker
2) Создать файл .env на основе "example.env" и прописать конфигурации
3) Создать файл redis.conf на основе "redis_example.conf" и прописать пароль как в .env
4) Для локальной разработки запустить "docker compose -f docker-compose.dev.yaml up --build" для прода "docker compose up --build"
5) Запустить миграции: "docker run -v "${PWD}/migrations:/migrations" --network host migrate/migrate -path=/migrations/ -database postgres://{user}:{password}@localhost:5432/book_stealer?sslmode=disable up" 
