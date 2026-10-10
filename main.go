package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/google/go-github/v69/github"
	"golang.org/x/oauth2"
)

const (
	renovateBotLogin = "renovate[bot]"
	issueTitle       = "Dependency Dashboard"
	labelName        = "dependencies"
	labelColor       = "0366d6"
)

func main() {
	dryRun := strings.ToLower(os.Getenv("GITHUB_DRY_RUN")) == "true" || strings.ToLower(os.Getenv("GITHUB_DRY_RUN")) == "1"
	if dryRun {
		log.Println("running in dry-run mode — no labels will be added")
	}

	token := os.Getenv("GITHUB_TOKEN")
	if token == "" {
		log.Fatal("GITHUB_TOKEN environment variable is required")
	}

	owner := os.Getenv("GITHUB_OWNER")
	if owner == "" {
		log.Fatal("GITHUB_OWNER environment variable is required")
	}

	repo := os.Getenv("GITHUB_REPO")

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	client := github.NewClient(oauth2.NewClient(ctx, ts))

	var repos []string
	if repo != "" {
		repos = []string{repo}
	} else {
		var err error
		repos, err = listRepos(ctx, client, owner)
		if err != nil {
			log.Fatalf("failed to list repositories: %v", err)
		}
		if len(repos) == 0 {
			log.Println("no repositories found for owner")
			return
		}
		log.Printf("found %d repositories for owner %q", len(repos), owner)
	}

	totalLabeled := 0
	totalSkipped := 0
	totalFound := 0

	for _, r := range repos {
		if err := ensureLabelExists(ctx, client, owner, r, labelName, labelColor); err != nil {
			log.Printf("failed to ensure label exists for %s/%s: %v", owner, r, err)
		}

		issues, err := fetchRenovateDependencyDashboardIssues(ctx, client, owner, r)
		if err != nil {
			log.Printf("failed to fetch issues for %s/%s: %v", owner, r, err)
			continue
		}

		if len(issues) == 0 {
			continue
		}

		for _, issue := range issues {
			totalFound++
			issueNumber := issue.GetNumber()
			existingLabels := issue.Labels

			if hasLabel(existingLabels, labelName) {
				totalSkipped++
				log.Printf("%s/%s: issue #%d already has %q label, skipping", owner, r, issueNumber, labelName)
				continue
			}

			if dryRun {
				log.Printf("%s/%s: [dry-run] would add %q label to issue #%d", owner, r, labelName, issueNumber)
				continue
			}

			if err := addLabel(ctx, client, owner, r, issueNumber, labelName); err != nil {
				log.Printf("failed to add label to issue #%d in %s/%s: %v", issueNumber, owner, r, err)
				continue
			}

			totalLabeled++
			log.Printf("%s/%s: added %q label to issue #%d", owner, r, labelName, issueNumber)
		}
	}

	log.Printf("done: %d issues found, %d labeled, %d already labeled", totalFound, totalLabeled, totalSkipped)
}

func fetchRenovateDependencyDashboardIssues(ctx context.Context, client *github.Client, owner, repo string) ([]*github.Issue, error) {
	listOpts := &github.IssueListByRepoOptions{
		State: "open",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var allIssues []*github.Issue
	for {
		issues, resp, err := client.Issues.ListByRepo(ctx, owner, repo, listOpts)
		if err != nil {
			return nil, fmt.Errorf("failed to list issues: %w", err)
		}
		allIssues = append(allIssues, issues...)

		if resp.NextPage == 0 {
			break
		}
		listOpts.Page = resp.NextPage
	}

	var renovateIssues []*github.Issue
	for _, issue := range allIssues {
		if issue.User == nil {
			continue
		}
		title := issue.GetTitle()
		author := issue.GetUser().GetLogin()

		if title == issueTitle && author == renovateBotLogin {
			renovateIssues = append(renovateIssues, issue)
		}
	}

	return renovateIssues, nil
}

func hasLabel(labels []*github.Label, name string) bool {
	for _, label := range labels {
		if strings.EqualFold(label.GetName(), name) {
			return true
		}
	}
	return false
}

func addLabel(ctx context.Context, client *github.Client, owner, repo string, number int, name string) error {
	_, _, err := client.Issues.AddLabelsToIssue(ctx, owner, repo, number, []string{name})
	if err != nil {
		return fmt.Errorf("failed to add label %q: %w", name, err)
	}
	return nil
}

func ensureLabelExists(ctx context.Context, client *github.Client, owner, repo, name, color string) error {
	// Check if the label already exists
	labels, _, err := client.Issues.ListLabels(ctx, owner, repo, &github.ListOptions{PerPage: 100})
	if err != nil {
		return fmt.Errorf("failed to list labels: %w", err)
	}

	for _, label := range labels {
		if strings.EqualFold(label.GetName(), name) {
			// Label exists, update color if needed
			existingColor := strings.ToLower(strings.TrimPrefix(label.GetColor(), "#"))
			if existingColor != strings.ToLower(color) {
				_, _, err := client.Issues.EditLabel(ctx, owner, repo, name, &github.Label{
					Name:  new(name),
					Color: new(color),
				})
				if err != nil {
					return fmt.Errorf("failed to update label color for %q: %w", name, err)
				}
				log.Printf("%s/%s: updated label %q color to %s", owner, repo, name, color)
			}
			return nil
		}
	}

	// Label doesn't exist, create it
	_, _, err = client.Issues.CreateLabel(ctx, owner, repo, &github.Label{
		Name:  new(name),
		Color: new(color),
	})
	if err != nil {
		return fmt.Errorf("failed to create label %q: %w", name, err)
	}
	log.Printf("%s/%s: created label %q with color %s", owner, repo, name, color)
	return nil
}

func listRepos(ctx context.Context, client *github.Client, owner string) ([]string, error) {
	orgOpts := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	var repos []string
	for {
		repoList, resp, err := client.Repositories.ListByOrg(ctx, owner, orgOpts)
		if err != nil {
			// If it's not an org, try listing by user
			userOpts := &github.RepositoryListByUserOptions{
				ListOptions: github.ListOptions{
					PerPage: 100,
					Page:    orgOpts.Page,
				},
			}
			repoList, resp, err = client.Repositories.ListByUser(ctx, owner, userOpts)
			if err != nil {
				return nil, fmt.Errorf("failed to list repositories: %w", err)
			}
			for _, r := range repoList {
				if r.GetArchived() {
					continue
				}
				repos = append(repos, r.GetName())
			}
			if resp.NextPage == 0 {
				break
			}
			orgOpts.Page = resp.NextPage
			continue
		}

		for _, r := range repoList {
			if r.GetArchived() {
				continue
			}
			repos = append(repos, r.GetName())
		}

		if resp.NextPage == 0 {
			break
		}
		orgOpts.Page = resp.NextPage
	}

	return repos, nil
}
