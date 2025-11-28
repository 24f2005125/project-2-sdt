package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	err := InitDB()
	if err != nil {
		log.Fatalf("Error initializing database: %v", err)
	}

	err = godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	/* Frontend */
	r.Static("/assets", "./public/assets")
	r.GET("/quiz-interface", func(c *gin.Context) {
		c.File("./public/index.html")
	})
	r.NoRoute(func(c *gin.Context) {
		c.File("./public/index.html")
	})

	ingestGroup := r.Group("/ingest")
	ingestGroup.Use(EnsureAuthenticated())
	ingestGroup.POST("", func(c *gin.Context) {
		var req TaskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(400, APIResponse[any]{
				Status:  "error",
				Message: "invalid_request_body",
				Error:   err.Error(),
				Data:    nil,
			})
			return
		}

		err := Ingest(req)
		if err != nil {
			c.JSON(500, APIResponse[any]{
				Status:  "error",
				Message: "ingest_failed",
				Error:   err.Error(),
				Data:    nil,
			})
			return
		}

		// Start the quiz session
		err = StartQuizSession(req)
		if err != nil {
			c.JSON(500, APIResponse[any]{
				Status:  "error",
				Message: "quiz_session_failed",
				Error:   err.Error(),
				Data:    nil,
			})
			return
		}

		c.JSON(200, APIResponse[any]{
			Status:  "success",
			Message: "quiz_session_started",
			Error:   "",
			Data:    nil,
		})
	})

	ingestGroup.GET("", func(c *gin.Context) {
		ingests, err := ListIngests()
		if err != nil {
			c.JSON(500, APIResponse[any]{
				Status:  "error",
				Message: "list_ingests_failed",
				Error:   err.Error(),
				Data:    nil,
			})
			return
		}

		c.JSON(200, APIResponse[[]Ingests]{
			Status:  "success",
			Message: "ingests_listed",
			Error:   "",
			Data:    ingests,
		})
	})

	ingestGroup.GET("/notification-accept", func(c *gin.Context) {
		idParam := c.Query("id")
		password := c.Query("password")

		var id uint
		_, err := fmt.Sscanf(idParam, "%d", &id)
		if err != nil {
			c.JSON(400, APIResponse[any]{
				Status:  "error",
				Message: "invalid_id_parameter",
				Error:   err.Error(),
				Data:    nil,
			})
			return
		}

		err = AcceptIngest(id, password)
		if err != nil {
			c.JSON(500, APIResponse[any]{
				Status:  "error",
				Message: "accept_ingest_failed",
				Error:   err.Error(),
				Data:    nil,
			})
			return
		}

		c.JSON(200, APIResponse[any]{
			Status:  "success",
			Message: "ingest_accepted",
			Error:   "",
			Data:    nil,
		})
	})

	// Quiz management endpoints
	quizGroup := r.Group("/quiz")
	quizGroup.Use(EnsureAuthenticated())

	// Get all quiz sessions
	quizGroup.GET("/sessions", func(c *gin.Context) {
		sessions, err := GetQuizSessions()
		if err != nil {
			c.JSON(500, APIResponse[any]{
				Status:  "error",
				Message: "failed_to_get_sessions",
				Error:   err.Error(),
				Data:    nil,
			})
			return
		}

		c.JSON(200, APIResponse[[]QuizSession]{
			Status:  "success",
			Message: "sessions_retrieved",
			Error:   "",
			Data:    sessions,
		})
	})

	// Get attempts for a specific session
	quizGroup.GET("/sessions/:id/attempts", func(c *gin.Context) {
		var sessionID uint
		_, err := fmt.Sscanf(c.Param("id"), "%d", &sessionID)
		if err != nil {
			c.JSON(400, APIResponse[any]{
				Status:  "error",
				Message: "invalid_session_id",
				Error:   err.Error(),
				Data:    nil,
			})
			return
		}

		attempts, err := GetQuizAttempts(sessionID)
		if err != nil {
			c.JSON(500, APIResponse[any]{
				Status:  "error",
				Message: "failed_to_get_attempts",
				Error:   err.Error(),
				Data:    nil,
			})
			return
		}

		c.JSON(200, APIResponse[[]QuizAttempt]{
			Status:  "success",
			Message: "attempts_retrieved",
			Error:   "",
			Data:    attempts,
		})
	})

	// Get pending attempts that need answers
	quizGroup.GET("/pending", func(c *gin.Context) {
		attempts, err := GetPendingAttempts()
		if err != nil {
			c.JSON(500, APIResponse[any]{
				Status:  "error",
				Message: "failed_to_get_pending_attempts",
				Error:   err.Error(),
				Data:    nil,
			})
			return
		}

		c.JSON(200, APIResponse[[]QuizAttempt]{
			Status:  "success",
			Message: "pending_attempts_retrieved",
			Error:   "",
			Data:    attempts,
		})
	})

	// Submit answer for a specific session
	quizGroup.POST("/sessions/:id/answer", func(c *gin.Context) {
		var sessionID uint
		_, err := fmt.Sscanf(c.Param("id"), "%d", &sessionID)
		if err != nil {
			c.JSON(400, APIResponse[any]{
				Status:  "error",
				Message: "invalid_session_id",
				Error:   err.Error(),
				Data:    nil,
			})
			return
		}

		var answerReq struct {
			Answer    interface{} `json:"answer"`
			SubmitURL string      `json:"submitUrl,omitempty"`
			Password  string      `json:"password"`
		}

		if err := c.ShouldBindJSON(&answerReq); err != nil {
			c.JSON(400, APIResponse[any]{
				Status:  "error",
				Message: "invalid_answer_format",
				Error:   err.Error(),
				Data:    nil,
			})
			return
		}

		var response *QuizResponse
		if answerReq.SubmitURL == "" {
			c.JSON(400, APIResponse[any]{
				Status:  "error",
				Message: "submit_url_required",
				Error:   "submitUrl is required",
				Data:    nil,
			})
			return
		}

		// Validate password
		expectedPassword := os.Getenv("QUIZ_ATTEMPT_PASSWORD")
		if answerReq.Password != expectedPassword {
			c.JSON(403, APIResponse[any]{
				Status:  "error",
				Message: "invalid_password",
				Error:   "Invalid quiz attempt password",
				Data:    nil,
			})
			return
		}

		response, err = SubmitManualAnswer(sessionID, answerReq.Answer, answerReq.SubmitURL)

		if err != nil {
			c.JSON(500, APIResponse[any]{
				Status:  "error",
				Message: "failed_to_submit_answer",
				Error:   err.Error(),
				Data:    nil,
			})
			return
		}

		c.JSON(200, APIResponse[*QuizResponse]{
			Status:  "success",
			Message: "answer_submitted",
			Error:   "",
			Data:    response,
		})
	})

	go func() {
		for {
			NotifyJob()
			time.Sleep(1 * time.Second)
		}
	}()

	r.Run(":8080")
}
