# Go Real-Time Chat

Real-time chat application built with Go, PostgreSQL, React, and WebSockets.

**Project status:** In development. The Go REST API supports user authentication, direct conversations, and message persistence. The React frontend and real-time WebSocket functionality are not yet implemented.

## Tech Stack

- **Backend:** Go (`net/http`)
- **Database:** PostgreSQL
- **Database driver:** pgx
- **Authentication:** JWT and bcrypt
- **Frontend (planned):** React + Vite
- **Real-time communication (planned):** WebSockets

## Requirements

Before running the project, install:

- Git
- Go (compatible with the version specified in `backend/go.mod`)
- PostgreSQL
- Node.js and npm (required when the frontend is implemented)

Development is currently performed using Ubuntu on WSL2, but the Go backend can also run on other supported operating systems.

## Getting Started

### 1. Clone the repository

```bash
git clone git@github.com:TomasCanevaro/go-chat.git
cd go-chat
```

### 2. Set up PostgreSQL

On Ubuntu or WSL, install PostgreSQL if necessary:

```bash
sudo apt update
sudo apt install postgresql postgresql-contrib
```

Start the database service:

```bash
sudo service postgresql start
```

Open PostgreSQL as the administrator:

```bash
sudo -u postgres psql
```

Create a dedicated application user and database:

```sql
CREATE USER chatuser WITH PASSWORD 'your_secure_password';
CREATE DATABASE chatdb OWNER chatuser;
```

Exit PostgreSQL:

```sql
\q
```

Verify that the new user can connect:

```bash
psql -h localhost -U chatuser -d chatdb
```

Enter the password you configured, then exit with `\q`.

### 3. Configure environment variables

Navigate to the backend:

```bash
cd backend
```

Create a `.env` file inside the `backend` directory:

```env
DATABASE_URL=postgres://chatuser:your_secure_password@localhost:5432/chatdb?sslmode=disable
PORT=8080
JWT_SECRET=your_generated_secret
```

Replace `your_secure_password` with the PostgreSQL password you created.

Generate a random JWT secret:

```bash
openssl rand -hex 32
```

Copy the generated value into `JWT_SECRET`.

The `sslmode=disable` setting is intended for the local development database. Production database connections should use an appropriate secure configuration.

### 4. Apply database migrations

Database migrations are stored in:

```text
backend/migrations/
```

Currently, migrations are applied manually.

From the `backend` directory, connect to PostgreSQL:

```bash
psql -h localhost -U chatuser -d chatdb
```

Apply the migrations in numerical order:

```sql
\i migrations/001_create_users.sql
\i migrations/002_create_chat_tables.sql
\i migrations/003_add_conversation_type.sql
\i migrations/004_add_direct_key.sql
```

Verify the tables:

```sql
\dt
```

The following tables should exist:

- `users`
- `conversations`
- `conversation_members`
- `messages`

Exit with `\q`.

**Note:** These migrations are intended for a fresh database and are not automatically tracked or reapplied by a migration tool yet.

### 5. Install Go dependencies

From `backend/`:

```bash
go mod download
```

### 6. Run the backend

```bash
go run .
```

Expected output:

```text
Connected to PostgreSQL!
Server running on http://localhost:8080
```

The API is now available at:

http://localhost:8080

Opening this URL should display:

```text
Go Chat API is running!
```

## API Endpoints

The following endpoints are currently implemented:

| Method | Endpoint | Description | Authentication |
|---|---|---|---|
| POST | `/api/register` | Register a user | No |
| POST | `/api/login` | Log in and receive a JWT | No |
| GET | `/api/me` | Retrieve the authenticated user | Yes |
| GET | `/api/users` | List registered users | No |
| GET | `/api/conversations` | List the user's direct conversations | Yes |
| POST | `/api/conversations` | Create or retrieve a direct conversation | Yes |
| GET | `/api/conversations/{id}/messages` | Retrieve conversation messages | Yes |
| POST | `/api/conversations/{id}/messages` | Send a message | Yes |

Protected endpoints require the following HTTP header:

```http
Authorization: Bearer YOUR_JWT_TOKEN
```

### Example: Register a user

```bash
curl -i -X POST http://localhost:8080/api/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "alice",
    "email": "alice@example.com",
    "password": "password123"
  }'
```

A successful registration returns HTTP `201 Created`.

### Example: Log in

```bash
curl -i -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "alice@example.com",
    "password": "password123"
  }'
```

A successful login returns the user's information and a JWT token.

### Example: Retrieve authenticated user

```bash
curl -i http://localhost:8080/api/me \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

Replace `YOUR_JWT_TOKEN` with the token received from login.

## Current Features

- User registration with bcrypt password hashing
- User login with JWT authentication
- Protected API endpoints
- PostgreSQL persistence
- Direct conversation creation
- Prevention of duplicate direct conversations using a unique conversation key
- Listing conversations belonging to the authenticated user
- Sending and retrieving messages
- Conversation membership authorization

## Planned Features

- React frontend with login and registration pages
- Chat interface
- Real-time messaging using WebSockets
- Online/offline presence
- Message history pagination
- Improved configuration and automated database migrations
- Automated tests
- Deployment configuration

## Development Notes

The project is currently under active development.

The REST API stores and retrieves messages through PostgreSQL. Real-time message delivery is not yet implemented.

Database migrations must currently be applied manually, and the API has not yet been prepared for production deployment.
