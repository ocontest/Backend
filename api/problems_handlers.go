package api

import (
	"github.com/gin-gonic/gin"
	"net/http"
	"ocontest/pkg"
	"ocontest/pkg/structs"
	"strconv"
)

func (h *handlers) CreateProblem(c *gin.Context) {
	logger := pkg.Log.WithField("handler", "createProblem")

	var reqData structs.RequestCreateProblem
	if err := c.ShouldBindJSON(&reqData); err != nil {
		logger.Warn("Failed to read request body", err)
		c.JSON(http.StatusBadRequest, gin.H{
			"error": pkg.ErrBadRequest.Error(),
		})
		return
	}

	resp, status := h.problemsHandler.CreateProblem(c, reqData)
	c.JSON(status, resp)
}

func (h *handlers) GetProblem(c *gin.Context) {
	//logger := pkg.Log.WithField("handler", "getProblem")

	problemID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "invalid id, id should be an integer",
		})
		return
	}
	resp, status := h.problemsHandler.GetProblem(c, problemID)
	if status == http.StatusOK {
		c.JSON(status, resp)
	} else {
		c.Status(status)
	}
}

func (h *handlers) ListProblems(c *gin.Context) {
	h.problemsHandler.ListProblem(c, structs.RequestListProblems{})
}