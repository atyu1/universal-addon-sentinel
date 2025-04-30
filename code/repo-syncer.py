import os
import requests

# GitHub API URL and token
GITHUB_API_URL = "https://api.github.com"
GITHUB_TOKEN = os.getenv("GITHUB_TOKEN")  # Provided by GitHub Actions

REPO_LIST_FILEPATH = "../repos.yaml"
CMP_FILE_LIST_FILEPATH = "../files.yaml"

def load_repositories():
    if not os.path.exists(yaml_file):
        print(f"❌ YAML file {yaml_file} not found!")
        return {}

    with open(REPO_LIST_FILEPATH, "r") as file:
        return yaml.safe_load(file)

def load_file_list():
    if not os.path.exists(file_path):
        print(f"⚠️ Configuration file {CMP_FILE_LIST_FILEPATH} not found!")
        return []
    with open(CMP_FILE_LIST_FILEPATH, "r") as file:
        return yaml.safe_load(file) 

def fetch_file_content(repo, file_path, branch="main"):
    url = f"{GITHUB_API_URL}/repos/{repo}/contents/{file_path}?ref={branch}"
    headers = {"Authorization": f"token {GITHUB_TOKEN}"}
    response = requests.get(url, headers=headers)

    if response.status_code == 200:
        content = response.json().get("content")
        return content.encode("utf-8").decode("utf-8")  # Decode base64 content
    else:
        print(f"Failed to fetch {file_path} from {repo}: {response.status_code} {response.text}")
        return None

def compare_files(parent_repo, sub_repos, file_cmp_list):
    all_in_sync = True

    for sub_repo in sub_repos:
        print(f"\nComparing files in {sub_repo} with {parent_repo}...\n")

        for file_path in file_cmp_list:
            parent_content = fetch_file_content(parent_repo, file_path)
            sub_repo_content = fetch_file_content(sub_repo, file_path)

            if parent_content and sub_repo_content:
                if parent_content == sub_repo_content:
                    print(f"✅ {file_path} is identical in both {parent_repo} and {sub_repo}.")
                else:
                    print(f"❌ {file_path} differs between {parent_repo} and {sub_repo}.")
                    all_in_sync = False
            else:
                print(f"⚠️ Could not compare {file_path} due to missing content in one of the repositories.")
                all_in_sync = False

    if all_in_sync:
        print("\n🎉 All files are in sync!")
    else:
        print("\n❌ Some files are not in sync. Please review the differences.")
        exit(1)  # Exit with an error code if files are not in sync

if __name__ == "__main__":
    if not GITHUB_TOKEN:
        print("❌ GitHub token is not set. Ensure GITHUB_TOKEN is available as an environment variable.")
    else:
        # Load repositories from the YAML file
        repos = load_repositories()
        files_to_compare = load_file_list()

        if not repos:
            print("❌ No repositories to compare. Please check your YAML file.")
        else:
            for parent_repo, sub_repos in repos.items():
                print(f"\n🔍 Processing Parent Repo: {parent_repo}")
                compare_files(parent_repo, sub_repos)
