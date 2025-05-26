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

var GITHUB_TOKEN string = os.Getenv("GITHUB_TOKEN") // Provided by GitHub Actions

type Repo struct {
	Name     string `yaml:"name"`
	Category string `yaml:"category"`
}

type Repos struct {
	Parent string `yaml:"parent"`
	Childs []Repo `yaml:"repos"`
}

type FileList struct {
	Common   []string `yaml:"common"`
	Irsa     []string `yaml:"addon-irsa"`
	IrsaOidc []string `yaml:"addon-irsa-oidc"`
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

func main() {

	if GITHUB_TOKEN == "" {
		fmt.Println("🐛 GitHub token is not set. Ensure GITHUB_TOKEN is available as an environment variable.")
		os.Exit(101)
	}

	var LaraRepos Repos
	err := LaraRepos.LoadData()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(LaraRepos)

	var LaraFiles FileList
	err = LaraFiles.LoadData()
	if err != nil {
		fmt.Println(err)
	}
	fmt.Println(LaraFiles)

	var wg sync.WaitGroup
	resultChan := make(chan PRCheckResult, len(LaraRepos.Childs))

	for _, r := range LaraRepos.Childs {
			wg.Add(1)
		  fmt.Println(r.Name)
			go checkRepoForPR(r.Name, &wg, resultChan)
	}

	wg.Wait()
	close(resultChan)

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
			}
	}

}
