package main

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"net/http"
	"sync"
	"encoding/json"
)

const GITHUB_API_URL string = "https://api.github.com"
const GITHUB_OWNER string = "lablabs"
const REPO_LIST_FILEPATH string = "configs/repos.yaml"
const CMP_FILE_LIST_FILEPATH string = "configs/files.yaml"
const PR_RAISED_LABEL string = "kind/sync"
const REPO_BRANCH = "main"

var GITHUB_TOKEN string = os.Getenv("GITHUB_TOKEN") // Provided by GitHub Actions

type Repo struct {
	Name     string `yaml:"name"`
	Category []string `yaml:"category"`
}

type Repos struct {
	Parent string `yaml:"parent"`
	Childs []Repo `yaml:"repos"`
}

type FileList struct {
	Common   []string `yaml:"common"`
	Irsa     []string `yaml:"addon-irsa"`
	Oidc []string `yaml:"addon-oidc"`
}

type Issue struct {
    Number      int             `json:"number"`
    Title       string          `json:"title"`
    PullRequest *PullRequestRef `json:"pull_request,omitempty"`
}

type PullRequestRef struct {
    URL string `json:"url"`
}

type PRCheckResult struct {
    Repo       string
    LabelFound bool
    PRNumber   int
    Title      string
    Error      error
}

type GitFile struct {
    Name string `json:"name"`
	  Checksum string `json:"sha"`
}

type ChecksumFile struct {
	  Filename string
	  Checksum string
	  Error    error
}

type RepoChecksumFile struct {
	  Repo string
	  ChecksumFiles []ChecksumFile
	  Error    error
}

func (R *Repos) LoadData() error {
	repos, err := os.ReadFile(REPO_LIST_FILEPATH)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(repos, &R)
	if err != nil {
		return err
	}

	return nil
}

func (F *FileList) LoadData() error {
	repos, err := os.ReadFile(CMP_FILE_LIST_FILEPATH)
	if err != nil {
		return err
	}

	err = yaml.Unmarshal(repos, &F)
	if err != nil {
		return err
	}

	return nil
}

// Fetch open PRs
func checkRepoForPR(repo string, wg *sync.WaitGroup, resultChan chan<- PRCheckResult) {
    defer wg.Done()
    url := fmt.Sprintf("https://api.github.com/repos/%s/issues?state=open&labels=%s&per_page=100", repo, PR_RAISED_LABEL)

    req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        resultChan <- PRCheckResult{Repo: fmt.Sprintf("%s", repo), Error: err}
        return
    }

    req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("Authorization", "Bearer "+GITHUB_TOKEN)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        resultChan <- PRCheckResult{Repo: fmt.Sprintf("%s", repo), Error: err}
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        resultChan <- PRCheckResult{Repo: fmt.Sprintf("%s", repo), Error: fmt.Errorf("GitHub API error: %s", resp.Status)}
        return
    }

    var issues []Issue
    if err := json.NewDecoder(resp.Body).Decode(&issues); err != nil {
        resultChan <- PRCheckResult{Repo: fmt.Sprintf("%s", repo), Error: err}
        return
    }

    for _, issue := range issues {
        if issue.PullRequest != nil {
            resultChan <- PRCheckResult{
                Repo:       fmt.Sprintf("%s", repo),
                LabelFound: true,
                PRNumber:   issue.Number,
                Title:      issue.Title,
            }
            return
        }
    }

    resultChan <- PRCheckResult{
        Repo:       fmt.Sprintf("%s", repo),
        LabelFound: false,
    }
}

func fetchFileChecksum(repo string, filename string, wg *sync.WaitGroup, resultChan chan<- ChecksumFile) {
    defer wg.Done()
    url := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s?ref=%s", repo, filename, REPO_BRANCH )

	  req, err := http.NewRequest("GET", url, nil)
    if err != nil {
        resultChan <- ChecksumFile{Filename: fmt.Sprintf("%s", repo), Error: err}
        return
    }

    req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("Authorization", "Bearer "+GITHUB_TOKEN)

    client := &http.Client{}
    resp, err := client.Do(req)
    if err != nil {
        resultChan <- ChecksumFile{Filename: fmt.Sprintf("%s", filename), Error: err}
        return
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        resultChan <- ChecksumFile{Filename: fmt.Sprintf("%s", filename), Error: fmt.Errorf("GitHub API error: %s", resp.Status)}
        return
    }

    var gitfile GitFile
    if err := json.NewDecoder(resp.Body).Decode(&gitfile); err != nil {
        resultChan <- ChecksumFile{Filename: fmt.Sprintf("%s", filename), Error: err}
        return
    }

    resultChan <- ChecksumFile{
      			Filename:filename, 
						Checksum:gitfile.Checksum,
	  }
}

// func checkRepoForFilesChecksums(repo string, fileList []string, wg *sync.WaitGroup, resultChan chan<- PRCheckResult) {
func checkRepoForFilesChecksums(repo string, fileList []string, wg *sync.WaitGroup, resultChan chan<- RepoChecksumFile) {
		defer wg.Done()
		
	  repoChecksums := RepoChecksumFile{Repo: repo}
		var wgFiles sync.WaitGroup
		resultChanFiles := make(chan ChecksumFile, len(fileList))

		for _, file := range fileList {
				wgFiles.Add(1)
				go fetchFileChecksum(repo, file, &wgFiles, resultChanFiles)
		}

		wgFiles.Wait()
		close(resultChanFiles)
    
    for files := range resultChanFiles {
       repoChecksums.ChecksumFiles = append(repoChecksums.ChecksumFiles, files)
    }
		resultChan <- repoChecksums
}

func compareFilesInRepo(parentRes RepoChecksumFile, repoRes RepoChecksumFile) {
	fmt.Printf("\n🔍 Processing Sub Repo: [%s] <==> Parent Repo: [%s] \n", repoRes.Repo, parentRes.Repo)
	for _, parentFiles := range parentRes.ChecksumFiles {
			if parentFiles.Error != nil {
					fmt.Printf("❌ [%s] Error: %v\n", parentRes.Repo, parentFiles.Error)
					continue
			}
			for _, repoFiles := range repoRes.ChecksumFiles {
				if repoRes.Error != nil {
						fmt.Printf("❌ [%s] Error: %v\n", repoRes.Repo, repoFiles.Error)
						continue
				}

				if parentFiles.Filename != repoFiles.Filename {
					continue
				}

				if parentFiles.Checksum == repoFiles.Checksum {
						fmt.Printf("✅ [%s] {%s} Match\n", repoRes.Repo, repoFiles.Filename)
				} else {
						fmt.Printf("🚫 [%s] {%s} Don't Match\n", repoRes.Repo, repoFiles.Filename)
				}
			}
	}
}

func main() {

	if GITHUB_TOKEN == "" {
		fmt.Println("🐛 GitHub token is not set. Ensure GITHUB_TOKEN is available as an environment variable.")
		os.Exit(101)
	}

	var LaraRepos Repos
	err := LaraRepos.LoadData()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	var LaraFiles FileList
	err = LaraFiles.LoadData()
	if err != nil {
		fmt.Println(err)
		os.Exit(2)
	}

	fmt.Printf("\n🔍 Checking Open PRs with label: %s\n", PR_RAISED_LABEL)
	var wg sync.WaitGroup
	resultChan := make(chan PRCheckResult, len(LaraRepos.Childs))

	for _, r := range LaraRepos.Childs {
			wg.Add(1)
			go checkRepoForPR(r.Name, &wg, resultChan)
	}

	wg.Wait()
	close(resultChan)

	var pRepos []string
	// Collect and print results
	for result := range resultChan {
			if result.Error != nil {
					fmt.Printf("❌ [%s] Error: %v\n", result.Repo, result.Error)
					continue
			}
			if result.LabelFound {
					fmt.Printf("✅ [%s] PR #%d has label '%s': %s\n", result.Repo, result.PRNumber, PR_RAISED_LABEL, result.Title)
			} else {
					fmt.Printf("🚫 [%s] No open PR found with label '%s'\n", result.Repo, PR_RAISED_LABEL)
				  pRepos = append(pRepos, result.Repo)
			}
	}

	resultChanRepo := make(chan RepoChecksumFile, len(pRepos)+1)
	wg.Add(1)
	go checkRepoForFilesChecksums(LaraRepos.Parent, LaraFiles.Common, &wg, resultChanRepo)

	for _, r := range pRepos {
			wg.Add(1)
			go checkRepoForFilesChecksums(r, LaraFiles.Common, &wg, resultChanRepo)
	}

	wg.Wait()
	close(resultChanRepo)
  
	resultMap := make(map[string]RepoChecksumFile)
	for result := range resultChanRepo {
	  resultMap[result.Repo] = result    
	}

	for k, v := range resultMap {
		if k == LaraRepos.Parent {
			continue
		}
		compareFilesInRepo(resultMap[LaraRepos.Parent], v)
	}
}
