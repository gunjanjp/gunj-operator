package handlers

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gunjanjp/gunj-operator/internal/gitops"
)

// WebhookHandler handles Git webhook events
type WebhookHandler struct {
	Client       client.Client
	Log          logr.Logger
	GitOpsManager *gitops.Manager
}

// NewWebhookHandler creates a new webhook handler
func NewWebhookHandler(client client.Client, log logr.Logger, gitOpsManager *gitops.Manager) *WebhookHandler {
	return &WebhookHandler{
		Client:        client,
		Log:           log.WithName("webhook-handler"),
		GitOpsManager: gitOpsManager,
	}
}

// RegisterRoutes registers webhook routes
func (h *WebhookHandler) RegisterRoutes(router *gin.RouterGroup) {
	webhook := router.Group("/webhook")
	{
		webhook.POST("/github", h.handleGitHubWebhook)
		webhook.POST("/gitlab", h.handleGitLabWebhook)
		webhook.POST("/bitbucket", h.handleBitbucketWebhook)
		webhook.POST("/gitea", h.handleGiteaWebhook)
		webhook.POST("/generic", h.handleGenericWebhook)
	}
}

// handleGitHubWebhook handles GitHub webhook events
func (h *WebhookHandler) handleGitHubWebhook(c *gin.Context) {
	log := h.Log.WithValues("provider", "github")
	
	// Verify signature
	signature := c.GetHeader("X-Hub-Signature-256")
	if signature == "" {
		signature = c.GetHeader("X-Hub-Signature")
	}
	
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Error(err, "Failed to read request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}
	
	// Get secret from configuration (in real implementation, this would be per-repository)
	secret := h.getWebhookSecret(c.Query("repository"))
	if secret != "" && !h.verifyGitHubSignature(signature, body, secret) {
		log.Error(nil, "Invalid webhook signature")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}
	
	// Parse event type
	eventType := c.GetHeader("X-GitHub-Event")
	log.V(1).Info("Received GitHub webhook", "event", eventType)
	
	// Convert to generic webhook event
	event, err := h.parseGitHubEvent(eventType, body)
	if err != nil {
		log.Error(err, "Failed to parse GitHub event")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse event"})
		return
	}
	
	// Process event
	if err := h.GitOpsManager.HandleGitWebhook(c.Request.Context(), event); err != nil {
		log.Error(err, "Failed to handle webhook event")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process event"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"status": "accepted"})
}

// handleGitLabWebhook handles GitLab webhook events
func (h *WebhookHandler) handleGitLabWebhook(c *gin.Context) {
	log := h.Log.WithValues("provider", "gitlab")
	
	// Verify token
	token := c.GetHeader("X-Gitlab-Token")
	expectedToken := h.getWebhookSecret(c.Query("repository"))
	if expectedToken != "" && token != expectedToken {
		log.Error(nil, "Invalid webhook token")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
		return
	}
	
	// Parse event type
	eventType := c.GetHeader("X-Gitlab-Event")
	log.V(1).Info("Received GitLab webhook", "event", eventType)
	
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Error(err, "Failed to read request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}
	
	// Convert to generic webhook event
	event, err := h.parseGitLabEvent(eventType, body)
	if err != nil {
		log.Error(err, "Failed to parse GitLab event")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse event"})
		return
	}
	
	// Process event
	if err := h.GitOpsManager.HandleGitWebhook(c.Request.Context(), event); err != nil {
		log.Error(err, "Failed to handle webhook event")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process event"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"status": "accepted"})
}

// handleBitbucketWebhook handles Bitbucket webhook events
func (h *WebhookHandler) handleBitbucketWebhook(c *gin.Context) {
	log := h.Log.WithValues("provider", "bitbucket")
	
	// Parse event type
	eventType := c.GetHeader("X-Event-Key")
	log.V(1).Info("Received Bitbucket webhook", "event", eventType)
	
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Error(err, "Failed to read request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}
	
	// Convert to generic webhook event
	event, err := h.parseBitbucketEvent(eventType, body)
	if err != nil {
		log.Error(err, "Failed to parse Bitbucket event")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse event"})
		return
	}
	
	// Process event
	if err := h.GitOpsManager.HandleGitWebhook(c.Request.Context(), event); err != nil {
		log.Error(err, "Failed to handle webhook event")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process event"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"status": "accepted"})
}

// handleGiteaWebhook handles Gitea webhook events
func (h *WebhookHandler) handleGiteaWebhook(c *gin.Context) {
	log := h.Log.WithValues("provider", "gitea")
	
	// Verify signature
	signature := c.GetHeader("X-Gitea-Signature")
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		log.Error(err, "Failed to read request body")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
		return
	}
	
	secret := h.getWebhookSecret(c.Query("repository"))
	if secret != "" && !h.verifyGiteaSignature(signature, body, secret) {
		log.Error(nil, "Invalid webhook signature")
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid signature"})
		return
	}
	
	// Parse event type
	eventType := c.GetHeader("X-Gitea-Event")
	log.V(1).Info("Received Gitea webhook", "event", eventType)
	
	// Convert to generic webhook event
	event, err := h.parseGiteaEvent(eventType, body)
	if err != nil {
		log.Error(err, "Failed to parse Gitea event")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Failed to parse event"})
		return
	}
	
	// Process event
	if err := h.GitOpsManager.HandleGitWebhook(c.Request.Context(), event); err != nil {
		log.Error(err, "Failed to handle webhook event")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process event"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"status": "accepted"})
}

// handleGenericWebhook handles generic webhook events
func (h *WebhookHandler) handleGenericWebhook(c *gin.Context) {
	log := h.Log.WithValues("provider", "generic")
	
	var event gitops.WebhookEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		log.Error(err, "Failed to parse generic webhook")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid webhook format"})
		return
	}
	
	log.V(1).Info("Received generic webhook", "event", event.Type)
	
	// Process event
	if err := h.GitOpsManager.HandleGitWebhook(c.Request.Context(), &event); err != nil {
		log.Error(err, "Failed to handle webhook event")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process event"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"status": "accepted"})
}

// verifyGitHubSignature verifies GitHub webhook signature
func (h *WebhookHandler) verifyGitHubSignature(signature string, body []byte, secret string) bool {
	if signature == "" {
		return false
	}
	
	// GitHub uses HMAC-SHA256 or HMAC-SHA1
	if strings.HasPrefix(signature, "sha256=") {
		expected := h.computeHMACSHA256(body, secret)
		return hmac.Equal([]byte(signature[7:]), []byte(expected))
	} else if strings.HasPrefix(signature, "sha1=") {
		expected := h.computeHMACSHA1(body, secret)
		return hmac.Equal([]byte(signature[5:]), []byte(expected))
	}
	
	return false
}

// verifyGiteaSignature verifies Gitea webhook signature
func (h *WebhookHandler) verifyGiteaSignature(signature string, body []byte, secret string) bool {
	expected := h.computeHMACSHA256(body, secret)
	return hmac.Equal([]byte(signature), []byte(expected))
}

// computeHMACSHA256 computes HMAC-SHA256
func (h *WebhookHandler) computeHMACSHA256(data []byte, secret string) string {
	h256 := hmac.New(sha256.New, []byte(secret))
	h256.Write(data)
	return hex.EncodeToString(h256.Sum(nil))
}

// computeHMACSHA1 computes HMAC-SHA1
func (h *WebhookHandler) computeHMACSHA1(data []byte, secret string) string {
	h1 := hmac.New(sha1.New, []byte(secret))
	h1.Write(data)
	return hex.EncodeToString(h1.Sum(nil))
}

// getWebhookSecret gets the webhook secret for a repository
func (h *WebhookHandler) getWebhookSecret(repository string) string {
	// In a real implementation, this would look up the secret for the specific repository
	// from a ConfigMap or Secret
	return ""
}

// parseGitHubEvent parses a GitHub webhook event
func (h *WebhookHandler) parseGitHubEvent(eventType string, body []byte) (*gitops.WebhookEvent, error) {
	var event gitops.WebhookEvent
	
	switch eventType {
	case "push":
		var pushEvent GitHubPushEvent
		if err := json.Unmarshal(body, &pushEvent); err != nil {
			return nil, err
		}
		
		event.Type = gitops.WebhookEventPush
		event.Repository = pushEvent.Repository.CloneURL
		event.Branch = strings.TrimPrefix(pushEvent.Ref, "refs/heads/")
		event.Commit = pushEvent.After
		event.Author = pushEvent.Pusher.Name
		if len(pushEvent.Commits) > 0 {
			event.Message = pushEvent.Commits[0].Message
		}
		
	case "pull_request":
		var prEvent GitHubPullRequestEvent
		if err := json.Unmarshal(body, &prEvent); err != nil {
			return nil, err
		}
		
		event.Type = gitops.WebhookEventPullRequest
		event.Repository = prEvent.Repository.CloneURL
		event.PullRequest = &gitops.PullRequest{
			Number:       prEvent.Number,
			Title:        prEvent.PullRequest.Title,
			Description:  prEvent.PullRequest.Body,
			State:        prEvent.PullRequest.State,
			Action:       prEvent.Action,
			SourceBranch: prEvent.PullRequest.Head.Ref,
			TargetBranch: prEvent.PullRequest.Base.Ref,
			Author:       prEvent.PullRequest.User.Login,
		}
		
	case "create":
		var createEvent GitHubCreateEvent
		if err := json.Unmarshal(body, &createEvent); err != nil {
			return nil, err
		}
		
		if createEvent.RefType == "tag" {
			event.Type = gitops.WebhookEventTag
			event.Repository = createEvent.Repository.CloneURL
			event.Tag = createEvent.Ref
		}
		
	case "release":
		var releaseEvent GitHubReleaseEvent
		if err := json.Unmarshal(body, &releaseEvent); err != nil {
			return nil, err
		}
		
		event.Type = gitops.WebhookEventRelease
		event.Repository = releaseEvent.Repository.CloneURL
		event.Release = &gitops.Release{
			Name:        releaseEvent.Release.Name,
			Tag:         releaseEvent.Release.TagName,
			Description: releaseEvent.Release.Body,
			Prerelease:  releaseEvent.Release.Prerelease,
			Draft:       releaseEvent.Release.Draft,
		}
	}
	
	return &event, nil
}

// parseGitLabEvent parses a GitLab webhook event
func (h *WebhookHandler) parseGitLabEvent(eventType string, body []byte) (*gitops.WebhookEvent, error) {
	var event gitops.WebhookEvent
	
	switch eventType {
	case "Push Hook":
		var pushEvent GitLabPushEvent
		if err := json.Unmarshal(body, &pushEvent); err != nil {
			return nil, err
		}
		
		event.Type = gitops.WebhookEventPush
		event.Repository = pushEvent.Project.GitHTTPURL
		event.Branch = strings.TrimPrefix(pushEvent.Ref, "refs/heads/")
		event.Commit = pushEvent.After
		event.Author = pushEvent.UserName
		if len(pushEvent.Commits) > 0 {
			event.Message = pushEvent.Commits[0].Message
		}
		
	case "Merge Request Hook":
		var mrEvent GitLabMergeRequestEvent
		if err := json.Unmarshal(body, &mrEvent); err != nil {
			return nil, err
		}
		
		event.Type = gitops.WebhookEventPullRequest
		event.Repository = mrEvent.Project.GitHTTPURL
		event.PullRequest = &gitops.PullRequest{
			Number:       mrEvent.ObjectAttributes.IID,
			Title:        mrEvent.ObjectAttributes.Title,
			Description:  mrEvent.ObjectAttributes.Description,
			State:        mrEvent.ObjectAttributes.State,
			Action:       mrEvent.ObjectAttributes.Action,
			SourceBranch: mrEvent.ObjectAttributes.SourceBranch,
			TargetBranch: mrEvent.ObjectAttributes.TargetBranch,
			Author:       mrEvent.User.Username,
		}
		
	case "Tag Push Hook":
		var tagEvent GitLabTagPushEvent
		if err := json.Unmarshal(body, &tagEvent); err != nil {
			return nil, err
		}
		
		event.Type = gitops.WebhookEventTag
		event.Repository = tagEvent.Project.GitHTTPURL
		event.Tag = strings.TrimPrefix(tagEvent.Ref, "refs/tags/")
	}
	
	return &event, nil
}

// parseBitbucketEvent parses a Bitbucket webhook event
func (h *WebhookHandler) parseBitbucketEvent(eventType string, body []byte) (*gitops.WebhookEvent, error) {
	var event gitops.WebhookEvent
	
	switch eventType {
	case "repo:push":
		var pushEvent BitbucketPushEvent
		if err := json.Unmarshal(body, &pushEvent); err != nil {
			return nil, err
		}
		
		if len(pushEvent.Push.Changes) > 0 {
			change := pushEvent.Push.Changes[0]
			event.Type = gitops.WebhookEventPush
			event.Repository = pushEvent.Repository.Links.Clone[0].Href
			if change.New.Type == "branch" {
				event.Branch = change.New.Name
			}
			event.Commit = change.New.Target.Hash
			event.Author = pushEvent.Actor.DisplayName
			event.Message = change.New.Target.Message
		}
		
	case "pullrequest:created", "pullrequest:updated":
		var prEvent BitbucketPullRequestEvent
		if err := json.Unmarshal(body, &prEvent); err != nil {
			return nil, err
		}
		
		event.Type = gitops.WebhookEventPullRequest
		event.Repository = prEvent.Repository.Links.Clone[0].Href
		event.PullRequest = &gitops.PullRequest{
			Number:       prEvent.PullRequest.ID,
			Title:        prEvent.PullRequest.Title,
			Description:  prEvent.PullRequest.Description,
			State:        prEvent.PullRequest.State,
			SourceBranch: prEvent.PullRequest.Source.Branch.Name,
			TargetBranch: prEvent.PullRequest.Destination.Branch.Name,
			Author:       prEvent.PullRequest.Author.DisplayName,
		}
	}
	
	return &event, nil
}

// parseGiteaEvent parses a Gitea webhook event
func (h *WebhookHandler) parseGiteaEvent(eventType string, body []byte) (*gitops.WebhookEvent, error) {
	// Gitea events are very similar to GitHub events
	return h.parseGitHubEvent(eventType, body)
}

// GitHub webhook event structures
type GitHubPushEvent struct {
	Ref        string `json:"ref"`
	Before     string `json:"before"`
	After      string `json:"after"`
	Repository struct {
		CloneURL string `json:"clone_url"`
	} `json:"repository"`
	Pusher struct {
		Name string `json:"name"`
	} `json:"pusher"`
	Commits []struct {
		Message string `json:"message"`
	} `json:"commits"`
}

type GitHubPullRequestEvent struct {
	Action      string `json:"action"`
	Number      int    `json:"number"`
	PullRequest struct {
		Title string `json:"title"`
		Body  string `json:"body"`
		State string `json:"state"`
		Head  struct {
			Ref string `json:"ref"`
		} `json:"head"`
		Base struct {
			Ref string `json:"ref"`
		} `json:"base"`
		User struct {
			Login string `json:"login"`
		} `json:"user"`
	} `json:"pull_request"`
	Repository struct {
		CloneURL string `json:"clone_url"`
	} `json:"repository"`
}

type GitHubCreateEvent struct {
	Ref      string `json:"ref"`
	RefType  string `json:"ref_type"`
	Repository struct {
		CloneURL string `json:"clone_url"`
	} `json:"repository"`
}

type GitHubReleaseEvent struct {
	Action  string `json:"action"`
	Release struct {
		Name       string `json:"name"`
		TagName    string `json:"tag_name"`
		Body       string `json:"body"`
		Prerelease bool   `json:"prerelease"`
		Draft      bool   `json:"draft"`
	} `json:"release"`
	Repository struct {
		CloneURL string `json:"clone_url"`
	} `json:"repository"`
}

// GitLab webhook event structures
type GitLabPushEvent struct {
	Ref      string `json:"ref"`
	Before   string `json:"before"`
	After    string `json:"after"`
	UserName string `json:"user_name"`
	Project  struct {
		GitHTTPURL string `json:"git_http_url"`
	} `json:"project"`
	Commits []struct {
		Message string `json:"message"`
	} `json:"commits"`
}

type GitLabMergeRequestEvent struct {
	User struct {
		Username string `json:"username"`
	} `json:"user"`
	Project struct {
		GitHTTPURL string `json:"git_http_url"`
	} `json:"project"`
	ObjectAttributes struct {
		IID          int    `json:"iid"`
		Title        string `json:"title"`
		Description  string `json:"description"`
		State        string `json:"state"`
		Action       string `json:"action"`
		SourceBranch string `json:"source_branch"`
		TargetBranch string `json:"target_branch"`
	} `json:"object_attributes"`
}

type GitLabTagPushEvent struct {
	Ref     string `json:"ref"`
	Project struct {
		GitHTTPURL string `json:"git_http_url"`
	} `json:"project"`
}

// Bitbucket webhook event structures
type BitbucketPushEvent struct {
	Push struct {
		Changes []struct {
			New struct {
				Type   string `json:"type"`
				Name   string `json:"name"`
				Target struct {
					Hash    string `json:"hash"`
					Message string `json:"message"`
				} `json:"target"`
			} `json:"new"`
		} `json:"changes"`
	} `json:"push"`
	Repository struct {
		Links struct {
			Clone []struct {
				Href string `json:"href"`
			} `json:"clone"`
		} `json:"links"`
	} `json:"repository"`
	Actor struct {
		DisplayName string `json:"display_name"`
	} `json:"actor"`
}

type BitbucketPullRequestEvent struct {
	PullRequest struct {
		ID          int    `json:"id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		State       string `json:"state"`
		Author      struct {
			DisplayName string `json:"display_name"`
		} `json:"author"`
		Source struct {
			Branch struct {
				Name string `json:"name"`
			} `json:"branch"`
		} `json:"source"`
		Destination struct {
			Branch struct {
				Name string `json:"name"`
			} `json:"branch"`
		} `json:"destination"`
	} `json:"pullrequest"`
	Repository struct {
		Links struct {
			Clone []struct {
				Href string `json:"href"`
			} `json:"clone"`
		} `json:"links"`
	} `json:"repository"`
}
