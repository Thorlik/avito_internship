# Запуск

## Запуск сервиса

```bash
git clone <repository-url>
cd avito_internship

docker-compose up -d
```

Сервис будет доступен по адресу `http://localhost:8080`

### Остановка

```bash
docker-compose down
```

### Пересборка после изменений

```bash
docker-compose up -d --build
```

## API Endpoints

- `POST /team/add` - Создать команду
- `GET /team/get?team_name=<name>` - Получить команду
- `POST /users/setIsActive` - Установить статус пользователя
- `GET /users/getReview?user_id=<id>` - Получить PR пользователя
- `POST /pullRequest/create` - Создать PR
- `POST /pullRequest/merge` - Смержить PR
- `POST /pullRequest/reassign` - Переназначить ревьювера
- `GET /statistics` - Статистика системы