# github-renovate-labeller

A CLI tool that automatically adds a `dependencies` label to Renovate bot's **Dependency Dashboard** issues across your GitHub repositories.

## What it does

- Iterates over all repositories for a given GitHub owner (user or organization).
- Finds open issues titled "Dependency Dashboard" created by `renovate[bot]`.
- Ensures a `dependencies` label (color `#0366d6`) exists in each repository, creating it if needed.
- Adds the label to any matching issue that doesn't already have it.

## Usage

```bash
GITHUB_TOKEN=<your-token> GITHUB_OWNER=<owner> github-renovate-labeller
```

### Environment variables

| Variable         | Required | Description                                                                                   |
|------------------|----------|-----------------------------------------------------------------------------------------------|
| `GITHUB_TOKEN`   | Yes      | GitHub personal access token with `repo` scope.                                               |
| `GITHUB_OWNER`   | Yes      | GitHub owner (user or organization name).                                                     |
|                  |          |                                                                                               |
| `GITHUB_REPO`    | No       | If set, only process this single repository. Otherwise all repos for the owner are processed. |
| `GITHUB_DRY_RUN` | No       | Set to `true` or `1` to preview changes without applying them.                                |

### Flags

```
-h, --help    Show help message
```

## Building

```bash
go build -o github-renovate-labeller .
```

## Example

```bash
# Label Dependency Dashboard issues in all repos for the "bakito" owner
GITHUB_TOKEN=ghp_... GITHUB_OWNER=bakito ./github-renovate-labeller

# Dry-run on a single repository
GITHUB_TOKEN=ghp_... GITHUB_OWNER=bakito GITHUB_REPO=my-repo GITHUB_DRY_RUN=true ./github-renovate-labeller
```

## License

MIT
