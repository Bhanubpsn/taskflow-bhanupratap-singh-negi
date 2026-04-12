package routes

import (
	"github.com/Bhanubpsn/taskflow-bhanupratap-singh-negi/controllers"
	"github.com/gin-gonic/gin"
)

func TaskRoutes(incomingRoutes gin.IRouter) {
	incomingRoutes.GET("/projects/:id/tasks", controllers.GetTasks())
	incomingRoutes.POST("/projects/:id/tasks", controllers.CreateTask())
	incomingRoutes.PATCH("/tasks/:id", controllers.UpdateTask())
	incomingRoutes.DELETE("/tasks/:id", controllers.DeleteTask())
}
