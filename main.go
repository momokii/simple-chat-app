package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/template/html/v2"
	"github.com/momokii/go-llmbridge/pkg/openai"
	"github.com/momokii/simple-chat-app/internal/database"
	"github.com/momokii/simple-chat-app/internal/handlers"
	"github.com/momokii/simple-chat-app/internal/middlewares"
	"github.com/momokii/simple-chat-app/internal/repository/message"
	"github.com/momokii/simple-chat-app/internal/repository/room"
	roommember "github.com/momokii/simple-chat-app/internal/repository/room_member"
	"github.com/momokii/simple-chat-app/internal/repository/room_train"
	"github.com/momokii/simple-chat-app/internal/repository/session"
	"github.com/momokii/simple-chat-app/internal/repository/user"
	"github.com/momokii/simple-chat-app/internal/ws"

	sso_conn_room_reserved "github.com/momokii/go-sso-web/pkg/repository/conn_room_credit_reserved"
	sso_user "github.com/momokii/go-sso-web/pkg/repository/user"
	sso_credit_reserved "github.com/momokii/go-sso-web/pkg/repository/user_credit_reserved"

	_ "github.com/joho/godotenv/autoload"
)

func main() {
	// llm client init
	gptClient, err := openai.New(
		os.Getenv("OA_APIKEY"),
		os.Getenv("OA_ORGANIZATIONID"),
		os.Getenv("OA_PROJECTID"),
		openai.WithModel("gpt-4o-mini"),
	)
	if err != nil {
		log.Println("Error when init openai client: ", err)
	} else {
		log.Println("OpenAI client is ready")
	}

	// db and session storage init
	database.InitDB()
	middlewares.InitSession()

	// repo init
	userRepo := user.NewUserRepo()
	roomRepo := room.NewRoomChatRepo()
	roomTrainRepo := room_train.NewRoomChatTrainRepo()
	messageRepo := message.NewMessageRepo()
	roomemberRepo := roommember.NewRoomMember()
	sessionRepo := session.NewSessionRepo()
	SSOCreditReservedRepo := sso_credit_reserved.NewUserCreditReserved()
	SSOConnReservedRoomRepo := sso_conn_room_reserved.NewConnRoomCreditReserved()
	SSOUser := sso_user.NewUserRepo()

	// handler init
	authHandler := handlers.NewAuthHandler(*userRepo, *sessionRepo)
	roomHandler := handlers.NewRoomChatHandler(*roomRepo, *roomTrainRepo, *roomemberRepo, gptClient, *SSOUser, *SSOCreditReservedRepo, *SSOConnReservedRoomRepo)
	userHandler := handlers.NewUserHandler(*userRepo)
	messageHandler := handlers.NewMessageHandler(*roomRepo, *messageRepo, gptClient, *roomTrainRepo, *SSOCreditReservedRepo)

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
	api := app.Group("/api")

	app.Use(middlewares.IsServerActive) // check if server is active
	app.Use(cors.New())
	app.Use(logger.New())
	app.Static("/web", "./web")

	// dashboard
	app.Get("/", middlewares.IsAuth, roomHandler.RoomMainDashboardView)

	// base http route

	// not using login and signup page, because using SSO
	// app.Get("/login", middlewares.IsNotAuth, authHandler.LoginView)
	// api.Post("/login", middlewares.IsNotAuth, authHandler.Login)

	// app.Get("/signup", middlewares.IsNotAuth, authHandler.SignUpView)
	// api.Post("/signup", middlewares.IsNotAuth, authHandler.SignUp)

	// auth sso
	app.Get("/auth/sso", middlewares.IsNotAuth, authHandler.SSOAuthLogin)

	api.Post("/logout", middlewares.IsAuth, authHandler.Logout)

	// room page
	app.Get("/rooms/:room_code/train", middlewares.IsAuth, roomHandler.RoomTrainChatView)
	api.Get("/rooms/:room_code/train/detail", middlewares.IsAuth, roomHandler.GetTrainRoomData)
	app.Get("/rooms/:room_code", middlewares.IsAuth, roomHandler.RoomChatView)
	api.Get("/rooms/:room_code", middlewares.IsAuth, roomHandler.GetRoomData)
	api.Get("/rooms", middlewares.IsAuth, roomHandler.GetRoomList)
	api.Post("/rooms/train", middlewares.IsAuth, roomHandler.CreateTrainRoom)
	api.Post("/rooms", middlewares.IsAuth, roomHandler.CreateRoom)
	api.Patch("/rooms", middlewares.IsAuth, roomHandler.EditRoom)
	api.Delete("/rooms", middlewares.IsAuth, roomHandler.DeleteRoom)

	api.Post("/rooms/members", middlewares.IsAuth, roomHandler.AddJoinRoom)
	api.Delete("/rooms/members", middlewares.IsAuth, roomHandler.RemoveRoomMember)

	app.Get("/ws/:room_code", adaptor.HTTPHandlerFunc(manager.ServeWS)) // websocket connection
	api.Get("/messages/:room_code", middlewares.IsAuth, messageHandler.GetMessageByRoom)
	api.Post("/messages/train/save", middlewares.IsAuth, messageHandler.SaveMessageLLM)
	api.Post("/messages/train", middlewares.IsAuth, messageHandler.SendMessageTrain)
	api.Post("/messages", middlewares.IsAuth, messageHandler.SaveNewMessage)

	api.Patch("/users", middlewares.IsAuth, userHandler.ChangeUsername)
	api.Patch("/users/password", middlewares.IsAuth, userHandler.ChangePassword)

	// setup graceful shutdown
	// ctx, cancel := context.WithCancel(context.Background())
	// sig := make(chan os.Signal, 1)
	// signal.Notify(sig, os.Interrupt)
	// signal.Notify(sig, syscall.SIGTERM)

	// go func() {
	// 	<-sig
	// 	log.Println("Shutting down app")
	// 	time.Sleep(5 * time.Second)
	// 	cancel()
	// }()

	// go func() {
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
	// }()

	// <-ctx.Done()

	// shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	// defer cancel()

	// if err := app.ShutdownWithContext(shutdownCtx); err != nil {
	// 	log.Fatal("Error when shutting down app, error: " + err.Error())
	// }
	// log.Println("App is down")
}
