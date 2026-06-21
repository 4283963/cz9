package services

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github-repo-manager/models"
)

const maxConcurrent = 3

type RepoTask struct {
	ID       uint
	Private  bool
	Template string
}

type RepoService struct {
	workspaceDir string
	scriptPath   string
	taskQueue    chan RepoTask
	once         sync.Once
	dbMutex      sync.Mutex
}

func NewRepoService() *RepoService {
	wd, _ := os.Getwd()
	s := &RepoService{
		workspaceDir: filepath.Join(wd, "workspace"),
		scriptPath:   filepath.Join(wd, "scripts", "create_repo.sh"),
		taskQueue:    make(chan RepoTask, 1000),
	}
	s.startWorkers()
	return s
}

func (s *RepoService) startWorkers() {
	s.once.Do(func() {
		for i := 0; i < maxConcurrent; i++ {
			go s.worker(i + 1)
		}
		log.Printf("Worker pool started with %d workers", maxConcurrent)
	})
}

func (s *RepoService) worker(id int) {
	log.Printf("Worker %d started", id)
	for task := range s.taskQueue {
		log.Printf("Worker %d: processing repo task id=%d", id, task.ID)
		var repo models.Repo
		if err := models.DB.First(&repo, task.ID).Error; err != nil {
			log.Printf("Worker %d: repo %d not found, skipping: %v", id, task.ID, err)
			continue
		}
		s.CreateRepo(&repo, task.Private, task.Template)
		log.Printf("Worker %d: finished repo %s (status=%s)", id, repo.Name, repo.Status)
	}
}

func (s *RepoService) Submit(task RepoTask) {
	s.taskQueue <- task
}

func (s *RepoService) QueueLen() int {
	return len(s.taskQueue)
}

type errorPattern struct {
	keywords []string
	message  string
}

var errorPatterns = []errorPattern{
	{
		keywords: []string{"already exists", "name already taken", "name is already", "422 Unprocessable"},
		message:  "仓库名已被占用",
	},
	{
		keywords: []string{"timeout", "connection timed out", "i/o timeout", "deadline exceeded"},
		message:  "网络超时",
	},
	{
		keywords: []string{"connection refused", "no such host", "could not resolve host", "network is unreachable"},
		message:  "网络连接失败",
	},
	{
		keywords: []string{"unauthorized", "401", "authentication required", "not logged in", "gh auth login"},
		message:  "GitHub 未登录或 token 过期",
	},
	{
		keywords: []string{"forbidden", "403", "permission denied"},
		message:  "没有权限操作",
	},
	{
		keywords: []string{"not found", "404", "does not exist"},
		message:  "模板仓库不存在",
	},
	{
		keywords: []string{"repository access denied", "access denied"},
		message:  "仓库访问被拒绝",
	},
	{
		keywords: []string{"too many requests", "rate limit", "429"},
		message:  "GitHub API 限流，请稍后再试",
	},
	{
		keywords: []string{"remote origin already exists"},
		message:  "Git 远程地址已存在",
	},
	{
		keywords: []string{"nothing to commit", "working tree clean"},
		message:  "没有可提交的内容",
	},
	{
		keywords: []string{"failed to push", "push failed", "error: failed to push"},
		message:  "代码推送失败",
	},
}

func AnalyzeError(logText string, err error) string {
	fullText := logText
	if err != nil {
		fullText += " " + err.Error()
	}
	fullText = strings.ToLower(fullText)

	for _, p := range errorPatterns {
		for _, kw := range p.keywords {
			if strings.Contains(fullText, strings.ToLower(kw)) {
				return p.message
			}
		}
	}

	if err != nil {
		msg := err.Error()
		if len(msg) > 50 {
			msg = msg[:47] + "..."
		}
		return msg
	}
	return "未知错误，查看日志详情"
}

func (s *RepoService) setError(repo *models.Repo, msg string) {
	repo.ErrorMessage = msg
}

func (s *RepoService) failRepo(repo *models.Repo, logMsg string, err error) {
	reason := AnalyzeError(repo.Log+logMsg, err)
	s.setError(repo, reason)
	s.updateStatus(repo, models.StatusFailed)
	if logMsg != "" {
		if err != nil {
			s.appendLog(repo, fmt.Sprintf("%s: %v", logMsg, err))
		} else {
			s.appendLog(repo, logMsg)
		}
	}
}

func (s *RepoService) appendLog(repo *models.Repo, msg string) {
	s.dbMutex.Lock()
	defer s.dbMutex.Unlock()
	now := time.Now().Format("15:04:05")
	line := fmt.Sprintf("[%s] %s\n", now, msg)
	repo.Log += line
	models.DB.Save(repo)
}

func (s *RepoService) updateStatus(repo *models.Repo, status models.RepoStatus) {
	s.dbMutex.Lock()
	defer s.dbMutex.Unlock()
	repo.Status = status
	repo.UpdatedAt = time.Now()
	if status == models.StatusSuccess || status == models.StatusFailed {
		now := time.Now()
		repo.FinishedAt = &now
	}
	models.DB.Save(repo)
}

func (s *RepoService) runCmd(repo *models.Repo, name string, args ...string) error {
	s.appendLog(repo, fmt.Sprintf("执行: %s %v", name, args))
	cmd := exec.Command(name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	cmd.Dir = s.workspaceDir
	err := cmd.Run()
	if stdout.Len() > 0 {
		s.appendLog(repo, fmt.Sprintf("STDOUT: %s", stdout.String()))
	}
	if stderr.Len() > 0 {
		s.appendLog(repo, fmt.Sprintf("STDERR: %s", stderr.String()))
	}
	return err
}

func (s *RepoService) ensureWorkspace() error {
	return os.MkdirAll(s.workspaceDir, 0755)
}

func (s *RepoService) CreateRepo(repo *models.Repo, private bool, template string) {
	s.ensureWorkspace()
	repo.ErrorMessage = ""
	s.appendLog(repo, fmt.Sprintf("开始创建仓库: %s", repo.Name))

	s.updateStatus(repo, models.StatusLocalInit)
	repoPath := filepath.Join(s.workspaceDir, repo.Name)
	if err := os.RemoveAll(repoPath); err != nil {
		s.appendLog(repo, fmt.Sprintf("清理旧目录警告: %v", err))
	}
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		s.failRepo(repo, "创建目录失败", err)
		return
	}

	initCmds := [][]string{
		{"-C", repoPath, "init"},
		{"-C", repoPath, "checkout", "-b", "main"},
	}
	for _, args := range initCmds {
		if err := s.runCmd(repo, "git", args...); err != nil {
			s.failRepo(repo, "本地初始化失败", err)
			return
		}
	}

	readmePath := filepath.Join(repoPath, "README.md")
	readmeContent := fmt.Sprintf("# %s\n\n%s\n", repo.Name, repo.Description)
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		s.failRepo(repo, "创建 README 失败", err)
		return
	}

	gitignorePath := filepath.Join(repoPath, ".gitignore")
	gitignoreContent := ".DS_Store\nnode_modules/\ndist/\n.env\n"
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		s.failRepo(repo, "创建 .gitignore 失败", err)
		return
	}

	s.appendLog(repo, "本地初始化完成")

	s.updateStatus(repo, models.StatusRemoteCreate)
	ghArgs := []string{"repo", "create", repo.Name, "--description", repo.Description, "--source", repoPath}
	if private {
		ghArgs = append(ghArgs, "--private")
	} else {
		ghArgs = append(ghArgs, "--public")
	}
	if template != "" {
		ghArgs = append(ghArgs, "--template", template)
	}
	if err := s.runCmd(repo, "gh", ghArgs...); err != nil {
		s.failRepo(repo, "远程创建失败", err)
		return
	}
	s.appendLog(repo, "远程仓库创建完成")

	s.updateStatus(repo, models.StatusFirstPush)
	pushCmds := [][]string{
		{"-C", repoPath, "add", "."},
		{"-C", repoPath, "commit", "-m", "chore: initial commit"},
		{"-C", repoPath, "push", "-u", "origin", "main"},
	}
	for _, args := range pushCmds {
		if err := s.runCmd(repo, "git", args...); err != nil {
			s.failRepo(repo, "首次推送失败", err)
			return
		}
	}

	s.updateStatus(repo, models.StatusSuccess)
	s.appendLog(repo, "仓库创建成功！")
}
