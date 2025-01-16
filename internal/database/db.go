package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

func InitDB() {
	var err error

	host := os.Getenv("HOST_POSTGRES")
	port := os.Getenv("PORT_POSTGRES")
	user := os.Getenv("USER_POSTGRES")
	password := os.Getenv("PASSWORD_POSTGRES")
	dbname := os.Getenv("DATABASE_POSTGRES")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s", host, port, user, password, dbname)

	// check if using localhost (ofc that mean using development mode) so disabled the sslmode
	if host == "localhost" {
		connStr += " sslmode=disable"
	}

	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		panic(err)
	}

	if err := DB.Ping(); err != nil {
		panic(err)
	}

	log.Println("Successfully connected to database")
}
