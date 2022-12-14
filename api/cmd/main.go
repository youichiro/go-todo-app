package main

import (
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/volatiletech/sqlboiler/boil"
	"github.com/youichiro/go-todo-app/internal/client"
	"github.com/youichiro/go-todo-app/internal/router"
)

func main() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	env := os.Getenv("GO_ENV")

	boil.DebugMode = true // なぜか効かない
	db := client.InitDB(env)
	defer db.Close()

	r := router.SetupRouter(db)
	err = r.Run("0.0.0.0:8080")
	if err != nil {
		panic(err.Error())
	}
}
