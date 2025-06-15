/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package webhook

import "time"

// Common event types

// PushEvent represents a git push event
type PushEvent struct {
	Repository string
	Branch     string
	Commit     string
	Author     string
	Message    string
	Timestamp  string
	Before     string
	After      string
	Commits    []CommitInfo
}

// PullRequestEvent represents a pull/merge request event
type PullRequestEvent struct {
	Repository   string
	Number       int
	Title        string
	State        string
	Action       string
	SourceBranch string
	TargetBranch string
	Author       string
	Merged       bool
	MergeCommit  string
}

// TagEvent represents a tag creation event
type TagEvent struct {
	Repository string
	Tag        string
	Commit     string
	Author     string
	Message    string
}

// CommitInfo represents information about a commit
type CommitInfo struct {
	ID        string
	Message   string
	Author    string
	Timestamp string
}

// Generic webhook event
type GenericWebhookEvent struct {
	Type       string    `json:"type"`
	Repository string    `json:"repository"`
	Branch     string    `json:"branch,omitempty"`
	Tag        string    `json:"tag,omitempty"`
	Commit     string    `json:"commit"`
	Author     string    `json:"author"`
	Message    string    `json:"message"`
	Timestamp  time.Time `json:"timestamp"`
}

// GitHub event types

// GitHubPushEvent represents a GitHub push event
type GitHubPushEvent struct {
	Ref        string `json:"ref"`
	Before     string `json:"before"`
	After      string `json:"after"`
	Repository struct {
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		CloneURL string `json:"clone_url"`
		SSHURL   string `json:"ssh_url"`
	} `json:"repository"`
	Pusher struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"pusher"`
	Sender struct {
		Login string `json:"login"`
	} `json:"sender"`
	HeadCommit struct {
		ID        string `json:"id"`
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`
		Author    struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
	} `json:"head_commit"`
	Commits []struct {
		ID        string `json:"id"`
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`
		Author    struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
	} `json:"commits"`
}

// GitHubPullRequestEvent represents a GitHub pull request event
type GitHubPullRequestEvent struct {
	Action      string `json:"action"`
	Number      int    `json:"number"`
	PullRequest struct {
		ID     int    `json:"id"`
		Number int    `json:"number"`
		State  string `json:"state"`
		Title  string `json:"title"`
		User   struct {
			Login string `json:"login"`
		} `json:"user"`
		Head struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"base"`
		Merged   bool   `json:"merged"`
		MergeCommitSHA string `json:"merge_commit_sha"`
	} `json:"pull_request"`
	Repository struct {
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		CloneURL string `json:"clone_url"`
	} `json:"repository"`
}

// GitHubCreateEvent represents a GitHub create event (for tags)
type GitHubCreateEvent struct {
	Ref     string `json:"ref"`
	RefType string `json:"ref_type"`
	SHA     string `json:"sha"`
	Repository struct {
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		CloneURL string `json:"clone_url"`
	} `json:"repository"`
	Sender struct {
		Login string `json:"login"`
	} `json:"sender"`
}

// GitLab event types

// GitLabPushEvent represents a GitLab push event
type GitLabPushEvent struct {
	ObjectKind string `json:"object_kind"`
	Before     string `json:"before"`
	After      string `json:"after"`
	Ref        string `json:"ref"`
	UserID     int    `json:"user_id"`
	UserName   string `json:"user_name"`
	UserEmail  string `json:"user_email"`
	Project    struct {
		ID              int    `json:"id"`
		Name            string `json:"name"`
		PathWithNamespace string `json:"path_with_namespace"`
		GitHTTPURL      string `json:"git_http_url"`
		GitSSHURL       string `json:"git_ssh_url"`
	} `json:"project"`
	Commits []struct {
		ID        string `json:"id"`
		Message   string `json:"message"`
		Timestamp string `json:"timestamp"`
		Author    struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
	} `json:"commits"`
}

// GitLabMergeRequestEvent represents a GitLab merge request event
type GitLabMergeRequestEvent struct {
	ObjectKind string `json:"object_kind"`
	User       struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		Username string `json:"username"`
		Email    string `json:"email"`
	} `json:"user"`
	Project struct {
		ID              int    `json:"id"`
		Name            string `json:"name"`
		PathWithNamespace string `json:"path_with_namespace"`
		GitHTTPURL      string `json:"git_http_url"`
		GitSSHURL       string `json:"git_ssh_url"`
	} `json:"project"`
	ObjectAttributes struct {
		ID           int    `json:"id"`
		IID          int    `json:"iid"`
		TargetBranch string `json:"target_branch"`
		SourceBranch string `json:"source_branch"`
		State        string `json:"state"`
		Title        string `json:"title"`
		Action       string `json:"action"`
		MergeCommitSHA string `json:"merge_commit_sha"`
	} `json:"object_attributes"`
}

// GitLabTagPushEvent represents a GitLab tag push event
type GitLabTagPushEvent struct {
	ObjectKind string `json:"object_kind"`
	Before     string `json:"before"`
	After      string `json:"after"`
	Ref        string `json:"ref"`
	UserID     int    `json:"user_id"`
	UserName   string `json:"user_name"`
	Project    struct {
		ID              int    `json:"id"`
		Name            string `json:"name"`
		PathWithNamespace string `json:"path_with_namespace"`
		GitHTTPURL      string `json:"git_http_url"`
		GitSSHURL       string `json:"git_ssh_url"`
	} `json:"project"`
}

// Bitbucket event types

// BitbucketPushEvent represents a Bitbucket push event
type BitbucketPushEvent struct {
	Push struct {
		Changes []struct {
			Forced bool `json:"forced"`
			Old    struct {
				Type   string `json:"type"`
				Name   string `json:"name"`
				Target struct {
					Hash string `json:"hash"`
				} `json:"target"`
			} `json:"old"`
			New struct {
				Type   string `json:"type"`
				Name   string `json:"name"`
				Target struct {
					Hash    string `json:"hash"`
					Message string `json:"message"`
					Date    string `json:"date"`
					Author  struct {
						User struct {
							DisplayName string `json:"display_name"`
							UUID        string `json:"uuid"`
						} `json:"user"`
					} `json:"author"`
				} `json:"target"`
			} `json:"new"`
		} `json:"changes"`
	} `json:"push"`
	Repository struct {
		Name  string `json:"name"`
		UUID  string `json:"uuid"`
		Links struct {
			Clone []struct {
				Href string `json:"href"`
				Name string `json:"name"`
			} `json:"clone"`
		} `json:"links"`
	} `json:"repository"`
	Actor struct {
		DisplayName string `json:"display_name"`
		UUID        string `json:"uuid"`
	} `json:"actor"`
}

// BitbucketPullRequestEvent represents a Bitbucket pull request event
type BitbucketPullRequestEvent struct {
	PullRequest struct {
		ID          int    `json:"id"`
		Title       string `json:"title"`
		State       string `json:"state"`
		Author      struct {
			DisplayName string `json:"display_name"`
			UUID        string `json:"uuid"`
		} `json:"author"`
		Source struct {
			Branch struct {
				Name string `json:"name"`
			} `json:"branch"`
			Commit struct {
				Hash string `json:"hash"`
			} `json:"commit"`
		} `json:"source"`
		Destination struct {
			Branch struct {
				Name string `json:"name"`
			} `json:"branch"`
			Commit struct {
				Hash string `json:"hash"`
			} `json:"commit"`
		} `json:"destination"`
		MergeCommit struct {
			Hash string `json:"hash"`
		} `json:"merge_commit"`
	} `json:"pullrequest"`
	Repository struct {
		Name  string `json:"name"`
		UUID  string `json:"uuid"`
		Links struct {
			Clone []struct {
				Href string `json:"href"`
				Name string `json:"name"`
			} `json:"clone"`
		} `json:"links"`
	} `json:"repository"`
}
