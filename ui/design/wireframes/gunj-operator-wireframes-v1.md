# Gunj Operator UI Wireframes & Screen Designs

**Version**: 1.0  
**Last Updated**: June 12, 2025  
**Status**: Design Phase  
**Designer**: AI-Assisted Design Process  

## 📋 Overview

This document presents the low-fidelity wireframes for all key screens in the Gunj Operator UI. Each wireframe focuses on layout, information architecture, and user flow rather than visual design details.

---

## 🖼️ Screen Inventory

### Core Screens
1. **Dashboard/Overview** - Main landing page with system health
2. **Platform List** - All deployed observability platforms
3. **Platform Details** - Individual platform management
4. **Platform Creation Wizard** - Step-by-step platform setup
5. **Component Configuration** - Detailed component settings
6. **Monitoring Dashboard** - Metrics and health visualization
7. **Logs Viewer** - Centralized log exploration
8. **Traces Explorer** - Distributed tracing interface
9. **Alerts Manager** - Alert rules and notifications
10. **Settings** - System and user preferences

### Additional Screens
11. **Login/Authentication** - SSO and standard login
12. **User Profile** - Account management
13. **Team Management** - RBAC and permissions
14. **API Keys** - API access management
15. **Audit Logs** - System activity tracking

---

## 📐 Wireframe Notation Guide

```
┌─────────────┐  Border/Container
│             │  
└─────────────┘  

[Button]         Primary action button
(Button)         Secondary action button
<Input Field>    Text input
▼ Dropdown       Select/dropdown menu
○ Radio          Radio button
☐ Checkbox       Checkbox
☑ Checked        Checked state
● Selected       Selected radio

[Tab 1] [Tab 2]  Tab navigation
━━━━━━━━━━━━━    Divider line
░░░░░░░░░░░░░    Loading/skeleton state
▲ ▼ ◀ ▶         Navigation arrows
⋮                More/menu icon
✕                Close icon
≡                Hamburger menu
🔍               Search icon
```

---

## 🎨 1. Dashboard/Overview Screen

### Desktop Layout (1440px)
```
┌─────────────────────────────────────────────────────────────────────────┐
│ ≡  Gunj Operator                    🔍 Search...        🔔 👤 John Doe ▼ │
├─────┬───────────────────────────────────────────────────────────────────┤
│     │                                                                     │
│  S  │  Welcome back, John! Here's your observability overview.          │
│  I  │                                                                     │
│  D  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ │
│  E  │  │ PLATFORMS   │ │ COMPONENTS  │ │ ALERTS      │ │ RESOURCES   │ │
│  B  │  │    12       │ │    48       │ │    3        │ │ CPU: 45%    │ │
│  A  │  │ ▲ 2 Active  │ │ ▲ 45 Healthy│ │ ▼ 2 Critical│ │ MEM: 62%    │ │
│  R  │  └─────────────┘ └─────────────┘ └─────────────┘ └─────────────┘ │
│     │                                                                     │
│  N  │  Platform Health Overview                     Quick Actions         │
│  A  │  ┌────────────────────────────────┐  ┌─────────────────────────┐ │
│  V  │  │ ████████████████████████ 85%   │  │ [+ Create Platform]     │ │
│     │  │ ████████░░░░░░░░░░░░░░░  15%   │  │ [⚙ Configure]          │ │
│  •  │  │                                 │  │ [📊 View Metrics]      │ │
│  D  │  │ ● Healthy (10)  ● Issues (2)   │  │ [📋 Documentation]     │ │
│  a  │  └────────────────────────────────┘  └─────────────────────────┘ │
│  s  │                                                                     │
│  h  │  Recent Activity                        System Metrics             │
│  b  │  ┌────────────────────────────────┐  ┌─────────────────────────┐ │
│  o  │  │ 10:23 Platform 'prod' updated  │  │     CPU Usage           │ │
│  a  │  │ 10:15 Alert resolved in 'dev'  │  │ ╱╲    ╱╲    ╱╲╱╲      │ │
│  r  │  │ 09:45 New version available    │  │╱  ╲__╱  ╲__╱            │ │
│  d  │  │ 09:30 Backup completed         │  │ 0%                 100% │ │
│     │  │ [View All Activity]            │  └─────────────────────────┘ │
│     │  └────────────────────────────────┘                               │
└─────┴───────────────────────────────────────────────────────────────────┘
```

### Mobile Layout (375px)
```
┌─────────────────────┐
│ ≡  Gunj Operator  🔍 │
├─────────────────────┤
│ Welcome back!       │
│                     │
│ ┌─────────┬────────┐│
│ │PLATFORMS│ALERTS  ││
│ │   12    │   3    ││
│ │ ▲ 2     │ ▼ 2    ││
│ └─────────┴────────┘│
│                     │
│ Platform Health     │
│ ┌──────────────────┐│
│ │ ████████████ 85% ││
│ │ ████░░░░░░░ 15% ││
│ │ ● OK  ● Issues   ││
│ └──────────────────┘│
│                     │
│ [+ Create Platform] │
│                     │
│ Recent Activity     │
│ ┌──────────────────┐│
│ │10:23 Platform... ││
│ │10:15 Alert...    ││
│ │09:45 New ver...  ││
│ └──────────────────┘│
└─────────────────────┘
```

---

## 📋 2. Platform List Screen

### Desktop Layout
```
┌─────────────────────────────────────────────────────────────────────────┐
│ ≡  Gunj Operator > Platforms        🔍 Search...       🔔 👤 John Doe ▼ │
├─────┬───────────────────────────────────────────────────────────────────┤
│  S  │  Observability Platforms                    [+ Create Platform]    │
│  I  │                                                                     │
│  D  │  🔍 <Search platforms...>  ▼ All Namespaces  ▼ All Status  [↻]   │
│  E  │                                                                     │
│  B  │  ┌─────────────────────────────────────────────────────────────┐ │
│  A  │  │ □ Name ↓    Namespace   Status    Version   Components   ⋮  │ │
│  R  │  ├─────────────────────────────────────────────────────────────┤ │
│     │  │ □ prod-platform                                              │ │
│  •  │  │    production   ● Ready    v2.48.0   P G L T A         ⋮  │ │
│     │  ├─────────────────────────────────────────────────────────────┤ │
│  P  │  │ □ staging-platform                                           │ │
│  l  │  │    staging      ● Ready    v2.48.0   P G L T           ⋮  │ │
│  a  │  ├─────────────────────────────────────────────────────────────┤ │
│  t  │  │ □ dev-platform                                               │ │
│  f  │  │    development  ◐ Updating v2.47.0   P G               ⋮  │ │
│  o  │  ├─────────────────────────────────────────────────────────────┤ │
│  r  │  │ □ test-platform                                              │ │
│  m  │  │    testing      ● Issues   v2.48.0   P G L T A         ⋮  │ │
│  s  │  └─────────────────────────────────────────────────────────────┘ │
│     │                                                                     │
│     │  Showing 4 of 12 platforms          [1] 2 3 ... 6  Rows: ▼ 10    │
│     │                                                                     │
│     │  Bulk Actions: [Delete] [Export] [Update]  (4 selected)          │
└─────┴───────────────────────────────────────────────────────────────────┘

Legend: P=Prometheus G=Grafana L=Loki T=Tempo A=Alertmanager
```

### Mobile Card View
```
┌─────────────────────┐
│ ≡  Platforms    [+] │
├─────────────────────┤
│ 🔍 <Search...>      │
│ ▼ All    ▼ All     │
├─────────────────────┤
│ ┌──────────────────┐│
│ │ prod-platform    ││
│ │ ● Ready          ││
│ │ production       ││
│ │ v2.48.0          ││
│ │ P G L T A       ││
│ │ [View] [⋮]      ││
│ └──────────────────┘│
│                     │
│ ┌──────────────────┐│
│ │ staging-platform ││
│ │ ● Ready          ││
│ │ staging          ││
│ │ v2.48.0          ││
│ │ P G L T          ││
│ │ [View] [⋮]      ││
│ └──────────────────┘│
└─────────────────────┘
```

---

## 🧙 3. Platform Creation Wizard

### Step 1: Basic Information
```
┌─────────────────────────────────────────────────────────────────────────┐
│ ≡  Gunj Operator > Create Platform                    🔔 👤 John Doe ▼ │
├─────┬───────────────────────────────────────────────────────────────────┤
│  S  │  Create New Observability Platform                                 │
│  I  │                                                                     │
│  D  │  [1. Basic] → 2. Components → 3. Configuration → 4. Review        │
│  E  │  ━━━━━━━━━━   ─ ─ ─ ─ ─ ─    ─ ─ ─ ─ ─ ─ ─ ─    ─ ─ ─ ─        │
│  B  │                                                                     │
│  A  │  Platform Information                                               │
│  R  │  ┌─────────────────────────────────────────────────────────────┐ │
│     │  │ Platform Name *                                               │ │
│     │  │ <my-observability-platform>                                   │ │
│  C  │  │                                                               │ │
│  r  │  │ Namespace *                    Environment                    │ │
│  e  │  │ ▼ Select namespace             ▼ Select environment          │ │
│  a  │  │                                                               │ │
│  t  │  │ Description                                                   │ │
│  e  │  │ <Describe your platform purpose...>                           │ │
│     │  │                                                               │ │
│     │  │ Labels                                                        │ │
│     │  │ Key              Value                                 [+Add] │ │
│     │  │ <team>           <platform-team>                        [✕]  │ │
│     │  │ <environment>    <production>                           [✕]  │ │
│     │  └─────────────────────────────────────────────────────────────┘ │
│     │                                                                     │
│     │                                      (Cancel)  [Next: Components] │
└─────┴───────────────────────────────────────────────────────────────────┘
```

### Step 2: Component Selection
```
┌─────────────────────────────────────────────────────────────────────────┐
│  Create New Observability Platform                                       │
│                                                                          │
│  1. Basic → [2. Components] → 3. Configuration → 4. Review             │
│  ━━━━━━━━    ━━━━━━━━━━━━━    ─ ─ ─ ─ ─ ─ ─ ─    ─ ─ ─ ─             │
│                                                                          │
│  Select Components                                                       │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │ ☑ Prometheus          ☑ Grafana            ☑ Loki              │    │
│  │   Metrics collection    Visualization        Log aggregation    │    │
│  │   ▼ v2.48.0            ▼ v10.2.0           ▼ v2.9.0           │    │
│  │                                                                 │    │
│  │ ☑ Tempo               ☑ Alertmanager       ☐ OpenTelemetry     │    │
│  │   Distributed tracing   Alert routing        Collector          │    │
│  │   ▼ v2.3.0            ▼ v0.26.0           ▼ v0.96.0          │    │
│  └────────────────────────────────────────────────────────────────┘    │
│                                                                          │
│  Presets: [Minimal] [Standard] [Full Stack] [Custom]                   │
│                                                                          │
│                          [← Back]  (Cancel)  [Next: Configuration]      │
└──────────────────────────────────────────────────────────────────────────┘
```

### Step 3: Configuration
```
┌─────────────────────────────────────────────────────────────────────────┐
│  Create New Observability Platform                                       │
│                                                                          │
│  1. Basic → 2. Components → [3. Configuration] → 4. Review             │
│  ━━━━━━━━    ━━━━━━━━━━━━    ━━━━━━━━━━━━━━━━    ─ ─ ─ ─             │
│                                                                          │
│  Component Configuration                                                 │
│  ┌────────────────────────────────────────────────────────────────┐    │
│  │ [Prometheus] [Grafana] [Loki] [Tempo] [Alertmanager]           │    │
│  │                                                                 │    │
│  │ Prometheus Configuration                                        │    │
│  │ ┌─────────────────────────────────────────────────────────┐   │    │
│  │ │ Resources                    Storage                      │   │    │
│  │ │ CPU Request  <1000m>        Size         <100Gi>        │   │    │
│  │ │ CPU Limit    <2000m>        Storage Class ▼ fast-ssd    │   │    │
│  │ │ Memory Req   <4Gi>          Retention     <30d>         │   │    │
│  │ │ Memory Limit <8Gi>                                       │   │    │
│  │ │                                                          │   │    │
│  │ │ High Availability           Advanced                     │   │    │
│  │ │ ☑ Enable HA (3 replicas)   [Configure Advanced →]       │   │    │
│  │ └─────────────────────────────────────────────────────────┘   │    │
│  └────────────────────────────────────────────────────────────────┘    │
│                                                                          │
│                          [← Back]  (Cancel)  [Next: Review & Create]    │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## 📊 4. Platform Details Screen

### Desktop Layout
```
┌─────────────────────────────────────────────────────────────────────────┐
│ ≡  Gunj Operator > Platforms > prod-platform         🔔 👤 John Doe ▼  │
├─────┬───────────────────────────────────────────────────────────────────┤
│  S  │  prod-platform                                    [Edit] [Delete] │
│  I  │  ● Ready  |  production  |  Created: 2 days ago  |  v2.48.0      │
│  D  │                                                                    │
│  E  │  [Overview] [Components] [Monitoring] [Logs] [Configuration]      │
│  B  │  ━━━━━━━━━━                                                       │
│  A  │                                                                    │
│  R  │  Platform Health                     Resource Usage               │
│     │  ┌───────────────────────┐  ┌─────────────────────────────────┐ │
│  •  │  │ All Systems           │  │ CPU            Memory           │ │
│  D  │  │ ● Operational         │  │ ▓▓▓▓▓▓░░ 75%  ▓▓▓▓▓▓▓░ 82%   │ │
│  e  │  │                       │  │ 6/8 cores      26/32 GB        │ │
│  t  │  │ 5/5 Components Active │  │                                 │ │
│  a  │  │ 0 Active Alerts       │  │ Storage        Network I/O      │ │
│  i  │  │ 99.9% Uptime (30d)    │  │ ▓▓▓░░░░░ 45%  ↑150 MB/s       │ │
│  l  │  └───────────────────────┘  │ 45/100 GB      ↓75 MB/s        │ │
│  s  │                             └─────────────────────────────────┘ │
│     │                                                                    │
│     │  Components                                                        │
│     │  ┌─────────────────────────────────────────────────────────────┐ │
│     │  │ Component    Status   Version   Replicas  CPU    Memory   ⋮ │ │
│     │  ├─────────────────────────────────────────────────────────────┤ │
│     │  │ Prometheus   ● Ready  v2.48.0   3/3       1.5    6GB      ⋮ │ │
│     │  │ Grafana      ● Ready  v10.2.0   2/2       0.5    2GB      ⋮ │ │
│     │  │ Loki         ● Ready  v2.9.0    3/3       1.0    4GB      ⋮ │ │
│     │  │ Tempo        ● Ready  v2.3.0    2/2       0.8    3GB      ⋮ │ │
│     │  │ Alertmanager ● Ready  v0.26.0   3/3       0.2    1GB      ⋮ │ │
│     │  └─────────────────────────────────────────────────────────────┘ │
└─────┴───────────────────────────────────────────────────────────────────┘
```

---

## 📈 5. Monitoring Dashboard

### Desktop Layout
```
┌─────────────────────────────────────────────────────────────────────────┐
│ ≡  Gunj Operator > prod-platform > Monitoring        🔔 👤 John Doe ▼  │
├─────┬───────────────────────────────────────────────────────────────────┤
│  S  │  Monitoring Dashboard                    ▼ Last 1 hour  [↻ Auto] │
│  I  │                                                                    │
│  D  │  [System] [Application] [Business] [Custom]                       │
│  E  │  ━━━━━━━                                                          │
│  B  │                                                                    │
│  A  │  Key Metrics                                                       │
│  R  │  ┌─────────────┬─────────────┬─────────────┬─────────────┐      │
│     │  │ Request Rate│ Error Rate  │ Latency P95 │ Availability│      │
│  M  │  │ 1.2K req/s  │ 0.05%       │ 145ms       │ 99.99%      │      │
│  o  │  │ ▲ +15%      │ ▼ -0.02%    │ ▲ +12ms     │ → 0%        │      │
│  n  │  └─────────────┴─────────────┴─────────────┴─────────────┘      │
│  i  │                                                                    │
│  t  │  Service Health Map                                                │
│  o  │  ┌─────────────────────────────────────────────────────────────┐ │
│  r  │  │     [Frontend]                                                │ │
│     │  │         ↓ 800 req/s                                          │ │
│     │  │     [API Gateway] ←→ [Auth Service]                          │ │
│     │  │      ↓ 750 req/s      50 req/s                              │ │
│     │  │   [Product API] ←→ [User API]                               │ │
│     │  │    ↓ 400 req/s     350 req/s                                │ │
│     │  │     [Database]                                               │ │
│     │  │                                                              │ │
│     │  │  ● Healthy  ◐ Warning  ● Error                              │ │
│     │  └─────────────────────────────────────────────────────────────┘ │
│     │                                                                    │
│     │  Time Series Graphs                                               │
│     │  ┌─────────────────────────┬─────────────────────────┐          │
│     │  │ CPU Usage               │ Memory Usage            │          │
│     │  │    ╱╲    ╱╲             │     ___________         │          │
│     │  │ __╱  ╲__╱  ╲___         │ ___/            \_      │          │
│     │  │                         │                         │          │
│     │  └─────────────────────────┴─────────────────────────┘          │
└─────┴───────────────────────────────────────────────────────────────────┘
```

---

## 📜 6. Logs Viewer

### Desktop Layout
```
┌─────────────────────────────────────────────────────────────────────────┐
│ ≡  Gunj Operator > prod-platform > Logs              🔔 👤 John Doe ▼  │
├─────┬───────────────────────────────────────────────────────────────────┤
│  S  │  Log Explorer                                                      │
│  I  │                                                                    │
│  D  │  🔍 <Search logs...>                    ▼ Last 15 minutes  [↻]   │
│  E  │                                                                    │
│  B  │  Filters                                                           │
│  A  │  ┌─────────────────────────────────────────────────────────────┐ │
│  R  │  │ ▼ All Services  ▼ All Levels  ▼ All Namespaces  [+ Add]    │ │
│     │  │                                                               │ │
│  L  │  │ Quick Filters: [Errors] [Warnings] [Slow Queries] [5XX]     │ │
│  o  │  └─────────────────────────────────────────────────────────────┘ │
│  g  │                                                                    │
│  s  │  Log Stream                                          Live ● [||]  │
│     │  ┌─────────────────────────────────────────────────────────────┐ │
│     │  │ 10:23:45 [api-gateway] INFO   Request processed in 45ms     │ │
│     │  │ 10:23:44 [user-service] ERROR  Database connection failed   │ │
│     │  │   Stack trace:                                              │ │
│     │  │   at connectDB (db.js:45:12)                               │ │
│     │  │   at initialize (app.js:23:5)                              │ │
│     │  │ 10:23:43 [api-gateway] WARN   Rate limit approaching       │ │
│     │  │ 10:23:42 [product-api] INFO   Cache hit for product-123    │ │
│     │  │ 10:23:41 [auth-service] INFO  Token validated successfully │ │
│     │  └─────────────────────────────────────────────────────────────┘ │
│     │                                                                    │
│     │  [Export Logs] [Save Query] [Share]     Showing 5 of 1,247 logs │
└─────┴───────────────────────────────────────────────────────────────────┘
```

---

## 🔔 7. Alerts Manager

### Desktop Layout
```
┌─────────────────────────────────────────────────────────────────────────┐
│ ≡  Gunj Operator > Alerts                            🔔 👤 John Doe ▼  │
├─────┬───────────────────────────────────────────────────────────────────┤
│  S  │  Alert Management                               [+ Create Alert]  │
│  I  │                                                                    │
│  D  │  [Active Alerts (3)] [Alert Rules (45)] [Notifications] [History] │
│  E  │  ━━━━━━━━━━━━━━━━━                                               │
│  B  │                                                                    │
│  A  │  Active Alerts                                                     │
│  R  │  ┌─────────────────────────────────────────────────────────────┐ │
│     │  │ Severity  Alert Name              Service      Started   ⋮   │ │
│  A  │  ├─────────────────────────────────────────────────────────────┤ │
│  l  │  │ ● CRIT   High Memory Usage       user-api     5m ago    ⋮   │ │
│  e  │  │          Memory usage > 90% for 5 minutes                   │ │
│  r  │  │          [View Details] [Acknowledge] [Silence]             │ │
│  t  │  ├─────────────────────────────────────────────────────────────┤ │
│  s  │  │ ● WARN   API Response Time       api-gateway  12m ago   ⋮   │ │
│     │  │          P95 latency > 500ms                                │ │
│     │  │          [View Details] [Acknowledge] [Silence]             │ │
│     │  ├─────────────────────────────────────────────────────────────┤ │
│     │  │ ● WARN   Disk Space Low          prometheus   1h ago    ⋮   │ │
│     │  │          Available space < 20%                              │ │
│     │  │          [View Details] [Acknowledge] [Silence]             │ │
│     │  └─────────────────────────────────────────────────────────────┘ │
│     │                                                                    │
│     │  Alert Summary                                                     │
│     │  Critical: 1  |  Warning: 2  |  Info: 0  |  Resolved Today: 8   │
└─────┴───────────────────────────────────────────────────────────────────┘
```

---

## ⚙️ 8. Settings Screen

### Desktop Layout
```
┌─────────────────────────────────────────────────────────────────────────┐
│ ≡  Gunj Operator > Settings                          🔔 👤 John Doe ▼  │
├─────┬───────────────────────────────────────────────────────────────────┤
│  S  │  Settings                                                          │
│  I  │                                                                    │
│  D  │  [General] [Security] [Integrations] [Notifications] [Advanced]   │
│  E  │  ━━━━━━━━                                                         │
│  B  │                                                                    │
│  A  │  General Settings                                                  │
│  R  │  ┌─────────────────────────────────────────────────────────────┐ │
│     │  │ Organization Name                                             │ │
│  S  │  │ <ACME Corporation>                                           │ │
│  e  │  │                                                               │ │
│  t  │  │ Default Namespace          Default Retention                 │ │
│  t  │  │ ▼ production               ▼ 30 days                         │ │
│  i  │  │                                                               │ │
│  n  │  │ Time Zone                  Date Format                       │ │
│  g  │  │ ▼ UTC                      ▼ YYYY-MM-DD                     │ │
│  s  │  │                                                               │ │
│     │  │ UI Preferences                                                │ │
│     │  │ ☑ Enable dark mode                                           │ │
│     │  │ ☑ Show advanced options                                      │ │
│     │  │ ☑ Enable keyboard shortcuts                                  │ │
│     │  │                                                               │ │
│     │  │ Resource Defaults                                             │ │
│     │  │ CPU Request    <1000m>     Memory Request    <2Gi>          │ │
│     │  │ CPU Limit      <2000m>     Memory Limit      <4Gi>          │ │
│     │  └─────────────────────────────────────────────────────────────┘ │
│     │                                                                    │
│     │                                   (Cancel)  [Save Changes]       │
└─────┴───────────────────────────────────────────────────────────────────┘
```

---

## 🔐 9. Login Screen

### Desktop Layout
```
┌─────────────────────────────────────────────────────────────────────────┐
│                                                                         │
│                                                                         │
│                          Gunj Operator                                  │
│                   Enterprise Observability Platform                     │
│                                                                         │
│                  ┌─────────────────────────────────┐                   │
│                  │                                 │                   │
│                  │  Welcome Back                   │                   │
│                  │                                 │                   │
│                  │  Email or Username              │                   │
│                  │  <user@example.com>             │                   │
│                  │                                 │                   │
│                  │  Password                       │                   │
│                  │  <••••••••••••>                │                   │
│                  │                                 │                   │
│                  │  ☐ Remember me                 │                   │
│                  │                                 │                   │
│                  │  [Sign In]                     │                   │
│                  │                                 │                   │
│                  │  ━━━━━━━━ OR ━━━━━━━━         │                   │
│                  │                                 │                   │
│                  │  [Sign in with SSO]            │                   │
│                  │  [Sign in with GitHub]         │                   │
│                  │                                 │                   │
│                  │  Forgot password? • Contact IT │                   │
│                  └─────────────────────────────────┘                   │
│                                                                         │
│                    Version 2.0.0 • Terms • Privacy                     │
└─────────────────────────────────────────────────────────────────────────┘
```

---

## 📱 Responsive Behavior Summary

### Navigation Patterns
- **Desktop**: Fixed sidebar (280px expanded, 72px collapsed)
- **Tablet**: Collapsible sidebar with overlay option
- **Mobile**: Full-screen drawer with hamburger trigger

### Data Display
- **Desktop**: Tables with all columns visible
- **Tablet**: Hidden secondary columns, priority info shown
- **Mobile**: Card-based layouts, stackable components

### Form Layouts
- **Desktop**: Multi-column forms, inline fields
- **Tablet**: Mixed single/dual column based on space
- **Mobile**: Single column, full-width fields

### Actions & Controls
- **Desktop**: All actions visible, hover states
- **Tablet**: Primary actions visible, secondary in menu
- **Mobile**: Minimal actions, expanded via menu

---

## 🎯 Interaction Patterns

### Loading States
```
┌─────────────────────┐
│ ░░░░░░░░░░░░░░░░░░ │  Skeleton loader
│ ░░░░░░░░░░░░░░░░░░ │
│ ░░░░░░░░░░░░░░░░░░ │
└─────────────────────┘
```

### Empty States
```
┌─────────────────────┐
│                     │
│    📭               │
│  No platforms yet   │
│                     │
│ [Create First One]  │
│                     │
└─────────────────────┘
```

### Error States
```
┌─────────────────────┐
│  ⚠️ Error           │
│                     │
│ Failed to load data │
│                     │
│ [Retry] [Details]   │
└─────────────────────┘
```

---

## 🚀 Next Steps

1. **Create High-Fidelity Mockups**: Transform wireframes into polished designs
2. **Build Interactive Prototypes**: Create clickable prototypes in Figma/HTML
3. **Component Library**: Design individual components in detail
4. **User Testing**: Validate designs with target users
5. **Developer Handoff**: Create detailed specifications for implementation

---

This wireframe document provides the foundation for the Gunj Operator UI design, focusing on functionality and user flow while maintaining flexibility for visual design refinements.