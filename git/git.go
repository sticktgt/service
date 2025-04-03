package git

import (
	"errors"
	"fmt"
	"gservice/configuration"
	"os"
	pth "path"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/google/uuid"
	gitignore "github.com/sabhiram/go-gitignore"
	"github.com/sirupsen/logrus"
)

// GitAuth - структура для хранения данных аутентификации
type GitAuth struct {
	Username string // Для HTTP ("token" для GitHub)
	Password string // Токен или пароль
	SSHKey   string // Путь к приватному ключу (если используем SSH)
	UseSSH   bool   // Флаг использования SSH
}

type GitAuthor struct {
	AuthorName  string
	AuthorEmail string
}

type GitCache struct {
	repoURL      string
	branch       string
	localPath    string
	repo         *git.Repository
	lastCommitID plumbing.Hash
	mu           sync.RWMutex
	initOnce     sync.Once
	initMu       sync.Mutex
	initErr      error
}

var log = logrus.New()

func loadGitAuthorConfig() (*GitAuthor, error) {
	auth := &GitAuthor{
		AuthorName:  configuration.Config.GetString("git.authorName"),
		AuthorEmail: configuration.Config.GetString("git.authorEmail"),
	}

	if auth.AuthorName == "" || auth.AuthorEmail == "" {
		return nil, fmt.Errorf("git author_name and author_email must be set in application.conf")
	}

	return auth, nil
}

// CloneRepositoryWithLocalFolderWipe Используются в сценарии клонирования репозиториев с проектными чартами
func CloneRepositoryWithLocalFolderWipe(auth GitAuth, repoURL, branch, localPath, processID string) (*git.Repository, error) {
	repository, rmErr := removeExistingFolder(localPath, processID)
	if rmErr != nil {
		return repository, rmErr
	}

	return PlainCloneRepository(auth, repoURL, branch, localPath, processID)
}

func PlainCloneRepository(auth GitAuth, repoURL string, branch string, localPath string, processID string) (*git.Repository, error) {
	log.WithField("processID", processID).Infof("Cloning repository to %s...", localPath)

	repo, err := git.PlainClone(localPath, false, &git.CloneOptions{
		URL:           repoURL,
		Progress:      os.Stdout,
		Auth:          getAuthMethod(auth),
		ReferenceName: plumbing.NewBranchReferenceName(branch),
		SingleBranch:  true,
		Depth:         1,
	})
	if err != nil {
		log.WithField("processID", processID).Errorf("failed to clone repository: %v", err)
		return nil, err
	}

	return repo, nil
}

func removeExistingFolder(localPath string, processID string) (*git.Repository, error) {
	log.WithField("processID", processID).Infof("Cleaning folder: %s before clone", localPath)

	if _, err := os.Stat(localPath); os.IsNotExist(err) {
		log.WithField("processID", processID).Infof("Repo folder does not exist, no need to remove")
		return nil, nil
	}

	// Удаляем папку, если она уже существует, чтобы избежать конфликтов
	if err := os.RemoveAll(localPath); err != nil {
		log.WithField("processID", processID).Errorf("failed to remove existing directory: %v", err)
		return nil, err
	}
	return nil, nil
}

func CommitAndPush(auth GitAuth, commitMessage string, repo *git.Repository, localPath string, branch, processID string, projectFolder string) error {
	gitAuthor, err := loadGitAuthorConfig()
	if err != nil {
		log.WithField("processID", processID).Errorf("error loading git author: %v", err)
		return err
	}

	worktree, err := repo.Worktree()
	if err != nil {
		log.WithField("processID", processID).Errorf("failed to get worktree: %v", err)
		return err
	}

	ignoreMatcher, _ := loadGitIgnore(localPath)

	err = filepath.Walk(localPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, _ := filepath.Rel(localPath, path)
		if ignoreMatcher != nil && ignoreMatcher.MatchesPath(relPath) {
			return nil
		}

		filePath := pth.Join(projectFolder, relPath)
		if !info.IsDir() {
			log.WithField("processID", processID).Info("Adding file: ", filePath)
			_, err := worktree.Add(filePath)
			if err != nil {
				log.WithField("processID", processID).Errorf("Warning: Failed to add file %s: %v", filePath, err)
			}
		}
		return nil
	})
	if err != nil {
		log.WithField("processID", processID).Errorf("failed to walk directory: %v", err)
		return err
	}

	status, err := worktree.Status()
	if err != nil {
		log.WithField("processID", processID).Errorf("failed to get status: %v", err)
		return err
	}

	if status.IsClean() {
		log.WithField("processID", processID).Info("No changes to commit")
		return nil
	}

	_, err = worktree.Commit(commitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  gitAuthor.AuthorName,
			Email: gitAuthor.AuthorEmail,
			When:  time.Now(),
		},
	})
	if err != nil {
		log.WithField("processID", processID).Errorf("failed to commit changes: %v", err)
		return err
	}

	err = repo.Push(&git.PushOptions{
		Auth:       getAuthMethod(auth),
		RemoteName: "origin",
		RefSpecs:   []config.RefSpec{config.RefSpec("refs/heads/" + branch + ":refs/heads/" + branch)},
		Progress:   os.Stdout,
		Force:      true,
	})
	if err != nil {
		log.WithField("processID", processID).Errorf("failed to push changes: %v", err)
		return err
	}

	log.WithField("processID", processID).Info("Successfully pushed to ", branch)
	return nil
}

// getAuthMethod выбирает метод аутентификации (HTTP или SSH)
func getAuthMethod(auth GitAuth) transport.AuthMethod {
	if auth.UseSSH {
		publicKeys, err := ssh.NewPublicKeysFromFile("git", auth.SSHKey, "")
		if err != nil {
			fmt.Println("Failed to load SSH key:", err)
			return nil
		}
		return publicKeys
	}
	return &http.BasicAuth{
		Username: auth.Username, // Обычно "token" для GitHub
		Password: auth.Password, // Личный токен
	}
}

func loadGitIgnore(repoPath string) (*gitignore.GitIgnore, error) {
	gitignorePath := filepath.Join(repoPath, ".gitignore")
	if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
		return nil, nil
	}
	content, err := os.ReadFile(gitignorePath)
	if err != nil {
		return nil, err
	}
	return gitignore.CompileIgnoreLines(string(content)), nil
}

func NewGitCache(repoURL, localPath string, branch string) *GitCache {
	return &GitCache{
		repoURL:   repoURL,
		localPath: localPath,
		branch:    branch,
	}
}

func (gc *GitCache) InitializeMetarepo(auth GitAuth, processID string) error {
	// Используем initMu, а не gc.mu, чтобы избежать блокировки внутри sync.Once
	gc.initMu.Lock()
	defer gc.initMu.Unlock()

	gc.initOnce.Do(func() {
		if _, err := os.Stat(gc.localPath); os.IsNotExist(err) {
			log.WithField("processID", processID).Infof("The metarepo does not exist locally, will plain clone")
			gc.repo, gc.initErr = PlainCloneRepository(auth, gc.repoURL, gc.branch, gc.localPath, uuid.New().String())
		} else {
			log.WithField("processID", processID).Infof("The metarepo exists locally, will plain open from filesystem")
			gc.repo, gc.initErr = git.PlainOpen(gc.localPath)
		}

		if gc.initErr != nil {
			return
		}

		ref, err := gc.repo.Head()
		if err != nil {
			gc.initErr = err
			log.WithField("processID", processID).Errorf("failed to get HEAD: %v", err)
			return
		}
		gc.lastCommitID = ref.Hash()
	})
	return gc.initErr
}

func (gc *GitCache) pullLatest(auth GitAuth, processID string) error {
	gc.mu.Lock()
	defer gc.mu.Unlock()

	w, err := gc.repo.Worktree()
	if err != nil {
		log.WithField("processID", processID).Errorf("failed to get worktree: %v", err)
		return err
	}

	status, err := w.Status()
	if err != nil {
		log.WithField("processID", processID).Errorf("failed to get status: %v", err)
		return err
	}

	if status.IsClean() {
		log.WithField("processID", processID).Info("No changes to commit")
		return nil
	}
	if err := w.Pull(&git.PullOptions{
		RemoteName: "origin",
		RemoteURL:  gc.repoURL,
		Auth:       getAuthMethod(auth),
		Progress:   os.Stdout,
	}); err != nil && !errors.Is(err, git.NoErrAlreadyUpToDate) {
		log.WithField("processID", processID).Errorf("pull failed: %+v", err)
		return err
	}
	log.WithField("processID", processID).Infof("The metarepo was pulled successfully")

	ref, err := gc.repo.Head()
	if err != nil {
		log.WithField("processID", processID).Errorf("failed to get HEAD after pull: %v", err)
		return err
	}
	log.WithField("processID", processID).Infof("The metarepo last commit ID is: %v", ref.Hash().String())
	log.WithField("processID", processID).Infof("The metarepo previous commit ID was: %v", gc.lastCommitID)
	gc.lastCommitID = ref.Hash()
	return nil
}

func (gc *GitCache) GetFile(filename string, auth GitAuth, processID string) ([]byte, error) {
	if err := gc.InitializeMetarepo(auth, processID); err != nil {
		log.WithField("processID", processID).Errorf("repository initialization failed: %v", err)
		return nil, err
	}

	if err := gc.pullLatest(auth, processID); err != nil {
		return nil, err
	}

	// Блокируем только для чтения файла
	gc.mu.RLock()
	defer gc.mu.RUnlock()

	filePath := filepath.Join(gc.localPath, filename)
	data, err := os.ReadFile(filePath)
	if err != nil {
		log.WithField("processID", processID).Errorf("failed to read file: %v", err)
		return nil, err
	}

	return data, nil
}
