# Go Real-Time Chat

Real time chat application written in Go.

## Tech Stack

- Go (backend)
- React (frontend)
- PostgreSQL (DB)
- WebSockets

## Requirements

- Go
- Node.js
- PostgreSQL

### Backend Setup

```bash
cd backend
```

Create a .env file:
```env
DATABASE_URL=postgres://YOUR_USER:YOUR_PASSWORD@localhost:5432/chatdb?sslmode=disable
PORT=8080
```

Install the Go dependencies:
```bash
go mod download
```

Make sure PostgreSQL is running:
```bash
sudo service postgresql start
```

Database:
The application uses PostgreSQL.

Database migrations are located in: backend/migrations/

For now, migrations are applied manually.

### Run the backend
```bash
go run .
```

The API will be available at:
http://localhost:8080