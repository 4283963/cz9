package services

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
	s.appendLog(repo, fmt.Sprintf("开始创建仓库: %s", repo.Name))

	s.updateStatus(repo, models.StatusLocalInit)
	repoPath := filepath.Join(s.workspaceDir, repo.Name)
	if err := os.RemoveAll(repoPath); err != nil {
		s.appendLog(repo, fmt.Sprintf("清理旧目录警告: %v", err))
	}
	if err := os.MkdirAll(repoPath, 0755); err != nil {
		s.updateStatus(repo, models.StatusFailed)
		s.appendLog(repo, fmt.Sprintf("创建目录失败: %v", err))
		return
	}

	initCmds := [][]string{
		{"-C", repoPath, "init"},
		{"-C", repoPath, "checkout", "-b", "main"},
	}
	for _, args := range initCmds {
		if err := s.runCmd(repo, "git", args...); err != nil {
			s.updateStatus(repo, models.StatusFailed)
			s.appendLog(repo, fmt.Sprintf("本地初始化失败: %v", err))
			return
		}
	}

	readmePath := filepath.Join(repoPath, "README.md")
	readmeContent := fmt.Sprintf("# %s\n\n%s\n", repo.Name, repo.Description)
	if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
		s.updateStatus(repo, models.StatusFailed)
		s.appendLog(repo, fmt.Sprintf("创建 README 失败: %v", err))
		return
	}

	gitignorePath := filepath.Join(repoPath, ".gitignore")
	gitignoreContent := ".DS_Store\nnode_modules/\ndist/\n.env\n"
	if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
		s.updateStatus(repo, models.StatusFailed)
		s.appendLog(repo, fmt.Sprintf("创建 .gitignore 失败: %v", err))
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
		s.updateStatus(repo, models.StatusFailed)
		s.appendLog(repo, fmt.Sprintf("远程创建失败: %v", err))
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
			s.updateStatus(repo, models.StatusFailed)
			s.appendLog(repo, fmt.Sprintf("首次推送失败: %v", err))
			return
		}
	}

	s.updateStatus(repo, models.StatusSuccess)
	s.appendLog(repo, "仓库创建成功！")
}
