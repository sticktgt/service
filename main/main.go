package main

import (
	"fmt"
	"gservice/configuration"
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

type GitCheckRequest struct {
	RepoName     string `json:"repoName" example:"https://github.com/Lockdain/metamart.git"`
	RepoUser     string `json:"repoUser" example:"elon@neof44.ru"`
	RepoPassword string `json:"repoPassword" example:"password or token"`
}

type GitCopyRequest struct {
	Repo1Password string `json:"repo1Password" example:"password or token"`
	Repo2Password string `json:"repo2Password" example:"password or token"`
	FileName      string `json:"fileName" example:"filename to copy"`
	SubFolderName string `json:"fsubFolderName" example:"subfolder for file"`
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
// @Param request body GitCheckRequest true "Параметры вызова"
// @Success 200 {object} SuccessResponse "Вызов прошел успешно"
// @Failure 400 {object} map[string]string
// @Failure 500 {object} ErrorResponse "Internal Server Error"
// @Router /gitcheck [post]
func gitcheckPostHandler(c *gin.Context) {
	ProcessID := uuid.New().String()

	var req GitCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	params := process.GitCheckFunctionParams{
		RepoName:     req.RepoName,
		RepoUser:     req.RepoUser,
		RepoPassword: req.RepoPassword,
		ProcessID:    ProcessID,
	}

	err := process.ProcessCheckGitFunction(params)
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

// @Summary Тестовый метод копирования файла из одного git-репозитория в другой
// @Description Метод для отладки работы с git-ом
// @Accept  json
// @Produce  json
// @Param request body GitCopyRequest true "Параметры вызова"
// @Success 200 {object} SuccessResponse "Вызов прошел успешно"
// @Failure 400 {object} map[string]string
// @Failure 500 {object} ErrorResponse "Internal Server Error"
// @Router /gitcopy [post]
func gitcopyPostHandler(c *gin.Context) {
	ProcessID := uuid.New().String()

	var req GitCopyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	params := process.GitCopyFunctionParams{

		Repo1Password:  req.Repo1Password,
		Repo2Password:  req.Repo2Password,
		FileName:       req.FileName,
		SubFolderName:  req.SubFolderName,
		Repo1UserName:  configuration.Config.GetString("repository1.username"),
		Repo2UserName:  configuration.Config.GetString("repository2.username"),
		Repo1Branch:    configuration.Config.GetString("repository1.branch"),
		Repo2Branch:    configuration.Config.GetString("repository2.branch"),
		Repo1Path:      configuration.Config.GetString("repository1.path"),
		Repo2Path:      configuration.Config.GetString("repository2.path"),
		Repo1LocalPath: configuration.Config.GetString("repository1.localpath"),
		Repo2LocaPath:  configuration.Config.GetString("repository2.localpath"),
		ProcessID:      ProcessID,
	}

	err := process.ProcessCopyGitFunction(params)
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

	configuration.Init()
	r.POST("/gitcheck", gitcheckPostHandler)
	r.POST("/gitcopy", gitcopyPostHandler)
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
