#!/usr/bin/env python3
"""
Cloud Native Maturity Report Generator
Gunj Operator Project
Version: 1.0
"""

import json
import datetime
import os
from typing import Dict, List, Tuple

class MaturityReportGenerator:
    def __init__(self, assessment_file: str = "maturity-assessment-report.json"):
        self.assessment_file = assessment_file
        self.report_data = self._load_assessment_data()
        
    def _load_assessment_data(self) -> Dict:
        """Load assessment data from JSON file"""
        if os.path.exists(self.assessment_file):
            with open(self.assessment_file, 'r') as f:
                return json.load(f)
        else:
            # Return default data if file doesn't exist
            return {
                "timestamp": datetime.datetime.utcnow().isoformat() + "Z",
                "project": "gunj-operator",
                "scores": {
                    "total": 0,
                    "percentage": 0,
                    "level1": 0,
                    "level2": 0,
                    "level3": 0,
                    "level4": 0,
                    "level5": 0
                },
                "maturityLevel": "0 - Traditional"
            }
    
    def generate_html_report(self) -> str:
        """Generate HTML report with visualizations"""
        html_template = """
<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Cloud Native Maturity Report - Gunj Operator</title>
    <script src="https://cdn.jsdelivr.net/npm/chart.js"></script>
    <style>
        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            margin: 0;
            padding: 20px;
            background-color: #f5f5f5;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            background-color: white;
            padding: 30px;
            border-radius: 10px;
            box-shadow: 0 2px 10px rgba(0,0,0,0.1);
        }
        h1 {
            color: #333;
            margin-bottom: 10px;
        }
        .timestamp {
            color: #666;
            font-size: 14px;
            margin-bottom: 30px;
        }
        .score-summary {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin-bottom: 40px;
        }
        .score-card {
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
            padding: 20px;
            border-radius: 8px;
            text-align: center;
        }
        .score-card h3 {
            margin: 0 0 10px 0;
            font-size: 16px;
            opacity: 0.9;
        }
        .score-value {
            font-size: 36px;
            font-weight: bold;
        }
        .maturity-level {
            background: #f0f0f0;
            padding: 20px;
            border-radius: 8px;
            text-align: center;
            margin-bottom: 30px;
        }
        .maturity-level h2 {
            margin: 0 0 10px 0;
            color: #333;
        }
        .current-level {
            font-size: 24px;
            color: #667eea;
            font-weight: bold;
        }
        .charts-container {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 30px;
            margin-bottom: 30px;
        }
        .chart-box {
            background: #f9f9f9;
            padding: 20px;
            border-radius: 8px;
        }
        .recommendations {
            background: #e8f5e9;
            padding: 20px;
            border-radius: 8px;
            border-left: 4px solid #4caf50;
        }
        .recommendations h3 {
            margin-top: 0;
            color: #2e7d32;
        }
        .recommendations ul {
            margin: 10px 0;
            padding-left: 20px;
        }
        .recommendations li {
            margin-bottom: 8px;
        }
        canvas {
            max-height: 300px;
        }
        .progress-bar {
            width: 100%;
            height: 30px;
            background-color: #e0e0e0;
            border-radius: 15px;
            overflow: hidden;
            margin: 20px 0;
        }
        .progress-fill {
            height: 100%;
            background: linear-gradient(90deg, #4caf50 0%, #8bc34a 100%);
            transition: width 0.3s ease;
            display: flex;
            align-items: center;
            justify-content: center;
            color: white;
            font-weight: bold;
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>Cloud Native Maturity Assessment Report</h1>
        <div class="timestamp">Generated on: {timestamp}</div>
        
        <div class="maturity-level">
            <h2>Current Maturity Level</h2>
            <div class="current-level">{maturity_level}</div>
            <div class="progress-bar">
                <div class="progress-fill" style="width: {percentage}%">{percentage}%</div>
            </div>
        </div>
        
        <div class="score-summary">
            <div class="score-card">
                <h3>Total Score</h3>
                <div class="score-value">{total_score}</div>
            </div>
            <div class="score-card">
                <h3>Level 1: Containerized</h3>
                <div class="score-value">{level1_score}/20</div>
            </div>
            <div class="score-card">
                <h3>Level 2: Orchestrated</h3>
                <div class="score-value">{level2_score}/35</div>
            </div>
            <div class="score-card">
                <h3>Level 3: Microservices</h3>
                <div class="score-value">{level3_score}/25</div>
            </div>
            <div class="score-card">
                <h3>Level 4: Cloud Native</h3>
                <div class="score-value">{level4_score}/15</div>
            </div>
            <div class="score-card">
                <h3>Level 5: Operations</h3>
                <div class="score-value">{level5_score}/20</div>
            </div>
        </div>
        
        <div class="charts-container">
            <div class="chart-box">
                <h3>Score Distribution</h3>
                <canvas id="radarChart"></canvas>
            </div>
            <div class="chart-box">
                <h3>Progress by Level</h3>
                <canvas id="barChart"></canvas>
            </div>
        </div>
        
        <div class="recommendations">
            <h3>Recommendations</h3>
            <ul>
                {recommendations}
            </ul>
        </div>
    </div>
    
    <script>
        // Radar Chart
        const radarCtx = document.getElementById('radarChart').getContext('2d');
        new Chart(radarCtx, {{
            type: 'radar',
            data: {{
                labels: ['Containerization', 'Orchestration', 'Microservices', 'Cloud Native', 'Operations'],
                datasets: [{{
                    label: 'Current Score',
                    data: [{level1_pct}, {level2_pct}, {level3_pct}, {level4_pct}, {level5_pct}],
                    backgroundColor: 'rgba(102, 126, 234, 0.2)',
                    borderColor: 'rgba(102, 126, 234, 1)',
                    borderWidth: 2
                }}]
            }},
            options: {{
                scales: {{
                    r: {{
                        beginAtZero: true,
                        max: 100
                    }}
                }}
            }}
        }});
        
        // Bar Chart
        const barCtx = document.getElementById('barChart').getContext('2d');
        new Chart(barCtx, {{
            type: 'bar',
            data: {{
                labels: ['Level 1', 'Level 2', 'Level 3', 'Level 4', 'Level 5'],
                datasets: [{{
                    label: 'Score',
                    data: [{level1_score}, {level2_score}, {level3_score}, {level4_score}, {level5_score}],
                    backgroundColor: [
                        'rgba(76, 175, 80, 0.6)',
                        'rgba(33, 150, 243, 0.6)',
                        'rgba(255, 193, 7, 0.6)',
                        'rgba(156, 39, 176, 0.6)',
                        'rgba(255, 87, 34, 0.6)'
                    ],
                    borderColor: [
                        'rgba(76, 175, 80, 1)',
                        'rgba(33, 150, 243, 1)',
                        'rgba(255, 193, 7, 1)',
                        'rgba(156, 39, 176, 1)',
                        'rgba(255, 87, 34, 1)'
                    ],
                    borderWidth: 1
                }}, {{
                    label: 'Max Score',
                    data: [20, 35, 25, 15, 20],
                    backgroundColor: 'rgba(200, 200, 200, 0.3)',
                    borderColor: 'rgba(200, 200, 200, 1)',
                    borderWidth: 1
                }}]
            }},
            options: {{
                scales: {{
                    y: {{
                        beginAtZero: true
                    }}
                }}
            }}
        }});
    </script>
</body>
</html>
        """
        
        # Calculate percentages
        level1_pct = (self.report_data['scores']['level1'] / 20) * 100
        level2_pct = (self.report_data['scores']['level2'] / 35) * 100
        level3_pct = (self.report_data['scores']['level3'] / 25) * 100
        level4_pct = (self.report_data['scores']['level4'] / 15) * 100
        level5_pct = (self.report_data['scores']['level5'] / 20) * 100
        
        # Generate recommendations
        recommendations = self._generate_recommendations()
        recommendations_html = '\n'.join([f'<li>{rec}</li>' for rec in recommendations])
        
        # Format the template
        return html_template.format(
            timestamp=self.report_data['timestamp'],
            maturity_level=self.report_data['maturityLevel'],
            percentage=self.report_data['scores']['percentage'],
            total_score=self.report_data['scores']['total'],
            level1_score=self.report_data['scores']['level1'],
            level2_score=self.report_data['scores']['level2'],
            level3_score=self.report_data['scores']['level3'],
            level4_score=self.report_data['scores']['level4'],
            level5_score=self.report_data['scores']['level5'],
            level1_pct=round(level1_pct),
            level2_pct=round(level2_pct),
            level3_pct=round(level3_pct),
            level4_pct=round(level4_pct),
            level5_pct=round(level5_pct),
            recommendations=recommendations_html
        )
    
    def _generate_recommendations(self) -> List[str]:
        """Generate recommendations based on scores"""
        recommendations = []
        scores = self.report_data['scores']
        
        # Level 1 recommendations
        if scores['level1'] < 20:
            if scores['level1'] < 5:
                recommendations.append("Create a Dockerfile for containerizing the application")
            if scores['level1'] < 10:
                recommendations.append("Implement multi-stage builds to optimize container size")
            if scores['level1'] < 15:
                recommendations.append("Configure containers to run as non-root user")
            recommendations.append("Use minimal base images (distroless or alpine)")
        
        # Level 2 recommendations
        if scores['level2'] < 35 and scores['level1'] >= 15:
            if scores['level2'] < 10:
                recommendations.append("Define Custom Resource Definitions (CRDs)")
            if scores['level2'] < 20:
                recommendations.append("Implement Kubernetes controller pattern")
            recommendations.append("Create Helm charts for deployment")
            recommendations.append("Configure RBAC policies")
        
        # Level 3 recommendations
        if scores['level3'] < 25 and scores['level2'] >= 25:
            recommendations.append("Decompose application into microservices")
            recommendations.append("Implement service mesh for inter-service communication")
            recommendations.append("Add distributed tracing with OpenTelemetry")
        
        # Level 4 recommendations
        if scores['level4'] < 15 and scores['level3'] >= 15:
            recommendations.append("Integrate with cloud provider services (AWS/Azure/GCP)")
            recommendations.append("Implement multi-cloud support")
            recommendations.append("Add cost optimization features")
        
        # Level 5 recommendations
        if scores['level5'] < 20 and scores['level4'] >= 10:
            recommendations.append("Implement GitOps workflow")
            recommendations.append("Add comprehensive E2E testing")
            recommendations.append("Implement predictive scaling and self-healing")
        
        if not recommendations:
            recommendations.append("Excellent! Continue maintaining high standards and consider contributing to CNCF")
        
        return recommendations
    
    def generate_markdown_report(self) -> str:
        """Generate Markdown report"""
        scores = self.report_data['scores']
        recommendations = self._generate_recommendations()
        
        markdown = f"""# Cloud Native Maturity Assessment Report

**Project**: Gunj Operator  
**Generated**: {self.report_data['timestamp']}  
**Maturity Level**: {self.report_data['maturityLevel']}  

## Executive Summary

The Gunj Operator project has achieved a maturity score of **{scores['total']}/115** ({scores['percentage']}%), 
placing it at **{self.report_data['maturityLevel']}**.

## Score Breakdown

| Level | Description | Score | Max | Percentage |
|-------|-------------|-------|-----|------------|
| 1 | Containerization | {scores['level1']} | 20 | {(scores['level1']/20)*100:.0f}% |
| 2 | Dynamic Orchestration | {scores['level2']} | 35 | {(scores['level2']/35)*100:.0f}% |
| 3 | Microservices Oriented | {scores['level3']} | 25 | {(scores['level3']/25)*100:.0f}% |
| 4 | Cloud Native Services | {scores['level4']} | 15 | {(scores['level4']/15)*100:.0f}% |
| 5 | Cloud Native Operations | {scores['level5']} | 20 | {(scores['level5']/20)*100:.0f}% |
| **Total** | | **{scores['total']}** | **115** | **{scores['percentage']}%** |

## Recommendations

"""
        for i, rec in enumerate(recommendations, 1):
            markdown += f"{i}. {rec}\n"
        
        markdown += """
## Next Steps

1. Address the recommendations above in priority order
2. Re-run the assessment after implementing changes
3. Track progress over time using the maturity dashboard
4. Share results with stakeholders and team members

---

*This report was automatically generated by the Cloud Native Maturity Assessment tool.*
"""
        return markdown
    
    def save_reports(self):
        """Save HTML and Markdown reports"""
        # Save HTML report
        html_content = self.generate_html_report()
        with open('maturity-report.html', 'w') as f:
            f.write(html_content)
        
        # Save Markdown report
        markdown_content = self.generate_markdown_report()
        with open('maturity-report.md', 'w') as f:
            f.write(markdown_content)
        
        print("Reports generated successfully!")
        print("- HTML Report: maturity-report.html")
        print("- Markdown Report: maturity-report.md")

if __name__ == "__main__":
    generator = MaturityReportGenerator()
    generator.save_reports()
