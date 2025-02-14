package handlers

import (
	"errors"
	"os"

	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/momokii/simple-chat-app/internal/database"
	"github.com/momokii/simple-chat-app/internal/middlewares"
	"github.com/momokii/simple-chat-app/internal/repository/session"
	"github.com/momokii/simple-chat-app/internal/repository/user"
	"github.com/momokii/simple-chat-app/pkg/utils"
	"golang.org/x/crypto/bcrypt"
)

type Auth struct {
	Username string `json:"username" validate:"required,min=5,max=25,alphanum"`
	Password string `json:"password" validate:"required,min=6,max=50,containsany=1234567890,containsany=QWERTYUIOPASDFGHJKLZXCVBNM"`
}

type AuthHandler struct {
	userRepo    user.UserRepo
	sessionRepo session.SessionRepo
}

func NewAuthHandler(userRepo user.UserRepo, sessionRepo session.SessionRepo) *AuthHandler {
	return &AuthHandler{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

// SSO func
func (h *AuthHandler) SSOAuthLogin(c *fiber.Ctx) error {
	// get jwt token from request
	token := c.Query("token")
	if token == "" {
		return errors.New("token is required")
	}

	// validate token
	token_data, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		return []byte(os.Getenv("JWT_SECRET")), nil
	})
	if err != nil {
		return errors.New("invalid token")
	}

	session_id := token_data.Claims.(jwt.MapClaims)["session_id"].(string)
	user_id := int(token_data.Claims.(jwt.MapClaims)["user_id"].(float64))

	// check session on db if valid or not
	tx, err := database.DB.Begin()
	if err != nil {
		return errors.New("Internal server error on setup db tx: " + err.Error())
	}
	defer func() {
		database.CommitOrRollback(tx, c, err)
	}()

	session_check, err := h.sessionRepo.FindSession(tx, session_id, user_id)
	if err != nil {
		return errors.New("Internal server error on find session: " + err.Error())
	}

	if session_check.Id == 0 && session_check.SessionId == "" && session_check.UserId == 0 {

		return errors.New("invalid, session not found")
	}

	// save session to fiber session data
	if err := middlewares.CreateSession(c, "id", user_id); err != nil {
		return errors.New("Internal server error on create session: " + err.Error())
	}

	if err := middlewares.CreateSession(c, "session_id", session_id); err != nil {
		return errors.New("Internal server error on create session: " + err.Error())
	}

	return c.Redirect("/")
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	// delete session here
	middlewares.DeleteSession(c)

	return utils.ResponseMessage(c, fiber.StatusOK, "Logout success")
}

// function below can use it when not using SSO
func (h *AuthHandler) LoginView(c *fiber.Ctx) error {
	return c.Render("login", fiber.Map{
		"Title": "Login - Chat Nge-Chat",
	})
}

func (h *AuthHandler) SignUpView(c *fiber.Ctx) error {
	return c.Render("signup", fiber.Map{
		"Title": "SignUp - Chat Nge-Chat",
	})
}

func (h *AuthHandler) SignUp(c *fiber.Ctx) error {
	auth := new(Auth)
	if err := c.BodyParser(auth); err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Invalid request")
	}

	if err := utils.ValidateStruct(auth); err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			switch err.Field() {
			case "Username":
				return utils.ResponseError(c, fiber.StatusBadRequest, "Username must be alphanumeric and between 3-25 characters")
			case "Password":
				return utils.ResponseError(c, fiber.StatusBadRequest, "Password must be alphanumeric and between 6-50 characters with minimum 1 number and 1 uppercase letter")
			}
		}
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, err.Error())
	}
	defer func() {
		database.CommitOrRollback(tx, c, err)
	}()

	// check if username already exist
	user, err := h.userRepo.FindByUsername(tx, auth.Username)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, err.Error())
	}

	if user.Id != 0 {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Username already exist")
	}

	// hashing password
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(auth.Password), 16)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, err.Error())
	}

	// add user to database
	user.Password = string(hashedPass)
	user.Username = auth.Username

	if err := h.userRepo.Create(tx, user); err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, err.Error())
	}

	return utils.ResponseMessage(c, fiber.StatusOK, "Signup success")
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	auth := new(Auth)
	if err := c.BodyParser(auth); err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Invalid request")
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, err.Error())
	}
	defer func() {
		database.CommitOrRollback(tx, c, err)
	}()

	userLog, err := h.userRepo.FindByUsername(tx, auth.Username)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, err.Error())
	}

	// check if user exist
	if userLog.Id == 0 {
		return utils.ResponseError(c, fiber.StatusUnauthorized, "Invalid username or password")
	}

	// password checking
	if err := bcrypt.CompareHashAndPassword([]byte(userLog.Password), []byte(auth.Password)); err != nil {
		return utils.ResponseError(c, fiber.StatusUnauthorized, "Invalid username or password")
	}

	// create token session here
	// sign := jwt.New(jwt.SigningMethodHS256)
	// claims := sign.Claims.(jwt.MapClaims)
	// claims["id"] = userLog.Id
	// claims["exp"] = time.Now().Add(time.Hour * 8).Unix()

	// _, err = sign.SignedString([]byte("secret"))
	// if err != nil {
	// 	return utils.ResponseError(c, fiber.StatusInternalServerError, err.Error())
	// }

	// create session here
	middlewares.CreateSession(c, "id", userLog.Id)

	// save token to cookie for browser use
	// c.Cookie(&fiber.Cookie{
	// 	Name:  "id",
	// 	Value: strconv.Itoa(userLog.Id),
	// })

	return utils.ResponseMessage(c, fiber.StatusOK, "Login success")
}
