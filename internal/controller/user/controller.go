package controller

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"timmygram/internal/model"
	userservice "timmygram/internal/service/user"
)

type UserController struct {
	userService *userservice.UserService
}

func NewUserController(userService *userservice.UserService) *UserController {
	return &UserController{userService: userService}
}

func (c *UserController) ShowUsersPage(ctx *gin.Context) {
	users, err := c.userService.ListUsers()
	if err != nil {
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	username := ctx.GetString("username")
	ctx.HTML(http.StatusOK, "users.html", gin.H{
		"title":    "Manage Users — TimmyGram",
		"page":     "users",
		"username": username,
		"is_owner": true,
		"users":    users,
	})
}

func (c *UserController) HandleCreateUser(ctx *gin.Context) {
	newUsername := strings.TrimSpace(ctx.PostForm("username"))
	password := ctx.PostForm("password")

	sessionUsername := ctx.GetString("username")
	renderErr := func(msg string, users []*model.User) {
		ctx.HTML(http.StatusUnprocessableEntity, "users.html", gin.H{
			"title":    "Manage Users — TimmyGram",
			"page":     "users",
			"username": sessionUsername,
			"is_owner": true,
			"users":    users,
			"error":    msg,
		})
	}

	users, err := c.userService.ListUsers()
	if err != nil {
		ctx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if newUsername == "" || password == "" {
		renderErr("Username and password are required.", users)
		return
	}
	if len(password) < 6 {
		renderErr("Password must be at least 6 characters.", users)
		return
	}

	if _, err := c.userService.CreateUser(newUsername, password); err != nil {
		if errors.Is(err, userservice.ErrUsernameTaken) {
			renderErr("That username is already taken.", users)
		} else {
			renderErr("Failed to create user. Please try again.", users)
		}
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/users")
}

func (c *UserController) HandleDeleteUser(ctx *gin.Context) {
	id, err := strconv.Atoi(ctx.Param("id"))
	if err != nil {
		ctx.AbortWithStatus(http.StatusBadRequest)
		return
	}

	if err := c.userService.DeleteUser(id); err != nil {
		if errors.Is(err, userservice.ErrCannotDelete) {
			ctx.AbortWithStatus(http.StatusForbidden)
		} else {
			ctx.AbortWithStatus(http.StatusInternalServerError)
		}
		return
	}

	ctx.Redirect(http.StatusSeeOther, "/users")
}
