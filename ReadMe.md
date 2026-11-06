# Go Chat

Real-time chat app built with Go, React and PostgreSQL.

## Requirements

- Git
- Go 1.27.1 or newer (see [`backend/go.mod`](backend/go.mod))
- Node.js 24 and npm
- PostgreSQL and its `psql` client
- OpenSSL to generate a local JWT secret

The commands below use Ubuntu / WSL and Bash. Run them from the repository root unless noted otherwise.

## 1. Clone

```bash
git clone https://github.com/TomasCanevaro/go-chat.git
cd go-chat
```

## 2. Set up PostgreSQL

If PostgreSQL is not installed:

```bash
sudo apt update
sudo apt install postgresql postgresql-contrib
```

Start it and open the administrator console:

```bash
sudo service postgresql start
sudo -u postgres psql
```

Create the application role and database. Replace `your_password` with a local development password:

```sql
CREATE USER chatuser WITH PASSWORD 'your_password';
CREATE DATABASE chatdb OWNER chatuser;
\q
```

Apply all migrations in order to the new database. Enter the password you just chose when prompted:

```bash
psql -h localhost -U chatuser -d chatdb -v ON_ERROR_STOP=1 --single-transaction \
  -f backend/migrations/001_create_users.sql \
  -f backend/migrations/002_create_chat_tables.sql \
  -f backend/migrations/003_add_conversation_type.sql \
  -f backend/migrations/004_add_direct_key.sql \
  -f backend/migrations/005_add_message_client_id.sql
```

Run these migrations once on a fresh database; the backend does not apply them automatically.

## 3. Configure the backend

```bash
cp backend/.env.example backend/.env
openssl rand -hex 32
```

Edit `backend/.env`, using the database password above and the generated secret:

```dotenv
DATABASE_URL=postgres://chatuser:your_password@localhost:5432/chatdb?sslmode=disable
PORT=8080
JWT_SECRET=paste_the_generated_secret_here
```

If your password contains URI-reserved characters such as `@`, URL-encode them in `DATABASE_URL`. Keep `.env` local; it is ignored by Git. `sslmode=disable` is for local development.

## 4. Start the backend

In one terminal:

```bash
cd backend
go mod download
go run .
```

Run from `backend/` so the server can load `.env`. Check [http://localhost:8080](http://localhost:8080): it should display `Go Chat API is running!`.

## 5. Start the frontend

In a second terminal, from the repository root:

```bash
cd frontend
npm i
npm run dev
```

Open [http://localhost:5173](http://localhost:5173). Vite proxies API and WebSocket requests to the backend on port 8080; no frontend `.env` is needed. Keep that backend port unless you also update [`frontend/vite.config.ts`](frontend/vite.config.ts).

Register two accounts, then log in using a normal window and an incognito window. Click **New conversation**, select the other user, and send a message. The recipient's sidebar updates automatically.

Both servers must be running to use the app. Stop each with **Ctrl+C**. On later runs, start PostgreSQL if needed, then use `go run .` and `npm run dev` from their respective directories.

## Checks (optional)

Frontend, from `frontend/`:

```bash
npm run lint
npm run build
```

Backend integration tests require a separate database named **`chatdb_test`**. Create it using the same role:

```bash
sudo -u postgres psql -c 'CREATE DATABASE chatdb_test OWNER chatuser;'
```

From the repository root, apply its migrations:

```bash
psql -h localhost -U chatuser -d chatdb_test -v ON_ERROR_STOP=1 --single-transaction \
  -f backend/migrations/001_create_users.sql \
  -f backend/migrations/002_create_chat_tables.sql \
  -f backend/migrations/003_add_conversation_type.sql \
  -f backend/migrations/004_add_direct_key.sql \
  -f backend/migrations/005_add_message_client_id.sql
```

Add this to `backend/.env`, using the same password as above:

```dotenv
TEST_DATABASE_URL=postgres://chatuser:your_password@localhost:5432/chatdb_test?sslmode=disable
```

With PostgreSQL running, execute from `backend/`:

```bash
go test ./...
```
