package routes

import (
	"github.com/Bhanubpsn/taskflow-bhanupratap-singh-negi/controllers"
	"github.com/gin-gonic/gin"
)

func ProjectRoutes(incomingRoutes gin.IRouter) {
	incomingRoutes.GET("/projects", controllers.GetProjects())
	incomingRoutes.POST("/projects", controllers.CreateProject())
	incomingRoutes.GET("/projects/:id", controllers.GetProjectByID())
	incomingRoutes.PATCH("/projects/:id", controllers.UpdateProject())
	incomingRoutes.DELETE("/projects/:id", controllers.DeleteProject())
}
