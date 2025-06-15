/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package security

import (
	"context"
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	observabilityv1beta1 "github.com/gunjanjp/gunj-operator/api/v1beta1"
)

var log = logf.Log.WithName("security-validator")

// PodSecurityLevel represents the Pod Security Standards levels
type PodSecurityLevel string

const (
	// PodSecurityLevelPrivileged allows unrestricted pods
	PodSecurityLevelPrivileged PodSecurityLevel = "privileged"
	// PodSecurityLevelBaseline allows minimally restrictive pods
	PodSecurityLevelBaseline PodSecurityLevel = "baseline"
	// PodSecurityLevelRestricted enforces pod hardening best practices
	PodSecurityLevelRestricted PodSecurityLevel = "restricted"
)

// SecurityValidator validates security policies for ObservabilityPlatform resources
type SecurityValidator struct {
	client.Client
	// DefaultSecurityLevel is the default pod security level to enforce
	DefaultSecurityLevel PodSecurityLevel
	// EnforceNonRoot enforces non-root user execution
	EnforceNonRoot bool
	// RequireSecurityAnnotations requires specific security annotations
	RequireSecurityAnnotations bool
	// AllowedCapabilities is the list of allowed capabilities
	AllowedCapabilities []string
	// RequiredSecurityAnnotations is the list of required security annotations
	RequiredSecurityAnnotations []string
	// NetworkPolicyRequired enforces network policy existence
	NetworkPolicyRequired bool
}

// NewSecurityValidator creates a new security validator
func NewSecurityValidator(client client.Client) *SecurityValidator {
	return &SecurityValidator{
		Client:               client,
		DefaultSecurityLevel: PodSecurityLevelRestricted,
		EnforceNonRoot:      true,
		RequireSecurityAnnotations: true,
		AllowedCapabilities: []string{
			"CHOWN",
			"DAC_OVERRIDE",
			"FOWNER",
			"FSETID",
			"KILL",
			"SETGID",
			"SETUID",
			"SETPCAP",
			"NET_BIND_SERVICE",
			"NET_RAW",
			"SYS_CHROOT",
			"MKNOD",
			"AUDIT_WRITE",
			"SETFCAP",
		},
		RequiredSecurityAnnotations: []string{
			"security.gunj-operator.io/pod-security-level",
			"security.gunj-operator.io/compliance-profile",
		},
		NetworkPolicyRequired: true,
	}
}

// ValidateSecurity performs comprehensive security validation
func (v *SecurityValidator) ValidateSecurity(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) field.ErrorList {
	var allErrs field.ErrorList

	// Get the security level for this platform
	securityLevel := v.getSecurityLevel(platform)
	log.V(1).Info("Validating security", "platform", platform.Name, "level", securityLevel)

	// Validate based on security level
	switch securityLevel {
	case PodSecurityLevelRestricted:
		allErrs = append(allErrs, v.validateRestrictedSecurity(ctx, platform)...)
	case PodSecurityLevelBaseline:
		allErrs = append(allErrs, v.validateBaselineSecurity(ctx, platform)...)
	case PodSecurityLevelPrivileged:
		// Minimal validation for privileged level
		log.V(1).Info("Privileged security level - minimal validation", "platform", platform.Name)
	}

	// Common validations regardless of security level
	allErrs = append(allErrs, v.validateCommonSecurity(ctx, platform)...)

	// Validate network policies if required
	if v.NetworkPolicyRequired {
		allErrs = append(allErrs, v.validateNetworkPolicies(ctx, platform)...)
	}

	// Validate required security annotations
	if v.RequireSecurityAnnotations {
		allErrs = append(allErrs, v.validateSecurityAnnotations(platform)...)
	}

	return allErrs
}

// getSecurityLevel determines the security level for the platform
func (v *SecurityValidator) getSecurityLevel(platform *observabilityv1beta1.ObservabilityPlatform) PodSecurityLevel {
	// Check annotations for override
	if platform.Annotations != nil {
		if level, ok := platform.Annotations["security.gunj-operator.io/pod-security-level"]; ok {
			switch PodSecurityLevel(level) {
			case PodSecurityLevelPrivileged, PodSecurityLevelBaseline, PodSecurityLevelRestricted:
				return PodSecurityLevel(level)
			}
		}
	}

	// Check if security spec has a level defined
	if platform.Spec.Security != nil && platform.Spec.Security.PodSecurityPolicy != "" {
		switch platform.Spec.Security.PodSecurityPolicy {
		case "privileged":
			return PodSecurityLevelPrivileged
		case "baseline":
			return PodSecurityLevelBaseline
		case "restricted":
			return PodSecurityLevelRestricted
		}
	}

	// Return default level
	return v.DefaultSecurityLevel
}

// validateRestrictedSecurity validates the most restrictive security policies
func (v *SecurityValidator) validateRestrictedSecurity(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) field.ErrorList {
	var allErrs field.ErrorList
	basePath := field.NewPath("spec")

	// Validate all components for restricted security
	if platform.Spec.Components.Prometheus != nil {
		allErrs = append(allErrs, v.validateComponentRestrictedSecurity(
			platform.Spec.Components.Prometheus.SecurityContext,
			basePath.Child("components", "prometheus", "securityContext"),
			"Prometheus",
		)...)
	}

	if platform.Spec.Components.Grafana != nil {
		allErrs = append(allErrs, v.validateComponentRestrictedSecurity(
			platform.Spec.Components.Grafana.SecurityContext,
			basePath.Child("components", "grafana", "securityContext"),
			"Grafana",
		)...)
	}

	if platform.Spec.Components.Loki != nil {
		allErrs = append(allErrs, v.validateComponentRestrictedSecurity(
			platform.Spec.Components.Loki.SecurityContext,
			basePath.Child("components", "loki", "securityContext"),
			"Loki",
		)...)
	}

	if platform.Spec.Components.Tempo != nil {
		allErrs = append(allErrs, v.validateComponentRestrictedSecurity(
			platform.Spec.Components.Tempo.SecurityContext,
			basePath.Child("components", "tempo", "securityContext"),
			"Tempo",
		)...)
	}

	return allErrs
}

// validateComponentRestrictedSecurity validates restricted security for a component
func (v *SecurityValidator) validateComponentRestrictedSecurity(secContext *observabilityv1beta1.SecurityContext, path *field.Path, componentName string) field.ErrorList {
	var allErrs field.ErrorList

	// Security context must be defined
	if secContext == nil {
		allErrs = append(allErrs, field.Required(
			path,
			fmt.Sprintf("%s must have security context defined for restricted security level", componentName),
		))
		return allErrs
	}

	// Pod Security Context validation
	if secContext.PodSecurityContext == nil {
		allErrs = append(allErrs, field.Required(
			path.Child("podSecurityContext"),
			"pod security context is required for restricted level",
		))
	} else {
		psc := secContext.PodSecurityContext

		// Must run as non-root
		if psc.RunAsNonRoot == nil || !*psc.RunAsNonRoot {
			allErrs = append(allErrs, field.Invalid(
				path.Child("podSecurityContext", "runAsNonRoot"),
				psc.RunAsNonRoot,
				"must be true for restricted security level",
			))
		}

		// Must have user ID >= 1000
		if psc.RunAsUser == nil || *psc.RunAsUser < 1000 {
			allErrs = append(allErrs, field.Invalid(
				path.Child("podSecurityContext", "runAsUser"),
				psc.RunAsUser,
				"must be >= 1000 for restricted security level",
			))
		}

		// Must have group ID >= 1000
		if psc.RunAsGroup == nil || *psc.RunAsGroup < 1000 {
			allErrs = append(allErrs, field.Invalid(
				path.Child("podSecurityContext", "runAsGroup"),
				psc.RunAsGroup,
				"must be >= 1000 for restricted security level",
			))
		}

		// Must have FSGroup >= 1000
		if psc.FSGroup == nil || *psc.FSGroup < 1000 {
			allErrs = append(allErrs, field.Invalid(
				path.Child("podSecurityContext", "fsGroup"),
				psc.FSGroup,
				"must be >= 1000 for restricted security level",
			))
		}

		// Seccomp profile must be RuntimeDefault or Localhost
		if psc.SeccompProfile == nil {
			allErrs = append(allErrs, field.Required(
				path.Child("podSecurityContext", "seccompProfile"),
				"seccomp profile is required for restricted security level",
			))
		} else {
			if psc.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault &&
				psc.SeccompProfile.Type != corev1.SeccompProfileTypeLocalhost {
				allErrs = append(allErrs, field.Invalid(
					path.Child("podSecurityContext", "seccompProfile", "type"),
					psc.SeccompProfile.Type,
					"must be RuntimeDefault or Localhost for restricted security level",
				))
			}
		}
	}

	// Container Security Context validation
	if secContext.ContainerSecurityContext == nil {
		allErrs = append(allErrs, field.Required(
			path.Child("containerSecurityContext"),
			"container security context is required for restricted level",
		))
	} else {
		csc := secContext.ContainerSecurityContext

		// Must not allow privilege escalation
		if csc.AllowPrivilegeEscalation == nil || *csc.AllowPrivilegeEscalation {
			allErrs = append(allErrs, field.Invalid(
				path.Child("containerSecurityContext", "allowPrivilegeEscalation"),
				csc.AllowPrivilegeEscalation,
				"must be false for restricted security level",
			))
		}

		// Must have read-only root filesystem
		if csc.ReadOnlyRootFilesystem == nil || !*csc.ReadOnlyRootFilesystem {
			allErrs = append(allErrs, field.Invalid(
				path.Child("containerSecurityContext", "readOnlyRootFilesystem"),
				csc.ReadOnlyRootFilesystem,
				"must be true for restricted security level",
			))
		}

		// Must drop ALL capabilities
		if csc.Capabilities == nil {
			allErrs = append(allErrs, field.Required(
				path.Child("containerSecurityContext", "capabilities"),
				"capabilities must be defined for restricted security level",
			))
		} else {
			// Check drop capabilities
			hasDropAll := false
			for _, cap := range csc.Capabilities.Drop {
				if string(cap) == "ALL" {
					hasDropAll = true
					break
				}
			}
			if !hasDropAll {
				allErrs = append(allErrs, field.Invalid(
					path.Child("containerSecurityContext", "capabilities", "drop"),
					csc.Capabilities.Drop,
					"must drop ALL capabilities for restricted security level",
				))
			}

			// Check added capabilities - only NET_BIND_SERVICE allowed
			for _, cap := range csc.Capabilities.Add {
				if string(cap) != "NET_BIND_SERVICE" {
					allErrs = append(allErrs, field.Invalid(
						path.Child("containerSecurityContext", "capabilities", "add"),
						cap,
						fmt.Sprintf("capability %s not allowed in restricted security level (only NET_BIND_SERVICE allowed)", cap),
					))
				}
			}
		}

		// Must run as non-root
		if csc.RunAsNonRoot == nil || !*csc.RunAsNonRoot {
			allErrs = append(allErrs, field.Invalid(
				path.Child("containerSecurityContext", "runAsNonRoot"),
				csc.RunAsNonRoot,
				"must be true for restricted security level",
			))
		}

		// Seccomp profile must be RuntimeDefault or Localhost
		if csc.SeccompProfile == nil {
			allErrs = append(allErrs, field.Required(
				path.Child("containerSecurityContext", "seccompProfile"),
				"seccomp profile is required for restricted security level",
			))
		} else {
			if csc.SeccompProfile.Type != corev1.SeccompProfileTypeRuntimeDefault &&
				csc.SeccompProfile.Type != corev1.SeccompProfileTypeLocalhost {
				allErrs = append(allErrs, field.Invalid(
					path.Child("containerSecurityContext", "seccompProfile", "type"),
					csc.SeccompProfile.Type,
					"must be RuntimeDefault or Localhost for restricted security level",
				))
			}
		}
	}

	return allErrs
}

// validateBaselineSecurity validates baseline security policies
func (v *SecurityValidator) validateBaselineSecurity(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) field.ErrorList {
	var allErrs field.ErrorList
	basePath := field.NewPath("spec")

	// Validate all components for baseline security
	if platform.Spec.Components.Prometheus != nil {
		allErrs = append(allErrs, v.validateComponentBaselineSecurity(
			platform.Spec.Components.Prometheus.SecurityContext,
			basePath.Child("components", "prometheus", "securityContext"),
			"Prometheus",
		)...)
	}

	if platform.Spec.Components.Grafana != nil {
		allErrs = append(allErrs, v.validateComponentBaselineSecurity(
			platform.Spec.Components.Grafana.SecurityContext,
			basePath.Child("components", "grafana", "securityContext"),
			"Grafana",
		)...)
	}

	if platform.Spec.Components.Loki != nil {
		allErrs = append(allErrs, v.validateComponentBaselineSecurity(
			platform.Spec.Components.Loki.SecurityContext,
			basePath.Child("components", "loki", "securityContext"),
			"Loki",
		)...)
	}

	if platform.Spec.Components.Tempo != nil {
		allErrs = append(allErrs, v.validateComponentBaselineSecurity(
			platform.Spec.Components.Tempo.SecurityContext,
			basePath.Child("components", "tempo", "securityContext"),
			"Tempo",
		)...)
	}

	return allErrs
}

// validateComponentBaselineSecurity validates baseline security for a component
func (v *SecurityValidator) validateComponentBaselineSecurity(secContext *observabilityv1beta1.SecurityContext, path *field.Path, componentName string) field.ErrorList {
	var allErrs field.ErrorList

	if secContext == nil {
		// Security context is recommended but not required for baseline
		return allErrs
	}

	// Container Security Context validation for baseline
	if secContext.ContainerSecurityContext != nil {
		csc := secContext.ContainerSecurityContext

		// Privileged must be false
		if csc.Privileged != nil && *csc.Privileged {
			allErrs = append(allErrs, field.Invalid(
				path.Child("containerSecurityContext", "privileged"),
				csc.Privileged,
				"privileged containers not allowed in baseline security level",
			))
		}

		// Check for dangerous capabilities
		if csc.Capabilities != nil {
			dangerousCapabilities := []string{
				"SYS_ADMIN", "SYS_PTRACE", "SYS_MODULE", "SYS_RAWIO",
				"SYS_PACCT", "SYS_BOOT", "SYS_NICE", "SYS_RESOURCE",
				"SYS_TIME", "SYS_TTY_CONFIG", "AUDIT_CONTROL",
				"MAC_ADMIN", "MAC_OVERRIDE", "NET_ADMIN", "SYSLOG",
				"DAC_READ_SEARCH", "LINUX_IMMUTABLE", "IPC_LOCK",
			}

			for _, cap := range csc.Capabilities.Add {
				for _, dangerous := range dangerousCapabilities {
					if string(cap) == dangerous {
						allErrs = append(allErrs, field.Invalid(
							path.Child("containerSecurityContext", "capabilities", "add"),
							cap,
							fmt.Sprintf("capability %s not allowed in baseline security level", cap),
						))
					}
				}
			}
		}

		// Check for host namespaces - not allowed in baseline
		// Note: These would typically be on the pod spec, but we validate the intent
		if secContext.HostNetwork {
			allErrs = append(allErrs, field.Invalid(
				path.Child("hostNetwork"),
				true,
				"host network not allowed in baseline security level",
			))
		}

		if secContext.HostPID {
			allErrs = append(allErrs, field.Invalid(
				path.Child("hostPID"),
				true,
				"host PID namespace not allowed in baseline security level",
			))
		}

		if secContext.HostIPC {
			allErrs = append(allErrs, field.Invalid(
				path.Child("hostIPC"),
				true,
				"host IPC namespace not allowed in baseline security level",
			))
		}
	}

	return allErrs
}

// validateCommonSecurity validates common security requirements
func (v *SecurityValidator) validateCommonSecurity(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) field.ErrorList {
	var allErrs field.ErrorList

	// Validate security spec if present
	if platform.Spec.Security != nil {
		secPath := field.NewPath("spec", "security")

		// Validate RBAC settings
		if platform.Spec.Security.RBAC != nil {
			if !platform.Spec.Security.RBAC.Create {
				// Warn that RBAC should be enabled
				log.V(1).Info("RBAC creation disabled - ensure manual RBAC setup", "platform", platform.Name)
			}
		}

		// Validate service account settings
		if platform.Spec.Security.ServiceAccount == "" {
			// Service account should be specified
			allErrs = append(allErrs, field.Invalid(
				secPath.Child("serviceAccount"),
				"",
				"service account should be specified for security",
			))
		}

		// Validate TLS settings
		if platform.Spec.Security.TLS.Enabled {
			if !platform.Spec.Security.TLS.AutoTLS && platform.Spec.Security.TLS.CertSecret == "" {
				allErrs = append(allErrs, field.Required(
					secPath.Child("tls", "certSecret"),
					"certificate secret required when TLS enabled without auto-TLS",
				))
			}
		}
	}

	// Check for sensitive environment variables
	allErrs = append(allErrs, v.validateEnvironmentVariables(platform)...)

	return allErrs
}

// validateNetworkPolicies validates network policy requirements
func (v *SecurityValidator) validateNetworkPolicies(ctx context.Context, platform *observabilityv1beta1.ObservabilityPlatform) field.ErrorList {
	var allErrs field.ErrorList
	secPath := field.NewPath("spec", "security")

	// Check if network policies are defined or referenced
	if platform.Spec.Security == nil || platform.Spec.Security.NetworkPolicy == nil {
		allErrs = append(allErrs, field.Required(
			secPath.Child("networkPolicy"),
			"network policy configuration required for security compliance",
		))
		return allErrs
	}

	np := platform.Spec.Security.NetworkPolicy
	npPath := secPath.Child("networkPolicy")

	if !np.Enabled {
		allErrs = append(allErrs, field.Invalid(
			npPath.Child("enabled"),
			false,
			"network policies must be enabled for security compliance",
		))
		return allErrs
	}

	// Validate ingress rules
	if len(np.Ingress) == 0 {
		allErrs = append(allErrs, field.Required(
			npPath.Child("ingress"),
			"at least one ingress rule must be defined",
		))
	}

	// Validate egress rules
	if len(np.Egress) == 0 {
		allErrs = append(allErrs, field.Required(
			npPath.Child("egress"),
			"at least one egress rule must be defined",
		))
	}

	// Validate that policies are not too permissive
	for i, rule := range np.Ingress {
		if rule.From == nil || len(rule.From) == 0 {
			allErrs = append(allErrs, field.Invalid(
				npPath.Child("ingress").Index(i).Child("from"),
				nil,
				"ingress rule must specify allowed sources",
			))
		}
	}

	for i, rule := range np.Egress {
		if rule.To == nil || len(rule.To) == 0 {
			// Check if this is a DNS-only egress rule
			isDNSOnly := false
			if rule.Ports != nil {
				for _, port := range rule.Ports {
					if port.Port != nil && port.Port.IntVal == 53 {
						isDNSOnly = true
						break
					}
				}
			}
			if !isDNSOnly {
				allErrs = append(allErrs, field.Invalid(
					npPath.Child("egress").Index(i).Child("to"),
					nil,
					"egress rule must specify allowed destinations",
				))
			}
		}
	}

	return allErrs
}

// validateSecurityAnnotations validates required security annotations
func (v *SecurityValidator) validateSecurityAnnotations(platform *observabilityv1beta1.ObservabilityPlatform) field.ErrorList {
	var allErrs field.ErrorList
	annotationsPath := field.NewPath("metadata", "annotations")

	if platform.Annotations == nil {
		platform.Annotations = make(map[string]string)
	}

	// Check for required security annotations
	for _, required := range v.RequiredSecurityAnnotations {
		if _, ok := platform.Annotations[required]; !ok {
			allErrs = append(allErrs, field.Required(
				annotationsPath.Child(required),
				fmt.Sprintf("security annotation '%s' is required", required),
			))
		}
	}

	// Validate specific annotation values
	if level, ok := platform.Annotations["security.gunj-operator.io/pod-security-level"]; ok {
		validLevels := []string{string(PodSecurityLevelPrivileged), string(PodSecurityLevelBaseline), string(PodSecurityLevelRestricted)}
		isValid := false
		for _, valid := range validLevels {
			if level == valid {
				isValid = true
				break
			}
		}
		if !isValid {
			allErrs = append(allErrs, field.Invalid(
				annotationsPath.Child("security.gunj-operator.io/pod-security-level"),
				level,
				fmt.Sprintf("must be one of: %s", strings.Join(validLevels, ", ")),
			))
		}
	}

	// Validate compliance profile
	if profile, ok := platform.Annotations["security.gunj-operator.io/compliance-profile"]; ok {
		validProfiles := []string{"cis", "nist", "pci-dss", "hipaa", "soc2", "custom"}
		isValid := false
		for _, valid := range validProfiles {
			if profile == valid {
				isValid = true
				break
			}
		}
		if !isValid {
			allErrs = append(allErrs, field.Invalid(
				annotationsPath.Child("security.gunj-operator.io/compliance-profile"),
				profile,
				fmt.Sprintf("must be one of: %s", strings.Join(validProfiles, ", ")),
			))
		}
	}

	return allErrs
}

// validateEnvironmentVariables checks for sensitive data in environment variables
func (v *SecurityValidator) validateEnvironmentVariables(platform *observabilityv1beta1.ObservabilityPlatform) field.ErrorList {
	var allErrs field.ErrorList

	// List of sensitive patterns to check in environment variable names
	sensitivePatterns := []string{
		"PASSWORD", "SECRET", "TOKEN", "KEY", "CERT", "CREDENTIAL",
		"PRIVATE", "APIKEY", "API_KEY", "ACCESS_KEY", "SECRET_KEY",
	}

	// Helper function to check env vars
	checkEnvVars := func(envVars []corev1.EnvVar, path *field.Path) {
		for i, env := range envVars {
			envName := strings.ToUpper(env.Name)
			
			// Check if the env var name contains sensitive patterns
			for _, pattern := range sensitivePatterns {
				if strings.Contains(envName, pattern) {
					// Ensure the value is from a secret, not plaintext
					if env.Value != "" {
						allErrs = append(allErrs, field.Invalid(
							path.Index(i).Child("value"),
							"[REDACTED]",
							fmt.Sprintf("environment variable '%s' appears to contain sensitive data and should use valueFrom.secretKeyRef instead of plaintext value", env.Name),
						))
					}
				}
			}
		}
	}

	// Check environment variables in each component
	if platform.Spec.Components.Prometheus != nil && platform.Spec.Components.Prometheus.ExtraEnvVars != nil {
		checkEnvVars(platform.Spec.Components.Prometheus.ExtraEnvVars,
			field.NewPath("spec", "components", "prometheus", "extraEnvVars"))
	}

	if platform.Spec.Components.Grafana != nil && platform.Spec.Components.Grafana.ExtraEnvVars != nil {
		checkEnvVars(platform.Spec.Components.Grafana.ExtraEnvVars,
			field.NewPath("spec", "components", "grafana", "extraEnvVars"))
	}

	if platform.Spec.Components.Loki != nil && platform.Spec.Components.Loki.ExtraEnvVars != nil {
		checkEnvVars(platform.Spec.Components.Loki.ExtraEnvVars,
			field.NewPath("spec", "components", "loki", "extraEnvVars"))
	}

	if platform.Spec.Components.Tempo != nil && platform.Spec.Components.Tempo.ExtraEnvVars != nil {
		checkEnvVars(platform.Spec.Components.Tempo.ExtraEnvVars,
			field.NewPath("spec", "components", "tempo", "extraEnvVars"))
	}

	return allErrs
}

// GenerateSecurityRecommendations generates security recommendations for a platform
func (v *SecurityValidator) GenerateSecurityRecommendations(platform *observabilityv1beta1.ObservabilityPlatform) []string {
	var recommendations []string

	// Get current security level
	level := v.getSecurityLevel(platform)

	// Recommend upgrading security level if not at restricted
	if level != PodSecurityLevelRestricted {
		recommendations = append(recommendations,
			fmt.Sprintf("Consider upgrading from '%s' to 'restricted' security level for enhanced security", level))
	}

	// Check for missing security context
	components := []struct {
		name string
		sec  *observabilityv1beta1.SecurityContext
	}{
		{"Prometheus", platform.Spec.Components.Prometheus.SecurityContext},
		{"Grafana", platform.Spec.Components.Grafana.SecurityContext},
		{"Loki", platform.Spec.Components.Loki.SecurityContext},
		{"Tempo", platform.Spec.Components.Tempo.SecurityContext},
	}

	for _, comp := range components {
		if comp.sec == nil {
			recommendations = append(recommendations,
				fmt.Sprintf("Add security context for %s component", comp.name))
			continue
		}

		// Check for specific security improvements
		if comp.sec.ContainerSecurityContext != nil {
			csc := comp.sec.ContainerSecurityContext
			if csc.ReadOnlyRootFilesystem == nil || !*csc.ReadOnlyRootFilesystem {
				recommendations = append(recommendations,
					fmt.Sprintf("Enable read-only root filesystem for %s", comp.name))
			}
			if csc.AllowPrivilegeEscalation == nil || *csc.AllowPrivilegeEscalation {
				recommendations = append(recommendations,
					fmt.Sprintf("Disable privilege escalation for %s", comp.name))
			}
		}
	}

	// Network policy recommendations
	if platform.Spec.Security == nil || platform.Spec.Security.NetworkPolicy == nil || !platform.Spec.Security.NetworkPolicy.Enabled {
		recommendations = append(recommendations,
			"Enable network policies to control traffic between components")
	}

	// TLS recommendations
	if platform.Spec.Security == nil || !platform.Spec.Security.TLS.Enabled {
		recommendations = append(recommendations,
			"Enable TLS for encrypted communication between components")
	}

	// Audit logging recommendations
	if platform.Spec.Security == nil || !platform.Spec.Security.AuditLogging {
		recommendations = append(recommendations,
			"Enable audit logging to track security-relevant events")
	}

	return recommendations
}
