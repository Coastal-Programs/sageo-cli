---
name: commit
description: Run checks, commit, push, and create a lightweight release tag
---

1. Run fast quality checks. Fix ALL errors before continuing:
   ```bash
   # Keep this lightweight: no release/build packaging steps.
   go vet ./...
   go test -race ./...
   ```

2. Review changes: run `git status`, `git diff --staged`, and `git diff`

3. Stage relevant files with `git add` (specific files, not `-A`)

4. Generate a high-quality commit summary from actual changes:
   - Create a one-line subject that starts with a verb (`Add`, `Update`, `Fix`, `Remove`, `Refactor`)
   - Also create a commit body with short bullet points grouped by type, only for real changes:
     - `Added:`
     - `Updated:`
     - `Fixed:`
     - `Docs:`
   - If a section has no items, omit it.
   - Keep bullets concrete (file/function/behavior), no vague text.

5. Commit with subject + body and push:
   ```bash
   subject="your generated one-line subject"
   {
     echo "$subject"
     echo
     echo "Added:"
     echo "- ..."
     echo
     echo "Updated:"
     echo "- ..."
     echo
     echo "Fixed:"
     echo "- ..."
     echo
     echo "Docs:"
     echo "- ..."
   } > /tmp/sageo_commit_msg.txt

   git commit -F /tmp/sageo_commit_msg.txt
   git push
   ```

6. Create and push the next lightweight semver patch tag (no binary build):
   ```bash
   latest_tag=$(git tag --list 'v*' --sort=-v:refname | head -n1)
   if [ -z "$latest_tag" ]; then
     next_tag="v0.1.0"
   else
     v="${latest_tag#v}"
     IFS='.' read -r major minor patch <<< "$v"
     major=${major:-0}
     minor=${minor:-1}
     patch=${patch:-0}
     next_tag="v${major}.${minor}.$((patch + 1))"
   fi

   git tag -a "$next_tag" -m "Release $next_tag"
   git push origin "$next_tag"
   ```

7. If GitHub CLI is available, publish a lightweight GitHub Release from that tag (no assets/builds):
   ```bash
   if command -v gh >/dev/null 2>&1; then
     gh release create "$next_tag" \
       --title "$next_tag" \
       --notes "Automated lightweight release (no binaries attached)."
   else
     echo "gh CLI not installed; tag pushed. Create release manually in GitHub UI if needed."
   fi
   ```
