# Cloud Native Maturity Dashboard Configuration
## Gunj Operator Project

**Version**: 1.0  
**Date**: June 12, 2025  
**Purpose**: Grafana dashboard configuration for maturity metrics  

---

## Dashboard Overview

This dashboard provides real-time visibility into the Cloud Native Maturity level of the Gunj Operator project.

### Key Metrics Tracked

1. **Overall Maturity Score** (0-115 points)
2. **Maturity Level** (0-5)
3. **Score by Category**
   - Containerization (Level 1)
   - Orchestration (Level 2)
   - Microservices (Level 3)
   - Cloud Native Services (Level 4)
   - Cloud Native Operations (Level 5)
4. **Trend Analysis**
5. **Gap Identification**

---

## Grafana Dashboard JSON

```json
{
  "dashboard": {
    "title": "Cloud Native Maturity - Gunj Operator",
    "uid": "gunj-maturity",
    "version": 1,
    "timezone": "browser",
    "panels": [
      {
        "id": 1,
        "gridPos": {"h": 8, "w": 6, "x": 0, "y": 0},
        "type": "stat",
        "title": "Current Maturity Level",
        "targets": [
          {
            "expr": "gunj_operator_maturity_level"
          }
        ],
        "options": {
          "graphMode": "area",
          "colorMode": "value",
          "mappings": [
            {"value": 0, "text": "Traditional"},
            {"value": 1, "text": "Containerized"},
            {"value": 2, "text": "Orchestrated"},
            {"value": 3, "text": "Microservices"},
            {"value": 4, "text": "Cloud Native"},
            {"value": 5, "text": "Operations"}
          ]
        }
      },
      {
        "id": 2,
        "gridPos": {"h": 8, "w": 6, "x": 6, "y": 0},
        "type": "gauge",
        "title": "Overall Score",
        "targets": [
          {
            "expr": "gunj_operator_maturity_score_total"
          }
        ],
        "options": {
          "max": 115,
          "thresholds": {
            "steps": [
              {"value": 0, "color": "red"},
              {"value": 30, "color": "orange"},
              {"value": 50, "color": "yellow"},
              {"value": 70, "color": "light-green"},
              {"value": 90, "color": "green"}
            ]
          }
        }
      },
      {
        "id": 3,
        "gridPos": {"h": 8, "w": 12, "x": 12, "y": 0},
        "type": "bargauge",
        "title": "Score by Level",
        "targets": [
          {
            "expr": "gunj_operator_maturity_score_by_level",
            "legendFormat": "{{level}}"
          }
        ]
      },
      {
        "id": 4,
        "gridPos": {"h": 10, "w": 24, "x": 0, "y": 8},
        "type": "graph",
        "title": "Maturity Trend",
        "targets": [
          {
            "expr": "gunj_operator_maturity_score_total",
            "legendFormat": "Total Score"
          }
        ],
        "yaxes": [
          {"max": 115, "min": 0}
        ]
      },
      {
        "id": 5,
        "gridPos": {"h": 8, "w": 12, "x": 0, "y": 18},
        "type": "table",
        "title": "Gap Analysis",
        "targets": [
          {
            "expr": "gunj_operator_maturity_gaps"
          }
        ]
      },
      {
        "id": 6,
        "gridPos": {"h": 8, "w": 12, "x": 12, "y": 18},
        "type": "piechart",
        "title": "Completion by Level",
        "targets": [
          {
            "expr": "gunj_operator_maturity_completion_by_level"
          }
        ]
      }
    ]
  }
}
```

---

## Prometheus Metrics

### Metric Definitions

```yaml
# Maturity level (0-5)
gunj_operator_maturity_level:
  type: gauge
  help: Current cloud native maturity level
  labels:
    - project: "gunj-operator"

# Total score
gunj_operator_maturity_score_total:
  type: gauge
  help: Total maturity assessment score out of 115
  labels:
    - project: "gunj-operator"

# Score by level
gunj_operator_maturity_score_by_level:
  type: gauge
  help: Maturity score broken down by level
  labels:
    - project: "gunj-operator"
    - level: "1|2|3|4|5"
    - level_name: "containerized|orchestrated|microservices|cloud_native|operations"

# Completion percentage by level
gunj_operator_maturity_completion_by_level:
  type: gauge
  help: Percentage completion for each maturity level
  labels:
    - project: "gunj-operator"
    - level: "1|2|3|4|5"

# Gap count
gunj_operator_maturity_gaps_total:
  type: gauge
  help: Number of identified gaps in maturity
  labels:
    - project: "gunj-operator"
    - level: "1|2|3|4|5"
    - severity: "critical|high|medium|low"

# Assessment timestamp
gunj_operator_maturity_last_assessment:
  type: gauge
  help: Timestamp of last maturity assessment
  labels:
    - project: "gunj-operator"
```

### Metrics Exporter

```go
package metrics

import (
    "github.com/prometheus/client_golang/prometheus"
    "github.com/prometheus/client_golang/prometheus/promauto"
)

var (
    // Maturity level gauge
    maturityLevel = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "gunj_operator_maturity_level",
            Help: "Current cloud native maturity level (0-5)",
        },
        []string{"project"},
    )

    // Total score gauge
    maturityScoreTotal = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "gunj_operator_maturity_score_total",
            Help: "Total maturity assessment score",
        },
        []string{"project"},
    )

    // Score by level
    maturityScoreByLevel = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "gunj_operator_maturity_score_by_level",
            Help: "Maturity score broken down by level",
        },
        []string{"project", "level", "level_name"},
    )

    // Completion percentage
    maturityCompletion = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "gunj_operator_maturity_completion_by_level",
            Help: "Percentage completion for each maturity level",
        },
        []string{"project", "level"},
    )

    // Gaps counter
    maturityGaps = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "gunj_operator_maturity_gaps_total",
            Help: "Number of identified gaps in maturity",
        },
        []string{"project", "level", "severity"},
    )
)

// UpdateMaturityMetrics updates all maturity-related metrics
func UpdateMaturityMetrics(assessment MaturityAssessment) {
    project := "gunj-operator"
    
    // Update level
    maturityLevel.WithLabelValues(project).Set(float64(assessment.Level))
    
    // Update total score
    maturityScoreTotal.WithLabelValues(project).Set(float64(assessment.TotalScore))
    
    // Update scores by level
    for level, score := range assessment.LevelScores {
        maturityScoreByLevel.WithLabelValues(
            project,
            level,
            getLevelName(level),
        ).Set(float64(score))
    }
    
    // Update completion percentages
    for level, percentage := range assessment.CompletionPercentages {
        maturityCompletion.WithLabelValues(
            project,
            level,
        ).Set(percentage)
    }
    
    // Update gaps
    for level, gaps := range assessment.GapsByLevel {
        for severity, count := range gaps {
            maturityGaps.WithLabelValues(
                project,
                level,
                severity,
            ).Set(float64(count))
        }
    }
}
```

---

## Alert Rules

```yaml
groups:
  - name: maturity_alerts
    interval: 1h
    rules:
      # Alert if maturity score drops
      - alert: MaturityScoreDecreased
        expr: |
          delta(gunj_operator_maturity_score_total[1h]) < -5
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Maturity score decreased by {{ $value }} points"
          description: "The cloud native maturity score has decreased, indicating potential regression"
      
      # Alert if below minimum threshold
      - alert: MaturityBelowThreshold
        expr: |
          gunj_operator_maturity_score_total < 30
        for: 1h
        labels:
          severity: critical
        annotations:
          summary: "Maturity score critically low"
          description: "Score is {{ $value }}/115, below minimum threshold of 30"
      
      # Alert on critical gaps
      - alert: CriticalMaturityGaps
        expr: |
          gunj_operator_maturity_gaps_total{severity="critical"} > 0
        for: 30m
        labels:
          severity: high
        annotations:
          summary: "Critical gaps in cloud native maturity"
          description: "{{ $value }} critical gaps identified at level {{ $labels.level }}"
```

---

## Integration with CI/CD

### Jenkins Pipeline

```groovy
pipeline {
    agent any
    
    stages {
        stage('Maturity Assessment') {
            steps {
                script {
                    sh './hack/maturity-assessment.sh'
                    
                    def assessment = readJSON file: 'maturity-assessment-report.json'
                    def score = assessment.scores.total
                    def level = assessment.maturityLevel
                    
                    // Update metrics
                    pushgateway(
                        url: "${PROMETHEUS_PUSHGATEWAY}",
                        job: 'maturity-assessment',
                        data: """
                            gunj_operator_maturity_level{project="gunj-operator"} ${assessment.scores.percentage / 20}
                            gunj_operator_maturity_score_total{project="gunj-operator"} ${score}
                        """
                    )
                    
                    // Add badge to README
                    addBadge(
                        icon: 'graph.png',
                        text: "Maturity: ${level}",
                        link: "${BUILD_URL}maturity-report.html"
                    )
                    
                    // Fail if below threshold
                    if (score < 30) {
                        error("Maturity score ${score} is below minimum threshold")
                    }
                }
            }
        }
    }
}
```

### GitLab CI

```yaml
maturity-assessment:
  stage: test
  script:
    - chmod +x hack/maturity-assessment.sh
    - ./hack/maturity-assessment.sh
    - python scripts/generate-maturity-report.py
  artifacts:
    reports:
      junit: maturity-assessment-report.xml
    paths:
      - maturity-report.html
      - maturity-report.md
    expire_in: 30 days
  rules:
    - if: $CI_COMMIT_BRANCH == $CI_DEFAULT_BRANCH
    - if: $CI_PIPELINE_SOURCE == "merge_request_event"
```

---

## Visualization Examples

### 1. Maturity Radar Chart
Shows score distribution across all five levels in a spider/radar chart format.

### 2. Progress Timeline
Displays maturity progression over time with milestone markers.

### 3. Gap Heatmap
Visualizes missing requirements across levels with severity indicators.

### 4. Completion Donut
Shows percentage completion for current maturity level.

### 5. Trend Line Graph
Tracks total score changes over last 90 days with regression line.

---

## Dashboard Access

### Local Development
```bash
# Start Grafana locally
docker run -d \
  -p 3000:3000 \
  --name=grafana \
  -v grafana-storage:/var/lib/grafana \
  grafana/grafana

# Import dashboard
curl -X POST http://admin:admin@localhost:3000/api/dashboards/db \
  -H "Content-Type: application/json" \
  -d @maturity-dashboard.json
```

### Production URL
- **Dashboard**: https://grafana.gunj-operator.io/d/gunj-maturity
- **Public Snapshot**: https://grafana.gunj-operator.io/s/maturity-public

---

*This dashboard configuration will be automatically deployed as part of the Gunj Operator observability stack.*
