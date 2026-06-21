package models

import (
	"time"
)

type RepoStatus string

const (
	StatusPending      RepoStatus = "pending"
	StatusLocalInit    RepoStatus = "local_init"
	StatusRemoteCreate RepoStatus = "remote_create"
	StatusFirstPush    RepoStatus = "first_push"
	StatusSuccess      RepoStatus = "success"
	StatusFailed       RepoStatus = "failed"
)

type Repo struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Name        string     `gorm:"not null;index" json:"name"`
	Description string     `json:"description"`
	Status      RepoStatus `gorm:"default:pending;index" json:"status"`
	Log         string     `gorm:"type:text" json:"log"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
}

type BatchCreateRequest struct {
	Prefix      string `json:"prefix" binding:"required"`
	Start       int    `json:"start" binding:"required,gte=0"`
	End         int    `json:"end" binding:"required,gtefield=Start"`
	Description string `json:"description"`
	Private     bool   `json:"private"`
	Template    string `json:"template"`
}

type RetryRequest struct {
	Description string `json:"description"`
	Private     bool   `json:"private"`
	Template    string `json:"template"`
}

func (r Repo) IsFinished() bool {
	return r.Status == StatusSuccess || r.Status == StatusFailed
}

func StatusText(s RepoStatus) string {
	switch s {
	case StatusPending:
		return "待处理"
	case StatusLocalInit:
		return "本地初始化中"
	case StatusRemoteCreate:
		return "远程创建中"
	case StatusFirstPush:
		return "首次推送中"
	case StatusSuccess:
		return "成功"
	case StatusFailed:
		return "失败"
	default:
		return string(s)
	}
}
