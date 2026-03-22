import csv
import os
import json
import yaml
import requests
from datetime import datetime, timezone

# GitHub API URL and token
GITHUB_API_URL = "https://api.github.com"
GITHUB_TOKEN = os.getenv("GITHUB_TOKEN")  # Provided by GitHub Actions
GITHUB_OWNER = "lablabs"
REPO_LIST_FILEPATH = "repos.yaml"
CMP_FILE_LIST_FILEPATH = "files.yaml"
PR_RAISED_LABEL = "kind/sync"
ENABLE_CSV_EXPORT = os.getenv("ENABLE_CSV_EXPORT", "false").lower() == "true"
CSV_EXPORT_DIR = "exports"

def load_repositories():
    if not os.path.exists(REPO_LIST_FILEPATH):
        print(f"❌ YAML file {REPO_LIST_FILEPATH} not found!")
        return {}

    with open(REPO_LIST_FILEPATH, "r") as file:
        return yaml.safe_load(file)

def load_file_list():
    if not os.path.exists(CMP_FILE_LIST_FILEPATH):
        print(f"⚠️ Configuration file {CMP_FILE_LIST_FILEPATH} not found!")
        return []
    with open(CMP_FILE_LIST_FILEPATH, "r") as file:
        return yaml.safe_load(file) 

def verify_pr_raised(repo):
    url = f"{GITHUB_API_URL}/search/issues"
    headers = {"Authorization": f"token {GITHUB_TOKEN}", "Accept": "application/vnd.github.v3+json"}
    query = f"repo:{repo} is:pr is:open label:{PR_RAISED_LABEL}"
    params = {"q": query}
    response = requests.get(url, headers=headers, params=params)

    number_of_prs = json.loads(response.text)["total_count"]

    if response.status_code == 200 and number_of_prs > 0:
        print(f"🛠️ skipping repo: {repo}, PR with label:{PR_RAISED_LABEL} already raised")
        return True
    else:
        return False

def fetch_file_content(repo, file_path, branch="main"):
    url = f"{GITHUB_API_URL}/repos/{repo}/contents/{file_path}?ref={branch}"
    headers = {"Authorization": f"token {GITHUB_TOKEN}"}
    response = requests.get(url, headers=headers)

    if response.status_code == 200:
        content = response.json().get("content")
        return content.encode("utf-8")  # Decode base64 content
    else:
        print(f"🐛 Failed to fetch {file_path} from {repo}: {response.status_code} {response.text}")
        return None

def get_used_files_by_repo(sub_repo, file_cmp_list):
    repo = ""
    file_list = file_cmp_list["common"][:]

    for repo_name, file_group in sub_repo.items():
        repo = repo_name
        file_list.extend(file_cmp_list.get(file_group, ""))

    return repo, file_list

def export_to_csv(results, sub_repo_name):
    os.makedirs(CSV_EXPORT_DIR, exist_ok=True)
    now = datetime.now(timezone.utc)
    timestamp = now.strftime("%Y-%m-%d_%H-%M-%S-") + f"{now.microsecond // 1000:03d}"
    safe_repo_name = sub_repo_name.replace("/", "_")
    csv_filepath = os.path.join(CSV_EXPORT_DIR, f"{safe_repo_name}_{timestamp}.csv")
    fieldnames = ["timestamp", "parent_repo", "sub_repo", "file_path", "status"]
    with open(csv_filepath, "w", newline="") as csvfile:
        writer = csv.DictWriter(csvfile, fieldnames=fieldnames)
        writer.writeheader()
        writer.writerows(results)
    print(f"\n📊 CSV report saved to {csv_filepath}")

def compare_files(parent_repo, sub_repos, file_cmp_list):
    all_in_sync = True
    run_timestamp = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ")

    for sub_repo in sub_repos:
        print(f"\n📄 Comparing files in {sub_repo} with {parent_repo}...\n")
        file_list_used_by_repo = []
        sub_repo_name, file_list_used_by_repo = get_used_files_by_repo(sub_repo, file_cmp_list)

        if verify_pr_raised(sub_repo_name):
            continue

        csv_results = []

        for file_path in file_list_used_by_repo:
            parent_content = fetch_file_content(parent_repo, file_path)
            sub_repo_content = fetch_file_content(sub_repo_name, file_path)
            status = "missing"

            if parent_content and sub_repo_content:
                if parent_content == sub_repo_content:
                    print(f"✅ {file_path} is identical in both {parent_repo} and {sub_repo_name}.")
                    status = "identical"
                else:
                    print(f"❌ {file_path} differs between {parent_repo} and {sub_repo_name}.")
                    all_in_sync = False
                    status = "differs"
            else:
                print(f"⚠️ Could not compare {file_path} due to missing content in one of the repositories.")
                all_in_sync = False
                status = "missing"

            csv_results.append({
                "timestamp": run_timestamp,
                "parent_repo": parent_repo,
                "sub_repo": sub_repo_name,
                "file_path": file_path,
                "status": status,
            })

        if ENABLE_CSV_EXPORT:
            export_to_csv(csv_results, sub_repo_name)

    if all_in_sync:
        print("\n🎉 All files are in sync!")
    else:
        print("\n❌ Some files are not in sync. Please review the differences.")
        exit(1)  # Exit with an error code if files are not in sync

if __name__ == "__main__":
    if not GITHUB_TOKEN:
        print("🐛 GitHub token is not set. Ensure GITHUB_TOKEN is available as an environment variable.")
    else:
        # Load repositories from the YAML file
        repos = load_repositories()
        files_to_compare = load_file_list()

        if not repos:
            print("🐛 No repositories to compare. Please check your YAML file.")
        else:
            for parent_repo, sub_repos in repos.items():
                print(f"\n🔍 Processing Parent Repo: {parent_repo}")
                compare_files(parent_repo, sub_repos, files_to_compare)
