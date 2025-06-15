/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package webhook

import (
	ctrl "sigs.k8s.io/controller-runtime"

	v1beta1webhook "github.com/gunjanjp/gunj-operator/internal/webhook/v1beta1"
	"github.com/gunjanjp/gunj-operator/internal/webhook/conversion"
)

// SetupWebhooksWithManager sets up all webhooks with the Manager
func SetupWebhooksWithManager(mgr ctrl.Manager) error {
	// Setup ObservabilityPlatform webhook
	observabilityPlatformWebhook := &v1beta1webhook.ObservabilityPlatformWebhook{}
	if err := observabilityPlatformWebhook.SetupWebhookWithManager(mgr); err != nil {
		return err
	}

	// Setup conversion webhook
	conversionWebhook, err := conversion.NewConversionWebhook(mgr)
	if err != nil {
		return err
	}
	if err := conversionWebhook.SetupWithManager(mgr); err != nil {
		return err
	}

	// Future webhooks can be added here
	// Example:
	// alertingRuleWebhook := &v1beta1webhook.AlertingRuleWebhook{}
	// if err := alertingRuleWebhook.SetupWebhookWithManager(mgr); err != nil {
	//     return err
	// }

	return nil
}
