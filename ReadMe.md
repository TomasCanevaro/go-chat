# Go Real-Time Chat

A real-time chat application built with **Go, PostgreSQL, React, and WebSockets**.

The project focuses on building a full-stack messaging application with secure authentication, persistent conversations, and real-time communication.

**Project status:** In development. The Go backend supports user authentication, direct conversations, PostgreSQL message persistence, and real-time messaging through WebSockets. Automated unit and integration tests are implemented. The React frontend is the next development phase.

## Tech Stack

### Backend
- **Go** — REST API and HTTP server using `net/http`
- **PostgreSQL** — Relational database
- **pgx** — PostgreSQL driver and connection pooling
- **JWT** — Token-based authentication
- **bcrypt** — Password hashing
- **WebSockets** — Real-time, bidirectional communication using `github.com/coder/websocket`

### Frontend (Planned)
- React
- Vite
- JavaScript

### Testing
- Go's built-in `testing` package
- `net/http/httptest`
- PostgreSQL integration tests
- WebSocket integration tests
- Go race detector

## Requirements

Before running the project, install:

- Git
- Go (compatible with the version specified in `backend/go.mod`)
- PostgreSQL
- Node.js and npm (required for the upcoming frontend)

Development is currently performed using Ubuntu on WSL2, but the Go backend can also run on other supported operating systems.

## Getting Started

### 1. Clone the repository

```bash
git clone git@github.com:TomasCanevaro/go-chat.git
cd go-chat
```

The SSH clone command requires GitHub SSH authentication to be configured.

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

Verify that the application user can connect:

```bash
psql -h localhost -U chatuser -d chatdb
```

Enter the password you configured, then exit using `\q`.

### 3. Configure environment variables

Navigate to the backend:

```bash
cd backend
```

Copy the example configuration:

```bash
cp .env.example .env
```

Configure the following variables in `backend/.env`:

```env
DATABASE_URL=postgres://chatuser:your_secure_password@localhost:5432/chatdb?sslmode=disable
PORT=8080
JWT_SECRET=your_generated_secret
```

Replace `your_secure_password` with your PostgreSQL password.

Generate a random JWT secret:

```bash
openssl rand -hex 32
```

Copy the generated value into `JWT_SECRET`.

**Security notes:**
- Never commit `.env` to Git.
- Use a strong, randomly generated JWT secret.
- `sslmode=disable` is intended for local development only.
- Production database connections should use an appropriate secure configuration.

### 4. Apply database migrations

Database migrations are stored in:

```text
backend/migrations/
```

Migrations are currently applied manually.

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

**Note:** These migrations are intended for a fresh database. Migration execution and version tracking are not yet automated.

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

`http://localhost:8080`

Opening this address should display:

```text
Go Chat API is running!
```

## REST API

The following endpoints are implemented:

| Method | Endpoint | Description | Authentication |
|---|---|---|---|
| POST | `/api/register` | Register a user | No |
| POST | `/api/login` | Log in and receive a JWT | No |
| GET | `/api/me` | Retrieve the authenticated user | Yes |
| GET | `/api/users` | List registered users | No |
| GET | `/api/conversations` | List the user's direct conversations | Yes |
| POST | `/api/conversations` | Create or retrieve a direct conversation | Yes |
| GET | `/api/conversations/{id}/messages` | Retrieve conversation messages | Yes |
| POST | `/api/conversations/{id}/messages` | Create and persist a message | Yes |

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

A successful login returns user information and a JWT token.

### Example: Retrieve authenticated user

```bash
curl -i http://localhost:8080/api/me \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

Replace `YOUR_JWT_TOKEN` with the token received during login.

## Real-Time Communication

The backend supports real-time messaging through WebSockets.

### WebSocket Endpoint

```text
ws://localhost:8080/ws
```

Clients establish a WebSocket connection and authenticate by sending a JWT as their first message.

### 1. Authenticate

After connecting, send:

```json
{
  "type": "auth",
  "token": "YOUR_JWT_TOKEN"
}
```

If authentication succeeds, the server responds:

```json
{
  "type": "auth_success",
  "message": "Authenticated successfully"
}
```

Invalid authentication attempts are rejected.

### 2. Send a message

After authentication, clients can send:

```json
{
  "type": "send_message",
  "conversation_id": 1,
  "content": "Hello!"
}
```

The backend:

1. Identifies the authenticated sender.
2. Validates the message content.
3. Verifies conversation membership.
4. Saves the message in PostgreSQL.
5. Broadcasts the saved message to connected conversation members.

The same message service handles messages created through both REST and WebSockets.

### 3. Receive a message

Connected conversation members receive a `new_message` event:

```json
{
  "type": "new_message",
  "message": {
    "id": 1,
    "conversation_id": 1,
    "sender_id": 2,
    "sender_username": "alice",
    "content": "Hello!",
    "created_at": "2026-10-27T12:00:00Z"
  }
}
```

Message IDs, user IDs, and timestamps are illustrative.

### 4. Error handling

If a message cannot be processed, the server sends an error event:

```json
{
  "type": "error",
  "message": "Conversation not found"
}
```

Errors are returned for cases such as:

- Invalid JSON
- Unknown message types
- Empty messages
- Messages exceeding the character limit
- Invalid or unauthorized conversations

### WebSocket Hub

The backend includes a WebSocket hub responsible for managing authenticated connections.

Its capabilities include:

- Registering and unregistering clients
- Supporting multiple connections per user
- Delivering events to specific users
- Broadcasting new messages to conversation members
- Synchronizing concurrent access using mutexes

**Current limitation:** WebSocket event delivery is best-effort. Disconnected clients must retrieve missed messages using the REST API. Automatic reconnection and delivery recovery are not yet implemented.

## Automated Testing

The backend includes unit and integration tests.

### Unit Tests

Unit tests cover:

- JWT generation and validation
- Invalid JWT signatures
- Malformed tokens
- Message content validation
- Whitespace normalization
- Unicode handling
- Message length limits
- WebSocket hub registration and removal
- Targeted message delivery
- Multiple connections per user
- Concurrent hub operations

### Integration Tests

Integration tests verify:

- Message persistence in PostgreSQL
- Conversation membership authorization
- Rejection of nonexistent conversations
- WebSocket authentication
- Real-time message delivery between authenticated users
- PostgreSQL persistence of WebSocket messages

### Configure the Test Database

Integration tests use a separate PostgreSQL database to avoid modifying development data.

Open PostgreSQL:

```bash
sudo -u postgres psql
```

Create the test database:

```sql
CREATE DATABASE chatdb_test OWNER chatuser;
```

Exit:

```sql
\q
```

From `backend/`, apply the migrations to the test database:

```bash
psql -h localhost -U chatuser -d chatdb_test \
  -f migrations/001_create_users.sql

psql -h localhost -U chatuser -d chatdb_test \
  -f migrations/002_create_chat_tables.sql

psql -h localhost -U chatuser -d chatdb_test \
  -f migrations/003_add_conversation_type.sql

psql -h localhost -U chatuser -d chatdb_test \
  -f migrations/004_add_direct_key.sql
```

Add the following variable to `backend/.env`:

```env
TEST_DATABASE_URL=postgres://chatuser:your_secure_password@localhost:5432/chatdb_test?sslmode=disable
```

The test database must be named `chatdb_test`.

### Run Tests

From `backend/`:

Run the complete test suite:

```bash
go test ./...
```

Run tests with verbose output:

```bash
go test ./... -v
```

Run only authentication tests:

```bash
go test ./internal/auth -v
```

Run only message service tests:

```bash
go test ./internal/services -v
```

Run only WebSocket tests:

```bash
go test ./internal/websocket -v
```

Run tests with Go's race detector:

```bash
go test -race ./...
```

The race detector helps identify unsafe concurrent memory access in code paths exercised by the tests.

**Note:** PostgreSQL must be running and the test database must be configured before executing integration tests. Some service integration tests skip when `TEST_DATABASE_URL` is missing, while WebSocket message integration tests require it.

## Current Features

### Authentication
- User registration
- bcrypt password hashing
- JWT-based authentication
- Protected REST endpoints
- Authenticated WebSocket connections

### Conversations
- Direct conversation creation
- Prevention of duplicate direct conversations
- Conversation listing
- Conversation membership authorization

### Messaging
- Sending messages through REST
- Sending messages through WebSockets
- PostgreSQL message persistence
- Retrieving conversation history
- Real-time message delivery
- Shared message service for REST and WebSocket operations
- Message validation and character limits

### WebSocket Infrastructure
- Authenticated client connections
- Connection registration and removal
- Multiple connections per user
- Targeted event delivery
- Concurrent connection management

### Testing
- JWT unit tests
- Message validation unit tests
- PostgreSQL integration tests
- WebSocket hub unit tests
- WebSocket authentication and messaging integration tests

## Planned Features

- React frontend using Vite
- Login and registration pages
- Conversation list and chat interface
- Real-time message updates in the frontend
- Automatic WebSocket reconnection
- Online/offline presence
- Message history pagination
- Improved WebSocket connection lifecycle and delivery handling
- Automated database migrations
- Additional automated tests
- Deployment configuration

## Development Notes

This project is under active development.

The backend currently supports persistent messaging through PostgreSQL and real-time communication through WebSockets.

REST and WebSocket message creation share the same service layer, ensuring consistent validation, authorization, and persistence behavior.

Database migrations are currently applied manually.

Although the backend includes automated tests, additional reliability, security, and deployment work is required before production use.

The next development phase focuses on implementing the React frontend.