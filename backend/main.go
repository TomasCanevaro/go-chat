package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"go-chat/backend/internal/auth"
	"go-chat/backend/internal/database"
	"go-chat/backend/internal/handlers"

	"github.com/joho/godotenv"
)

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "Go Chat API is running!")
}

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	jwtSecret := os.Getenv("JWT_SECRET")

	if jwtSecret == "" {
		log.Fatal("JWT_SECRET is not set")
	}

	db, err := database.Connect(databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("Connected to PostgreSQL!")

	userHandler := &handlers.UserHandler{
		DB:        db,
		JWTSecret: jwtSecret,
	}

	conversationHandler := &handlers.ConversationHandler{
		DB: db,
	}

	messageHandler := &handlers.MessageHandler{
		DB: db,
	}

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/api/users", userHandler.GetUsers)
	http.HandleFunc("/api/register", userHandler.Register)
	http.HandleFunc("/api/login", userHandler.Login)
	http.HandleFunc(
		"/api/me",
		auth.Middleware(jwtSecret, userHandler.Me),
	)
	http.HandleFunc(
		"/api/conversations",
		auth.Middleware(jwtSecret, conversationHandler.Handle),
	)
	http.HandleFunc(
		"GET /api/conversations/{id}/messages",
		auth.Middleware(jwtSecret, messageHandler.List),
	)
	http.HandleFunc(
		"POST /api/conversations/{id}/messages",
		auth.Middleware(jwtSecret, messageHandler.Create),
	)

	port := os.Getenv("PORT")
	fmt.Printf("Server running on http://localhost:%s\n", port)

	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal(err)
	}
}
