# Accessibility Standards
## Gunj Operator - Enterprise Observability Platform

**Document Version**: 1.0  
**Date**: June 12, 2025  
**Owner**: UI/UX Team  
**Status**: Phase 1.4.2 - CNCF Compliance Planning  
**WCAG Target**: 2.1 Level AA  

---

## Executive Summary

This document establishes accessibility standards for the Gunj Operator project, ensuring the platform is usable by people with disabilities. We follow Web Content Accessibility Guidelines (WCAG) 2.1 Level AA as our baseline, with select AAA criteria where feasible.

### Core Principles (POUR)
- **Perceivable**: Information must be presentable in ways users can perceive
- **Operable**: Interface components must be operable
- **Understandable**: Information and UI operation must be understandable
- **Robust**: Content must be robust enough for various assistive technologies

---

## 1. WCAG 2.1 Compliance Requirements

### 1.1 Level A Requirements (Mandatory)

#### Perceivable
- [ ] **1.1.1 Non-text Content**: All images, icons, and charts have text alternatives
  ```jsx
  // CORRECT
  <img src="platform-status.png" alt="Platform status: 3 healthy, 1 warning" />
  <Icon name="warning" aria-label="Warning: Configuration issue detected" />
  
  // INCORRECT
  <img src="platform-status.png" />
  <Icon name="warning" />
  ```

- [ ] **1.2.1 Audio-only and Video-only**: Provide alternatives for time-based media
- [ ] **1.2.2 Captions**: Provide captions for videos
- [ ] **1.2.3 Audio Description**: Provide audio description or text alternative
- [ ] **1.3.1 Info and Relationships**: Information conveyed through presentation is available in text
  ```jsx
  // CORRECT: Semantic HTML
  <nav aria-label="Main navigation">
    <ul>
      <li><a href="/platforms">Platforms</a></li>
      <li><a href="/monitoring">Monitoring</a></li>
    </ul>
  </nav>
  ```

- [ ] **1.3.2 Meaningful Sequence**: Content order is logical and intuitive
- [ ] **1.3.3 Sensory Characteristics**: Instructions don't rely solely on sensory characteristics
- [ ] **1.4.1 Use of Color**: Color is not the only visual means of conveying information
  ```jsx
  // CORRECT: Status with icon and text
  <StatusIndicator status="error">
    <ErrorIcon /> Error: Service unavailable
  </StatusIndicator>
  
  // INCORRECT: Color only
  <div className="red-background" />
  ```

- [ ] **1.4.2 Audio Control**: Users can pause/stop audio that plays automatically

#### Operable
- [ ] **2.1.1 Keyboard**: All functionality available via keyboard
  ```jsx
  // CORRECT: Keyboard accessible button
  <button 
    onClick={handleClick}
    onKeyDown={(e) => e.key === 'Enter' && handleClick()}
    tabIndex={0}
  >
    Deploy Platform
  </button>
  ```

- [ ] **2.1.2 No Keyboard Trap**: Keyboard focus can move away from any component
- [ ] **2.1.4 Character Key Shortcuts**: Single character shortcuts can be disabled/remapped
- [ ] **2.2.1 Timing Adjustable**: Time limits can be adjusted
- [ ] **2.2.2 Pause, Stop, Hide**: Moving content can be paused
- [ ] **2.3.1 Three Flashes**: No content flashes more than 3 times per second
- [ ] **2.4.1 Bypass Blocks**: Skip navigation links provided
  ```jsx
  // Skip to main content link
  <a href="#main" className="skip-link">Skip to main content</a>
  ```

- [ ] **2.4.2 Page Titled**: Pages have descriptive titles
- [ ] **2.4.3 Focus Order**: Focus order preserves meaning
- [ ] **2.4.4 Link Purpose**: Link purpose clear from text or context
- [ ] **2.5.1 Pointer Gestures**: Multipoint/path gestures have alternatives
- [ ] **2.5.2 Pointer Cancellation**: Down-events don't execute functions
- [ ] **2.5.3 Label in Name**: Visible labels match accessible names
- [ ] **2.5.4 Motion Actuation**: Motion-operated functions have alternatives

#### Understandable
- [ ] **3.1.1 Language of Page**: Page language is specified
  ```html
  <html lang="en">
  ```

- [ ] **3.2.1 On Focus**: Focus doesn't trigger context changes
- [ ] **3.2.2 On Input**: Input doesn't trigger unexpected context changes
- [ ] **3.3.1 Error Identification**: Errors are clearly identified
  ```jsx
  // CORRECT: Clear error messaging
  <FormField error>
    <Label htmlFor="platform-name">Platform Name *</Label>
    <Input 
      id="platform-name" 
      aria-invalid="true"
      aria-describedby="platform-name-error"
    />
    <ErrorMessage id="platform-name-error">
      Platform name is required and must be alphanumeric
    </ErrorMessage>
  </FormField>
  ```

- [ ] **3.3.2 Labels or Instructions**: Labels provided for user input

#### Robust
- [ ] **4.1.1 Parsing**: Valid HTML/markup
- [ ] **4.1.2 Name, Role, Value**: UI components have accessible names and roles
- [ ] **4.1.3 Status Messages**: Status messages announced to screen readers
  ```jsx
  // CORRECT: Announce status changes
  <div role="status" aria-live="polite" aria-atomic="true">
    Platform deployment completed successfully
  </div>
  ```

### 1.2 Level AA Requirements (Target)

#### Perceivable
- [ ] **1.2.4 Captions (Live)**: Live audio content has captions
- [ ] **1.2.5 Audio Description**: Audio description for video content
- [ ] **1.3.4 Orientation**: Content not restricted to single orientation
- [ ] **1.3.5 Identify Input Purpose**: Input purpose can be programmatically determined
  ```jsx
  <Input 
    type="email"
    name="email"
    autoComplete="email"
    aria-label="Email address"
  />
  ```

- [ ] **1.4.3 Contrast (Minimum)**: 4.5:1 for normal text, 3:1 for large text
- [ ] **1.4.4 Resize Text**: Text can resize to 200% without horizontal scrolling
- [ ] **1.4.5 Images of Text**: Use actual text rather than images
- [ ] **1.4.10 Reflow**: Content reflows to single column at 320px
- [ ] **1.4.11 Non-text Contrast**: 3:1 contrast for UI components
- [ ] **1.4.12 Text Spacing**: Content remains readable with adjusted spacing
- [ ] **1.4.13 Content on Hover**: Hover content is dismissible, hoverable, persistent

#### Operable
- [ ] **2.4.5 Multiple Ways**: Multiple ways to locate pages
- [ ] **2.4.6 Headings and Labels**: Descriptive headings and labels
- [ ] **2.4.7 Focus Visible**: Keyboard focus is clearly visible
  ```css
  /* CORRECT: Visible focus indicator */
  :focus {
    outline: 3px solid #4A90E2;
    outline-offset: 2px;
  }
  
  /* Never remove focus indicators without replacement */
  ```

#### Understandable
- [ ] **3.1.2 Language of Parts**: Language changes are marked
- [ ] **3.2.3 Consistent Navigation**: Navigation is consistent
- [ ] **3.2.4 Consistent Identification**: Components identified consistently
- [ ] **3.3.3 Error Suggestion**: Error corrections suggested
- [ ] **3.3.4 Error Prevention**: Reversible, checked, or confirmed for important actions

### 1.3 Level AAA Considerations (Enhanced)

Selected AAA criteria we implement:
- [ ] **1.4.6 Contrast (Enhanced)**: 7:1 for normal text, 4.5:1 for large text
- [ ] **2.1.3 Keyboard (No Exception)**: All functionality keyboard accessible
- [ ] **2.2.3 No Timing**: No time limits
- [ ] **2.4.8 Location**: User's location within site is indicated
- [ ] **3.3.5 Help**: Context-sensitive help available

---

## 2. UI Component Accessibility

### 2.1 Interactive Components

#### Buttons
```jsx
// Accessible button component
const Button = ({ children, onClick, disabled, loading, ...props }) => {
  return (
    <button
      onClick={onClick}
      disabled={disabled || loading}
      aria-busy={loading}
      aria-disabled={disabled || loading}
      {...props}
    >
      {loading && <Spinner aria-label="Loading" />}
      {children}
    </button>
  );
};
```

#### Forms
```jsx
// Accessible form field
const FormField = ({ label, error, required, helpText, children }) => {
  const fieldId = useId();
  const errorId = `${fieldId}-error`;
  const helpId = `${fieldId}-help`;
  
  return (
    <div className="form-field">
      <label htmlFor={fieldId}>
        {label}
        {required && <span aria-label="required">*</span>}
      </label>
      {React.cloneElement(children, {
        id: fieldId,
        'aria-invalid': !!error,
        'aria-describedby': [
          error && errorId,
          helpText && helpId
        ].filter(Boolean).join(' '),
        'aria-required': required
      })}
      {error && (
        <div id={errorId} role="alert" className="error">
          {error}
        </div>
      )}
      {helpText && (
        <div id={helpId} className="help-text">
          {helpText}
        </div>
      )}
    </div>
  );
};
```

#### Modals/Dialogs
```jsx
// Accessible modal component
const Modal = ({ isOpen, onClose, title, children }) => {
  const titleId = useId();
  
  useEffect(() => {
    if (isOpen) {
      // Trap focus within modal
      const previousFocus = document.activeElement;
      
      return () => {
        // Restore focus on close
        previousFocus?.focus();
      };
    }
  }, [isOpen]);
  
  if (!isOpen) return null;
  
  return (
    <div 
      role="dialog"
      aria-modal="true"
      aria-labelledby={titleId}
      className="modal"
    >
      <div className="modal-content">
        <h2 id={titleId}>{title}</h2>
        <button
          onClick={onClose}
          aria-label="Close dialog"
          className="close-button"
        >
          <CloseIcon />
        </button>
        {children}
      </div>
    </div>
  );
};
```

#### Data Tables
```jsx
// Accessible data table
const DataTable = ({ caption, data, columns }) => {
  return (
    <table role="table">
      <caption>{caption}</caption>
      <thead>
        <tr role="row">
          {columns.map(col => (
            <th 
              key={col.key}
              scope="col"
              aria-sort={col.sortable ? col.sortDirection : undefined}
            >
              {col.header}
            </th>
          ))}
        </tr>
      </thead>
      <tbody>
        {data.map((row, i) => (
          <tr key={row.id} role="row">
            {columns.map(col => (
              <td key={col.key} role="cell">
                {row[col.key]}
              </td>
            ))}
          </tr>
        ))}
      </tbody>
    </table>
  );
};
```

### 2.2 Navigation Patterns

#### Main Navigation
```jsx
const Navigation = () => {
  const location = useLocation();
  
  return (
    <nav aria-label="Main navigation">
      <ul role="list">
        <li>
          <NavLink 
            to="/platforms"
            aria-current={location.pathname === '/platforms' ? 'page' : undefined}
          >
            Platforms
          </NavLink>
        </li>
        <li>
          <NavLink 
            to="/monitoring"
            aria-current={location.pathname === '/monitoring' ? 'page' : undefined}
          >
            Monitoring
          </NavLink>
        </li>
      </ul>
    </nav>
  );
};
```

#### Breadcrumbs
```jsx
const Breadcrumbs = ({ items }) => {
  return (
    <nav aria-label="Breadcrumb">
      <ol className="breadcrumb">
        {items.map((item, index) => (
          <li key={item.path}>
            {index === items.length - 1 ? (
              <span aria-current="page">{item.label}</span>
            ) : (
              <Link to={item.path}>{item.label}</Link>
            )}
          </li>
        ))}
      </ol>
    </nav>
  );
};
```

### 2.3 Feedback Patterns

#### Loading States
```jsx
const LoadingSpinner = ({ label = "Loading..." }) => {
  return (
    <div 
      role="status"
      aria-live="polite"
      aria-label={label}
      className="spinner"
    >
      <span className="visually-hidden">{label}</span>
      <SpinnerIcon aria-hidden="true" />
    </div>
  );
};
```

#### Error Messages
```jsx
const ErrorBoundary = ({ children }) => {
  const [hasError, setHasError] = useState(false);
  
  if (hasError) {
    return (
      <div role="alert" className="error-boundary">
        <h2>Something went wrong</h2>
        <p>We're sorry, but an error occurred. Please try refreshing the page.</p>
        <button onClick={() => window.location.reload()}>
          Refresh Page
        </button>
      </div>
    );
  }
  
  return children;
};
```

---

## 3. Keyboard Navigation

### 3.1 Navigation Patterns

#### Tab Order
```jsx
// Correct tab order management
const Dashboard = () => {
  return (
    <div>
      <SkipLink href="#main">Skip to main content</SkipLink>
      <Header tabIndex={-1} /> {/* Not in tab order */}
      <Nav />
      <main id="main" tabIndex={-1}> {/* Programmatic focus target */}
        <h1>Dashboard</h1>
        <PlatformList />
      </main>
      <Footer />
    </div>
  );
};
```

#### Keyboard Shortcuts
```jsx
// Accessible keyboard shortcuts
const KeyboardShortcuts = () => {
  useEffect(() => {
    const handleKeyboard = (e) => {
      // Only when no input is focused
      if (document.activeElement.tagName === 'INPUT') return;
      
      // Alt + key combinations
      if (e.altKey) {
        switch(e.key) {
          case 'p': // Alt+P: Go to platforms
            navigate('/platforms');
            break;
          case 's': // Alt+S: Focus search
            document.getElementById('search')?.focus();
            break;
          case '/': // Alt+/: Show shortcuts help
            setShowShortcuts(true);
            break;
        }
      }
    };
    
    window.addEventListener('keydown', handleKeyboard);
    return () => window.removeEventListener('keydown', handleKeyboard);
  }, []);
  
  return null;
};
```

### 3.2 Focus Management

#### Focus Trap
```jsx
// Focus trap utility
const useFocusTrap = (isActive) => {
  const containerRef = useRef(null);
  
  useEffect(() => {
    if (!isActive) return;
    
    const container = containerRef.current;
    const focusableElements = container.querySelectorAll(
      'a[href], button, textarea, input, select, [tabindex]:not([tabindex="-1"])'
    );
    
    const firstElement = focusableElements[0];
    const lastElement = focusableElements[focusableElements.length - 1];
    
    const handleTab = (e) => {
      if (e.key !== 'Tab') return;
      
      if (e.shiftKey) {
        if (document.activeElement === firstElement) {
          e.preventDefault();
          lastElement.focus();
        }
      } else {
        if (document.activeElement === lastElement) {
          e.preventDefault();
          firstElement.focus();
        }
      }
    };
    
    container.addEventListener('keydown', handleTab);
    firstElement?.focus();
    
    return () => container.removeEventListener('keydown', handleTab);
  }, [isActive]);
  
  return containerRef;
};
```

#### Roving Tabindex
```jsx
// Roving tabindex for toolbars
const Toolbar = ({ tools }) => {
  const [activeIndex, setActiveIndex] = useState(0);
  
  const handleKeyDown = (e, index) => {
    let newIndex = index;
    
    switch(e.key) {
      case 'ArrowRight':
        newIndex = (index + 1) % tools.length;
        break;
      case 'ArrowLeft':
        newIndex = (index - 1 + tools.length) % tools.length;
        break;
      case 'Home':
        newIndex = 0;
        break;
      case 'End':
        newIndex = tools.length - 1;
        break;
      default:
        return;
    }
    
    e.preventDefault();
    setActiveIndex(newIndex);
    document.getElementById(`tool-${newIndex}`)?.focus();
  };
  
  return (
    <div role="toolbar" aria-label="Platform actions">
      {tools.map((tool, index) => (
        <button
          key={tool.id}
          id={`tool-${index}`}
          tabIndex={index === activeIndex ? 0 : -1}
          onKeyDown={(e) => handleKeyDown(e, index)}
          onClick={tool.action}
          aria-label={tool.label}
        >
          <tool.Icon />
        </button>
      ))}
    </div>
  );
};
```

---

## 4. Screen Reader Compatibility

### 4.1 ARIA Usage

#### Live Regions
```jsx
// Announcing dynamic content
const StatusAnnouncer = () => {
  return (
    <>
      {/* Polite announcements */}
      <div 
        role="status"
        aria-live="polite"
        aria-atomic="true"
        className="visually-hidden"
        id="status-announcer"
      />
      
      {/* Urgent announcements */}
      <div 
        role="alert"
        aria-live="assertive"
        aria-atomic="true"
        className="visually-hidden"
        id="alert-announcer"
      />
    </>
  );
};

// Usage
const announce = (message, urgent = false) => {
  const announcer = document.getElementById(
    urgent ? 'alert-announcer' : 'status-announcer'
  );
  announcer.textContent = message;
  
  // Clear after announcement
  setTimeout(() => {
    announcer.textContent = '';
  }, 1000);
};
```

#### Landmark Roles
```jsx
const Layout = () => {
  return (
    <div>
      <header role="banner">
        <Logo />
        <Navigation />
      </header>
      
      <div className="container">
        <aside role="complementary" aria-label="Platform filters">
          <Filters />
        </aside>
        
        <main role="main" aria-label="Platform list">
          <h1>Observability Platforms</h1>
          <PlatformList />
        </main>
      </div>
      
      <footer role="contentinfo">
        <Copyright />
      </footer>
    </div>
  );
};
```

### 4.2 Content Structure

#### Heading Hierarchy
```jsx
// Correct heading structure
const PlatformDetail = ({ platform }) => {
  return (
    <article>
      <h1>{platform.name}</h1>
      
      <section aria-labelledby="status-heading">
        <h2 id="status-heading">Status</h2>
        <StatusDashboard />
      </section>
      
      <section aria-labelledby="components-heading">
        <h2 id="components-heading">Components</h2>
        
        <article>
          <h3>Prometheus</h3>
          <PrometheusConfig />
        </article>
        
        <article>
          <h3>Grafana</h3>
          <GrafanaConfig />
        </article>
      </section>
    </article>
  );
};
```

---

## 5. Color and Contrast

### 5.1 Color Palette

```css
/* Accessible color system */
:root {
  /* Primary colors with AA/AAA compliance */
  --color-primary: #0066CC;        /* 4.5:1 on white */
  --color-primary-dark: #004499;   /* 7:1 on white */
  
  /* Status colors */
  --color-success: #008844;        /* 4.5:1 on white */
  --color-warning: #CC6600;        /* 4.5:1 on white */
  --color-error: #CC0000;          /* 4.5:1 on white */
  --color-info: #0066CC;           /* 4.5:1 on white */
  
  /* Neutral colors */
  --color-text: #1A1A1A;           /* 17:1 on white */
  --color-text-secondary: #595959; /* 7:1 on white */
  --color-text-disabled: #999999;  /* 4.5:1 on white */
  
  /* Background colors */
  --color-bg: #FFFFFF;
  --color-bg-secondary: #F5F5F5;
  --color-bg-hover: #E8E8E8;
}

/* Dark mode with maintained contrast */
@media (prefers-color-scheme: dark) {
  :root {
    --color-primary: #4D94FF;      /* 4.5:1 on black */
    --color-text: #E6E6E6;         /* 14:1 on black */
    --color-bg: #1A1A1A;
  }
}
```

### 5.2 Contrast Requirements

#### Text Contrast
```css
/* Ensure minimum contrast ratios */
.text-normal {
  color: var(--color-text);         /* 17:1 - exceeds AAA */
  font-size: 16px;
}

.text-large {
  color: var(--color-text-secondary); /* 7:1 - meets AAA for large text */
  font-size: 24px;
  font-weight: 400;
}

/* Interactive elements */
.button {
  background: var(--color-primary);
  color: white;                     /* 4.5:1 minimum */
  border: 2px solid transparent;
}

.button:focus {
  border-color: var(--color-primary-dark);
  outline: 3px solid var(--color-primary);
  outline-offset: 2px;
}
```

---

## 6. Testing Framework

### 6.1 Automated Testing

#### Jest + Testing Library
```javascript
// Accessibility testing utilities
import { render, screen } from '@testing-library/react';
import { axe, toHaveNoViolations } from 'jest-axe';

expect.extend(toHaveNoViolations);

describe('Button Component', () => {
  it('should be accessible', async () => {
    const { container } = render(
      <Button onClick={jest.fn()}>
        Deploy Platform
      </Button>
    );
    
    const results = await axe(container);
    expect(results).toHaveNoViolations();
  });
  
  it('should have proper ARIA attributes when loading', () => {
    render(
      <Button loading onClick={jest.fn()}>
        Deploy Platform
      </Button>
    );
    
    const button = screen.getByRole('button');
    expect(button).toHaveAttribute('aria-busy', 'true');
    expect(button).toHaveAttribute('aria-disabled', 'true');
  });
});
```

#### Cypress E2E Testing
```javascript
// Cypress accessibility tests
describe('Platform Dashboard Accessibility', () => {
  beforeEach(() => {
    cy.visit('/platforms');
    cy.injectAxe(); // Inject axe-core
  });
  
  it('should have no accessibility violations', () => {
    cy.checkA11y(); // Check entire page
  });
  
  it('should be keyboard navigable', () => {
    // Tab through interface
    cy.get('body').tab();
    cy.focused().should('have.attr', 'href', '#main');
    
    // Tab to first platform
    cy.tab();
    cy.focused().should('contain', 'Platform 1');
    
    // Activate with Enter
    cy.focused().type('{enter}');
    cy.url().should('include', '/platforms/1');
  });
  
  it('should announce status changes', () => {
    cy.get('[data-testid="deploy-button"]').click();
    
    // Check live region updated
    cy.get('[role="status"]')
      .should('contain', 'Deployment started');
  });
});
```

### 6.2 Manual Testing

#### Screen Reader Testing Checklist
```markdown
## Screen Reader Testing Protocol

### NVDA (Windows)
- [ ] All content is announced in logical order
- [ ] Interactive elements announce their role and state
- [ ] Form labels are associated correctly
- [ ] Error messages are announced
- [ ] Dynamic content updates are announced
- [ ] Landmarks allow quick navigation

### JAWS (Windows)
- [ ] Headings hierarchy is correct
- [ ] Tables have proper headers
- [ ] Links are descriptive
- [ ] Buttons vs links used appropriately
- [ ] Modal dialogs trap focus correctly

### VoiceOver (macOS/iOS)
- [ ] Rotor navigation works correctly
- [ ] Touch gestures work on mobile
- [ ] Images have appropriate alt text
- [ ] Complex widgets are operable

### TalkBack (Android)
- [ ] All functionality available via gestures
- [ ] Focus indicators visible
- [ ] Reading order is logical
```

#### Keyboard Testing Protocol
```markdown
## Keyboard Navigation Testing

### General Navigation
- [ ] Tab key moves through all interactive elements
- [ ] Shift+Tab moves backwards
- [ ] No keyboard traps
- [ ] Skip links work correctly

### Component-Specific
- [ ] Enter/Space activate buttons
- [ ] Arrow keys navigate menus
- [ ] Escape closes modals/menus
- [ ] Home/End keys work in lists

### Custom Shortcuts
- [ ] All shortcuts documented
- [ ] Don't conflict with browser/AT shortcuts
- [ ] Can be disabled/remapped
```

### 6.3 Accessibility Audit Tools

```javascript
// Automated audit script
const auditAccessibility = async () => {
  const lighthouse = require('lighthouse');
  const chromeLauncher = require('chrome-launcher');
  
  const chrome = await chromeLauncher.launch({chromeFlags: ['--headless']});
  const options = {
    logLevel: 'info',
    output: 'json',
    onlyCategories: ['accessibility'],
    port: chrome.port
  };
  
  const runnerResult = await lighthouse('http://localhost:3000', options);
  
  // Process results
  const score = runnerResult.lhr.categories.accessibility.score * 100;
  console.log(`Accessibility score: ${score}`);
  
  // Generate report
  const reportHtml = runnerResult.report;
  fs.writeFileSync('accessibility-audit.html', reportHtml);
  
  await chrome.kill();
};
```

---

## 7. Implementation Checklist

### Development Phase
- [ ] Install accessibility linting (eslint-plugin-jsx-a11y)
- [ ] Configure automated testing (jest-axe, cypress-axe)
- [ ] Set up color contrast analyzer
- [ ] Create component library with accessibility built-in
- [ ] Document keyboard shortcuts
- [ ] Create skip navigation links

### Testing Phase
- [ ] Run automated accessibility tests
- [ ] Perform manual keyboard testing
- [ ] Test with screen readers (NVDA, JAWS, VoiceOver)
- [ ] Verify color contrast ratios
- [ ] Test with browser zoom at 200%
- [ ] Test responsive design down to 320px

### Deployment Phase
- [ ] Include accessibility in PR checklist
- [ ] Monitor accessibility metrics
- [ ] Set up user feedback mechanism
- [ ] Document known issues
- [ ] Plan for regular audits

---

## 8. Resources and Tools

### Development Tools
- **axe DevTools**: Browser extension for testing
- **WAVE**: Web Accessibility Evaluation Tool
- **Lighthouse**: Automated auditing
- **Pa11y**: Command line accessibility testing
- **Stark**: Figma/Sketch plugin for designers

### Screen Readers
- **NVDA**: Free Windows screen reader
- **JAWS**: Commercial Windows screen reader
- **VoiceOver**: Built-in macOS/iOS screen reader
- **TalkBack**: Android screen reader
- **Orca**: Linux screen reader

### References
- [WCAG 2.1 Guidelines](https://www.w3.org/WAI/WCAG21/quickref/)
- [ARIA Authoring Practices](https://www.w3.org/WAI/ARIA/apg/)
- [WebAIM Resources](https://webaim.org/resources/)
- [A11y Project](https://www.a11yproject.com/)
- [MDN Accessibility](https://developer.mozilla.org/en-US/docs/Web/Accessibility)

---

**This document is a living standard and will be updated as accessibility best practices evolve.**

*For questions about accessibility, contact accessibility@gunjoperator.io*
