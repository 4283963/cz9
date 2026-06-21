package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github-repo-manager/models"
	"github-repo-manager/services"

	"github.com/gin-gonic/gin"
)

type RepoHandler struct {
	service *services.RepoService
}

func NewRepoHandler() *RepoHandler {
	return &RepoHandler{
		service: services.NewRepoService(),
	}
}

func (h *RepoHandler) ListRepos(c *gin.Context) {
	var repos []models.Repo
	status := c.Query("status")
	query := models.DB.Order("id DESC")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if err := query.Find(&repos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": repos})
}

func (h *RepoHandler) GetRepo(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var repo models.Repo
	if err := models.DB.First(&repo, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repo not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": repo})
}

func (h *RepoHandler) BatchCreate(c *gin.Context) {
	var req models.BatchCreateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var repos []models.Repo
	for i := req.Start; i <= req.End; i++ {
		name := fmt.Sprintf("%s%d", req.Prefix, i)
		repo := models.Repo{
			Name:        name,
			Description: req.Description,
			Status:      models.StatusPending,
		}
		if err := models.DB.Create(&repo).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		repos = append(repos, repo)
	}

	for _, repo := range repos {
		h.service.Submit(services.RepoTask{
			ID:       repo.ID,
			Private:  req.Private,
			Template: req.Template,
		})
	}

	c.JSON(http.StatusAccepted, gin.H{
		"message": fmt.Sprintf("已提交 %d 个仓库创建任务，后台正在后台排队处理中（并发 %d）", len(repos), 3),
		"queue":   h.service.QueueLen(),
		"data":    repos,
	})
}

func (h *RepoHandler) RetryRepo(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var repo models.Repo
	if err := models.DB.First(&repo, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repo not found"})
		return
	}

	var req models.RetryRequest
	if c.ShouldBindJSON(&req) == nil {
		if req.Description != "" {
			repo.Description = req.Description
		}
	}

	repo.Status = models.StatusPending
	repo.Log = ""
	repo.FinishedAt = nil
	models.DB.Save(&repo)

	h.service.Submit(services.RepoTask{
		ID:       repo.ID,
		Private:  req.Private,
		Template: req.Template,
	})

	c.JSON(http.StatusAccepted, gin.H{
		"message": fmt.Sprintf("已重新提交仓库 %s 的创建任务，排队中", repo.Name),
		"queue":   h.service.QueueLen(),
		"data":    repo,
	})
}

func (h *RepoHandler) DeleteRepo(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	if err := models.DB.Delete(&models.Repo{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已删除"})
}

func (h *RepoHandler) Stats(c *gin.Context) {
	var total, pending, running, success, failed int64
	models.DB.Model(&models.Repo{}).Count(&total)
	models.DB.Model(&models.Repo{}).Where("status = ?", models.StatusPending).Count(&pending)
	models.DB.Model(&models.Repo{}).Where("status IN ?", []models.RepoStatus{
		models.StatusLocalInit, models.StatusRemoteCreate, models.StatusFirstPush,
	}).Count(&running)
	models.DB.Model(&models.Repo{}).Where("status = ?", models.StatusSuccess).Count(&success)
	models.DB.Model(&models.Repo{}).Where("status = ?", models.StatusFailed).Count(&failed)

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"total":   total,
			"pending": pending,
			"running": running,
			"success": success,
			"failed":  failed,
		},
	})
}

func (h *RepoHandler) DeleteAll(c *gin.Context) {
	if err := models.DB.Exec("DELETE FROM repos").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "已清空所有记录"})
}
