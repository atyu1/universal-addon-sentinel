package main

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"io"
)

const GITHUB_API_URL string = "https://api.github.com"
const GITHUB_TOKEN string = os.getenv("GITHUB_TOKEN")  // Provided by GitHub Actions
const GITHUB_OWNER string = "lablabs"
const REPO_LIST_FILEPATH string = "configs/repos.yaml"
const CMP_FILE_LIST_FILEPATH string = "configs/files.yaml"
const PR_RAISED_LABEL string = "kind/sync"

type Repo struct {
	name string `yaml: "name"`
	category string `yaml: "category"`
}

type Repos struct {
	parent string `yaml:"parent"`
	childs []repo
}

type FileList struct {
	commonFiles []string 	`yaml: "common"`
	irsaFiles []string  `yaml: "addon-irsa"`
	oidcFiles []string `yaml: "addon-irsa-oidc"`
}

func main()  {
  var LaraFiles FileList
	var LaraRepos Repos

  LaraFiles, err := os.ReadFile(REPO_LIST_FILEPATH)

}
