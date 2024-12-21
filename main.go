package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html/v2"
	"github.com/momokii/simple-chat-app/internal/database"
	"github.com/momokii/simple-chat-app/internal/handlers"
	"github.com/momokii/simple-chat-app/internal/middlewares"
	"github.com/momokii/simple-chat-app/internal/models"
	"github.com/momokii/simple-chat-app/internal/repository/room"
	"github.com/momokii/simple-chat-app/internal/repository/user"
	"github.com/momokii/simple-chat-app/internal/ws"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	// db and session storage init
	database.InitDB()
	middlewares.InitSession()

	// repo init
	userRepo := user.NewUserRepo()
	roomRepo := room.NewRoomChatRepo()

	// handler init
	authHandler := handlers.NewAuthHandler(*userRepo)
	roomHandler := handlers.NewRoomChatHandler(*roomRepo)
	userHandler := handlers.NewUserHandler(*userRepo)

	// init websocket manager
	manager := ws.NewManager()

	engine := html.New("./web", ".html")
	app := fiber.New(fiber.Config{
		Views: engine,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).Render("error", fiber.Map{
				"Code":    code,
				"Message": err.Error(),
			})
		},
	})

	app.Use(cors.New())
	app.Use(logger.New())
	app.Static("/web", "./web")

	// websocket connection
	app.Get("/ws", adaptor.HTTPHandlerFunc(manager.ServeWS))

	// dashboard
	app.Get("/", middlewares.IsAuth, func(c *fiber.Ctx) error {
		user := c.Locals("user").(models.UserSession)

		// user := models.UserSession{
		// 	Id:       1,
		// 	Username: "test",
		// }

		return c.Render("dashboard", fiber.Map{
			"Title": "RoomPage - Chat Nge-Chat",
			"User":  user,
		})
	})

	app.Get("/chat", func(c *fiber.Ctx) error {
		return c.Render("index", fiber.Map{
			"Title": "RoomPage - Chat Nge-Chat",
		})
	})

	// base http route
	app.Get("/login", middlewares.IsNotAuth, authHandler.LoginView)
	app.Post("/login", middlewares.IsNotAuth, authHandler.Login)

	app.Get("/signup", middlewares.IsNotAuth, authHandler.SignUpView)
	app.Post("/signup", middlewares.IsNotAuth, authHandler.SignUp)

	app.Post("/logout", middlewares.IsAuth, authHandler.Logout)

	app.Get("/rooms", middlewares.IsAuth, roomHandler.GetRoomList)
	app.Post("/rooms", middlewares.IsAuth, roomHandler.CreateRoom)
	app.Patch("/rooms", middlewares.IsAuth, roomHandler.EditRoom)
	app.Delete("/rooms", middlewares.IsAuth, roomHandler.DeleteRoom)

	app.Patch("/users", middlewares.IsAuth, userHandler.ChangeUsername)
	app.Patch("/users/password", middlewares.IsAuth, userHandler.ChangePassword)

	// if using tls for https for accomodate wss in websocket can use below code to using it on local development, because browser will block wss connection if not using https
	// and when deploy on like gcp can just use app.Listen, because gcp will handle tls for us and the wss can still work
	if os.Getenv("APP_ENV") == "development" {
		log.Println("Running on development mode")
		if err := app.ListenTLS(":3000", "server.crt", "server.key"); err != nil {
			log.Println("Error running app: ", err)
		}
	} else if os.Getenv("APP_ENV") == "production" {
		log.Println("Running on production mode")
		if err := app.Listen(":3000"); err != nil {
			log.Println("Error running app: ", err)
		}
	} else {
		log.Println("APP_ENV not set")
	}
}
