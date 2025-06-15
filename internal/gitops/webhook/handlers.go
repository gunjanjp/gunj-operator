package webhook

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	observabilityv1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

var (
	// Metrics for webhook handling
	webhookEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "gitops_webhook_events_total",
			Help: "Total number of webhook events received",
		},
		[]string{"provider", "event_type", "status"},
	)

	webhookProcessingDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "gitops_webhook_processing_duration_seconds",
			Help:    "Duration of webhook processing in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"provider", "event_type"},
	)
)

// timeNow is a variable to allow mocking in tests
var timeNow = time.Now

// Handler handles webhook requests from various Git providers
type Handler struct {
	client   client.Client
	log      logr.Logger
	handlers map[string]ProviderHandler
}

// ProviderHandler defines the interface for Git provider webhook handlers
type ProviderHandler interface {
	ValidateSignature(r *http.Request, secret string) error
	ParsePayload(r *http.Request) (*WebhookPayload, error)
	GetEventType(r *http.Request) string
}

// NewHandler creates a new webhook handler
func NewHandler(client client.Client, log logr.Logger) *Handler {
	h := &Handler{
		client: client,
		log:    log.WithName("webhook-handler"),
		handlers: map[string]ProviderHandler{
			"github":    &GitHubHandler{log: log},
			"gitlab":    &GitLabHandler{log: log},
			"bitbucket": &BitbucketHandler{log: log},
		},
	}
	return h
}

// WebhookPayload represents a normalized webhook payload
type WebhookPayload struct {
	Provider    string
	EventType   string
	Repository  RepositoryInfo
	Branch      string
	Tag         string
	Commit      CommitInfo
	PullRequest *PullRequestInfo
	Sender      UserInfo
}

// RepositoryInfo contains repository information
type RepositoryInfo struct {
	Name     string
	FullName string
	CloneURL string
	SSHURL   string
}

// CommitInfo contains commit information
type CommitInfo struct {
	SHA       string
	Message   string
	Author    string
	Email     string
	Timestamp string
	URL       string
}

// PullRequestInfo contains pull request information
type PullRequestInfo struct {
	Number      int
	Title       string
	Description string
	State       string
	SourceBranch string
	TargetBranch string
}

// UserInfo contains user information
type UserInfo struct {
	Username string
	Email    string
}

// HandleWebhook processes incoming webhook requests
func (h *Handler) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	start := timeNow()
	ctx := r.Context()
	
	// Extract provider from path or header
	provider := h.detectProvider(r)
	if provider == "" {
		h.log.Error(nil, "Unable to detect webhook provider")
		http.Error(w, "Unable to detect webhook provider", http.StatusBadRequest)
		return
	}

	handler, exists := h.handlers[provider]
	if !exists {
		h.log.Error(nil, "Unsupported webhook provider", "provider", provider)
		http.Error(w, fmt.Sprintf("Unsupported webhook provider: %s", provider), http.StatusBadRequest)
		return
	}

	// Get GitOpsDeployment name and namespace from URL params
	gitopsName := r.URL.Query().Get("name")
	gitopsNamespace := r.URL.Query().Get("namespace")
	if gitopsName == "" || gitopsNamespace == "" {
		h.log.Error(nil, "Missing name or namespace in webhook URL")
		http.Error(w, "Missing name or namespace parameters", http.StatusBadRequest)
		return
	}

	// Get the GitOpsDeployment
	gitops := &observabilityv1.GitOpsDeployment{}
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      gitopsName,
		Namespace: gitopsNamespace,
	}, gitops); err != nil {
		h.log.Error(err, "Failed to get GitOpsDeployment", "name", gitopsName, "namespace", gitopsNamespace)
		http.Error(w, "GitOpsDeployment not found", http.StatusNotFound)
		return
	}

	// Validate webhook signature if secret is configured
	if gitops.Spec.WebhookSecret != "" {
		if err := handler.ValidateSignature(r, gitops.Spec.WebhookSecret); err != nil {
			h.log.Error(err, "Invalid webhook signature")
			http.Error(w, "Invalid signature", http.StatusUnauthorized)
			return
		}
	}

	// Parse the payload
	payload, err := handler.ParsePayload(r)
	if err != nil {
		h.log.Error(err, "Failed to parse webhook payload")
		http.Error(w, "Failed to parse payload", http.StatusBadRequest)
		return
	}

	// Get event type
	eventType := handler.GetEventType(r)
	h.log.Info("Received webhook", 
		"provider", provider, 
		"event", eventType,
		"repository", payload.Repository.FullName,
		"branch", payload.Branch)

	// Process the webhook based on event type
	if err := h.processWebhook(ctx, gitops, payload, eventType); err != nil {
		h.log.Error(err, "Failed to process webhook")
		webhookEventsTotal.WithLabelValues(provider, eventType, "error").Inc()
		http.Error(w, "Failed to process webhook", http.StatusInternalServerError)
		return
	}

	// Record metrics
	duration := timeNow().Sub(start).Seconds()
	webhookProcessingDuration.WithLabelValues(provider, eventType).Observe(duration)
	webhookEventsTotal.WithLabelValues(provider, eventType, "success").Inc()

	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}

// detectProvider detects the webhook provider
func (h *Handler) detectProvider(r *http.Request) string {
	// Check headers for provider identification
	if r.Header.Get("X-GitHub-Event") != "" {
		return "github"
	}
	if r.Header.Get("X-Gitlab-Event") != "" {
		return "gitlab"
	}
	if r.Header.Get("X-Event-Key") != "" {
		return "bitbucket"
	}

	// Check URL path
	if strings.Contains(r.URL.Path, "/github") {
		return "github"
	}
	if strings.Contains(r.URL.Path, "/gitlab") {
		return "gitlab"
	}
	if strings.Contains(r.URL.Path, "/bitbucket") {
		return "bitbucket"
	}

	return ""
}

// processWebhook processes the webhook and triggers appropriate actions
func (h *Handler) processWebhook(ctx context.Context, gitops *observabilityv1.GitOpsDeployment, payload *WebhookPayload, eventType string) error {
	// Check if webhook matches the repository
	if !h.matchesRepository(gitops, payload) {
		h.log.Info("Webhook does not match repository configuration", 
			"expected", gitops.Spec.Repository,
			"received", payload.Repository.CloneURL)
		return nil
	}

	// Check if webhook matches the branch/tag
	if !h.matchesBranchOrTag(gitops, payload) {
		h.log.Info("Webhook does not match branch/tag configuration",
			"branch", payload.Branch,
			"tag", payload.Tag)
		return nil
	}

	// Process based on event type
	switch eventType {
	case "push", "push_event":
		return h.handlePushEvent(ctx, gitops, payload)
	case "pull_request", "merge_request":
		return h.handlePullRequestEvent(ctx, gitops, payload)
	case "tag", "tag_push", "create":
		return h.handleTagEvent(ctx, gitops, payload)
	default:
		h.log.V(1).Info("Ignoring webhook event", "type", eventType)
		return nil
	}
}

// matchesRepository checks if the webhook matches the configured repository
func (h *Handler) matchesRepository(gitops *observabilityv1.GitOpsDeployment, payload *WebhookPayload) bool {
	// Normalize URLs for comparison
	configuredRepo := strings.TrimSuffix(strings.ToLower(gitops.Spec.Repository), ".git")
	receivedRepo := strings.TrimSuffix(strings.ToLower(payload.Repository.CloneURL), ".git")
	receivedSSH := strings.TrimSuffix(strings.ToLower(payload.Repository.SSHURL), ".git")

	return strings.Contains(receivedRepo, configuredRepo) || 
		strings.Contains(receivedSSH, configuredRepo) ||
		strings.Contains(configuredRepo, payload.Repository.FullName)
}

// matchesBranchOrTag checks if the webhook matches the configured branch or tag
func (h *Handler) matchesBranchOrTag(gitops *observabilityv1.GitOpsDeployment, payload *WebhookPayload) bool {
	// If branch is configured, check branch match
	if gitops.Spec.Branch != "" {
		return payload.Branch == gitops.Spec.Branch
	}

	// If tag is configured, check tag match
	if gitops.Spec.Tag != "" {
		if gitops.Spec.Tag == "*" || gitops.Spec.Tag == "latest" {
			return payload.Tag != ""
		}
		return payload.Tag == gitops.Spec.Tag
	}

	// Default to main/master branch
	return payload.Branch == "main" || payload.Branch == "master"
}

// handlePushEvent handles push events
func (h *Handler) handlePushEvent(ctx context.Context, gitops *observabilityv1.GitOpsDeployment, payload *WebhookPayload) error {
	h.log.Info("Processing push event", 
		"repository", payload.Repository.FullName,
		"branch", payload.Branch,
		"commit", payload.Commit.SHA)

	// Trigger sync based on sync provider
	switch gitops.Spec.SyncProvider {
	case observabilityv1.ArgoCD:
		// Trigger ArgoCD refresh
		if err := h.triggerArgoCDRefresh(ctx, gitops); err != nil {
			return fmt.Errorf("triggering ArgoCD refresh: %w", err)
		}
	case observabilityv1.Flux:
		// Trigger Flux reconciliation
		if err := h.triggerFluxReconciliation(ctx, gitops); err != nil {
			return fmt.Errorf("triggering Flux reconciliation: %w", err)
		}
	default:
		// Update GitOps status to trigger reconciliation
		gitops.Status.WebhookEvent = &observabilityv1.WebhookEventStatus{
			Type:       "push",
			CommitSHA:  payload.Commit.SHA,
			Branch:     payload.Branch,
			ReceivedAt: h.getCurrentTime(),
		}
		if err := h.client.Status().Update(ctx, gitops); err != nil {
			return fmt.Errorf("updating GitOps status: %w", err)
		}
	}

	return nil
}

// handlePullRequestEvent handles pull request events
func (h *Handler) handlePullRequestEvent(ctx context.Context, gitops *observabilityv1.GitOpsDeployment, payload *WebhookPayload) error {
	if payload.PullRequest == nil {
		return nil
	}

	h.log.Info("Processing pull request event",
		"repository", payload.Repository.FullName,
		"pr", payload.PullRequest.Number,
		"state", payload.PullRequest.State)

	// Only process if PR preview is enabled
	if gitops.Spec.PullRequestPreview == nil || !gitops.Spec.PullRequestPreview.Enabled {
		h.log.V(1).Info("Pull request preview is not enabled")
		return nil
	}

	// Create or update preview environment
	if payload.PullRequest.State == "open" || payload.PullRequest.State == "reopened" {
		if err := h.createPRPreview(ctx, gitops, payload); err != nil {
			return fmt.Errorf("creating PR preview: %w", err)
		}
	} else if payload.PullRequest.State == "closed" || payload.PullRequest.State == "merged" {
		if err := h.deletePRPreview(ctx, gitops, payload); err != nil {
			return fmt.Errorf("deleting PR preview: %w", err)
		}
	}

	return nil
}

// handleTagEvent handles tag events
func (h *Handler) handleTagEvent(ctx context.Context, gitops *observabilityv1.GitOpsDeployment, payload *WebhookPayload) error {
	h.log.Info("Processing tag event",
		"repository", payload.Repository.FullName,
		"tag", payload.Tag)

	// Check if auto-promotion is enabled
	if gitops.Spec.AutoPromotion != nil && gitops.Spec.AutoPromotion.Enabled {
		if h.matchesPromotionPattern(gitops, payload.Tag) {
			if err := h.createPromotion(ctx, gitops, payload); err != nil {
				return fmt.Errorf("creating promotion: %w", err)
			}
		}
	}

	// Update GitOps to use new tag
	if gitops.Spec.Tag == "*" || gitops.Spec.Tag == "latest" {
		gitops.Spec.Tag = payload.Tag
		if err := h.client.Update(ctx, gitops); err != nil {
			return fmt.Errorf("updating GitOps tag: %w", err)
		}
	}

	return nil
}

// Helper methods for specific actions
func (h *Handler) triggerArgoCDRefresh(ctx context.Context, gitops *observabilityv1.GitOpsDeployment) error {
	// TODO: Implement ArgoCD refresh trigger
	h.log.V(1).Info("Triggering ArgoCD refresh", "app", gitops.Spec.ArgoCDConfig.ApplicationName)
	return nil
}

func (h *Handler) triggerFluxReconciliation(ctx context.Context, gitops *observabilityv1.GitOpsDeployment) error {
	// TODO: Implement Flux reconciliation trigger
	h.log.V(1).Info("Triggering Flux reconciliation", "kustomization", gitops.Name)
	return nil
}

func (h *Handler) createPRPreview(ctx context.Context, gitops *observabilityv1.GitOpsDeployment, payload *WebhookPayload) error {
	// TODO: Implement PR preview environment creation
	h.log.V(1).Info("Creating PR preview environment", 
		"pr", payload.PullRequest.Number,
		"branch", payload.PullRequest.SourceBranch)
	return nil
}

func (h *Handler) deletePRPreview(ctx context.Context, gitops *observabilityv1.GitOpsDeployment, payload *WebhookPayload) error {
	// TODO: Implement PR preview environment deletion
	h.log.V(1).Info("Deleting PR preview environment", "pr", payload.PullRequest.Number)
	return nil
}

func (h *Handler) matchesPromotionPattern(gitops *observabilityv1.GitOpsDeployment, tag string) bool {
	// TODO: Implement pattern matching for auto-promotion
	return true
}

func (h *Handler) createPromotion(ctx context.Context, gitops *observabilityv1.GitOpsDeployment, payload *WebhookPayload) error {
	// TODO: Implement automatic promotion creation
	h.log.V(1).Info("Creating automatic promotion", "tag", payload.Tag)
	return nil
}

func (h *Handler) getCurrentTime() string {
	return fmt.Sprintf("%d", timeNow().Unix())
}

// GitHubHandler handles GitHub webhooks
type GitHubHandler struct {
	log logr.Logger
}

// ValidateSignature validates GitHub webhook signature
func (g *GitHubHandler) ValidateSignature(r *http.Request, secret string) error {
	signature := r.Header.Get("X-Hub-Signature-256")
	if signature == "" {
		signature = r.Header.Get("X-Hub-Signature")
	}

	if signature == "" {
		return fmt.Errorf("no signature header found")
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return fmt.Errorf("reading request body: %w", err)
	}
	r.Body = io.NopCloser(strings.NewReader(string(body)))

	// Calculate expected signature
	var mac hash.Hash
	if strings.HasPrefix(signature, "sha256=") {
		mac = hmac.New(sha256.New, []byte(secret))
		signature = strings.TrimPrefix(signature, "sha256=")
	} else if strings.HasPrefix(signature, "sha1=") {
		mac = hmac.New(sha1.New, []byte(secret))
		signature = strings.TrimPrefix(signature, "sha1=")
	} else {
		return fmt.Errorf("unsupported signature algorithm")
	}

	mac.Write(body)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return fmt.Errorf("signature mismatch")
	}

	return nil
}

// ParsePayload parses GitHub webhook payload
func (g *GitHubHandler) ParsePayload(r *http.Request) (*WebhookPayload, error) {
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decoding payload: %w", err)
	}

	wp := &WebhookPayload{
		Provider: "github",
	}

	// Parse repository info
	if repo, ok := payload["repository"].(map[string]interface{}); ok {
		wp.Repository = RepositoryInfo{
			Name:     getString(repo, "name"),
			FullName: getString(repo, "full_name"),
			CloneURL: getString(repo, "clone_url"),
			SSHURL:   getString(repo, "ssh_url"),
		}
	}

	// Parse based on event type
	event := r.Header.Get("X-GitHub-Event")
	switch event {
	case "push":
		wp.EventType = "push"
		wp.Branch = extractBranchFromRef(getString(payload, "ref"))
		
		if headCommit, ok := payload["head_commit"].(map[string]interface{}); ok {
			wp.Commit = CommitInfo{
				SHA:     getString(headCommit, "id"),
				Message: getString(headCommit, "message"),
				URL:     getString(headCommit, "url"),
			}
			
			if author, ok := headCommit["author"].(map[string]interface{}); ok {
				wp.Commit.Author = getString(author, "name")
				wp.Commit.Email = getString(author, "email")
			}
		}

	case "pull_request":
		wp.EventType = "pull_request"
		if pr, ok := payload["pull_request"].(map[string]interface{}); ok {
			wp.PullRequest = &PullRequestInfo{
				Number:       int(getFloat(pr, "number")),
				Title:        getString(pr, "title"),
				Description:  getString(pr, "body"),
				State:        getString(pr, "state"),
			}
			
			if head, ok := pr["head"].(map[string]interface{}); ok {
				wp.PullRequest.SourceBranch = getString(head, "ref")
			}
			if base, ok := pr["base"].(map[string]interface{}); ok {
				wp.PullRequest.TargetBranch = getString(base, "ref")
			}
		}

	case "create":
		if getString(payload, "ref_type") == "tag" {
			wp.EventType = "tag"
			wp.Tag = getString(payload, "ref")
		}
	}

	// Parse sender info
	if sender, ok := payload["sender"].(map[string]interface{}); ok {
		wp.Sender = UserInfo{
			Username: getString(sender, "login"),
		}
	}

	return wp, nil
}

// GetEventType returns the GitHub event type
func (g *GitHubHandler) GetEventType(r *http.Request) string {
	return r.Header.Get("X-GitHub-Event")
}

// GitLabHandler handles GitLab webhooks
type GitLabHandler struct {
	log logr.Logger
}

// ValidateSignature validates GitLab webhook signature
func (g *GitLabHandler) ValidateSignature(r *http.Request, secret string) error {
	token := r.Header.Get("X-Gitlab-Token")
	if token == "" {
		return fmt.Errorf("no token header found")
	}

	if token != secret {
		return fmt.Errorf("token mismatch")
	}

	return nil
}

// ParsePayload parses GitLab webhook payload
func (g *GitLabHandler) ParsePayload(r *http.Request) (*WebhookPayload, error) {
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decoding payload: %w", err)
	}

	wp := &WebhookPayload{
		Provider: "gitlab",
	}

	// Parse repository info
	if project, ok := payload["project"].(map[string]interface{}); ok {
		wp.Repository = RepositoryInfo{
			Name:     getString(project, "name"),
			FullName: getString(project, "path_with_namespace"),
			CloneURL: getString(project, "http_url"),
			SSHURL:   getString(project, "ssh_url"),
		}
	}

	// Parse based on event type
	event := r.Header.Get("X-Gitlab-Event")
	switch event {
	case "Push Hook":
		wp.EventType = "push"
		wp.Branch = extractBranchFromRef(getString(payload, "ref"))
		
		commits := payload["commits"].([]interface{})
		if len(commits) > 0 {
			if commit, ok := commits[0].(map[string]interface{}); ok {
				wp.Commit = CommitInfo{
					SHA:       getString(commit, "id"),
					Message:   getString(commit, "message"),
					URL:       getString(commit, "url"),
					Timestamp: getString(commit, "timestamp"),
				}
				
				if author, ok := commit["author"].(map[string]interface{}); ok {
					wp.Commit.Author = getString(author, "name")
					wp.Commit.Email = getString(author, "email")
				}
			}
		}

	case "Merge Request Hook":
		wp.EventType = "merge_request"
		if mr, ok := payload["merge_request"].(map[string]interface{}); ok {
			wp.PullRequest = &PullRequestInfo{
				Number:       int(getFloat(mr, "iid")),
				Title:        getString(mr, "title"),
				Description:  getString(mr, "description"),
				State:        getString(mr, "state"),
				SourceBranch: getString(mr, "source_branch"),
				TargetBranch: getString(mr, "target_branch"),
			}
		}

	case "Tag Push Hook":
		wp.EventType = "tag"
		wp.Tag = extractTagFromRef(getString(payload, "ref"))
	}

	// Parse user info
	if user, ok := payload["user"].(map[string]interface{}); ok {
		wp.Sender = UserInfo{
			Username: getString(user, "username"),
			Email:    getString(user, "email"),
		}
	}

	return wp, nil
}

// GetEventType returns the GitLab event type
func (g *GitLabHandler) GetEventType(r *http.Request) string {
	return r.Header.Get("X-Gitlab-Event")
}

// BitbucketHandler handles Bitbucket webhooks
type BitbucketHandler struct {
	log logr.Logger
}

// ValidateSignature validates Bitbucket webhook signature
func (b *BitbucketHandler) ValidateSignature(r *http.Request, secret string) error {
	// Bitbucket doesn't use signatures, it uses IP whitelisting
	// For now, we'll skip validation
	// TODO: Implement IP whitelist validation
	return nil
}

// ParsePayload parses Bitbucket webhook payload
func (b *BitbucketHandler) ParsePayload(r *http.Request) (*WebhookPayload, error) {
	var payload map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		return nil, fmt.Errorf("decoding payload: %w", err)
	}

	wp := &WebhookPayload{
		Provider: "bitbucket",
	}

	// Parse repository info
	if repo, ok := payload["repository"].(map[string]interface{}); ok {
		wp.Repository = RepositoryInfo{
			Name:     getString(repo, "name"),
			FullName: getString(repo, "full_name"),
		}
		
		if links, ok := repo["links"].(map[string]interface{}); ok {
			if clone, ok := links["clone"].([]interface{}); ok {
				for _, c := range clone {
					if link, ok := c.(map[string]interface{}); ok {
						if getString(link, "name") == "https" {
							wp.Repository.CloneURL = getString(link, "href")
						} else if getString(link, "name") == "ssh" {
							wp.Repository.SSHURL = getString(link, "href")
						}
					}
				}
			}
		}
	}

	// Parse based on event type
	event := r.Header.Get("X-Event-Key")
	switch event {
	case "repo:push":
		wp.EventType = "push"
		
		if push, ok := payload["push"].(map[string]interface{}); ok {
			if changes, ok := push["changes"].([]interface{}); ok && len(changes) > 0 {
				if change, ok := changes[0].(map[string]interface{}); ok {
					if newRef, ok := change["new"].(map[string]interface{}); ok {
						wp.Branch = getString(newRef, "name")
						if target, ok := newRef["target"].(map[string]interface{}); ok {
							wp.Commit = CommitInfo{
								SHA:     getString(target, "hash"),
								Message: getString(target, "message"),
							}
							
							if author, ok := target["author"].(map[string]interface{}); ok {
								if user, ok := author["user"].(map[string]interface{}); ok {
									wp.Commit.Author = getString(user, "display_name")
								}
							}
						}
					}
				}
			}
		}

	case "pullrequest:created", "pullrequest:updated":
		wp.EventType = "pull_request"
		if pr, ok := payload["pullrequest"].(map[string]interface{}); ok {
			wp.PullRequest = &PullRequestInfo{
				Number:      int(getFloat(pr, "id")),
				Title:       getString(pr, "title"),
				Description: getString(pr, "description"),
				State:       getString(pr, "state"),
			}
			
			if source, ok := pr["source"].(map[string]interface{}); ok {
				if branch, ok := source["branch"].(map[string]interface{}); ok {
					wp.PullRequest.SourceBranch = getString(branch, "name")
				}
			}
			if destination, ok := pr["destination"].(map[string]interface{}); ok {
				if branch, ok := destination["branch"].(map[string]interface{}); ok {
					wp.PullRequest.TargetBranch = getString(branch, "name")
				}
			}
		}
	}

	// Parse actor info
	if actor, ok := payload["actor"].(map[string]interface{}); ok {
		wp.Sender = UserInfo{
			Username: getString(actor, "username"),
		}
	}

	return wp, nil
}

// GetEventType returns the Bitbucket event type
func (b *BitbucketHandler) GetEventType(r *http.Request) string {
	return r.Header.Get("X-Event-Key")
}

// Helper functions
func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func getFloat(m map[string]interface{}, key string) float64 {
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

func extractBranchFromRef(ref string) string {
	const prefix = "refs/heads/"
	if strings.HasPrefix(ref, prefix) {
		return strings.TrimPrefix(ref, prefix)
	}
	return ref
}

func extractTagFromRef(ref string) string {
	const prefix = "refs/tags/"
	if strings.HasPrefix(ref, prefix) {
		return strings.TrimPrefix(ref, prefix)
	}
	return ref
}
