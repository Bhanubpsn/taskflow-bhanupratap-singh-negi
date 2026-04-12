package routes

import (
	"github.com/Bhanubpsn/taskflow-bhanupratap-singh-negi/controllers"
	"github.com/gin-gonic/gin"
)

func UserRoutes(incomingRoutes gin.IRouter) {
	incomingRoutes.POST("/auth/register", controllers.Register())
	incomingRoutes.POST("/auth/login", controllers.Login())
}
