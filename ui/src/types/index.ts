// Core types for the Gunj Operator UI

export interface Platform {
  id: string;
  metadata: {
    name: string;
    namespace: string;
    uid: string;
    createdAt: string;
    updatedAt: string;
    version: string;
    labels?: Record<string, string>;
    annotations?: Record<string, string>;
  };
  spec: PlatformSpec;
  status: PlatformStatus;
}

export interface PlatformSpec {
  components: {
    prometheus?: PrometheusSpec;
    grafana?: GrafanaSpec;
    loki?: LokiSpec;
    tempo?: TempoSpec;
  };
  global?: GlobalConfig;
  alerting?: AlertingConfig;
}

export interface PrometheusSpec {
  enabled: boolean;
  version: string;
  replicas?: number;
  resources?: ResourceRequirements;
  storage?: StorageConfig;
  retention?: string;
  externalLabels?: Record<string, string>;
  remoteWrite?: RemoteWriteConfig[];
}

export interface GrafanaSpec {
  enabled: boolean;
  version: string;
  adminPassword?: string;
  ingress?: IngressConfig;
  resources?: ResourceRequirements;
  persistence?: StorageConfig;
}

export interface LokiSpec {
  enabled: boolean;
  version: string;
  storage?: StorageConfig;
  s3?: S3Config;
  resources?: ResourceRequirements;
  retention?: RetentionConfig;
}

export interface TempoSpec {
  enabled: boolean;
  version: string;
  storage?: StorageConfig;
  resources?: ResourceRequirements;
}

export interface GlobalConfig {
  externalLabels?: Record<string, string>;
  logLevel?: 'debug' | 'info' | 'warn' | 'error';
}

export interface ResourceRequirements {
  requests?: {
    cpu?: string;
    memory?: string;
  };
  limits?: {
    cpu?: string;
    memory?: string;
  };
}

export interface StorageConfig {
  size: string;
  storageClassName?: string;
}

export interface IngressConfig {
  enabled: boolean;
  host: string;
  tlsSecret?: string;
  annotations?: Record<string, string>;
}

export interface S3Config {
  enabled: boolean;
  bucketName: string;
  region: string;
  endpoint?: string;
}

export interface RetentionConfig {
  days: number;
  compactInterval?: string;
}

export interface AlertingConfig {
  alertmanager?: {
    enabled: boolean;
    config: any; // AlertManager config
  };
}

export interface RemoteWriteConfig {
  url: string;
  remoteTimeout?: string;
  headers?: Record<string, string>;
}

export interface PlatformStatus {
  phase: 'Pending' | 'Installing' | 'Ready' | 'Failed' | 'Upgrading';
  message?: string;
  conditions: Condition[];
  componentStatuses?: Record<string, ComponentStatus>;
  lastReconciled?: string;
  observedGeneration?: number;
}

export interface Condition {
  type: string;
  status: 'True' | 'False' | 'Unknown';
  lastTransitionTime: string;
  reason?: string;
  message?: string;
}

export interface ComponentStatus {
  ready: boolean;
  message?: string;
  version?: string;
  endpoints?: string[];
}

// Filter and sorting types
export interface PlatformFilters {
  namespace: string | null;
  status: string | null;
  search: string;
  labels?: Record<string, string>;
}

export interface SortCriteria {
  field: 'name' | 'namespace' | 'status' | 'createdAt' | 'updatedAt';
  direction: 'asc' | 'desc';
}

// WebSocket types
export interface WebSocketMessage {
  type: string;
  channel?: string;
  data?: any;
}

export interface PlatformEvent {
  type: 'platform.created' | 'platform.updated' | 'platform.deleted' | 'platform.status.changed';
  data: any;
  timestamp: string;
}

export interface MetricsEvent {
  platformId: string;
  component: string;
  data: MetricData[];
}

export interface MetricData {
  name: string;
  value: number;
  timestamp: number;
  labels?: Record<string, string>;
}

export interface AlertEvent {
  data: Alert;
}

export interface Alert {
  id: string;
  name: string;
  severity: 'critical' | 'warning' | 'info';
  description: string;
  platformId: string;
  component: string;
  startsAt: string;
  endsAt?: string;
  labels: Record<string, string>;
}

// API types
export interface LoginCredentials {
  email: string;
  password: string;
}

export interface CreatePlatformData {
  name: string;
  namespace: string;
  spec: PlatformSpec;
  labels?: Record<string, string>;
}

export interface PlatformUpdate {
  type: 'created' | 'updated' | 'deleted';
  platform?: Platform;
  platformId?: string;
}

// UI State types
export interface User {
  id: string;
  email: string;
  name: string;
  roles: string[];
  permissions: string[];
  avatarUrl?: string;
}

export interface Theme {
  mode: 'light' | 'dark' | 'system';
  primaryColor?: string;
}

export interface Notification {
  id: string;
  type: 'success' | 'error' | 'warning' | 'info';
  message: string;
  description?: string;
  duration?: number;
  action?: {
    label: string;
    onClick: () => void;
  };
  timestamp: number;
}