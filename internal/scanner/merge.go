// Package scanner provides repository scanning functionality.
//
// The merge subpackage handles combining local and GitHub scan results.
package scanner

import (
	"fmt"
	"time"

	"github.com/alexcatdad/catscan/internal/cache"
	"github.com/alexcatdad/catscan/internal/model"
)

// Merge combines local and GitHub scan results into a unified list of Repo objects.
//
// Local fields populate clone state and git state.
// GitHub fields populate everything else.
// Repos that exist on GitHub but not locally get cloned=false.
// Repos that exist locally but not on GitHub appear with minimal data.
// Lifecycle status is computed during merge.
func Merge(
	localRepos map[string]LocalRepo,
	githubRepos []GitHubRepo,
	scanPath string,
	state cache.RepoState,
	thresholds model.LifecycleThresholds,
) []model.Repo {
	// Build a map of GitHub repos by name for easy lookup
	githubMap := make(map[string]GitHubRepo)
	for _, ghRepo := range githubRepos {
		githubMap[ghRepo.Name] = ghRepo
	}

	// Collect all unique repo names
	allNames := make(map[string]struct{})
	for name := range localRepos {
		allNames[name] = struct{}{}
	}
	for name := range githubMap {
		allNames[name] = struct{}{}
	}

	// Build unified repo list
	var result []model.Repo
	for name := range allNames {
		repo := model.Repo{Name: name}

		// Get GitHub data if available
		ghRepo, hasGitHub := githubMap[name]
		localRepo, hasLocal := localRepos[name]

		if hasGitHub {
			// Identity
			if ghRepo.PrimaryLanguage != nil {
				repo.FullName = fmt.Sprintf("%s/%s", ghRepo.PrimaryLanguage.Name, name)
				repo.Language = ghRepo.PrimaryLanguage.Name
			} else {
				repo.FullName = name
			}
			repo.Visibility = parseVisibility(ghRepo.Visibility)
			repo.Description = ghRepo.Description
			repo.HomepageURL = ghRepo.HomepageURL

			// Extract topic names from nested objects
			if ghRepo.Topics != nil {
				topics := make([]string, 0, len(ghRepo.Topics))
				for _, t := range ghRepo.Topics {
					topics = append(topics, t.Name)
				}
				repo.Topics = topics
			}

			// Parse pushedAt for lifecycle calculation
			if ghRepo.PushedAt != "" {
				if pushTime, err := time.Parse(time.RFC3339, ghRepo.PushedAt); err == nil {
					repo.GitHubLastPush = pushTime
				}
			}

			// Activity data from per-repo GitHub fetches
			repo.OpenPRs = ghRepo.OpenPRs
			repo.ActionsStatus = model.ActionsStatus(ghRepo.ActionsStatus)

			// Completeness info
			repo.Completeness.HasDescription = ghRepo.Description != ""
			repo.Completeness.HasTopics = len(ghRepo.Topics) > 0
			repo.Completeness.HasHomepage = ghRepo.HomepageURL != ""
			if ghRepo.FilePresence != nil {
				repo.Completeness.HasReadme = ghRepo.FilePresence.HasREADME
				repo.Completeness.HasLicense = ghRepo.FilePresence.HasLICENSE
				repo.Completeness.HasClaudeMd = ghRepo.FilePresence.HasCLAUDEmd
				repo.Completeness.HasAgentsMd = ghRepo.FilePresence.HasAGENTSmd
				repo.Completeness.HasProjectJson = ghRepo.FilePresence.HasProjectJson
			}

			// Release info
			if ghRepo.LatestRelease != nil {
				pubTime, _ := time.Parse(time.RFC3339, ghRepo.LatestRelease.PublishedAt)
				repo.LatestRelease = &model.ReleaseInfo{
					TagName:     ghRepo.LatestRelease.TagName,
					PublishedAt: pubTime,
				}

				// Check if this is a new release
				if stateEntry, ok := state[name]; ok && stateEntry != nil {
					repo.NewRelease = stateEntry.LastSeenReleaseTag != ghRepo.LatestRelease.TagName
				} else {
					repo.NewRelease = true
				}
			}

			// Default branch name (for non-cloned repos)
			if !hasLocal && ghRepo.DefaultBranch != nil {
				repo.Branch = ghRepo.DefaultBranch.Name
			}
		}

		// Local data
		if hasLocal {
			repo.Cloned = true
			repo.LocalPath = localRepo.Path
			repo.Branch = localRepo.Branch
			repo.Dirty = localRepo.Dirty
			repo.LocalLastCommit = localRepo.LastCommit
		} else {
			repo.Cloned = false
			repo.LocalPath = fmt.Sprintf("%s/%s", scanPath, name)
		}

		// Compute lifecycle
		repo.Lifecycle = repo.ComputeLifecycle(thresholds)

		result = append(result, repo)
	}

	return result
}

// parseVisibility converts GitHub visibility string to model.Visibility.
func parseVisibility(v string) model.Visibility {
	switch v {
	case "public":
		return model.VisibilityPublic
	case "private":
		return model.VisibilityPrivate
	default:
		return model.VisibilityPrivate
	}
}
