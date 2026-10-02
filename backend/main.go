package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

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

	db, err := database.Connect(databaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("Connected to PostgreSQL!")

	userHandler := &handlers.UserHandler{
		DB: db,
	}

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/api/users", userHandler.GetUsers)
	http.HandleFunc("/api/register", userHandler.Register)
	http.HandleFunc("/api/login", userHandler.Login)

	port := os.Getenv("PORT")
	fmt.Printf("Server running on http://localhost:%s\n", port)

	err = http.ListenAndServe(":"+port, nil)
	if err != nil {
		log.Fatal(err)
	}
}