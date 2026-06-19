---
name: auto-archive-and-push
description: "Use when the user wants to automatically archive all active openspec (spex) changes, sync/update the main specs, and commit and push the changes to GitHub with a Conventional Commit message. Make sure to trigger this skill when the user mentions auto-archiving specs, sync and push, commit and push after archiving, or automatically running spex-archive followed by git commit and push."
license: MIT
compatibility: Requires openspec CLI and git.
metadata:
  author: spex
  version: "1.0"
---

Archive all active changes, update main specifications, commit all changes following conventional commit guidelines, and push to GitHub.

## Steps

1. **Find all active changes**
   Run `openspec list --json` to get the list of active openspec changes.
   Parse the list to find the names of all active changes.
   If no active changes are found, notify the user but proceed to Step 3 if there are other uncommitted changes in git.

2. **Archive and sync specs automatically**
   For each active change:
   a. Run `openspec archive -y <name>` to archive the change and update the main specs.
   b. Look for a brainstorm plan reference in the change's proposal file at `openspec/changes/<name>/proposal.md` under a `## Source` section.
      Specifically, search for: `Derived from brainstorm plan: .spex/plans/<filename>`
      If found:
      - Create the archive directory `.spex/plans/archive/` if it doesn't exist:
        `mkdir -p .spex/plans/archive/`
      - Move the brainstorm file to the archive directory:
        `mv .spex/plans/<filename> .spex/plans/archive/<filename>`
      - Report brainstorm plan archived.

3. **Stage and commit files**
   Check `git status --porcelain` to identify modified and untracked files (such as updated main specs under `openspec/specs/`, archived changes under `openspec/changes/archive/`, and archived brainstorm plans under `.spex/plans/archive/`).
   
   If there are changes:
   a. Stage all modified and untracked files:
      `git add -A`
   b. Determine a suitable commit message. The commit message MUST strictly adhere to the Conventional Commits guidelines defined in the project:
      - Format: `<type>[optional scope]: <description>`
      - Use `docs` or `chore` type, e.g., `docs(spec): archive completed openspec changes and sync specs` or `docs(spec): archive <change-name> and update specs`.
      - Description must be clear and lowercase.
   c. Commit the staged changes:
      `git commit -m "<commit-message>"`

4. **Push to GitHub**
   a. Identify the current branch name:
      `git branch --show-current`
   b. Push the commits to the remote repository:
      `git push origin <branch-name>`

5. **Display Summary**
   Display a detailed summary showing:
   - All archived changes
   - Main specs updated/synced
   - Brainstorm plans archived
   - Git commit message used
   - Push status (successful / branch name)

## Output Format
```
## Auto-Archive & Sync Summary

**Archived Changes:**
- <change-name-1>
- <change-name-2>

**Updated Main Specs:**
- <spec-path-1>
- <spec-path-2>

**Brainstorm Plans Archived:**
- `.spex/plans/archive/<filename>`

**Git Commit:**
- Message: `<commit-message>`
- Branch: `<branch-name>`
- Status: Successfully pushed to GitHub

All active changes have been archived, specifications synced, and changes pushed to origin.
```
