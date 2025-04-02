package main

import (
	"fmt"
	_ "gservice/main/docs"
	"gservice/process"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

var log = logrus.New()

type GitRequest struct {
	RepoName     string `json:"repoName" example:"https://github.com/Lockdain/metamart.git"`
	RepoUser     string `json:"repoUser" example:"elon@neof44.ru"`
	RepoPassword string `json:"repoPassword" example:"password or token"`
}

type ErrorResponse struct {
	Message   string `json:"message" example:"An error occurred"`
	Err       string `json:"err,omitempty" example:"detailed error message"`
	ProcessID string `json:"processID" example:"GUID"`
}

type SuccessResponse struct {
	Message   string `json:"message"   example:"Executed successfully"`
	ProcessID string `json:"processID" example:"GUID"`
}

// @Summary Тестовый метод соединения с git
// @Description Метод для отладки работы с git-ом
// @Accept  json
// @Produce  json
// @Param request body GitRequest true "Параметры вызова"
// @Success 200 {object} SuccessResponse "Вызов прошел успешно"
// @Failure 400 {object} map[string]string
// @Failure 500 {object} ErrorResponse "Internal Server Error"
// @Router /work [post]
func workPostHandler(c *gin.Context) {
	ProcessID := uuid.New().String()

	var req GitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	params := process.GitFunctionParams{
		RepoName:     req.RepoName,
		RepoUser:     req.RepoUser,
		RepoPassword: req.RepoPassword,
		ProcessID:    ProcessID,
	}

	err := process.ProcessGitFunction(params)
	if err != nil {
		log.WithField("processID", ProcessID).Errorf("An error occurred: %v", err)
		errorResponse := ErrorResponse{
			Message:   "An error occurred",
			Err:       err.Error(),
			ProcessID: ProcessID,
		}
		c.JSON(http.StatusInternalServerError, errorResponse)
		return
	}

	response := SuccessResponse{
		Message:   "Success",
		ProcessID: ProcessID,
	}
	c.JSON(http.StatusOK, response)
}

func main() {
	//	log.WithField("processID", ProcessID).Info("process started")
	//	defer log.WithField("processID", ProcessID).Info("process stopped")

	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(customLogger())
	r.Use(gin.Recovery())
	r.POST("/work", workPostHandler)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	log.Info("Application started")
	err := r.Run(":8080")
	if err != nil {
		fmt.Println("Error starting web server:", err)
	}
}

func customLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path != "/healthz" {
			log.Printf("%s %s\n", c.Request.Method, c.Request.URL.Path)
		}
		c.Next()
	}
}
