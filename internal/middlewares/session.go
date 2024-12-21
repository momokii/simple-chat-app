package middlewares

import (
	"log"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	"github.com/momokii/simple-chat-app/internal/database"
	"github.com/momokii/simple-chat-app/internal/models"
	"github.com/momokii/simple-chat-app/internal/repository/user"
)

var Store *session.Store

func InitSession() {
	Store = session.New(session.Config{
		Expiration:     7 * time.Hour,
		CookieSecure:   true,
		CookieHTTPOnly: true,
	})

	log.Println("Session store initialized")
}

func CreateSession(c *fiber.Ctx, key string, value interface{}) error {
	sess, err := Store.Get(c)
	if err != nil {
		return err
	}
	defer sess.Save()

	sess.Set(key, value)

	return nil
}

func DeleteSession(c *fiber.Ctx) error {
	sess, err := Store.Get(c)
	if err != nil {
		return err
	}
	defer sess.Save()

	sess.Destroy()

	return nil
}

func CheckSession(c *fiber.Ctx, key string) (interface{}, error) {
	sess, err := Store.Get(c)
	if err != nil {
		return nil, err
	}

	return sess.Get(key), nil
}

func IsNotAuth(c *fiber.Ctx) error {
	token, err := CheckSession(c, "id")
	if err != nil {
		DeleteSession(c)
		return c.Redirect("/login")
	}

	if token != nil {
		return c.Redirect("/")
	}

	return c.Next()
}

func IsAuth(c *fiber.Ctx) error {
	tokenid, err := CheckSession(c, "id")
	if err != nil {
		DeleteSession(c)
		return c.Redirect("/login")
	}

	if tokenid == nil {
		DeleteSession(c)
		return c.Redirect("/login")
	}

	tx, err := database.DB.Begin()
	if err != nil {
		DeleteSession(c)
		return c.Redirect("/login")
	}
	defer func() {
		database.CommitOrRollback(tx, c, err)
	}()

	userRepo := user.NewUserRepo()
	userData, err := userRepo.FindByID(tx, tokenid.(int))
	if err != nil {
		DeleteSession(c)
		return c.Redirect("/login")
	}

	userSession := models.UserSession{
		Id:       userData.Id,
		Username: userData.Username,
	}

	// store information for next data
	c.Locals("user", userSession)

	return c.Next()
}
