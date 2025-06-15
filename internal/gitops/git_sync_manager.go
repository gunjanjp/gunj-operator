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

package gitops

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

// GitSyncManager manages Git repository synchronization
type GitSyncManager struct {
	client client.Client
	log    logr.Logger
}

// NewGitSyncManager creates a new Git sync manager
func NewGitSyncManager(client client.Client, log logr.Logger) *GitSyncManager {
	return &GitSyncManager{
		client: client,
		log:    log.WithName("git-sync-manager"),
	}
}

// GitCredentials represents Git authentication credentials
type GitCredentials struct {
	Username string
	Password string
	SSHKey   string
}

// GitSyncConfig represents Git synchronization configuration
type GitSyncConfig struct {
	Repository   string
	Branch       string
	Path         string
	PollInterval string
	Credentials  *GitCredentials
	WebhookURL   string
}

// SetupSync sets up Git repository synchronization
func (m *GitSyncManager) SetupSync(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, config *GitSyncConfig) error {
	log := m.log.WithValues("deployment", fmt.Sprintf("%s/%s", deployment.Namespace, deployment.Name))

	// Create Git credentials secret if needed
	if config.Credentials != nil {
		if err := m.createGitCredentialsSecret(ctx, deployment, config.Credentials); err != nil {
			return fmt.Errorf("failed to create Git credentials secret: %w", err)
		}
	}

	// Create Git sync ConfigMap for configuration
	if err := m.createGitSyncConfigMap(ctx, deployment, config); err != nil {
		return fmt.Errorf("failed to create Git sync ConfigMap: %w", err)
	}

	// Create Git sync CronJob for periodic synchronization
	if err := m.createGitSyncCronJob(ctx, deployment, config); err != nil {
		return fmt.Errorf("failed to create Git sync CronJob: %w", err)
	}

	// Setup webhook receiver if configured
	if deployment.Spec.Repository.Webhook != nil && deployment.Spec.Repository.Webhook.Enabled {
		if err := m.setupWebhookReceiver(ctx, deployment); err != nil {
			return fmt.Errorf("failed to setup webhook receiver: %w", err)
		}
	}

	log.Info("Git sync setup completed successfully")
	return nil
}

// createGitCredentialsSecret creates a secret for Git credentials
func (m *GitSyncManager) createGitCredentialsSecret(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, creds *GitCredentials) error {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-git-creds", deployment.Name),
			Namespace: deployment.Namespace,
			Labels: map[string]string{
				"observability.io/gitops-deployment": deployment.Name,
				"observability.io/managed-by":        "gunj-operator",
			},
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{},
	}

	// Add credentials based on type
	if creds.SSHKey != "" {
		secret.Data["ssh-privatekey"] = []byte(creds.SSHKey)
	} else if creds.Username != "" && creds.Password != "" {
		secret.Data["username"] = []byte(creds.Username)
		secret.Data["password"] = []byte(creds.Password)
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(deployment, secret, m.client.Scheme()); err != nil {
		return err
	}

	// Create or update secret
	existing := &corev1.Secret{}
	err := m.client.Get(ctx, types.NamespacedName{Name: secret.Name, Namespace: secret.Namespace}, existing)
	if err == nil {
		existing.Data = secret.Data
		return m.client.Update(ctx, existing)
	} else if errors.IsNotFound(err) {
		return m.client.Create(ctx, secret)
	}
	return err
}

// createGitSyncConfigMap creates a ConfigMap for Git sync configuration
func (m *GitSyncManager) createGitSyncConfigMap(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, config *GitSyncConfig) error {
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-git-sync-config", deployment.Name),
			Namespace: deployment.Namespace,
			Labels: map[string]string{
				"observability.io/gitops-deployment": deployment.Name,
				"observability.io/managed-by":        "gunj-operator",
			},
		},
		Data: map[string]string{
			"repository":    config.Repository,
			"branch":        config.Branch,
			"path":          config.Path,
			"pollInterval":  config.PollInterval,
			"deploymentName": deployment.Name,
		},
	}

	// Add webhook URL if configured
	if config.WebhookURL != "" {
		configMap.Data["webhookURL"] = config.WebhookURL
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(deployment, configMap, m.client.Scheme()); err != nil {
		return err
	}

	// Create or update ConfigMap
	existing := &corev1.ConfigMap{}
	err := m.client.Get(ctx, types.NamespacedName{Name: configMap.Name, Namespace: configMap.Namespace}, existing)
	if err == nil {
		existing.Data = configMap.Data
		return m.client.Update(ctx, existing)
	} else if errors.IsNotFound(err) {
		return m.client.Create(ctx, configMap)
	}
	return err
}

// createGitSyncCronJob creates a CronJob for periodic Git synchronization
func (m *GitSyncManager) createGitSyncCronJob(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment, config *GitSyncConfig) error {
	// Parse poll interval to cron schedule
	cronSchedule := m.intervalToCron(config.PollInterval)

	// Create CronJob
	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-git-sync", deployment.Name),
			Namespace: deployment.Namespace,
			Labels: map[string]string{
				"observability.io/gitops-deployment": deployment.Name,
				"observability.io/managed-by":        "gunj-operator",
			},
		},
		Spec: batchv1.CronJobSpec{
			Schedule:          cronSchedule,
			ConcurrencyPolicy: batchv1.ForbidConcurrent,
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"observability.io/gitops-deployment": deployment.Name,
						"observability.io/job-type":          "git-sync",
					},
				},
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"observability.io/gitops-deployment": deployment.Name,
								"observability.io/job-type":          "git-sync",
							},
						},
						Spec: m.buildGitSyncPodSpec(deployment, config),
					},
				},
			},
		},
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(deployment, cronJob, m.client.Scheme()); err != nil {
		return err
	}

	// Create or update CronJob
	existing := &batchv1.CronJob{}
	err := m.client.Get(ctx, types.NamespacedName{Name: cronJob.Name, Namespace: cronJob.Namespace}, existing)
	if err == nil {
		existing.Spec = cronJob.Spec
		return m.client.Update(ctx, existing)
	} else if errors.IsNotFound(err) {
		return m.client.Create(ctx, cronJob)
	}
	return err
}

// buildGitSyncPodSpec builds the Pod spec for Git sync jobs
func (m *GitSyncManager) buildGitSyncPodSpec(deployment *observabilityv1beta1.GitOpsDeployment, config *GitSyncConfig) corev1.PodSpec {
	volumes := []corev1.Volume{
		{
			Name: "git-sync-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: fmt.Sprintf("%s-git-sync-config", deployment.Name),
					},
				},
			},
		},
		{
			Name: "workspace",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "git-sync-config",
			MountPath: "/etc/git-sync",
		},
		{
			Name:      "workspace",
			MountPath: "/workspace",
		},
	}

	// Add credentials volume if needed
	if config.Credentials != nil {
		volumes = append(volumes, corev1.Volume{
			Name: "git-credentials",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: fmt.Sprintf("%s-git-creds", deployment.Name),
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      "git-credentials",
			MountPath: "/etc/git-secret",
			ReadOnly:  true,
		})
	}

	// Build container spec
	container := corev1.Container{
		Name:  "git-sync",
		Image: "k8s.gcr.io/git-sync/git-sync:v3.6.2",
		Env: []corev1.EnvVar{
			{
				Name:  "GIT_SYNC_REPO",
				Value: config.Repository,
			},
			{
				Name:  "GIT_SYNC_BRANCH",
				Value: config.Branch,
			},
			{
				Name:  "GIT_SYNC_ROOT",
				Value: "/workspace",
			},
			{
				Name:  "GIT_SYNC_DEST",
				Value: "repo",
			},
			{
				Name:  "GIT_SYNC_ONE_TIME",
				Value: "true",
			},
		},
		VolumeMounts: volumeMounts,
		Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("50m"),
				corev1.ResourceMemory: resource.MustParse("64Mi"),
			},
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:    resource.MustParse("100m"),
				corev1.ResourceMemory: resource.MustParse("128Mi"),
			},
		},
	}

	// Add authentication environment variables
	if config.Credentials != nil {
		if config.Credentials.SSHKey != "" {
			container.Env = append(container.Env,
				corev1.EnvVar{
					Name:  "GIT_SYNC_SSH",
					Value: "true",
				},
				corev1.EnvVar{
					Name:  "GIT_SSH_KEY_FILE",
					Value: "/etc/git-secret/ssh-privatekey",
				},
			)
		} else if config.Credentials.Username != "" {
			container.Env = append(container.Env,
				corev1.EnvVar{
					Name: "GIT_SYNC_USERNAME",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: fmt.Sprintf("%s-git-creds", deployment.Name),
							},
							Key: "username",
						},
					},
				},
				corev1.EnvVar{
					Name: "GIT_SYNC_PASSWORD",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: fmt.Sprintf("%s-git-creds", deployment.Name),
							},
							Key: "password",
						},
					},
				},
			)
		}
	}

	// Add post-sync container to trigger reconciliation
	postSyncContainer := corev1.Container{
		Name:  "post-sync",
		Image: "bitnami/kubectl:latest",
		Command: []string{
			"/bin/bash",
			"-c",
			fmt.Sprintf(`
				# Trigger reconciliation by updating annotation
				kubectl annotate gitopsdeployment %s -n %s \
					observability.io/last-sync-time="$(date -u +%%Y-%%m-%%dT%%H:%%M:%%SZ)" \
					--overwrite
			`, deployment.Name, deployment.Namespace),
		},
		VolumeMounts: []corev1.VolumeMount{
			{
				Name:      "workspace",
				MountPath: "/workspace",
			},
		},
	}

	return corev1.PodSpec{
		RestartPolicy:      corev1.RestartPolicyOnFailure,
		ServiceAccountName: "git-sync-sa", // This SA needs to be created with proper RBAC
		Volumes:            volumes,
		Containers:         []corev1.Container{container, postSyncContainer},
	}
}

// setupWebhookReceiver sets up a webhook receiver for real-time Git updates
func (m *GitSyncManager) setupWebhookReceiver(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) error {
	// Create webhook receiver Service
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-webhook", deployment.Name),
			Namespace: deployment.Namespace,
			Labels: map[string]string{
				"observability.io/gitops-deployment": deployment.Name,
				"observability.io/managed-by":        "gunj-operator",
			},
		},
		Spec: corev1.ServiceSpec{
			Selector: map[string]string{
				"observability.io/gitops-deployment": deployment.Name,
				"observability.io/component":         "webhook-receiver",
			},
			Ports: []corev1.ServicePort{
				{
					Name:     "webhook",
					Port:     8080,
					Protocol: corev1.ProtocolTCP,
				},
			},
		},
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(deployment, service, m.client.Scheme()); err != nil {
		return err
	}

	// Create service
	if err := m.client.Create(ctx, service); err != nil && !errors.IsAlreadyExists(err) {
		return err
	}

	// The actual webhook receiver deployment would be created here
	// For now, we'll just log that it should be created
	m.log.Info("Webhook receiver setup completed", "deployment", deployment.Name)

	return nil
}

// TriggerSync triggers an immediate Git synchronization
func (m *GitSyncManager) TriggerSync(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) error {
	// Create a one-time Job to sync immediately
	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-git-sync-%d", deployment.Name, time.Now().Unix()),
			Namespace: deployment.Namespace,
			Labels: map[string]string{
				"observability.io/gitops-deployment": deployment.Name,
				"observability.io/job-type":          "git-sync-manual",
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"observability.io/gitops-deployment": deployment.Name,
						"observability.io/job-type":          "git-sync-manual",
					},
				},
				Spec: m.buildGitSyncPodSpec(deployment, &GitSyncConfig{
					Repository:   deployment.Spec.Repository.URL,
					Branch:       deployment.Spec.Repository.Branch,
					Path:         deployment.Spec.Repository.Path,
					PollInterval: deployment.Spec.Repository.PollInterval,
				}),
			},
		},
	}

	// Set owner reference
	if err := controllerutil.SetControllerReference(deployment, job, m.client.Scheme()); err != nil {
		return err
	}

	// Create job
	if err := m.client.Create(ctx, job); err != nil {
		return fmt.Errorf("failed to create sync job: %w", err)
	}

	m.log.Info("Manual sync triggered", "deployment", deployment.Name, "job", job.Name)
	return nil
}

// GetLastSyncTime gets the last successful sync time
func (m *GitSyncManager) GetLastSyncTime(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) (*time.Time, error) {
	// List recent sync jobs
	jobList := &batchv1.JobList{}
	if err := m.client.List(ctx, jobList,
		client.InNamespace(deployment.Namespace),
		client.MatchingLabels{
			"observability.io/gitops-deployment": deployment.Name,
			"observability.io/job-type":          "git-sync",
		}); err != nil {
		return nil, err
	}

	var lastSync *time.Time
	for _, job := range jobList.Items {
		if job.Status.Succeeded > 0 && job.Status.CompletionTime != nil {
			if lastSync == nil || job.Status.CompletionTime.After(lastSync.Time) {
				lastSync = &job.Status.CompletionTime.Time
			}
		}
	}

	return lastSync, nil
}

// CleanupSync cleans up Git sync resources
func (m *GitSyncManager) CleanupSync(ctx context.Context, deployment *observabilityv1beta1.GitOpsDeployment) error {
	// Delete CronJob
	cronJob := &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-git-sync", deployment.Name),
			Namespace: deployment.Namespace,
		},
	}
	if err := m.client.Delete(ctx, cronJob); err != nil && !errors.IsNotFound(err) {
		m.log.Error(err, "Failed to delete Git sync CronJob")
	}

	// Delete ConfigMap
	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-git-sync-config", deployment.Name),
			Namespace: deployment.Namespace,
		},
	}
	if err := m.client.Delete(ctx, configMap); err != nil && !errors.IsNotFound(err) {
		m.log.Error(err, "Failed to delete Git sync ConfigMap")
	}

	// Delete credentials secret
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-git-creds", deployment.Name),
			Namespace: deployment.Namespace,
		},
	}
	if err := m.client.Delete(ctx, secret); err != nil && !errors.IsNotFound(err) {
		m.log.Error(err, "Failed to delete Git credentials secret")
	}

	// Delete webhook service if exists
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-webhook", deployment.Name),
			Namespace: deployment.Namespace,
		},
	}
	if err := m.client.Delete(ctx, service); err != nil && !errors.IsNotFound(err) {
		m.log.Error(err, "Failed to delete webhook service")
	}

	m.log.Info("Git sync cleanup completed", "deployment", deployment.Name)
	return nil
}

// intervalToCron converts a duration interval to a cron schedule
func (m *GitSyncManager) intervalToCron(interval string) string {
	duration, err := time.ParseDuration(interval)
	if err != nil {
		// Default to every 5 minutes
		return "*/5 * * * *"
	}

	minutes := int(duration.Minutes())
	if minutes < 1 {
		minutes = 1
	} else if minutes > 59 {
		// For intervals longer than an hour, run hourly
		return "0 * * * *"
	}

	return fmt.Sprintf("*/%d * * * *", minutes)
}
