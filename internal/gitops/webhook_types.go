package gitops

import (
	"time"
)

// WebhookEventType represents the type of Git webhook event
type WebhookEventType string

const (
	// WebhookEventPush represents a git push event
	WebhookEventPush WebhookEventType = "push"
	// WebhookEventPullRequest represents a pull request event
	WebhookEventPullRequest WebhookEventType = "pull_request"
	// WebhookEventTag represents a tag event
	WebhookEventTag WebhookEventType = "tag"
	// WebhookEventRelease represents a release event
	WebhookEventRelease WebhookEventType = "release"
)

// WebhookEvent represents a Git webhook event
type WebhookEvent struct {
	// Type is the type of event
	Type WebhookEventType `json:"type"`

	// Repository is the Git repository URL
	Repository string `json:"repository"`

	// Branch is the branch name (for push events)
	Branch string `json:"branch,omitempty"`

	// Tag is the tag name (for tag events)
	Tag string `json:"tag,omitempty"`

	// Commit is the commit SHA
	Commit string `json:"commit,omitempty"`

	// Author is the author of the change
	Author string `json:"author,omitempty"`

	// Message is the commit/tag message
	Message string `json:"message,omitempty"`

	// Timestamp is when the event occurred
	Timestamp time.Time `json:"timestamp"`

	// PullRequest contains PR details
	PullRequest *PullRequest `json:"pull_request,omitempty"`

	// Release contains release details
	Release *Release `json:"release,omitempty"`

	// Changes contains file changes
	Changes []FileChange `json:"changes,omitempty"`
}

// PullRequest represents pull request details
type PullRequest struct {
	// Number is the PR number
	Number int `json:"number"`

	// Title is the PR title
	Title string `json:"title"`

	// Description is the PR description
	Description string `json:"description"`

	// State is the PR state (open, closed, merged)
	State string `json:"state"`

	// Action is the PR action (opened, closed, reopened, synchronize)
	Action string `json:"action"`

	// SourceBranch is the source branch
	SourceBranch string `json:"source_branch"`

	// TargetBranch is the target branch
	TargetBranch string `json:"target_branch"`

	// Author is the PR author
	Author string `json:"author"`

	// Labels are the PR labels
	Labels []string `json:"labels"`

	// Reviewers are the requested reviewers
	Reviewers []string `json:"reviewers"`

	// Approved indicates if the PR is approved
	Approved bool `json:"approved"`

	// MergeCommit is the merge commit SHA (if merged)
	MergeCommit string `json:"merge_commit,omitempty"`
}

// Release represents a release event
type Release struct {
	// Name is the release name
	Name string `json:"name"`

	// Tag is the release tag
	Tag string `json:"tag"`

	// Description is the release description
	Description string `json:"description"`

	// Prerelease indicates if this is a pre-release
	Prerelease bool `json:"prerelease"`

	// Draft indicates if this is a draft
	Draft bool `json:"draft"`

	// Assets are the release assets
	Assets []ReleaseAsset `json:"assets,omitempty"`
}

// ReleaseAsset represents a release asset
type ReleaseAsset struct {
	// Name is the asset name
	Name string `json:"name"`

	// URL is the download URL
	URL string `json:"url"`

	// Size is the asset size in bytes
	Size int64 `json:"size"`

	// ContentType is the MIME type
	ContentType string `json:"content_type"`
}

// FileChange represents a file change in a commit
type FileChange struct {
	// Path is the file path
	Path string `json:"path"`

	// Action is the change action (added, modified, deleted)
	Action string `json:"action"`

	// Additions is the number of lines added
	Additions int `json:"additions"`

	// Deletions is the number of lines deleted
	Deletions int `json:"deletions"`
}

// WebhookProvider represents a Git webhook provider
type WebhookProvider string

const (
	// WebhookProviderGitHub represents GitHub webhooks
	WebhookProviderGitHub WebhookProvider = "github"
	// WebhookProviderGitLab represents GitLab webhooks
	WebhookProviderGitLab WebhookProvider = "gitlab"
	// WebhookProviderBitbucket represents Bitbucket webhooks
	WebhookProviderBitbucket WebhookProvider = "bitbucket"
	// WebhookProviderGitea represents Gitea webhooks
	WebhookProviderGitea WebhookProvider = "gitea"
)

// WebhookConfig represents webhook configuration
type WebhookConfig struct {
	// Provider is the webhook provider
	Provider WebhookProvider `json:"provider"`

	// Secret is the webhook secret for validation
	Secret string `json:"secret"`

	// Events are the events to listen for
	Events []WebhookEventType `json:"events"`

	// Filters are optional event filters
	Filters *WebhookFilters `json:"filters,omitempty"`
}

// WebhookFilters represents webhook event filters
type WebhookFilters struct {
	// Branches to include
	Branches []string `json:"branches,omitempty"`

	// Tags to include (supports patterns)
	Tags []string `json:"tags,omitempty"`

	// Paths to watch (supports patterns)
	Paths []string `json:"paths,omitempty"`

	// ExcludePaths to ignore
	ExcludePaths []string `json:"exclude_paths,omitempty"`

	// Authors to include
	Authors []string `json:"authors,omitempty"`

	// Labels required on PRs
	Labels []string `json:"labels,omitempty"`
}
