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

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/go-logr/logr"
	"github.com/gorilla/mux"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	gitopsv1beta1 "github.com/gunjanjp/gunj-operator/pkg/gitops/api/v1beta1"
)

// Handler handles Git webhook events
type Handler struct {
	Client         client.Client
	Log            logr.Logger
	EventProcessor EventProcessor
}

// EventProcessor processes webhook events
type EventProcessor interface {
	ProcessPushEvent(ctx context.Context, event *PushEvent) error
	ProcessPullRequestEvent(ctx context.Context, event *PullRequestEvent) error
	ProcessTagEvent(ctx context.Context, event *TagEvent) error
}

// NewHandler creates a new webhook handler
func NewHandler(client client.Client, log logr.Logger, processor EventProcessor) *Handler {
	return &Handler{
		Client:         client,
		Log:            log.WithName("webhook-handler"),
		EventProcessor: processor,
	}
}

// RegisterRoutes registers webhook routes
func (h *Handler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/webhook/github", h.handleGitHubWebhook).Methods("POST")
	router.HandleFunc("/webhook/gitlab", h.handleGitLabWebhook).Methods("POST")
	router.HandleFunc("/webhook/bitbucket", h.handleBitbucketWebhook).Methods("POST")
	router.HandleFunc("/webhook/generic", h.handleGenericWebhook).Methods("POST")
}

// handleGitHubWebhook handles GitHub webhook events
func (h *Handler) handleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	log := h.Log.WithValues("provider", "github")

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error(err, "Failed to read request body")
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Get event type
	eventType := r.Header.Get("X-GitHub-Event")
	if eventType == "" {
		log.Error(nil, "Missing X-GitHub-Event header")
		http.Error(w, "Missing event type", http.StatusBadRequest)
		return
	}

	// Get signature
	signature := r.Header.Get("X-Hub-Signature-256")
	
	// Get webhook secret from query parameter (platform name)
	platformName := r.URL.Query().Get("platform")
	if platformName == "" {
		log.Error(nil, "Missing platform query parameter")
		http.Error(w, "Missing platform", http.StatusBadRequest)
		return
	}

	// Verify signature if provided
	if signature != "" {
		secret, err := h.getWebhookSecret(r.Context(), platformName)
		if err != nil {
			log.Error(err, "Failed to get webhook secret")
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		if !h.verifyGitHubSignature(body, signature, secret) {
			log.Error(nil, "Invalid signature")
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// Process event
	switch eventType {
	case "push":
		var event GitHubPushEvent
		if err := json.Unmarshal(body, &event); err != nil {
			log.Error(err, "Failed to unmarshal push event")
			http.Error(w, "Invalid event payload", http.StatusBadRequest)
			return
		}
		
		pushEvent := h.convertGitHubPushEvent(&event)
		if err := h.EventProcessor.ProcessPushEvent(r.Context(), pushEvent); err != nil {
			log.Error(err, "Failed to process push event")
			http.Error(w, "Failed to process event", http.StatusInternalServerError)
			return
		}

	case "pull_request":
		var event GitHubPullRequestEvent
		if err := json.Unmarshal(body, &event); err != nil {
			log.Error(err, "Failed to unmarshal pull request event")
			http.Error(w, "Invalid event payload", http.StatusBadRequest)
			return
		}
		
		prEvent := h.convertGitHubPullRequestEvent(&event)
		if err := h.EventProcessor.ProcessPullRequestEvent(r.Context(), prEvent); err != nil {
			log.Error(err, "Failed to process pull request event")
			http.Error(w, "Failed to process event", http.StatusInternalServerError)
			return
		}

	case "create":
		var event GitHubCreateEvent
		if err := json.Unmarshal(body, &event); err != nil {
			log.Error(err, "Failed to unmarshal create event")
			http.Error(w, "Invalid event payload", http.StatusBadRequest)
			return
		}
		
		if event.RefType == "tag" {
			tagEvent := h.convertGitHubTagEvent(&event)
			if err := h.EventProcessor.ProcessTagEvent(r.Context(), tagEvent); err != nil {
				log.Error(err, "Failed to process tag event")
				http.Error(w, "Failed to process event", http.StatusInternalServerError)
				return
			}
		}

	default:
		log.Info("Ignoring event type", "type", eventType)
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}

// handleGitLabWebhook handles GitLab webhook events
func (h *Handler) handleGitLabWebhook(w http.ResponseWriter, r *http.Request) {
	log := h.Log.WithValues("provider", "gitlab")

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error(err, "Failed to read request body")
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Get event type
	eventType := r.Header.Get("X-Gitlab-Event")
	if eventType == "" {
		log.Error(nil, "Missing X-Gitlab-Event header")
		http.Error(w, "Missing event type", http.StatusBadRequest)
		return
	}

	// Get token
	token := r.Header.Get("X-Gitlab-Token")
	
	// Get platform name from query parameter
	platformName := r.URL.Query().Get("platform")
	if platformName == "" {
		log.Error(nil, "Missing platform query parameter")
		http.Error(w, "Missing platform", http.StatusBadRequest)
		return
	}

	// Verify token if provided
	if token != "" {
		secret, err := h.getWebhookSecret(r.Context(), platformName)
		if err != nil {
			log.Error(err, "Failed to get webhook secret")
			http.Error(w, "Internal error", http.StatusInternalServerError)
			return
		}

		if token != secret {
			log.Error(nil, "Invalid token")
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}
	}

	// Process event
	switch eventType {
	case "Push Hook":
		var event GitLabPushEvent
		if err := json.Unmarshal(body, &event); err != nil {
			log.Error(err, "Failed to unmarshal push event")
			http.Error(w, "Invalid event payload", http.StatusBadRequest)
			return
		}
		
		pushEvent := h.convertGitLabPushEvent(&event)
		if err := h.EventProcessor.ProcessPushEvent(r.Context(), pushEvent); err != nil {
			log.Error(err, "Failed to process push event")
			http.Error(w, "Failed to process event", http.StatusInternalServerError)
			return
		}

	case "Merge Request Hook":
		var event GitLabMergeRequestEvent
		if err := json.Unmarshal(body, &event); err != nil {
			log.Error(err, "Failed to unmarshal merge request event")
			http.Error(w, "Invalid event payload", http.StatusBadRequest)
			return
		}
		
		prEvent := h.convertGitLabMergeRequestEvent(&event)
		if err := h.EventProcessor.ProcessPullRequestEvent(r.Context(), prEvent); err != nil {
			log.Error(err, "Failed to process merge request event")
			http.Error(w, "Failed to process event", http.StatusInternalServerError)
			return
		}

	case "Tag Push Hook":
		var event GitLabTagPushEvent
		if err := json.Unmarshal(body, &event); err != nil {
			log.Error(err, "Failed to unmarshal tag push event")
			http.Error(w, "Invalid event payload", http.StatusBadRequest)
			return
		}
		
		tagEvent := h.convertGitLabTagEvent(&event)
		if err := h.EventProcessor.ProcessTagEvent(r.Context(), tagEvent); err != nil {
			log.Error(err, "Failed to process tag event")
			http.Error(w, "Failed to process event", http.StatusInternalServerError)
			return
		}

	default:
		log.Info("Ignoring event type", "type", eventType)
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}

// handleBitbucketWebhook handles Bitbucket webhook events
func (h *Handler) handleBitbucketWebhook(w http.ResponseWriter, r *http.Request) {
	log := h.Log.WithValues("provider", "bitbucket")

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error(err, "Failed to read request body")
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Get event type
	eventType := r.Header.Get("X-Event-Key")
	if eventType == "" {
		log.Error(nil, "Missing X-Event-Key header")
		http.Error(w, "Missing event type", http.StatusBadRequest)
		return
	}

	// Process event
	switch eventType {
	case "repo:push":
		var event BitbucketPushEvent
		if err := json.Unmarshal(body, &event); err != nil {
			log.Error(err, "Failed to unmarshal push event")
			http.Error(w, "Invalid event payload", http.StatusBadRequest)
			return
		}
		
		pushEvent := h.convertBitbucketPushEvent(&event)
		if err := h.EventProcessor.ProcessPushEvent(r.Context(), pushEvent); err != nil {
			log.Error(err, "Failed to process push event")
			http.Error(w, "Failed to process event", http.StatusInternalServerError)
			return
		}

	case "pullrequest:created", "pullrequest:updated":
		var event BitbucketPullRequestEvent
		if err := json.Unmarshal(body, &event); err != nil {
			log.Error(err, "Failed to unmarshal pull request event")
			http.Error(w, "Invalid event payload", http.StatusBadRequest)
			return
		}
		
		prEvent := h.convertBitbucketPullRequestEvent(&event)
		if err := h.EventProcessor.ProcessPullRequestEvent(r.Context(), prEvent); err != nil {
			log.Error(err, "Failed to process pull request event")
			http.Error(w, "Failed to process event", http.StatusInternalServerError)
			return
		}

	default:
		log.Info("Ignoring event type", "type", eventType)
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}

// handleGenericWebhook handles generic webhook events
func (h *Handler) handleGenericWebhook(w http.ResponseWriter, r *http.Request) {
	log := h.Log.WithValues("provider", "generic")

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error(err, "Failed to read request body")
		http.Error(w, "Failed to read request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse generic event
	var event GenericWebhookEvent
	if err := json.Unmarshal(body, &event); err != nil {
		log.Error(err, "Failed to unmarshal event")
		http.Error(w, "Invalid event payload", http.StatusBadRequest)
		return
	}

	// Process based on event type
	switch event.Type {
	case "push":
		pushEvent := &PushEvent{
			Repository: event.Repository,
			Branch:     event.Branch,
			Commit:     event.Commit,
			Author:     event.Author,
			Message:    event.Message,
			Timestamp:  event.Timestamp,
		}
		
		if err := h.EventProcessor.ProcessPushEvent(r.Context(), pushEvent); err != nil {
			log.Error(err, "Failed to process push event")
			http.Error(w, "Failed to process event", http.StatusInternalServerError)
			return
		}

	default:
		log.Info("Ignoring event type", "type", event.Type)
	}

	w.WriteHeader(http.StatusOK)
	fmt.Fprint(w, "OK")
}

// verifyGitHubSignature verifies GitHub webhook signature
func (h *Handler) verifyGitHubSignature(payload []byte, signature, secret string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	
	signature = strings.TrimPrefix(signature, "sha256=")
	
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedMAC := mac.Sum(nil)
	expectedSignature := hex.EncodeToString(expectedMAC)
	
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

// getWebhookSecret gets the webhook secret for a platform
func (h *Handler) getWebhookSecret(ctx context.Context, platformName string) (string, error) {
	// This is simplified - in a real implementation, you would:
	// 1. Look up the ObservabilityPlatform by name
	// 2. Get its GitOps configuration
	// 3. Retrieve the webhook secret from the referenced secret
	
	// For now, return a placeholder
	return "webhook-secret", nil
}

// Event conversion functions

func (h *Handler) convertGitHubPushEvent(event *GitHubPushEvent) *PushEvent {
	return &PushEvent{
		Repository: event.Repository.CloneURL,
		Branch:     strings.TrimPrefix(event.Ref, "refs/heads/"),
		Commit:     event.After,
		Author:     event.Pusher.Name,
		Message:    event.HeadCommit.Message,
		Timestamp:  event.HeadCommit.Timestamp,
	}
}

func (h *Handler) convertGitHubPullRequestEvent(event *GitHubPullRequestEvent) *PullRequestEvent {
	return &PullRequestEvent{
		Repository:  event.Repository.CloneURL,
		Number:      event.PullRequest.Number,
		Title:       event.PullRequest.Title,
		State:       event.PullRequest.State,
		Action:      event.Action,
		SourceBranch: event.PullRequest.Head.Ref,
		TargetBranch: event.PullRequest.Base.Ref,
		Author:      event.PullRequest.User.Login,
	}
}

func (h *Handler) convertGitHubTagEvent(event *GitHubCreateEvent) *TagEvent {
	return &TagEvent{
		Repository: event.Repository.CloneURL,
		Tag:        event.Ref,
		Commit:     event.SHA,
		Author:     event.Sender.Login,
	}
}

func (h *Handler) convertGitLabPushEvent(event *GitLabPushEvent) *PushEvent {
	return &PushEvent{
		Repository: event.Project.GitHTTPURL,
		Branch:     strings.TrimPrefix(event.Ref, "refs/heads/"),
		Commit:     event.After,
		Author:     event.UserName,
		Message:    event.Commits[0].Message,
		Timestamp:  event.Commits[0].Timestamp,
	}
}

func (h *Handler) convertGitLabMergeRequestEvent(event *GitLabMergeRequestEvent) *PullRequestEvent {
	return &PullRequestEvent{
		Repository:   event.Project.GitHTTPURL,
		Number:       event.ObjectAttributes.IID,
		Title:        event.ObjectAttributes.Title,
		State:        event.ObjectAttributes.State,
		Action:       event.ObjectAttributes.Action,
		SourceBranch: event.ObjectAttributes.SourceBranch,
		TargetBranch: event.ObjectAttributes.TargetBranch,
		Author:       event.User.Username,
	}
}

func (h *Handler) convertGitLabTagEvent(event *GitLabTagPushEvent) *TagEvent {
	return &TagEvent{
		Repository: event.Project.GitHTTPURL,
		Tag:        strings.TrimPrefix(event.Ref, "refs/tags/"),
		Commit:     event.After,
		Author:     event.UserName,
	}
}

func (h *Handler) convertBitbucketPushEvent(event *BitbucketPushEvent) *PushEvent {
	return &PushEvent{
		Repository: event.Repository.Links.Clone[0].Href,
		Branch:     event.Push.Changes[0].New.Name,
		Commit:     event.Push.Changes[0].New.Target.Hash,
		Author:     event.Actor.DisplayName,
		Message:    event.Push.Changes[0].New.Target.Message,
		Timestamp:  event.Push.Changes[0].New.Target.Date,
	}
}

func (h *Handler) convertBitbucketPullRequestEvent(event *BitbucketPullRequestEvent) *PullRequestEvent {
	return &PullRequestEvent{
		Repository:   event.Repository.Links.Clone[0].Href,
		Number:       event.PullRequest.ID,
		Title:        event.PullRequest.Title,
		State:        event.PullRequest.State,
		Action:       "updated",
		SourceBranch: event.PullRequest.Source.Branch.Name,
		TargetBranch: event.PullRequest.Destination.Branch.Name,
		Author:       event.PullRequest.Author.DisplayName,
	}
}
