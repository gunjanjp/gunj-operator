/*
Copyright 2025 Gunjan Patil.

Licensed under the MIT License.
*/

package v1alpha1

import (
	"sigs.k8s.io/controller-runtime/pkg/conversion"
	
	"github.com/gunjanjp/gunj-operator/api/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

// ConvertTo converts this ObservabilityPlatform to the Hub version (v1beta1).
func (src *ObservabilityPlatform) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*v1beta1.ObservabilityPlatform)

	// Copy ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// Convert Spec
	if err := convertObservabilityPlatformSpecToV1Beta1(&src.Spec, &dst.Spec); err != nil {
		return err
	}

	// Convert Status
	if err := convertObservabilityPlatformStatusToV1Beta1(&src.Status, &dst.Status); err != nil {
		return err
	}

	return nil
}

// ConvertFrom converts from the Hub version (v1beta1) to this version.
func (dst *ObservabilityPlatform) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*v1beta1.ObservabilityPlatform)

	// Copy ObjectMeta
	dst.ObjectMeta = src.ObjectMeta

	// Convert Spec
	if err := convertObservabilityPlatformSpecFromV1Beta1(&src.Spec, &dst.Spec); err != nil {
		return err
	}

	// Convert Status
	if err := convertObservabilityPlatformStatusFromV1Beta1(&src.Status, &dst.Status); err != nil {
		return err
	}

	return nil
}

// Spec conversion functions

func convertObservabilityPlatformSpecToV1Beta1(src *ObservabilityPlatformSpec, dst *v1beta1.ObservabilityPlatformSpec) error {
	// Direct field mappings
	dst.Paused = src.Paused

	// Convert Components
	if err := convertComponentsToV1Beta1(&src.Components, &dst.Components); err != nil {
		return err
	}

	// Convert Global config
	if err := convertGlobalConfigToV1Beta1(&src.Global, &dst.Global); err != nil {
		return err
	}

	// Convert Alerting
	if src.Alerting != nil {
		dst.Alerting = v1beta1.AlertingConfig{}
		if src.Alerting.AlertManager != nil {
			dst.Alerting.Alertmanager = &v1beta1.AlertmanagerSpec{
				Enabled:  src.Alerting.AlertManager.Enabled,
				Replicas: src.Alerting.AlertManager.Replicas,
				Config:   src.Alerting.AlertManager.Config,
			}
		}
		for _, rule := range src.Alerting.Rules {
			dst.Alerting.Rules = append(dst.Alerting.Rules, convertAlertingRuleToV1Beta1(rule))
		}
	}

	// Convert HighAvailability
	if src.HighAvailability != nil {
		dst.HighAvailability = &v1beta1.HighAvailabilityConfig{
			Enabled:     src.HighAvailability.Enabled,
			MinReplicas: src.HighAvailability.MinReplicas,
		}
	}

	// Convert Backup
	if src.Backup != nil {
		dst.Backup = &v1beta1.BackupConfig{
			Enabled:       src.Backup.Enabled,
			Schedule:      src.Backup.Schedule,
			RetentionDays: src.Backup.RetentionDays,
			Destination:   convertBackupDestinationToV1Beta1(src.Backup.Destination),
		}
	}

	// Security is new in v1beta1, set defaults
	dst.Security = &v1beta1.SecurityConfig{
		TLS: v1beta1.TLSConfig{
			Enabled: true,
		},
		PodSecurityPolicy: true,
		NetworkPolicy:     true,
	}

	return nil
}

func convertObservabilityPlatformSpecFromV1Beta1(src *v1beta1.ObservabilityPlatformSpec, dst *ObservabilityPlatformSpec) error {
	// Direct field mappings
	dst.Paused = src.Paused

	// Convert Components
	if err := convertComponentsFromV1Beta1(&src.Components, &dst.Components); err != nil {
		return err
	}

	// Convert Global config
	if err := convertGlobalConfigFromV1Beta1(&src.Global, &dst.Global); err != nil {
		return err
	}

	// Convert Alerting
	if src.Alerting.Alertmanager != nil || len(src.Alerting.Rules) > 0 {
		dst.Alerting = &AlertingConfig{}
		if src.Alerting.Alertmanager != nil {
			dst.Alerting.AlertManager = &AlertManagerSpec{
				Enabled:  src.Alerting.Alertmanager.Enabled,
				Replicas: src.Alerting.Alertmanager.Replicas,
				Config:   src.Alerting.Alertmanager.Config,
			}
		}
		for _, rule := range src.Alerting.Rules {
			dst.Alerting.Rules = append(dst.Alerting.Rules, convertAlertingRuleFromV1Beta1(rule))
		}
	}

	// Convert HighAvailability
	if src.HighAvailability != nil {
		dst.HighAvailability = &HighAvailabilityConfig{
			Enabled:     src.HighAvailability.Enabled,
			MinReplicas: src.HighAvailability.MinReplicas,
		}
		// Note: AntiAffinity is lost in conversion from v1beta1 to v1alpha1
	}

	// Convert Backup
	if src.Backup != nil {
		dst.Backup = &BackupConfig{
			Enabled:       src.Backup.Enabled,
			Schedule:      src.Backup.Schedule,
			RetentionDays: src.Backup.RetentionDays,
			Destination:   convertBackupDestinationFromV1Beta1(src.Backup.Destination),
		}
	}

	// Note: Security configuration is lost when converting from v1beta1 to v1alpha1

	return nil
}

// Component conversion functions

func convertComponentsToV1Beta1(src *Components, dst *v1beta1.Components) error {
	// Convert Prometheus
	if src.Prometheus != nil {
		dst.Prometheus = &v1beta1.PrometheusSpec{
			Enabled:  src.Prometheus.Enabled,
			Version:  src.Prometheus.Version,
			Replicas: src.Prometheus.Replicas,
			Retention: src.Prometheus.Retention,
		}
		
		// Convert Resources
		dst.Prometheus.Resources = convertResourceRequirementsToV1Beta1(src.Prometheus.Resources)
		
		// Convert Storage
		if src.Prometheus.Storage != nil {
			dst.Prometheus.Storage = v1beta1.StorageSpec{
				Size:             resource.MustParse(src.Prometheus.Storage.Size),
				StorageClassName: src.Prometheus.Storage.StorageClassName,
			}
		}
		
		// Copy ExternalLabels from CustomConfig if present
		if val, ok := src.Prometheus.CustomConfig["externalLabels"]; ok {
			dst.Prometheus.ExternalLabels = map[string]string{"custom": val}
		}
		
		// Convert RemoteWrite
		for _, rw := range src.Prometheus.RemoteWrite {
			dst.Prometheus.RemoteWrite = append(dst.Prometheus.RemoteWrite, v1beta1.RemoteWriteSpec{
				URL:           rw.URL,
				RemoteTimeout: rw.RemoteTimeout,
				Headers:       rw.Headers,
			})
		}
		
		dst.Prometheus.ServiceMonitorSelector = src.Prometheus.ServiceMonitorSelector
	}

	// Convert Grafana
	if src.Grafana != nil {
		dst.Grafana = &v1beta1.GrafanaSpec{
			Enabled:       src.Grafana.Enabled,
			Version:       src.Grafana.Version,
			Replicas:      src.Grafana.Replicas,
			AdminPassword: src.Grafana.AdminPassword,
		}
		
		// Convert Resources
		dst.Grafana.Resources = convertResourceRequirementsToV1Beta1(src.Grafana.Resources)
		
		// Convert Ingress
		if src.Grafana.Ingress != nil {
			dst.Grafana.Ingress = &v1beta1.IngressConfig{
				Enabled:   src.Grafana.Ingress.Enabled,
				ClassName: src.Grafana.Ingress.ClassName,
				Host:      src.Grafana.Ingress.Host,
				Path:      src.Grafana.Ingress.Path,
			}
			if src.Grafana.Ingress.TLS != nil {
				dst.Grafana.Ingress.TLS = &v1beta1.IngressTLS{
					Enabled:    src.Grafana.Ingress.TLS.Enabled,
					SecretName: src.Grafana.Ingress.TLS.SecretName,
				}
			}
		}
		
		// Convert DataSources
		for _, ds := range src.Grafana.DataSources {
			dst.Grafana.DataSources = append(dst.Grafana.DataSources, v1beta1.DataSourceConfig{
				Name:      ds.Name,
				Type:      ds.Type,
				URL:       ds.URL,
				Access:    ds.Access,
				IsDefault: ds.IsDefault,
				JSONData:  ds.JSONData,
			})
		}
		
		// Convert Dashboards
		for _, db := range src.Grafana.Dashboards {
			dst.Grafana.Dashboards = append(dst.Grafana.Dashboards, v1beta1.DashboardConfig{
				Name:      db.Name,
				Folder:    db.Folder,
				ConfigMap: db.ConfigMap,
				URL:       db.URL,
			})
		}
	}

	// Convert Loki
	if src.Loki != nil {
		dst.Loki = &v1beta1.LokiSpec{
			Enabled:   src.Loki.Enabled,
			Version:   src.Loki.Version,
			Replicas:  1, // Default since v1alpha1 doesn't have replicas
			Retention: src.Loki.Retention,
		}
		
		// Convert Resources
		dst.Loki.Resources = convertResourceRequirementsToV1Beta1(src.Loki.Resources)
		
		// Convert Storage
		if src.Loki.Storage != nil {
			dst.Loki.Storage = v1beta1.StorageSpec{
				Size:             resource.MustParse(src.Loki.Storage.Size),
				StorageClassName: src.Loki.Storage.StorageClassName,
			}
		}
		
		// Convert S3
		if src.Loki.S3 != nil {
			dst.Loki.S3 = &v1beta1.S3Config{
				Enabled:         src.Loki.S3.Enabled,
				BucketName:      src.Loki.S3.BucketName,
				Region:          src.Loki.S3.Region,
				Endpoint:        src.Loki.S3.Endpoint,
				AccessKeyID:     src.Loki.S3.AccessKeyID,
				SecretAccessKey: src.Loki.S3.SecretAccessKey,
			}
		}
	}

	// Convert Tempo
	if src.Tempo != nil {
		dst.Tempo = &v1beta1.TempoSpec{
			Enabled:   src.Tempo.Enabled,
			Version:   src.Tempo.Version,
			Replicas:  1, // Default since v1alpha1 doesn't have replicas
			Retention: src.Tempo.Retention,
		}
		
		// Convert Resources
		dst.Tempo.Resources = convertResourceRequirementsToV1Beta1(src.Tempo.Resources)
		
		// Convert Storage
		if src.Tempo.Storage != nil {
			dst.Tempo.Storage = v1beta1.StorageSpec{
				Size:             resource.MustParse(src.Tempo.Storage.Size),
				StorageClassName: src.Tempo.Storage.StorageClassName,
			}
		}
	}

	// Convert OpenTelemetryCollector
	if src.OpenTelemetryCollector != nil {
		dst.OpenTelemetryCollector = &v1beta1.OpenTelemetryCollectorSpec{
			Enabled:  src.OpenTelemetryCollector.Enabled,
			Version:  src.OpenTelemetryCollector.Version,
			Replicas: src.OpenTelemetryCollector.Replicas,
			Config:   src.OpenTelemetryCollector.Config,
		}
		
		// Convert Resources
		dst.OpenTelemetryCollector.Resources = convertResourceRequirementsToV1Beta1(src.OpenTelemetryCollector.Resources)
	}

	return nil
}

func convertComponentsFromV1Beta1(src *v1beta1.Components, dst *Components) error {
	// Convert Prometheus
	if src.Prometheus != nil {
		dst.Prometheus = &PrometheusSpec{
			Enabled:   src.Prometheus.Enabled,
			Version:   src.Prometheus.Version,
			Replicas:  src.Prometheus.Replicas,
			Retention: src.Prometheus.Retention,
			ServiceMonitorSelector: src.Prometheus.ServiceMonitorSelector,
		}
		
		// Convert Resources
		dst.Prometheus.Resources = convertResourceRequirementsFromV1Beta1(src.Prometheus.Resources)
		
		// Convert Storage
		if src.Prometheus.Storage.Size.String() != "0" {
			dst.Prometheus.Storage = &StorageConfig{
				Size:             src.Prometheus.Storage.Size.String(),
				StorageClassName: src.Prometheus.Storage.StorageClassName,
			}
		}
		
		// Convert RemoteWrite
		for _, rw := range src.Prometheus.RemoteWrite {
			dst.Prometheus.RemoteWrite = append(dst.Prometheus.RemoteWrite, RemoteWriteSpec{
				URL:           rw.URL,
				RemoteTimeout: rw.RemoteTimeout,
				Headers:       rw.Headers,
			})
		}
		
		// Note: ExternalLabels and AdditionalScrapeConfigs are lost in conversion
	}

	// Convert Grafana
	if src.Grafana != nil {
		dst.Grafana = &GrafanaSpec{
			Enabled:       src.Grafana.Enabled,
			Version:       src.Grafana.Version,
			Replicas:      src.Grafana.Replicas,
			AdminPassword: src.Grafana.AdminPassword,
		}
		
		// Convert Resources
		dst.Grafana.Resources = convertResourceRequirementsFromV1Beta1(src.Grafana.Resources)
		
		// Convert Ingress
		if src.Grafana.Ingress != nil {
			dst.Grafana.Ingress = &IngressConfig{
				Enabled:   src.Grafana.Ingress.Enabled,
				ClassName: src.Grafana.Ingress.ClassName,
				Host:      src.Grafana.Ingress.Host,
				Path:      src.Grafana.Ingress.Path,
			}
			if src.Grafana.Ingress.TLS != nil {
				dst.Grafana.Ingress.TLS = &IngressTLS{
					Enabled:    src.Grafana.Ingress.TLS.Enabled,
					SecretName: src.Grafana.Ingress.TLS.SecretName,
				}
			}
		}
		
		// Convert DataSources
		for _, ds := range src.Grafana.DataSources {
			dst.Grafana.DataSources = append(dst.Grafana.DataSources, DataSourceConfig{
				Name:      ds.Name,
				Type:      ds.Type,
				URL:       ds.URL,
				Access:    ds.Access,
				IsDefault: ds.IsDefault,
				JSONData:  ds.JSONData,
			})
		}
		
		// Convert Dashboards
		for _, db := range src.Grafana.Dashboards {
			dst.Grafana.Dashboards = append(dst.Grafana.Dashboards, DashboardConfig{
				Name:      db.Name,
				Folder:    db.Folder,
				ConfigMap: db.ConfigMap,
				URL:       db.URL,
			})
		}
		
		// Note: Plugins and SMTP are lost in conversion
	}

	// Convert Loki
	if src.Loki != nil {
		dst.Loki = &LokiSpec{
			Enabled:   src.Loki.Enabled,
			Version:   src.Loki.Version,
			Retention: src.Loki.Retention,
		}
		
		// Convert Resources
		dst.Loki.Resources = convertResourceRequirementsFromV1Beta1(src.Loki.Resources)
		
		// Convert Storage
		if src.Loki.Storage.Size.String() != "0" {
			dst.Loki.Storage = &StorageConfig{
				Size:             src.Loki.Storage.Size.String(),
				StorageClassName: src.Loki.Storage.StorageClassName,
			}
		}
		
		// Convert S3
		if src.Loki.S3 != nil {
			dst.Loki.S3 = &S3Config{
				Enabled:         src.Loki.S3.Enabled,
				BucketName:      src.Loki.S3.BucketName,
				Region:          src.Loki.S3.Region,
				Endpoint:        src.Loki.S3.Endpoint,
				AccessKeyID:     src.Loki.S3.AccessKeyID,
				SecretAccessKey: src.Loki.S3.SecretAccessKey,
			}
		}
		
		// Note: CompactorEnabled is lost in conversion
	}

	// Convert Tempo
	if src.Tempo != nil {
		dst.Tempo = &TempoSpec{
			Enabled:   src.Tempo.Enabled,
			Version:   src.Tempo.Version,
			Retention: src.Tempo.Retention,
		}
		
		// Convert Resources
		dst.Tempo.Resources = convertResourceRequirementsFromV1Beta1(src.Tempo.Resources)
		
		// Convert Storage
		if src.Tempo.Storage.Size.String() != "0" {
			dst.Tempo.Storage = &StorageConfig{
				Size:             src.Tempo.Storage.Size.String(),
				StorageClassName: src.Tempo.Storage.StorageClassName,
			}
		}
		
		// Note: SearchEnabled is lost in conversion
	}

	// Convert OpenTelemetryCollector
	if src.OpenTelemetryCollector != nil {
		dst.OpenTelemetryCollector = &OpenTelemetryCollectorSpec{
			Enabled:  src.OpenTelemetryCollector.Enabled,
			Version:  src.OpenTelemetryCollector.Version,
			Replicas: src.OpenTelemetryCollector.Replicas,
			Config:   src.OpenTelemetryCollector.Config,
		}
		
		// Convert Resources
		dst.OpenTelemetryCollector.Resources = convertResourceRequirementsFromV1Beta1(src.OpenTelemetryCollector.Resources)
	}

	return nil
}

// Resource conversion helpers

func convertResourceRequirementsToV1Beta1(src ResourceRequirements) corev1.ResourceRequirements {
	result := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{},
		Limits:   corev1.ResourceList{},
	}
	
	if src.Requests.Memory != "" {
		result.Requests[corev1.ResourceMemory] = resource.MustParse(src.Requests.Memory)
	}
	if src.Requests.CPU != "" {
		result.Requests[corev1.ResourceCPU] = resource.MustParse(src.Requests.CPU)
	}
	if src.Limits.Memory != "" {
		result.Limits[corev1.ResourceMemory] = resource.MustParse(src.Limits.Memory)
	}
	if src.Limits.CPU != "" {
		result.Limits[corev1.ResourceCPU] = resource.MustParse(src.Limits.CPU)
	}
	
	return result
}

func convertResourceRequirementsFromV1Beta1(src corev1.ResourceRequirements) ResourceRequirements {
	result := ResourceRequirements{
		Requests: ResourceList{},
		Limits:   ResourceList{},
	}
	
	if mem, ok := src.Requests[corev1.ResourceMemory]; ok {
		result.Requests.Memory = mem.String()
	}
	if cpu, ok := src.Requests[corev1.ResourceCPU]; ok {
		result.Requests.CPU = cpu.String()
	}
	if mem, ok := src.Limits[corev1.ResourceMemory]; ok {
		result.Limits.Memory = mem.String()
	}
	if cpu, ok := src.Limits[corev1.ResourceCPU]; ok {
		result.Limits.CPU = cpu.String()
	}
	
	return result
}

// Global config conversion

func convertGlobalConfigToV1Beta1(src *GlobalConfig, dst *v1beta1.GlobalConfig) error {
	dst.ExternalLabels = src.ExternalLabels
	dst.LogLevel = src.LogLevel
	dst.NodeSelector = src.NodeSelector
	
	// Convert Tolerations
	for _, tol := range src.Tolerations {
		dst.Tolerations = append(dst.Tolerations, corev1.Toleration{
			Key:               tol.Key,
			Operator:          corev1.TolerationOperator(tol.Operator),
			Value:             tol.Value,
			Effect:            corev1.TaintEffect(tol.Effect),
			TolerationSeconds: tol.TolerationSeconds,
		})
	}
	
	// Note: SecurityContext is not directly convertible, would need custom logic
	// Note: Affinity and ImagePullSecrets are new in v1beta1
	
	return nil
}

func convertGlobalConfigFromV1Beta1(src *v1beta1.GlobalConfig, dst *GlobalConfig) error {
	dst.ExternalLabels = src.ExternalLabels
	dst.LogLevel = src.LogLevel
	dst.NodeSelector = src.NodeSelector
	
	// Convert Tolerations
	for _, tol := range src.Tolerations {
		dst.Tolerations = append(dst.Tolerations, Toleration{
			Key:               tol.Key,
			Operator:          string(tol.Operator),
			Value:             tol.Value,
			Effect:            string(tol.Effect),
			TolerationSeconds: tol.TolerationSeconds,
		})
	}
	
	// Note: Affinity and ImagePullSecrets are lost in conversion
	
	return nil
}

// Alerting conversion

func convertAlertingRuleToV1Beta1(src AlertingRule) v1beta1.AlertingRule {
	result := v1beta1.AlertingRule{
		Name: src.Name,
	}
	
	for _, group := range src.Groups {
		ruleGroup := v1beta1.AlertRuleGroup{
			Name:     group.Name,
			Interval: group.Interval,
		}
		
		for _, rule := range group.Rules {
			ruleGroup.Rules = append(ruleGroup.Rules, v1beta1.AlertRule{
				Alert:       rule.Alert,
				Expr:        rule.Expr,
				For:         rule.For,
				Labels:      rule.Labels,
				Annotations: rule.Annotations,
			})
		}
		
		result.Groups = append(result.Groups, ruleGroup)
	}
	
	return result
}

func convertAlertingRuleFromV1Beta1(src v1beta1.AlertingRule) AlertingRule {
	result := AlertingRule{
		Name: src.Name,
	}
	
	for _, group := range src.Groups {
		ruleGroup := AlertRuleGroup{
			Name:     group.Name,
			Interval: group.Interval,
		}
		
		for _, rule := range group.Rules {
			ruleGroup.Rules = append(ruleGroup.Rules, AlertRule{
				Alert:       rule.Alert,
				Expr:        rule.Expr,
				For:         rule.For,
				Labels:      rule.Labels,
				Annotations: rule.Annotations,
			})
		}
		
		result.Groups = append(result.Groups, ruleGroup)
	}
	
	return result
}

// Backup destination conversion

func convertBackupDestinationToV1Beta1(src BackupDestination) v1beta1.BackupDestination {
	result := v1beta1.BackupDestination{
		Type: src.Type,
	}
	
	if src.S3 != nil {
		result.S3 = &v1beta1.S3Config{
			Enabled:         src.S3.Enabled,
			BucketName:      src.S3.BucketName,
			Region:          src.S3.Region,
			Endpoint:        src.S3.Endpoint,
			AccessKeyID:     src.S3.AccessKeyID,
			SecretAccessKey: src.S3.SecretAccessKey,
		}
	}
	
	if src.GCS != nil {
		result.GCS = &v1beta1.GCSConfig{
			BucketName:        src.GCS.BucketName,
			ServiceAccountKey: src.GCS.ServiceAccountKey,
		}
	}
	
	if src.Azure != nil {
		result.Azure = &v1beta1.AzureConfig{
			ContainerName:    src.Azure.ContainerName,
			StorageAccount:   src.Azure.StorageAccount,
			StorageAccessKey: src.Azure.StorageAccessKey,
		}
	}
	
	return result
}

func convertBackupDestinationFromV1Beta1(src v1beta1.BackupDestination) BackupDestination {
	result := BackupDestination{
		Type: src.Type,
	}
	
	if src.S3 != nil {
		result.S3 = &S3Config{
			Enabled:         src.S3.Enabled,
			BucketName:      src.S3.BucketName,
			Region:          src.S3.Region,
			Endpoint:        src.S3.Endpoint,
			AccessKeyID:     src.S3.AccessKeyID,
			SecretAccessKey: src.S3.SecretAccessKey,
		}
	}
	
	if src.GCS != nil {
		result.GCS = &GCSConfig{
			BucketName:        src.GCS.BucketName,
			ServiceAccountKey: src.GCS.ServiceAccountKey,
		}
	}
	
	if src.Azure != nil {
		result.Azure = &AzureConfig{
			ContainerName:    src.Azure.ContainerName,
			StorageAccount:   src.Azure.StorageAccount,
			StorageAccessKey: src.Azure.StorageAccessKey,
		}
	}
	
	return result
}

// Status conversion

func convertObservabilityPlatformStatusToV1Beta1(src *ObservabilityPlatformStatus, dst *v1beta1.ObservabilityPlatformStatus) error {
	dst.Phase = src.Phase
	dst.Message = src.Message
	dst.ObservedGeneration = src.ObservedGeneration
	dst.LastReconcileTime = src.LastReconcileTime
	dst.Conditions = src.Conditions
	dst.Endpoints = src.Endpoints
	
	// Convert ComponentStatus
	if src.ComponentStatus != nil {
		dst.ComponentStatus = make(map[string]v1beta1.ComponentStatus)
		for k, v := range src.ComponentStatus {
			dst.ComponentStatus[k] = v1beta1.ComponentStatus{
				Phase:      v.Phase,
				Version:    v.Version,
				Ready:      v.ReadyReplicas,
				Message:    "",
				LastUpdate: v.LastUpdateTime,
			}
		}
	}
	
	// Set default health status
	dst.Health = v1beta1.HealthStatus{
		Status: "Unknown",
	}
	
	return nil
}

func convertObservabilityPlatformStatusFromV1Beta1(src *v1beta1.ObservabilityPlatformStatus, dst *ObservabilityPlatformStatus) error {
	dst.Phase = src.Phase
	dst.Message = src.Message
	dst.ObservedGeneration = src.ObservedGeneration
	dst.LastReconcileTime = src.LastReconcileTime
	dst.Conditions = src.Conditions
	dst.Endpoints = src.Endpoints
	
	// Convert ComponentStatus
	if src.ComponentStatus != nil {
		dst.ComponentStatus = make(map[string]ComponentStatus)
		for k, v := range src.ComponentStatus {
			dst.ComponentStatus[k] = ComponentStatus{
				Phase:          v.Phase,
				Version:        v.Version,
				ReadyReplicas:  v.Ready,
				LastUpdateTime: v.LastUpdate,
			}
		}
	}
	
	// Note: Version and Health are lost in conversion
	
	return nil
}
