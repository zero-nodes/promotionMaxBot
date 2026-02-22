# 🚀 Promotion Max Bot

Бот для активации купонов в мессенджере **Max** через [max-bot-api-client-go](https://github.com/max-messenger/max-bot-api-client-go).  
Пользователи отправляют фото чеков 📸, которые проходят модерацию ✅. Подтверждённые чеки превращаются в активные купоны 🎟️ и учитываются в рейтинговой таблице 🏆.


## ✨ Основные возможности

- 📝 **Регистрация и главное меню** для пользователей  
- 🎟️ **Активация купонов** с проверкой дневного лимита  
- ✅ **Модерация чеков** для администратора (подтвердить / отклонить)  
- 📊 **Рейтинговая таблица** и ссылки на канал  


## 🛠 Установка

1. **Клонировать репозиторий**  
```bash
git clone https://github.com/your-repo/promotion-max-bot.git
cd promotion-max-bot
````

2. **Создать `.env` с токенами и БД**

```env
TOKEN=your_max_bot_token
DB_URL=postgres://user:password@localhost:5432/dbname?sslmode=disable
THIS_API_TOKEN=some_secret_token_for_http_api
```

3. **Настроить `settings.json`** с текстами, ссылками и лимитами

4. **Собрать и запустить**

```bash
go mod download
go build -o bot .
./bot
```


## 🚀 Использование

Пользователь отправляет чек 📸 → бот сохраняет → администратор проверяет ✅ → купон становится активным 🎟️.


## 💻 Технологии

* **Go 1.21+**
* **PostgreSQL**
* **Max Bot API клиент**
* **Gorilla Mux** для HTTP-сервера



