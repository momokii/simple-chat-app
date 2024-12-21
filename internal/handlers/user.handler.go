package handlers

import (
	"github.com/go-playground/validator/v10"
	"github.com/gofiber/fiber/v2"
	"github.com/momokii/simple-chat-app/internal/database"
	"github.com/momokii/simple-chat-app/internal/models"
	"github.com/momokii/simple-chat-app/internal/repository/user"
	"github.com/momokii/simple-chat-app/pkg/utils"
	"golang.org/x/crypto/bcrypt"
)

type UserHandler struct {
	userRepo user.UserRepo
}

func NewUserHandler(userRepo user.UserRepo) *UserHandler {
	return &UserHandler{
		userRepo: userRepo,
	}
}

func (h *UserHandler) ChangeUsername(c *fiber.Ctx) error {
	// var txError error
	user := c.Locals("user").(models.UserSession)

	userInput := new(models.UserChangeUsernameInput)
	if err := c.BodyParser(&userInput); err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Invalid request")
	}

	if err := utils.ValidateStruct(userInput); err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			switch err.Field() {
			case "Id":
				return utils.ResponseError(c, fiber.StatusBadRequest, "Invalid request, invalid id")
			case "Username":
				return utils.ResponseError(c, fiber.StatusBadRequest, "Username must be alphanumeric and between 3-25 characters")
			}
		}
	}

	// if new username is same with current username, just return success
	if user.Username == userInput.Username {
		return utils.ResponseMessage(c, fiber.StatusOK, "Success Change Username")
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to start transaction")
	}
	defer func() {
		database.CommitOrRollback(tx, c, err)
	}()

	// check if user is exist or not
	userCheck, err := h.userRepo.FindByID(tx, user.Id)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to check user")
	}

	if userCheck.Id == 0 {
		return utils.ResponseError(c, fiber.StatusBadRequest, "User not found")
	}

	// check if id inputted is same with user id
	if userCheck.Id != user.Id {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Invalid request")
	}

	// check if username already exist
	isUsernameExist, err := h.userRepo.FindByUsername(tx, userInput.Username)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to check username")
	}

	if isUsernameExist.Id > 0 {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Username already exist")
	}

	// if free to use, update username
	updateUser := models.User{
		Id:       user.Id,
		Username: userInput.Username,
		Password: userCheck.Password,
	}

	if err := h.userRepo.Update(tx, &updateUser); err != nil {
		// txError = err
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to change username")
	}

	return utils.ResponseMessage(c, fiber.StatusOK, "Success Change Username")
}

func (h *UserHandler) ChangePassword(c *fiber.Ctx) error {
	user := c.Locals("user").(models.UserSession)

	passInput := new(models.UserChangePasswordInput)
	if err := c.BodyParser(&passInput); err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Invalid request")
	}

	if err := utils.ValidateStruct(passInput); err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			switch err.Field() {
			case "Id":
				return utils.ResponseError(c, fiber.StatusBadRequest, "Invalid request")
			case "Password":
				return utils.ResponseError(c, fiber.StatusBadRequest, "Password must be filled")
			case "NewPassword":
				return utils.ResponseError(c, fiber.StatusBadRequest, "New Password must be alphanumeric and between 6-50 characters, contains number and uppercase")
			}
		}
	}

	tx, err := database.DB.Begin()
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to start transaction")
	}
	defer func() {
		database.CommitOrRollback(tx, c, err)
	}()

	// check if id inputted is same with user id
	if user.Id != passInput.Id {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Invalid request")
	}

	// check if user exist
	userCheck, err := h.userRepo.FindByID(tx, user.Id)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to check user")
	}

	if userCheck.Id == 0 {
		return utils.ResponseError(c, fiber.StatusBadRequest, "User not found")
	}

	// check if password is correct
	if err := bcrypt.CompareHashAndPassword([]byte(userCheck.Password), []byte(passInput.Password)); err != nil {
		return utils.ResponseError(c, fiber.StatusBadRequest, "Invalid current password value")
	}

	// hash password
	hashedPass, err := bcrypt.GenerateFromPassword([]byte(passInput.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to hash new password")
	}

	// update password
	if err := h.userRepo.UpdatePassword(tx, &models.User{
		Id:       user.Id,
		Username: userCheck.Username,
		Password: string(hashedPass),
	}); err != nil {
		return utils.ResponseError(c, fiber.StatusInternalServerError, "Failed to change password")
	}

	return utils.ResponseMessage(c, fiber.StatusOK, "Success Change Password")
}
