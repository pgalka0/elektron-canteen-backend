package routers

import (
	"elektron-canteen/api/controllers"
	"elektron-canteen/api/data/menu"
	"elektron-canteen/api/data/user"
	"elektron-canteen/api/mid"
	"net/http"

	"github.com/gin-gonic/gin"
)

type MenuRouter struct {
	router     *gin.Engine
	controller controllers.MenuController
}

func NewMenuRouter(r *gin.Engine, c controllers.MenuController) *MenuRouter {
	return &MenuRouter{
		router:     r,
		controller: c,
	}
}

func (r *MenuRouter) Initialize() {
	mr := r.router.Group("/menu")

	mr.GET("/:day", r.getMenuByDay)
	mr.GET("/range/:start_day/:end_day", r.getRangeMenus)

	mr.POST("", mid.Role(user.ADMIN_ROLE), r.createMenu)
	mr.PATCH("", mid.Role(user.ADMIN_ROLE), r.updateMenu)
	mr.DELETE("/:day", mid.Role(user.ADMIN_ROLE), r.deleteMenu)
}

func (r *MenuRouter) getRangeMenus(c *gin.Context) {
	startDay := c.Param("start_day")
	endDay := c.Param("end_day")
	menus, err := r.controller.GetRangeMenus(startDay, endDay)
	if err != nil {
		responseWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"start_day": startDay, "end_day": endDay, "menus": menus})
}

func (r *MenuRouter) getMenuByDay(c *gin.Context) {
	day := c.Param("day")

	m, err := r.controller.GetByDay(day)
	if err != nil {
		responseWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": m})
}

func (r *MenuRouter) deleteMenu(c *gin.Context) {
	day := c.Param("day")

	if err := r.controller.Delete(day); err != nil {
		responseWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "deleted successfully"})
}

func (r *MenuRouter) createMenu(c *gin.Context) {
	var mr menu.AddRequest

	if err := c.BindJSON(&mr); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.controller.Add(mr); err != nil {
		responseWithError(c, err)
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Menu created successfully"})
}

func (r *MenuRouter) updateMenu(c *gin.Context) {
	var m menu.Menu

	if err := c.BindJSON(&m); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := r.controller.Update(m); err != nil {
		responseWithError(c, err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Menu updated successfully"})
}
