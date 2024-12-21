package database

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/gofiber/fiber/v2"
)

func CommitOrRollback(tx *sql.Tx, c *fiber.Ctx, err error) {
	if p := recover(); p != nil {
		tx.Rollback()
		panic(p)
	} else if err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			log.Printf("error rollback transaction: %v (original error: %v)\n", rbErr, err)
		}
		log.Println("Rollback, error transaction: ", err)
	} else {
		if cErr := tx.Commit(); cErr != nil {
			err = fmt.Errorf("Error Commit Transaction: %v (Original Error: %w)", cErr, err)
		}
	}
}
