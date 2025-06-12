# Gunj Operator UI - Interactive Prototype Documentation

**Version**: 1.0  
**Last Updated**: June 12, 2025  
**Status**: Design Phase  

## 🎯 Interactive Prototype Overview

This document outlines the interactive behaviors, micro-interactions, and user flows for the Gunj Operator UI prototype. Use this as a guide when building the interactive prototype in Figma, Framer, or other prototyping tools.

---

## 🔄 Core Interactions

### 1. Navigation Patterns

#### Sidebar Navigation
```
Interaction: Hover
- Background: fade to rgba(primary, 0.08)
- Text color: primary
- Duration: 200ms ease-out

Interaction: Click
- Background: rgba(primary, 0.12)
- Left border: 3px solid primary
- Route transition: slide-fade 300ms

Interaction: Collapse/Expand
- Width: 280px → 72px
- Icons: remain centered
- Text: fade out at 150ms
- Total duration: 300ms ease-in-out
```

#### Tab Navigation
```
Interaction: Click
- Underline: animate width 0 → 100%
- Color: text.secondary → primary
- Background: ripple effect from click point
- Duration: 300ms cubic-bezier(0.4, 0, 0.2, 1)

Interaction: Keyboard (Arrow Keys)
- Focus outline: 2px solid primary
- Automatic scroll into view
- Announce to screen readers
```

### 2. Button Interactions

#### Primary Button
```
State: Default
- Background: primary.main
- Elevation: 0

State: Hover
- Background: primary.dark
- Elevation: 2
- Cursor: pointer
- Transform: translateY(-1px)
- Duration: 200ms

State: Active
- Background: primary.darker
- Elevation: 0
- Transform: translateY(0)
- Ripple: from click point

State: Loading
- Content: fade out
- Spinner: fade in + rotate
- Disabled: true
- Min-width: maintained
```

#### Icon Button
```
State: Hover
- Background: rgba(0,0,0,0.04)
- Scale: 1.1
- Rotation: 15deg (for refresh icon)

State: Active
- Scale: 0.95
- Ripple: circular from center
```

### 3. Form Interactions

#### Text Input
```
State: Focus
- Border: 2px solid primary
- Label: shrink + move up
- Helper text: fade in
- Duration: 200ms

State: Error
- Border: 2px solid error
- Label/Helper: color error
- Shake: 2px horizontal, 100ms x3

State: Success
- Border: briefly flash success color
- Check icon: scale in from 0
```

#### Select Dropdown
```
Interaction: Open
- Dropdown: expand from top
- Backdrop: fade in 0 → 0.5
- Options: stagger fade in (50ms each)
- Max height: 300px with scroll

Interaction: Option Hover
- Background: rgba(0,0,0,0.04)
- Cursor: pointer

Interaction: Selection
- Check mark: slide in from left
- Dropdown: collapse with 200ms delay
```

### 4. Card Interactions

#### Platform Card
```
State: Hover
- Elevation: 1 → 4
- Transform: translateY(-2px)
- Border: subtle primary glow
- Action buttons: fade in
- Duration: 200ms

State: Click
- Ripple: from click point
- Scale: 0.98 → 1
- Navigate: after 150ms

State: Loading
- Content: skeleton shimmer
- Actions: disabled
```

### 5. Data Visualizations

#### Charts
```
Interaction: Hover on Data Point
- Tooltip: fade in at cursor
- Data point: scale 1.5x
- Line: increase stroke width
- Related values: highlight

Interaction: Pan/Zoom
- Drag: pan chart
- Scroll: zoom in/out
- Double-click: reset view
- Touch: pinch to zoom

Interaction: Legend Click
- Toggle series visibility
- Fade animation: 300ms
- Update scale: smooth transition
```

#### Progress Indicators
```
Linear Progress:
- Fill: animate from 0 to value
- Stripes: animate background position
- Duration: 1000ms ease-out

Circular Progress:
- Stroke: animate dash-offset
- Rotate: continuous 2s linear
- Pulse: scale 0.95-1.05 for attention
```

---

## 🎬 Micro-Interactions

### Status Changes
```
Healthy → Warning:
1. Icon: morph circle → triangle
2. Color: fade green → orange
3. Pulse: 2x at 1.2x scale
4. Duration: 600ms total

Warning → Critical:
1. Icon: morph triangle → X
2. Color: fade orange → red
3. Shake: 5px horizontal
4. Pulse: continuous glow
```

### Real-time Updates
```
New Data Point:
1. Fade in with slight scale
2. Push animation for lists
3. Highlight briefly (2s)
4. Auto-scroll if needed

Value Change:
1. Number: count up/down animation
2. Background: brief flash
3. Trend arrow: rotate + fade
```

### Notifications
```
Toast Appearance:
1. Slide in from right + fade
2. Progress bar if auto-dismiss
3. Hover: pause timer
4. Swipe right: dismiss
5. Stack: push others down

Alert Banner:
1. Slide down from top
2. Push content down
3. Close: slide up + fade
4. Important: add shake
```

---

## 🗺️ User Flow Animations

### 1. Platform Creation Flow

```
Step Transitions:
- Direction: slide left
- Previous step: fade + scale 0.95
- New step: slide in + fade
- Progress bar: animate fill
- Duration: 400ms

Validation:
- Error: shake + highlight field
- Success: check mark appears
- Progress: enable next button

Completion:
- Success animation: confetti or checkmark
- Redirect: after 2s
- Show creation progress
```

### 2. Component Configuration

```
Panel Open:
- Overlay: fade in backdrop
- Panel: slide in from right
- Content: stagger load sections
- Width: 480px on desktop

Save Changes:
- Button: loading state
- Form: disabled state
- Success: toast + close panel
- Error: inline messages
```

### 3. Dashboard Load

```
Initial Load:
1. Skeleton screens for cards
2. Stagger reveal cards (100ms each)
3. Animate in numbers
4. Start real-time updates

Refresh:
1. Subtle pulse on refresh icon
2. Cards: brief loading shimmer
3. Update values with transitions
4. Success feedback
```

---

## 📱 Touch Interactions

### Mobile Gestures

#### Swipe Actions
```
Platform Card Swipe:
- Threshold: 30% of card width
- Left: reveal edit/delete actions
- Right: quick view panel
- Rubber band: if no action
- Snap back: spring animation
```

#### Pull to Refresh
```
Gesture:
1. Pull down: reveal spinner
2. Threshold: 80px
3. Release: spinner + loading
4. Complete: success checkmark
5. Reset: fade out after 1s
```

#### Long Press
```
Actions:
- Duration: 500ms
- Haptic: subtle feedback
- Menu: appear at touch point
- Options: fade in with scale
```

### Touch Feedback
```
All Tappable Elements:
- Active state: 0.95 scale
- Ripple: from touch point
- No hover states on touch
- Larger hit areas (44px min)
```

---

## ⚡ Performance Optimizations

### Lazy Loading
```
Images/Charts:
- Placeholder: blur or skeleton
- Load: fade in when visible
- Progressive: for large images

Routes:
- Show progress bar
- Maintain previous view
- Smooth transition when ready
```

### Optimistic Updates
```
User Actions:
1. Immediate UI update
2. Show pending state
3. Sync in background
4. Rollback if error
5. Show final state
```

### Debounced Interactions
```
Search:
- Delay: 300ms after typing
- Loading: inline spinner
- Results: fade in

Resize:
- Debounce: 150ms
- Smooth reflow
- Maintain scroll position
```

---

## 🎨 Theme Transitions

### Light/Dark Mode Switch
```
Animation:
1. Ripple from switch position
2. Colors: smooth transition 400ms
3. Shadows: adjust opacity
4. Icons: some may change
5. Store preference

Special Effects:
- Sun/Moon icon morph
- Background gradient shift
- Delayed element updates for effect
```

---

## 🔊 Sound Design (Optional)

### Subtle Audio Feedback
```
Success Actions:
- Soft chime: 200ms
- Volume: 20%
- Frequency: 800Hz

Error Actions:
- Low tone: 150ms
- Volume: 25%
- Frequency: 200Hz

Notifications:
- Gentle pop: 100ms
- Volume: 30%
- Frequency: 600Hz
```

---

## 🧪 Prototype Testing Scenarios

### Critical Paths to Prototype

1. **First-Time User Experience**
   - Welcome tour
   - Platform creation wizard
   - Initial dashboard view
   - Help tooltips

2. **Daily Operations**
   - Quick platform status check
   - Responding to alerts
   - Updating configurations
   - Viewing metrics

3. **Troubleshooting Flow**
   - Alert → Investigation → Resolution
   - Log searching
   - Metric correlation
   - Fix application

4. **Administrative Tasks**
   - User management
   - Global settings
   - Audit log review
   - Cost analysis

### Interactive States to Include

1. **Empty States**
   - No platforms
   - No data
   - No search results
   - Connection lost

2. **Loading States**
   - Initial load
   - Data refresh
   - Action processing
   - Route transitions

3. **Error States**
   - Form validation
   - API errors
   - Permission denied
   - Resource limits

4. **Success States**
   - Creation complete
   - Update successful
   - Action confirmed
   - Goal achieved

---

## 🚀 Prototype Deliverables

### Figma Structure
```
📁 Gunj Operator UI
├── 📁 Design System
│   ├── 📄 Colors & Typography
│   ├── 📄 Components
│   ├── 📄 Icons
│   └── 📄 Patterns
├── 📁 Screens
│   ├── 📁 Desktop
│   ├── 📁 Tablet
│   └── 📁 Mobile
├── 📁 Flows
│   ├── 📄 Platform Creation
│   ├── 📄 Monitoring
│   └── 📄 Configuration
└── 📁 Prototype
    ├── 📄 Desktop Prototype
    ├── 📄 Mobile Prototype
    └── 📄 Interaction Guide
```

### Handoff Requirements

1. **Design Tokens**
   - Export as JSON
   - Include all values
   - Version controlled

2. **Component Specs**
   - Detailed measurements
   - Interaction states
   - Animation timing

3. **Assets**
   - SVG icons
   - Lottie animations
   - Image assets (2x, 3x)

4. **Documentation**
   - Interaction videos
   - Edge case handling
   - Accessibility notes

---

This interactive prototype documentation provides comprehensive guidance for creating engaging, intuitive, and performant user interactions in the Gunj Operator UI.
