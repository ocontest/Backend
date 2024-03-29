package api

import (
	"net/http"

	"github.com/ocontest/backend/internal/oc/auth"
	"github.com/ocontest/backend/internal/oc/contests"
	"github.com/ocontest/backend/internal/oc/problems"
	"github.com/ocontest/backend/internal/oc/submissions"

	"github.com/gin-gonic/gin"
)

type handlers struct {
	authHandler        auth.AuthHandler
	problemsHandler    problems.ProblemsHandler
	contestsHandler    contests.ContestsHandler
	submissionsHandler submissions.Handler
}

func AddRoutes(r *gin.Engine, authHandler auth.AuthHandler, problemHandler problems.ProblemsHandler, submissionsHandler submissions.Handler,
	contestsHandler contests.ContestsHandler) {
	h := handlers{
		authHandler:        authHandler,
		problemsHandler:    problemHandler,
		submissionsHandler: submissionsHandler,
		contestsHandler:    contestsHandler,
	}

	r.Use(h.corsHandler)
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong2",
		})
	})

	r.OPTIONS("/*cors", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	v1 := r.Group("/v1")
	{
		authGroup := v1.Group("/auth")
		{
			authGroup.POST("/register", h.registerUser)
			authGroup.POST("/verify", h.verifyEmail)
			authGroup.POST("/otp", h.GetOTPForLogin)
			authGroup.POST("/login", h.loginUser)
			authGroup.POST("/renew_token", h.AuthMiddleware(), h.renewToken)
			authGroup.POST("/edit_user", h.AuthMiddleware(), h.editUser)
			authGroup.GET("", h.AuthMiddleware(), h.getOwnUser)
			authGroup.GET("/:id", h.AuthMiddleware(), h.getUser)
		}
		problemGroup := v1.Group("/problems", h.AuthMiddleware())
		{
			problemGroup.POST("", h.CreateProblem)
			problemGroup.GET("/:id", h.GetProblem)
			problemGroup.GET("", h.ListProblems)
			problemGroup.PUT("/:id", h.UpdateProblem)
			problemGroup.DELETE("/:id", h.DeleteProblem)
			problemGroup.POST("/:id/testcase", h.AddTestCase)
			problemGroup.GET("/:id/testcase", h.GetTestCase)
			problemGroup.GET("/:id/submissions", h.ListSubmissions)
		}
		contestGroup := v1.Group("/contests", h.AuthMiddleware())
		{
			contestGroup.POST("", h.CreateContest)
			contestGroup.GET("", h.ListContests)
			contestGroup.GET("/:id", h.GetContest)
			contestGroup.GET("/:id/scoreboard", h.GetContestScoreboard)
			contestGroup.PUT("/:id", h.UpdateContest)
			contestGroup.DELETE("/:contest_id", h.DeleteContest)
			contestGroup.POST("/:contest_id/problems/:problem_id", h.AddProblemContest)
			contestGroup.DELETE("/:contest_id/problems/:problem_id", h.RemoveProblemContest)
			contestGroup.PATCH("/:contest_id", h.PatchContest)
			contestGroup.GET("/:id/submissions", h.ListContestSubmissions)
			contestGroup.GET("/:id/problems/:problem_id/submissions", h.ListContestProblemSubmissions)
		}

		submissionGroup := v1.Group("/submissions", h.AuthMiddleware())
		{
			submissionGroup.GET("/:id", h.GetSubmission)
			submissionGroup.POST("/", h.Submit)
			submissionGroup.GET("/:id/results", h.GetSubmissionResult)
		}
		v1.POST("/problems/:id/submit", h.AuthMiddleware(), h.Submit)
	}
}
