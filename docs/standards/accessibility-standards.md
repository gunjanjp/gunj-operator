# Gunj Operator Accessibility Standards v1.0

**Version**: 1.0  
**Last Updated**: June 12, 2025  
**Project**: Gunj Operator - Enterprise Observability Platform  
**Contact**: gunjanjp@gmail.com  
**Status**: Official Accessibility Guidelines  

---

## 📋 Executive Summary

This document defines comprehensive accessibility standards for the Gunj Operator project to ensure our platform is usable by people with diverse abilities. We are committed to meeting WCAG 2.1 Level AA standards and exceeding them where possible.

### 🎯 Accessibility Goals

1. **Universal Access**: Enable all users to deploy and manage observability infrastructure
2. **WCAG Compliance**: Meet or exceed WCAG 2.1 Level AA standards
3. **Inclusive Design**: Build accessibility into every component from the start
4. **Continuous Improvement**: Regular audits and user feedback integration

---

## 🌐 Web Content Accessibility Guidelines (WCAG) 2.1

### Level A Requirements (Minimum)

#### 1. Perceivable

**1.1 Text Alternatives**
- All non-text content must have text alternatives
- Images must have descriptive alt text
- Icons must have accessible labels
- Charts must have text descriptions

```typescript
// CORRECT: Accessible image component
<img 
  src="/platform-status.png" 
  alt="Platform status showing 3 healthy components: Prometheus (ready), Grafana (ready), and Loki (installing)"
/>

// CORRECT: Icon with accessible label
<IconButton aria-label="Delete platform">
  <DeleteIcon />
</IconButton>
```

**1.2 Time-based Media**
- Provide captions for all video content
- Offer transcripts for audio content
- Include audio descriptions for visual information

**1.3 Adaptable**
- Content must be presentable in different ways without losing meaning
- Use semantic HTML structure
- Ensure proper reading order

```html
<!-- CORRECT: Semantic structure -->
<main>
  <h1>Observability Platforms</h1>
  <section aria-labelledby="platform-list-heading">
    <h2 id="platform-list-heading">Active Platforms</h2>
    <ul role="list">
      <li role="listitem">...</li>
    </ul>
  </section>
</main>
```

**1.4 Distinguishable**
- Use of color must not be the only visual means of conveying information
- Provide sufficient color contrast (minimum 4.5:1 for normal text)
- Text must be resizable up to 200% without loss of functionality

```css
/* CORRECT: High contrast colors */
:root {
  --text-primary: #1a1a1a;      /* Contrast ratio: 15.5:1 on white */
  --text-secondary: #505050;     /* Contrast ratio: 7.8:1 on white */
  --error-color: #d32f2f;        /* Contrast ratio: 5.9:1 on white */
  --success-color: #2e7d32;      /* Contrast ratio: 5.8:1 on white */
}
```

### Level AA Requirements (Target)

#### 2. Operable

**2.1 Keyboard Accessible**
- All functionality must be available via keyboard
- No keyboard traps
- Provide keyboard shortcuts for common actions

```typescript
// CORRECT: Keyboard navigation hook
export const useKeyboardNavigation = () => {
  useEffect(() => {
    const handleKeyPress = (e: KeyboardEvent) => {
      // Ctrl/Cmd + K for search
      if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
        e.preventDefault();
        focusSearchInput();
      }
      // Escape to close modals
      if (e.key === 'Escape') {
        closeActiveModal();
      }
    };
    
    document.addEventListener('keydown', handleKeyPress);
    return () => document.removeEventListener('keydown', handleKeyPress);
  }, []);
};
```

**2.2 Enough Time**
- Users must be able to extend time limits
- Auto-updating content must be pausable
- Session timeouts must provide warnings

```typescript
// CORRECT: Accessible auto-refresh
const AutoRefreshControl: React.FC = () => {
  const [isPaused, setIsPaused] = useState(false);
  
  return (
    <div role="region" aria-label="Auto-refresh controls">
      <button
        onClick={() => setIsPaused(!isPaused)}
        aria-pressed={isPaused}
        aria-label={isPaused ? "Resume auto-refresh" : "Pause auto-refresh"}
      >
        {isPaused ? <PlayIcon /> : <PauseIcon />}
      </button>
      <span aria-live="polite">
        {isPaused ? "Auto-refresh paused" : "Auto-refresh active (30 seconds)"}
      </span>
    </div>
  );
};
```

**2.3 Seizures and Physical Reactions**
- No content that flashes more than 3 times per second
- Provide warnings for potentially problematic content
- Allow users to disable animations

**2.4 Navigable**
- Provide skip links for repetitive content
- Use descriptive page titles
- Focus order must be logical
- Link purpose must be clear from context

```typescript
// CORRECT: Skip navigation
const SkipLinks: React.FC = () => (
  <div className="skip-links">
    <a href="#main-content" className="skip-link">
      Skip to main content
    </a>
    <a href="#primary-navigation" className="skip-link">
      Skip to navigation
    </a>
  </div>
);
```

**2.5 Input Modalities**
- Provide alternatives to complex gestures
- Ensure sufficient target sizes (minimum 44x44 pixels)
- Support both mouse and touch inputs

#### 3. Understandable

**3.1 Readable**
- Language of page must be programmatically determined
- Clear and simple language
- Define abbreviations and jargon

```html
<!DOCTYPE html>
<html lang="en">
<head>
  <title>Gunj Operator - Platform Management</title>
</head>
```

**3.2 Predictable**
- Navigation must be consistent
- Components must behave predictably
- No automatic context changes without warning

**3.3 Input Assistance**
- Label all form inputs clearly
- Provide helpful error messages
- Include input instructions

```typescript
// CORRECT: Accessible form field
<FormControl error={!!errors.name}>
  <InputLabel htmlFor="platform-name">
    Platform Name *
  </InputLabel>
  <Input
    id="platform-name"
    name="name"
    aria-describedby="name-helper-text name-error-text"
    aria-required="true"
    aria-invalid={!!errors.name}
  />
  <FormHelperText id="name-helper-text">
    Must be lowercase alphanumeric (max 63 characters)
  </FormHelperText>
  {errors.name && (
    <FormHelperText id="name-error-text" role="alert">
      {errors.name}
    </FormHelperText>
  )}
</FormControl>
```

#### 4. Robust

**4.1 Compatible**
- Use valid, well-formed markup
- Ensure compatibility with assistive technologies
- Provide name, role, and value for all UI components

---

## 🎨 UI Component Standards

### Component Accessibility Checklist

Every UI component must:
- [ ] Have appropriate ARIA labels/descriptions
- [ ] Be keyboard navigable
- [ ] Have sufficient color contrast
- [ ] Work with screen readers
- [ ] Have visible focus indicators
- [ ] Support high contrast mode
- [ ] Be tested with assistive technologies

### Platform Cards

```typescript
interface PlatformCardProps {
  platform: Platform;
  onSelect: (platform: Platform) => void;
}

const PlatformCard: React.FC<PlatformCardProps> = ({ platform, onSelect }) => {
  const statusLabel = getStatusLabel(platform.status);
  
  return (
    <Card
      role="article"
      aria-label={`Platform ${platform.name}`}
      tabIndex={0}
      onClick={() => onSelect(platform)}
      onKeyPress={(e) => {
        if (e.key === 'Enter' || e.key === ' ') {
          e.preventDefault();
          onSelect(platform);
        }
      }}
      sx={{
        '&:focus': {
          outline: '2px solid var(--focus-color)',
          outlineOffset: '2px',
        },
      }}
    >
      <CardContent>
        <Typography variant="h3" component="h2">
          {platform.name}
        </Typography>
        <div
          role="status"
          aria-label={`Status: ${statusLabel}`}
          aria-live="polite"
        >
          <StatusIcon status={platform.status} />
          <span className="visually-hidden">{statusLabel}</span>
        </div>
      </CardContent>
    </Card>
  );
};
```

### Data Tables

```typescript
const PlatformTable: React.FC = () => {
  return (
    <Table role="table" aria-label="Observability platforms">
      <TableHead>
        <TableRow>
          <TableCell scope="col">Name</TableCell>
          <TableCell scope="col">Namespace</TableCell>
          <TableCell scope="col">Status</TableCell>
          <TableCell scope="col">
            <span className="visually-hidden">Actions</span>
          </TableCell>
        </TableRow>
      </TableHead>
      <TableBody>
        {platforms.map((platform) => (
          <TableRow key={platform.id}>
            <TableCell scope="row">{platform.name}</TableCell>
            <TableCell>{platform.namespace}</TableCell>
            <TableCell>
              <span role="status" aria-live="polite">
                {platform.status}
              </span>
            </TableCell>
            <TableCell>
              <IconButton
                aria-label={`Edit platform ${platform.name}`}
                onClick={() => handleEdit(platform)}
              >
                <EditIcon />
              </IconButton>
            </TableCell>
          </TableRow>
        ))}
      </TableBody>
    </Table>
  );
};
```

### Modals and Dialogs

```typescript
const DeleteConfirmDialog: React.FC<DialogProps> = ({ open, platform, onConfirm, onCancel }) => {
  const cancelButtonRef = useRef<HTMLButtonElement>(null);
  
  return (
    <Dialog
      open={open}
      onClose={onCancel}
      aria-labelledby="delete-dialog-title"
      aria-describedby="delete-dialog-description"
      initialFocus={cancelButtonRef}
    >
      <DialogTitle id="delete-dialog-title">
        Delete Platform
      </DialogTitle>
      <DialogContent>
        <DialogContentText id="delete-dialog-description">
          Are you sure you want to delete the platform "{platform.name}"? 
          This action cannot be undone and will remove all associated resources.
        </DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button ref={cancelButtonRef} onClick={onCancel}>
          Cancel
        </Button>
        <Button onClick={onConfirm} color="error">
          Delete Platform
        </Button>
      </DialogActions>
    </Dialog>
  );
};
```

---

## 📊 Charts and Visualizations

### Chart Accessibility Requirements

1. **Text Alternatives**
   - Provide summary of data in text format
   - Include data tables as alternative views
   - Use descriptive titles and labels

2. **Color and Contrast**
   - Don't rely solely on color
   - Use patterns or shapes in addition to colors
   - Ensure sufficient contrast between elements

3. **Keyboard Navigation**
   - Make data points focusable
   - Provide keyboard shortcuts for exploration
   - Show data values on focus

```typescript
const AccessibleChart: React.FC<ChartProps> = ({ data, title }) => {
  const chartId = useId();
  const summaryId = `${chartId}-summary`;
  
  return (
    <div role="img" aria-labelledby={`${chartId}-title`} aria-describedby={summaryId}>
      <h3 id={`${chartId}-title`}>{title}</h3>
      
      {/* Visual chart */}
      <LineChart data={data} />
      
      {/* Text summary for screen readers */}
      <div id={summaryId} className="visually-hidden">
        {generateChartSummary(data)}
      </div>
      
      {/* Data table alternative */}
      <details>
        <summary>View data as table</summary>
        <DataTable data={data} />
      </details>
    </div>
  );
};
```

---

## 🔊 Screen Reader Support

### ARIA Best Practices

1. **Use semantic HTML first**
   - Prefer native HTML elements over ARIA
   - Only use ARIA to enhance, not replace semantics

2. **ARIA Labels and Descriptions**
   ```typescript
   // Use aria-label for short descriptions
   <button aria-label="Close dialog">×</button>
   
   // Use aria-describedby for longer descriptions
   <input
     aria-describedby="password-help"
     type="password"
   />
   <span id="password-help">
     Password must be at least 8 characters
   </span>
   ```

3. **Live Regions**
   ```typescript
   // Status messages
   <div role="status" aria-live="polite">
     Platform created successfully
   </div>
   
   // Error messages
   <div role="alert" aria-live="assertive">
     Failed to create platform: Invalid configuration
   </div>
   ```

4. **Landmarks**
   ```html
   <header role="banner">...</header>
   <nav role="navigation">...</nav>
   <main role="main">...</main>
   <aside role="complementary">...</aside>
   <footer role="contentinfo">...</footer>
   ```

---

## ⌨️ Keyboard Navigation

### Navigation Patterns

1. **Tab Order**
   - Logical flow from left to right, top to bottom
   - Group related controls
   - Skip repetitive content

2. **Keyboard Shortcuts**
   ```typescript
   const KEYBOARD_SHORTCUTS = {
     'Ctrl+K, Cmd+K': 'Open search',
     'Ctrl+N, Cmd+N': 'Create new platform',
     'Escape': 'Close dialog/dropdown',
     'Ctrl+S, Cmd+S': 'Save changes',
     '?': 'Show keyboard shortcuts',
     '/': 'Focus search input',
   };
   ```

3. **Focus Management**
   ```typescript
   const useFocusManagement = () => {
     const previousFocus = useRef<HTMLElement | null>(null);
     
     const saveFocus = () => {
       previousFocus.current = document.activeElement as HTMLElement;
     };
     
     const restoreFocus = () => {
       previousFocus.current?.focus();
     };
     
     return { saveFocus, restoreFocus };
   };
   ```

---

## 🎨 Visual Design Standards

### Color Contrast Requirements

```css
/* Minimum contrast ratios */
:root {
  /* Normal text (4.5:1) */
  --text-normal: #212121;        /* 16.1:1 on white */
  
  /* Large text (3:1) */
  --text-large: #424242;         /* 10.9:1 on white */
  
  /* UI components (3:1) */
  --border-color: #757575;       /* 4.6:1 on white */
  
  /* Disabled states */
  --text-disabled: #9e9e9e;      /* 2.8:1 - exempt from WCAG */
}

/* Focus indicators */
:focus {
  outline: 2px solid var(--focus-color);
  outline-offset: 2px;
}

/* High contrast mode support */
@media (prefers-contrast: high) {
  :root {
    --text-normal: #000000;
    --background: #ffffff;
    --border-color: #000000;
  }
}
```

### Responsive Design

```css
/* Touch target sizes */
.touch-target {
  min-width: 44px;
  min-height: 44px;
  display: inline-flex;
  align-items: center;
  justify-content: center;
}

/* Spacing for mobile */
@media (max-width: 768px) {
  .interactive-element {
    margin: 8px;
    padding: 12px;
  }
}
```

---

## 🧪 Testing Procedures

### Automated Testing

1. **Jest/React Testing Library**
   ```typescript
   test('platform card is accessible', () => {
     const { getByRole } = render(<PlatformCard platform={mockPlatform} />);
     
     const card = getByRole('article', { name: /platform production/i });
     expect(card).toBeInTheDocument();
     
     const status = getByRole('status');
     expect(status).toHaveAttribute('aria-label', 'Status: Ready');
   });
   ```

2. **Axe-core Integration**
   ```typescript
   import { axe, toHaveNoViolations } from 'jest-axe';
   
   expect.extend(toHaveNoViolations);
   
   test('no accessibility violations', async () => {
     const { container } = render(<App />);
     const results = await axe(container);
     expect(results).toHaveNoViolations();
   });
   ```

### Manual Testing Checklist

- [ ] **Keyboard Only**: Navigate entire app without mouse
- [ ] **Screen Reader**: Test with NVDA/JAWS (Windows), VoiceOver (Mac)
- [ ] **Zoom**: Test at 200% zoom level
- [ ] **High Contrast**: Enable high contrast mode
- [ ] **Color Blindness**: Use simulator tools
- [ ] **Mobile**: Test with mobile screen readers

### Browser/AT Combinations

1. **Windows**
   - NVDA + Firefox
   - JAWS + Chrome
   - Narrator + Edge

2. **macOS**
   - VoiceOver + Safari
   - VoiceOver + Chrome

3. **Mobile**
   - TalkBack + Chrome (Android)
   - VoiceOver + Safari (iOS)

---

## 📝 Documentation Accessibility

### Accessible Documentation

1. **Structure**
   - Use proper heading hierarchy
   - Include table of contents
   - Provide text alternatives for diagrams

2. **Code Examples**
   ```markdown
   ```yaml
   # Platform configuration example
   apiVersion: observability.io/v1beta1
   kind: ObservabilityPlatform
   metadata:
     name: production
   spec:
     components:
       prometheus:
         enabled: true
   ```
   
   Alternative text: YAML configuration showing 
   ObservabilityPlatform resource with Prometheus enabled
   ```

3. **Videos and Images**
   - Provide captions for videos
   - Include transcripts
   - Use descriptive alt text

---

## 🌍 Internationalization (i18n)

### RTL Language Support

```css
/* RTL layout support */
[dir="rtl"] {
  .card {
    text-align: right;
  }
  
  .icon-before {
    margin-left: 8px;
    margin-right: 0;
  }
}
```

### Translatable Content

```typescript
// All user-facing text must be translatable
const messages = {
  'platform.create.title': 'Create New Platform',
  'platform.create.description': 'Deploy a new observability platform',
  'platform.status.ready': 'Platform is ready',
  'platform.status.error': 'Platform has errors',
};
```

---

## 🚀 Implementation Roadmap

### Phase 1: Foundation (Weeks 1-2)
- [ ] Set up automated accessibility testing
- [ ] Create component templates
- [ ] Establish color palette
- [ ] Define focus management patterns

### Phase 2: Component Development (Weeks 3-6)
- [ ] Build accessible component library
- [ ] Implement keyboard navigation
- [ ] Add ARIA attributes
- [ ] Create screen reader announcements

### Phase 3: Testing & Validation (Weeks 7-8)
- [ ] Conduct automated testing
- [ ] Perform manual testing
- [ ] Get user feedback
- [ ] Fix identified issues

### Phase 4: Documentation (Week 9)
- [ ] Document patterns
- [ ] Create developer guide
- [ ] Build testing procedures
- [ ] Provide training materials

---

## 📚 Resources

### Standards and Guidelines
- [WCAG 2.1](https://www.w3.org/WAI/WCAG21/quickref/)
- [ARIA Authoring Practices](https://www.w3.org/WAI/ARIA/apg/)
- [WebAIM Resources](https://webaim.org/resources/)

### Testing Tools
- [axe DevTools](https://www.deque.com/axe/devtools/)
- [WAVE](https://wave.webaim.org/)
- [Lighthouse](https://developers.google.com/web/tools/lighthouse)
- [Pa11y](https://pa11y.org/)

### Component Libraries
- [Reach UI](https://reach.tech/)
- [React Aria](https://react-spectrum.adobe.com/react-aria/)
- [Headless UI](https://headlessui.dev/)

---

## 🎯 Success Metrics

### Compliance Metrics
- **WCAG Compliance**: 100% Level AA
- **Automated Test Pass Rate**: > 95%
- **Keyboard Navigation Coverage**: 100%
- **Screen Reader Compatibility**: 100%

### User Metrics
- **Task Completion Rate**: > 90% for users with disabilities
- **Time to Complete Tasks**: Within 20% of baseline
- **User Satisfaction**: > 4.5/5 rating
- **Support Tickets**: < 5% accessibility-related

---

**This document ensures the Gunj Operator is accessible to all users, regardless of their abilities.**

*For questions about accessibility, contact gunjanjp@gmail.com or join our accessibility channel.*