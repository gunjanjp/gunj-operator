# Gunj Operator UI Component Library

**Version**: 1.0  
**Last Updated**: June 12, 2025  
**Status**: Design Phase  

## 📚 Component Library Overview

This document showcases all the UI components designed for the Gunj Operator, organized by category. Each component includes usage guidelines, variations, and implementation notes.

---

## 🎨 Foundation Components

### Colors

#### Primary Palette
- **Primary**: `#1976D2` - Main brand color (Kubernetes blue)
- **Primary Light**: `#42A5F5` - Hover states, highlights
- **Primary Dark**: `#1565C0` - Active states, emphasis
- **Secondary**: `#00ACC1` - Accent color (Teal)

#### Component Brand Colors
- **Prometheus**: `#E6522C` - Orange
- **Grafana**: `#F46800` - Dark orange  
- **Loki**: `#F5A623` - Yellow
- **Tempo**: `#7B61FF` - Purple
- **Alertmanager**: `#FF6B6B` - Red gradient

#### Semantic Colors
- **Success**: `#2E7D32` - Green for healthy states
- **Warning**: `#F57C00` - Orange for warnings
- **Error**: `#D32F2F` - Red for errors/critical
- **Info**: `#0288D1` - Blue for information

### Typography

#### Font Stack
```css
font-family: 'Inter', -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
font-family: 'JetBrains Mono', 'Consolas', monospace; /* Code */
```

#### Type Scale
- **H1**: 48px / 300 weight - Page titles
- **H2**: 34px / 300 weight - Section headers
- **H3**: 24px / 400 weight - Card titles
- **H4**: 20px / 400 weight - Subsections
- **H5**: 18px / 400 weight - Component headers
- **H6**: 16px / 500 weight - Labels
- **Body**: 14px / 400 weight - Default text
- **Caption**: 12px / 400 weight - Help text

---

## 🔘 Interactive Components

### Buttons

#### Primary Button
- **Usage**: Main actions (Create, Save, Submit)
- **States**: Default, Hover, Active, Disabled, Loading
- **Sizes**: Small (32px), Medium (40px), Large (48px)

```html
<!-- Primary Button -->
<button class="button button-primary">
  <span class="material-icons">add</span>
  Create Platform
</button>

<!-- Variants -->
<button class="button button-primary button-small">Small</button>
<button class="button button-primary" disabled>Disabled</button>
<button class="button button-primary button-loading">
  <span class="spinner"></span>
  Loading...
</button>
```

#### Secondary Button
- **Usage**: Secondary actions, cancel operations
- **Border**: 1px solid with primary color

```html
<button class="button button-secondary">
  <span class="material-icons">settings</span>
  Configure
</button>
```

#### Icon Button
- **Usage**: Toolbar actions, compact spaces
- **Size**: 40x40px standard, 32x32px small

```html
<button class="icon-button">
  <span class="material-icons">refresh</span>
</button>
```

### Form Elements

#### Text Input
- **States**: Default, Focus, Error, Disabled
- **Helper text**: Below input for guidance
- **Validation**: Real-time or on blur

```html
<div class="form-group">
  <label class="form-label">
    Platform Name <span class="required">*</span>
  </label>
  <input type="text" class="form-input" placeholder="my-platform">
  <div class="form-helper">Lowercase letters, numbers, and hyphens</div>
</div>
```

#### Select Dropdown
- **Custom styling**: Consistent across browsers
- **Icon**: Chevron down indicator

```html
<select class="form-select">
  <option>Select namespace</option>
  <option>production</option>
  <option>staging</option>
</select>
```

#### Toggle Switch
- **Usage**: Binary on/off settings
- **Label**: Always on the right

```html
<label class="toggle-switch">
  <input type="checkbox">
  <span class="toggle-slider"></span>
</label>
<span>Enable high availability</span>
```

#### Checkbox & Radio
- **Size**: 18x18px minimum
- **Focus**: Visible outline

```html
<label class="checkbox-label">
  <input type="checkbox" class="checkbox">
  <span>Enable Prometheus</span>
</label>
```

---

## 📊 Data Display Components

### Cards

#### Basic Card
- **Shadow**: elevation-1 (2px)
- **Radius**: 12px
- **Padding**: 24px

```html
<div class="card">
  <div class="card-content">
    <h3>Card Title</h3>
    <p>Card content goes here</p>
  </div>
</div>
```

#### Platform Card
- **Usage**: Platform list display
- **Sections**: Header, details, actions

```html
<div class="platform-card">
  <div class="platform-card-header">
    <h3>prod-platform</h3>
    <span class="status-badge ready">Ready</span>
  </div>
  <div class="platform-card-details">
    <!-- Details -->
  </div>
  <div class="platform-card-actions">
    <button class="button button-primary">View</button>
  </div>
</div>
```

### Status Indicators

#### Status Badge
- **Variants**: Ready, Updating, Error, Warning
- **Icon**: 8px status dot

```html
<span class="status-badge ready">
  <span class="status-dot"></span>
  Ready
</span>
```

#### Component Icons
- **Size**: 24x24px standard
- **Colors**: Brand colors per component

```html
<div class="component-icons">
  <div class="component-icon prometheus" title="Prometheus">P</div>
  <div class="component-icon grafana" title="Grafana">G</div>
  <div class="component-icon loki" title="Loki">L</div>
</div>
```

### Progress Indicators

#### Linear Progress
- **Usage**: Loading states, completion tracking
- **Height**: 4px or 8px

```html
<div class="progress-bar">
  <div class="progress-fill" style="width: 75%"></div>
</div>
```

#### Circular Progress
- **Usage**: Indeterminate loading
- **Sizes**: 20px, 40px, 60px

```html
<div class="spinner"></div>
```

---

## 📋 Tables & Lists

### Data Table
- **Header**: Sticky, sortable columns
- **Rows**: Hover state, selectable
- **Responsive**: Converts to cards on mobile

```html
<table class="data-table">
  <thead>
    <tr>
      <th>Name</th>
      <th>Status</th>
      <th>Version</th>
    </tr>
  </thead>
  <tbody>
    <tr>
      <td>prod-platform</td>
      <td><span class="status-badge ready">Ready</span></td>
      <td>v2.48.0</td>
    </tr>
  </tbody>
</table>
```

### List Items
- **Padding**: 16px vertical, 24px horizontal
- **Divider**: 1px bottom border
- **Actions**: Right-aligned

---

## 🔔 Feedback Components

### Alerts
- **Types**: Info, Success, Warning, Error
- **Dismissible**: Optional close button
- **Icons**: Leading icon for type

```html
<div class="alert alert-info">
  <span class="material-icons">info</span>
  <span>Platform deployment in progress</span>
</div>
```

### Tooltips
- **Trigger**: Hover or focus
- **Position**: Auto-adjust to viewport
- **Delay**: 600ms show, 0ms hide

```html
<span class="tooltip" data-tooltip="Additional information">
  <span class="material-icons">help_outline</span>
</span>
```

### Modals
- **Backdrop**: Semi-transparent overlay
- **Sizes**: Small (400px), Medium (600px), Large (800px)
- **Actions**: Right-aligned in footer

```html
<div class="modal">
  <div class="modal-header">
    <h3>Modal Title</h3>
    <button class="icon-button">
      <span class="material-icons">close</span>
    </button>
  </div>
  <div class="modal-body">
    <!-- Content -->
  </div>
  <div class="modal-footer">
    <button class="button button-secondary">Cancel</button>
    <button class="button button-primary">Confirm</button>
  </div>
</div>
```

---

## 🧭 Navigation Components

### Sidebar Navigation
- **Width**: 280px expanded, 72px collapsed
- **Items**: Icon + label
- **Active state**: Left border indicator

```html
<nav class="sidebar-nav">
  <a href="#" class="nav-item active">
    <span class="material-icons">dashboard</span>
    <span class="nav-text">Dashboard</span>
  </a>
</nav>
```

### Tabs
- **Indicator**: Bottom border on active
- **Scrollable**: Horizontal scroll on overflow

```html
<div class="tabs">
  <button class="tab active">Overview</button>
  <button class="tab">Components</button>
  <button class="tab">Monitoring</button>
</div>
```

### Breadcrumbs
- **Separator**: Chevron right icon
- **Current**: Bold, no link

```html
<nav class="breadcrumbs">
  <a href="#">Home</a>
  <span class="material-icons">chevron_right</span>
  <a href="#">Platforms</a>
  <span class="material-icons">chevron_right</span>
  <span>prod-platform</span>
</nav>
```

---

## 📱 Responsive Patterns

### Mobile Adaptations

#### Navigation Drawer
- **Trigger**: Hamburger menu
- **Width**: 280px or full screen
- **Overlay**: Dark backdrop

#### Card View
- **Stack**: Single column
- **Touch**: 44px minimum targets
- **Swipe**: Optional gestures

#### Simplified Tables
- **Hide**: Secondary columns
- **Priority**: Key information visible

### Breakpoint Behaviors

| Component | Mobile (<600px) | Tablet (600-959px) | Desktop (960px+) |
|-----------|----------------|-------------------|------------------|
| Navigation | Drawer | Collapsible | Fixed sidebar |
| Grid | 1 column | 2 columns | 4+ columns |
| Tables | Cards | Simplified | Full table |
| Forms | Stacked | Mixed layout | Multi-column |
| Buttons | Full width | Auto width | Auto width |

---

## ♿ Accessibility Guidelines

### Focus Management
- **Visible focus**: 2px outline
- **Tab order**: Logical flow
- **Skip links**: For keyboard navigation

### ARIA Labels
- **Required**: For icon-only buttons
- **Live regions**: For dynamic updates
- **Roles**: Semantic HTML preferred

### Color Contrast
- **Normal text**: 4.5:1 minimum
- **Large text**: 3:1 minimum
- **Interactive**: 3:1 minimum

---

## 🎬 Animation Guidelines

### Timing
- **Micro**: 100-200ms (hovers, selections)
- **Standard**: 300ms (most transitions)
- **Complex**: 400-600ms (page transitions)

### Easing
- **Standard**: `cubic-bezier(0.4, 0, 0.2, 1)`
- **Deceleration**: `cubic-bezier(0.0, 0, 0.2, 1)`
- **Acceleration**: `cubic-bezier(0.4, 0, 1, 1)`

### Common Animations
- **Fade**: Opacity transitions
- **Scale**: Growth for emphasis
- **Slide**: Directional movement
- **Ripple**: Touch feedback

---

## 🛠️ Implementation Notes

### CSS Architecture
- **Methodology**: BEM with utility classes
- **Variables**: CSS custom properties
- **Preprocessor**: Optional (Sass/Less)

### Component Structure
```css
/* Component */
.component { }

/* Component variant */
.component--variant { }

/* Component element */
.component__element { }

/* Component state */
.component.is-active { }

/* Utility classes */
.u-text-center { }
.u-mt-2 { }
```

### Performance Considerations
- **Bundle size**: Tree-shake unused components
- **Images**: Lazy load, responsive images
- **Fonts**: Preload critical fonts
- **CSS**: Critical CSS inline

---

## 📦 Component Checklist

When implementing a new component:

- [ ] Follows design system guidelines
- [ ] Responsive across all breakpoints
- [ ] Accessible (ARIA, keyboard, contrast)
- [ ] Documented with examples
- [ ] Tested across browsers
- [ ] Performance optimized
- [ ] Theme-aware (light/dark)
- [ ] Internationalization ready

---

This component library serves as the foundation for building a consistent, accessible, and performant user interface for the Gunj Operator platform.