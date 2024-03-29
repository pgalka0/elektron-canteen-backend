package routers

import (
	"elektron-canteen/api/controllers"
	"elektron-canteen/api/mid"
	jwtutil "elektron-canteen/foundation/jwt"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"net/http"
)

type UserRouter struct {
	router     *gin.Engine
	controller controllers.UserController
}

func NewUserRouter(r *gin.Engine, c controllers.UserController) *UserRouter {
	return &UserRouter{
		router:     r,
		controller: c,
	}
}

func (r *UserRouter) Initialize() {
	r.router.GET("/user", mid.Auth(), r.getUserData)
}

func (r *UserRouter) getUserData(c *gin.Context) {
	cookie, err := c.Cookie("token")
	if err != nil {
		responseWithError(c, err)
	}

	claims, err := jwtutil.DecodeIntoClaims(cookie)
	if err != nil {
		responseWithError(c, err)
	}

	userID, err := primitive.ObjectIDFromHex(claims["user"].(string))
	if err != nil {
		panic(err)
	}

	user, err := r.controller.GetByID(userID)
	if err != nil {
		responseWithError(c, err)
		return
	}

	user.Password = ""

	c.JSON(http.StatusOK, gin.H{
		"user": user,
	})
}
