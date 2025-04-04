package process

import (
	"fmt"
	"gservice/git"
	"os"
	"path"
	"regexp"
	"strings"
	"sync"

	"github.com/sirupsen/logrus"
)

type GitCheckFunctionParams struct {
	RepoName     string
	RepoUser     string
	RepoPassword string
	ProcessID    string
}

type GitCopyFunctionParams struct {
	FileName       string
	SubFolderName  string
	Repo1UserName  string
	Repo2UserName  string
	Repo1Branch    string
	Repo2Branch    string
	Repo1Path      string
	Repo2Path      string
	Repo1LocalPath string
	Repo2LocaPath  string
	Repo1Password  string
	Repo2Password  string
	ProcessID      string
}

var mu sync.Mutex
var log = logrus.New()

func ProcessCheckGitFunction(params GitCheckFunctionParams) (err error) {
	log.WithField("processID", params.ProcessID).Info("process started")

	// Simulate some processing
	if params.RepoName == "" || params.RepoUser == "" || params.RepoPassword == "" {
		return fmt.Errorf("missing required parameters")
	}

	// Simulate success
	log.WithField("processID", params.ProcessID).Info("check process completed successfully")
	return nil
}

func ProcessCopyGitFunction(params GitCopyFunctionParams) (err error) {
	log := logrus.New()
	log.WithField("processID", params.ProcessID).Info("process started")

	// Simulate some processing
	if params.Repo1Password == "" || params.Repo2Password == "" || params.FileName == "" {
		return fmt.Errorf("missing required parameters")
	}

	authRepo1 := createGitAuth(
		params.Repo1UserName,
		params.Repo1Password)
	authRepo2 := createGitAuth(
		params.Repo2UserName,
		params.Repo2Password)

	repo1Cache := createGitCache(params.Repo1Path, params.Repo1LocalPath, params.Repo1Branch)
	filedata, err := repo1Cache.GetFile(params.FileName, authRepo1, params.ProcessID)
	if err != nil {
		log.WithField("processID", params.ProcessID).Errorf("Unable to get metafile from git: %v", err)
		return err
	}

	repoDir := prepareProjectRepoName(
		params.Repo2Path,
		params.Repo2Branch,
		params.ProcessID)

	mu.Lock()
	defer mu.Unlock()
	volumeRepo2Subdir := path.Join(params.Repo2LocaPath, repoDir)
	repo, err := git.CloneRepositoryWithLocalFolderWipe(authRepo2, params.Repo2Path, params.Repo2Branch, volumeRepo2Subdir, params.ProcessID) // ???
	if err != nil {
		log.WithField("processID", params.ProcessID).Errorf("Unable to clone repository %s %v", params.Repo2Path, err)
		return err
	}

	projectSubfolder := prepareProjectFolderName(volumeRepo2Subdir, params.SubFolderName)

	// Check if the subfolder exists, if not, create it
	if _, err := os.Stat(projectSubfolder); os.IsNotExist(err) {
		err = os.MkdirAll(projectSubfolder, os.ModePerm)
		if err != nil {
			log.WithField("processID", params.ProcessID).Errorf("Unable to create subfolder: %v", err)
			return err
		}
	}

	// Delete the local file if it already exists
	localFilePath := projectSubfolder + "/" + params.FileName
	if _, err := os.Stat(localFilePath); err == nil {
		err = os.Remove(localFilePath)
		if err != nil {
			log.WithField("processID", params.ProcessID).Errorf("Unable to delete existing file: %v", err)
			return err
		}
	}
	log.WithField("processID", params.ProcessID).Info("saving file to local path: ", localFilePath)
	// Save filedata to a local file
	err = os.WriteFile(localFilePath, filedata, 0644)
	if err != nil {
		log.WithField("processID", params.ProcessID).Errorf("Unable to save file to local path: %v", err)
		return err
	}

	defer func() {
		if err := eraseFolder(volumeRepo2Subdir, params.ProcessID); err != nil {
			log.WithField("processID", params.ProcessID).Error("unable to delete temporary repository folder", err)
		}
	}()
	/*
		w, err := repo.Worktree()
		if err != nil {
			log.WithField("processID", params.ProcessID).Errorf("failed to get worktree: %v", err)
			return err
		}

		status, err := w.Status()
		if err != nil {
			log.WithField("processID", params.ProcessID).Errorf("failed to get status: %v", err)
			return err
		}
		log.WithField("processID", params.ProcessID).Infof("The repo status is: %v", status)
	*/
	err = git.CommitAndPush(authRepo2, "commit "+params.ProcessID, repo, projectSubfolder, params.Repo2Branch, params.ProcessID, params.SubFolderName)
	if err != nil {
		log.WithField("processID", params.ProcessID).Error("Error commiting resources to GIT:", err)
		return err
	}
	log.WithField("processID", params.ProcessID).Info("copy process completed successfully")
	return nil
}

func createGitAuth(gitUsername string, gitToken string) git.GitAuth {
	authChartRepo := git.GitAuth{
		Username: gitUsername,
		Password: gitToken,
	}
	return authChartRepo
}

func createGitCache(gitPath, localPath, branch string) *git.GitCache {
	gitCache := git.NewGitCache(
		gitPath,
		localPath,
		branch,
	)
	return gitCache
}

func prepareProjectRepoName(repo string, branch string, processID string) string {
	httpRegex := regexp.MustCompile(`(?i)https?://`)
	repo = httpRegex.ReplaceAllString(repo, "")

	replacer := strings.NewReplacer("/", "_", ".", "_", ":", "_")
	sanitizedRepo := replacer.Replace(repo + "-" + branch)

	return path.Join(".", sanitizedRepo+"_"+processID)
}

func eraseFolder(dirPath string, processID string) (err error) {
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		log.WithField("processID", processID).Error("Could not find the folder: ", dirPath)
		return err
	} else {
		err := os.RemoveAll(dirPath)
		if err != nil {
			log.WithField("processID", processID).Error("The error occurred during erase", dirPath)
			return err
		} else {
			log.WithField("processID", processID).Info("The temporary local folder is deleted: ", dirPath)
			return nil
		}
	}
}

func prepareProjectFolderName(projectRepoName string, projectSubfolder string) string {
	return path.Join(projectRepoName, projectSubfolder)
}
