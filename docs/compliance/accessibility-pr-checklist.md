# Accessibility Checklist for Pull Request Reviews

**Project**: Gunj Operator  
**Version**: 1.0  
**WCAG Target**: 2.1 Level AA  

Use this checklist when reviewing PRs that include UI changes.

---

## 🎯 Quick Checks (Required for All PRs)

### Automated Testing
- [ ] ESLint jsx-a11y rules pass without errors
- [ ] jest-axe tests pass
- [ ] No accessibility violations in CI pipeline

### Basic Accessibility
- [ ] All images have appropriate `alt` text
- [ ] Form inputs have associated labels
- [ ] Buttons have accessible names
- [ ] Color is not the only means of conveying information
- [ ] Interactive elements have focus indicators

---

## 📋 Detailed Review Checklist

### 1. Keyboard Navigation
- [ ] **Tab Order**: Can navigate all interactive elements with Tab key
- [ ] **Focus Visible**: Focus indicator is clearly visible
- [ ] **No Keyboard Traps**: Can exit all components with keyboard
- [ ] **Skip Links**: Skip navigation links present if needed
- [ ] **Shortcuts**: Custom keyboard shortcuts don't conflict with screen readers

```jsx
// ✅ Good
<button onKeyDown={handleKeyDown} tabIndex={0}>

// ❌ Bad  
<div onClick={handleClick}> // Not keyboard accessible
```

### 2. Screen Reader Support
- [ ] **Semantic HTML**: Uses appropriate HTML5 elements
- [ ] **ARIA Labels**: Interactive elements have labels when text not visible
- [ ] **ARIA Roles**: Custom components have appropriate roles
- [ ] **Live Regions**: Dynamic content updates announced
- [ ] **Landmarks**: Page regions properly marked

```jsx
// ✅ Good
<nav aria-label="Main navigation">
<main>
<button aria-label="Close dialog">

// ❌ Bad
<div class="navigation"> // Missing semantic meaning
```

### 3. Visual Design
- [ ] **Color Contrast**: Text meets WCAG AA ratios (4.5:1 normal, 3:1 large)
- [ ] **Focus Indicators**: 3:1 contrast ratio for focus outlines
- [ ] **Text Resizing**: Content reflows at 200% zoom
- [ ] **Responsive**: Works down to 320px width
- [ ] **Motion**: Respects prefers-reduced-motion

```css
/* ✅ Good */
color: #1A1A1A; /* 17:1 on white */
outline: 3px solid #0066CC;

/* ❌ Bad */
color: #999999; /* May not meet contrast on all backgrounds */
outline: none; /* Never remove without replacement */
```

### 4. Forms
- [ ] **Labels**: All inputs have visible labels or aria-label
- [ ] **Error Messages**: Clear error identification and suggestions
- [ ] **Required Fields**: Marked with text and aria-required
- [ ] **Field Instructions**: Help text associated with inputs
- [ ] **Error Prevention**: Confirmation for destructive actions

```jsx
// ✅ Good
<FormField
  label="Email Address"
  error={errors.email}
  required
  hint="We'll never share your email"
>
  <input type="email" />
</FormField>
```

### 5. Images & Media
- [ ] **Alt Text**: Descriptive for informative images, empty for decorative
- [ ] **Complex Images**: Long descriptions provided when needed
- [ ] **Videos**: Captions and audio descriptions available
- [ ] **Audio**: Transcripts provided
- [ ] **Animations**: Can be paused/stopped

```jsx
// ✅ Good
<img src="chart.png" alt="Sales increased 25% from Q1 to Q2" />
<img src="decoration.png" alt="" /> // Decorative

// ❌ Bad
<img src="chart.png" alt="chart" /> // Not descriptive
```

### 6. Interactive Components
- [ ] **Buttons vs Links**: Buttons for actions, links for navigation
- [ ] **Touch Targets**: Minimum 44x44px
- [ ] **Loading States**: Announced to screen readers
- [ ] **Status Updates**: Success/error messages announced
- [ ] **Modals**: Focus trapped and returned on close

```jsx
// ✅ Good
<button aria-busy={loading}>
  {loading && <Spinner />}
  Save Changes
</button>
```

### 7. Data Tables
- [ ] **Headers**: Proper `<th>` elements with scope
- [ ] **Caption**: Table purpose described
- [ ] **Sort Indicators**: Sort state announced
- [ ] **Row Actions**: Accessible to keyboard
- [ ] **Responsive**: Usable on mobile

```jsx
// ✅ Good
<table>
  <caption>Platform resource usage</caption>
  <thead>
    <tr>
      <th scope="col">Platform</th>
      <th scope="col" aria-sort="ascending">CPU</th>
    </tr>
  </thead>
</table>
```

### 8. Error Handling
- [ ] **Error Identification**: Errors clearly marked
- [ ] **Error Description**: Helpful error messages
- [ ] **Error Association**: Errors linked to fields
- [ ] **Success Feedback**: Positive actions confirmed
- [ ] **Recovery**: Can recover from errors

### 9. Documentation
- [ ] **Component Docs**: Accessibility features documented
- [ ] **Keyboard Shortcuts**: All shortcuts documented
- [ ] **Screen Reader Notes**: Special instructions noted
- [ ] **Examples**: Include accessible examples

---

## 🧪 Testing Requirements

### Manual Testing
- [ ] **Keyboard Only**: Navigate entire feature with keyboard
- [ ] **Screen Reader**: Test with at least one screen reader
- [ ] **Zoom**: Test at 200% browser zoom
- [ ] **Color**: Test with Windows High Contrast mode
- [ ] **Mobile**: Test with mobile screen reader

### Browser/AT Combinations to Test
Priority combinations for testing:
1. NVDA + Chrome/Firefox (Windows)
2. VoiceOver + Safari (macOS)
3. TalkBack + Chrome (Android)
4. VoiceOver + Safari (iOS)

---

## 🚫 Common Issues to Check

### Anti-Patterns to Avoid
- [ ] No positive `tabindex` values (1, 2, 3, etc.)
- [ ] No `placeholder` as sole label
- [ ] No auto-playing media with sound
- [ ] No focus outline removal without alternative
- [ ] No clickable `<div>` or `<span>` without role="button"
- [ ] No empty headings or labels
- [ ] No skipped heading levels (h1 → h3)

### Code Smells
```jsx
// 🚫 Avoid these patterns:

// Positive tabindex
<button tabIndex="1">

// Placeholder as label
<input placeholder="Enter email" /> // No label!

// Click handlers on non-interactive elements
<div onClick={handleClick}>Click me</div>

// Removing focus indicators
button:focus { outline: none; }

// Auto-focus without user action
useEffect(() => { inputRef.current.focus() }, [])
```

---

## ✅ PR Approval Criteria

### Must Fix (Blocks Merge)
- [ ] Keyboard navigation works
- [ ] Screen reader can access all content
- [ ] No WCAG Level A violations
- [ ] Color contrast meets AA standards
- [ ] Forms are accessible

### Should Fix (Before Next Release)
- [ ] Minor WCAG AA issues
- [ ] Performance optimizations for AT
- [ ] Enhanced keyboard shortcuts
- [ ] Improved error messages

### Nice to Have (Future Enhancement)
- [ ] WCAG AAA compliance
- [ ] Advanced ARIA patterns
- [ ] Animated focus indicators
- [ ] Accessibility preferences

---

## 📚 Resources for Reviewers

### Quick References
- [WCAG 2.1 Quick Reference](https://www.w3.org/WAI/WCAG21/quickref/)
- [ARIA Authoring Practices](https://www.w3.org/WAI/ARIA/apg/)
- [WebAIM Color Contrast Checker](https://webaim.org/resources/contrastchecker/)

### Testing Tools
- [axe DevTools](https://www.deque.com/axe/devtools/) - Browser extension
- [WAVE](https://wave.webaim.org/) - Web accessibility evaluation
- [Lighthouse](https://developers.google.com/web/tools/lighthouse) - Chrome DevTools

### Screen Readers
- [NVDA](https://www.nvaccess.org/) - Free Windows screen reader
- [VoiceOver](https://support.apple.com/guide/voiceover/welcome/mac) - Built into macOS/iOS

---

## 💬 Review Comments Templates

### For Missing Alt Text
```
The image on line X needs descriptive alt text for screen reader users. 
Example: `alt="Platform CPU usage graph showing 75% utilization"`
```

### For Missing Labels
```
The input field on line X needs an associated label.
Either add a visible `<label>` or use `aria-label` for screen readers.
```

### For Contrast Issues
```
The text color (#XXX) on background (#YYY) has a contrast ratio of X:1, 
which doesn't meet WCAG AA standards (4.5:1 required).
Consider using our color variables that ensure proper contrast.
```

### For Keyboard Access
```
This interactive element needs keyboard support.
Add `tabIndex={0}` and handle both click and keyboard events.
```

---

**Remember**: Accessibility is not optional. It's a core requirement that ensures our platform is usable by everyone. When in doubt, ask for help or consult the accessibility team.
