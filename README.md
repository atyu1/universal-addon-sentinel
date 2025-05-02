# Universal Addon Sentinel

    Universal Addon Sentinel is a GitHub Action-based automation tool designed to compare files between a parent repository and its sub-repositories. 
Primary focus was to use for terraform-aws-eks-universal-addon project to ensure that repos copied from this project are in sync, when there are changes on universal repo. But can be used for other projects due to its dynamic form.
It ensures synchronization and consistency across repositories by dynamically loading repository relationships and selected files for comparison. This project leverages `git diff` and YAML configuration files to provide detailed insights into differences between repositories.

---

## Features

- **Dynamic Repository Loading**: Uses `repos.yaml` to define parent and sub-repositories dynamically.
- **Flexible File Selection**: Configurable file list in `files.yaml` to specify which files to compare.
- **Detailed Output**: Provides a clear summary of file differences or confirms synchronization.
- **GitHub Actions Integration**: Automates comparisons as part of CI/CD pipelines.

---

## File Descriptions

### `files.yaml`
This file contains the list of files that should be compared between the parent and sub-repositories.  
Example structure:

```yaml
common:
  - README.md
  - script.py
  - configs/config.yaml

my-part:
  - variables.tf
```

- part **common** is required to present in every repo, this is file list which is automatically in all sub repos
- part **my-part** is dynamic extension to ensure if some sub repos are not using all files

### `repos.yaml`
This file defines the relationship between the parent repository and its sub-repositories.  
Example structure:

```yaml
atyu1/universal-addon-sentinel:
  - atyu1/sub-repo-1: "my-part"
  - atyu1/sub-repo-2: ""
  - atyu1/sub-repo-3: ""
```

- The **key** is the parent repository.
- The **values** are the list of sub-repositories to be compared against the parent.
- In this example, as you can see, on top of common file liest, sub-repo-1 use more common files with parent, described in **my-part**

---

## Example Output

### 1. **No Difference Found**
If all files in the sub-repositories match the parent repository, the output will look like this:
```
🔍 Processing Parent Repo: atyu1/universal-addon-sentinel

Comparing files in atyu1/sub-repo-1 with atyu1/universal-addon-sentinel...

✅ README.md is identical in both atyu1/universal-addon-sentinel and atyu1/sub-repo-1.
✅ script.py is identical in both atyu1/universal-addon-sentinel and atyu1/sub-repo-1.
✅ configs/config.yaml is identical in both atyu1/universal-addon-sentinel and atyu1/sub-repo-1.

🎉 All files are in sync!
```

### 2. **Differences Found**
If there are discrepancies between the parent and sub-repositories, the output will highlight the differences:
```
🔍 Processing Parent Repo: atyu1/universal-addon-sentinel

Comparing files in atyu1/sub-repo-2 with atyu1/universal-addon-sentinel...

✅ README.md is identical in both atyu1/universal-addon-sentinel and atyu1/sub-repo-2.
❌ script.py differs between atyu1/universal-addon-sentinel and atyu1/sub-repo-2.
⚠️ Could not compare configs/config.yaml due to missing content in one of the repositories.

❌ Some files are not in sync. Please review the differences.
```

---

## How to Use

1. **Set up the Configuration Files**:
   - Define the list of files to compare in `files.yaml`.
   - Define the repository relationships in `repos.yaml`.

2. **Run the GitHub Action**:
   - Push changes to your repository to trigger the workflow.
   - Alternatively, trigger the workflow manually using `workflow_dispatch`.

3. **Review the Output**:
   - Check the GitHub Actions logs for detailed results.
   - Download the difference reports if artifacts are configured in the workflow.

---

## License
This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
