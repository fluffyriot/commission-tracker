package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) RootHandler(c *gin.Context) {

	if h.DBInitErr != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": h.DBInitErr.Error(),
		})
		return
	}

	if h.KeyB64Err1 != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": h.KeyB64Err1.Error(),
		})
		return
	}

	if h.KeyB64Err2 != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": h.KeyB64Err2.Error(),
		})
		return
	}

	if h.InstVerErr != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": h.InstVerErr.Error(),
		})
		return
	}

	ctx := c.Request.Context()

	users, err := h.DB.GetAllUsers(ctx)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": err.Error(),
		})
		return
	}

	if len(users) == 0 {
		c.HTML(http.StatusOK, "user-setup.html", nil)
		return
	}

	user := users[0]

	sources, err := h.DB.GetUserSources(ctx, user.ID)
	if err != nil {
		c.HTML(http.StatusInternalServerError, "error.html", gin.H{
			"error": err.Error(),
		})
		return
	}
	c.HTML(http.StatusOK, "index.html", gin.H{
		"username": user.Username,
		"user_id":  user.ID,
		"sources":  sources,
	})
}
