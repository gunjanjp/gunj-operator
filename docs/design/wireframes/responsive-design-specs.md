# Gunj Operator UI - Responsive Design Specifications

**Version**: 1.0  
**Last Updated**: June 12, 2025  
**Status**: Design Phase  

## 📱 Responsive Design Strategy

This document outlines the responsive design approach for the Gunj Operator UI, ensuring optimal user experience across all device sizes and orientations.

---

## 🎯 Breakpoint System

### Core Breakpoints

```scss
// Base breakpoints aligned with MUI
$breakpoints: (
  xs: 0px,      // Mobile portrait
  sm: 600px,    // Mobile landscape / Small tablet
  md: 960px,    // Tablet / Small laptop
  lg: 1280px,   // Desktop
  xl: 1920px    // Large desktop
);

// Custom breakpoints for specific needs
$custom-breakpoints: (
  mobile: 375px,      // iPhone SE/8 min
  phablet: 414px,    // Large phones
  tablet: 768px,     // iPad portrait
  laptop: 1024px,    // Small laptops
  desktop: 1440px,   // Standard desktop
  wide: 2560px       // 4K displays
);
```

### Device Categories

| Category | Width Range | Typical Devices | Layout Columns |
|----------|------------|-----------------|----------------|
| Mobile | 320-599px | Phones | 4 |
| Tablet | 600-959px | Tablets, small laptops | 8 |
| Desktop | 960-1279px | Laptops, desktops | 12 |
| Wide | 1280px+ | Large monitors | 12 |

---

## 📐 Layout Adaptations

### 1. Navigation Responsive Behavior

#### Mobile (< 600px)
```
Header:
- Height: 56px
- Logo: Centered or left-aligned
- Menu: Hamburger icon right
- Search: Hidden or icon only

Navigation:
- Type: Full-screen drawer
- Trigger: Hamburger menu
- Width: 100% or 280px slide-in
- Backdrop: Dark overlay
```

#### Tablet (600-959px)
```
Header:
- Height: 64px
- Logo: Left-aligned
- Menu: Hamburger or visible items
- Search: Collapsed/expandable

Navigation:
- Type: Collapsible sidebar
- Default: Collapsed (icons only)
- Expanded: 280px on demand
- Behavior: Push content or overlay
```

#### Desktop (960px+)
```
Sidebar:
- Width: 280px (expanded)
- Collapsed: 72px (icons only)
- Position: Fixed left
- Behavior: Push content

Header:
- Full search bar
- All actions visible
- User menu expanded
```

### 2. Grid System Responsive Rules

```scss
// Mobile: Stack everything
@media (max-width: 599px) {
  .grid-container {
    grid-template-columns: 1fr;
    gap: 16px;
  }
}

// Tablet: 2 columns for cards
@media (min-width: 600px) and (max-width: 959px) {
  .grid-container {
    grid-template-columns: repeat(2, 1fr);
    gap: 20px;
  }
}

// Desktop: Flexible columns
@media (min-width: 960px) {
  .grid-container {
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 24px;
  }
}
```

### 3. Content Width Constraints

```scss
.content-container {
  width: 100%;
  margin: 0 auto;
  padding: 0 16px;
  
  @media (min-width: 600px) {
    padding: 0 24px;
  }
  
  @media (min-width: 960px) {
    padding: 0 32px;
    max-width: 1440px;
  }
  
  @media (min-width: 1920px) {
    padding: 0 48px;
    max-width: 1680px;
  }
}
```

---

## 📊 Component Responsive Patterns

### Platform Cards

#### Mobile Layout
```
┌─────────────────────┐
│ Platform Name       │
│ Status Badge        │
├─────────────────────┤
│ Namespace: prod     │
│ Version: v2.48.0    │
│ Components: 5       │
├─────────────────────┤
│ [View] [Actions ⋮] │
└─────────────────────┘
```

#### Tablet Layout
```
┌─────────────────────────────┐
│ Platform Name    Status     │
│ Namespace: prod             │
│ Version: v2.48.0            │
│ Components: P G L T A       │
│ [View] [Edit] [More ⋮]     │
└─────────────────────────────┘
```

#### Desktop Layout
```
┌───────────────────────────────────────────┐
│ Platform Name          Status: Ready      │
│ Namespace: production  Created: 2 days ago│
│ Version: v2.48.0      Updated: 1 hour ago│
│ Components: [P][G][L][T][A]              │
│ Resources: 8 CPU, 32GB RAM, 100GB Storage│
│ [View] [Edit] [Clone] [Delete]           │
└───────────────────────────────────────────┘
```

### Data Tables

#### Mobile: Card View
```typescript
// Transform table to cards on mobile
<Box sx={{ display: { xs: 'block', md: 'none' } }}>
  {rows.map((row) => (
    <Card key={row.id} sx={{ mb: 2 }}>
      <CardContent>
        <Typography variant="h6">{row.name}</Typography>
        <Typography color="textSecondary">
          Status: {row.status}
        </Typography>
        <Typography variant="body2">
          Version: {row.version}
        </Typography>
      </CardContent>
      <CardActions>
        <Button size="small">View</Button>
        <IconButton size="small">
          <MoreVertIcon />
        </IconButton>
      </CardActions>
    </Card>
  ))}
</Box>
```

#### Tablet+: Table View with Responsive Columns
```typescript
// Show/hide columns based on screen size
const columns = [
  { field: 'name', headerName: 'Name', minWidth: 150, flex: 1 },
  { 
    field: 'status', 
    headerName: 'Status', 
    width: 120,
    hideable: false  // Always show
  },
  { 
    field: 'version', 
    headerName: 'Version', 
    width: 100,
    hide: { xs: true, sm: false }  // Hide on mobile
  },
  { 
    field: 'created', 
    headerName: 'Created', 
    width: 150,
    hide: { xs: true, sm: true, md: false }  // Desktop only
  },
];
```

### Forms

#### Mobile: Single Column
```
┌─────────────────────┐
│ Platform Name       │
│ [________________] │
│                    │
│ Namespace          │
│ [________________] │
│                    │
│ Environment        │
│ [Select ▼]         │
│                    │
│ [Cancel] [Create]  │
└─────────────────────┘
```

#### Desktop: Multi-Column
```
┌─────────────────────────────────────┐
│ Platform Name      Namespace        │
│ [_______________] [_______________] │
│                                     │
│ Environment        Team             │
│ [Select ▼]        [Select ▼]       │
│                                     │
│ Description                         │
│ [_________________________________]│
│ [_________________________________]│
│                                     │
│              [Cancel] [Create]      │
└─────────────────────────────────────┘
```

### Charts & Visualizations

#### Responsive Chart Sizing
```typescript
const getChartDimensions = (width: number) => {
  if (width < 600) {
    return { height: 200, showLegend: false, simplified: true };
  } else if (width < 960) {
    return { height: 300, showLegend: true, simplified: false };
  }
  return { height: 400, showLegend: true, simplified: false };
};
```

#### Mobile Optimizations
- Simplified axes labels
- Touch-friendly tooltips
- Horizontal scroll for time series
- Collapsible legends
- Reduced data points

---

## 🎨 Typography Scaling

### Responsive Type Scale

```scss
// Base font sizes
$font-sizes: (
  xs: (
    h1: 2rem,      // 32px
    h2: 1.5rem,    // 24px
    h3: 1.25rem,   // 20px
    body: 0.875rem // 14px
  ),
  sm: (
    h1: 2.5rem,    // 40px
    h2: 1.875rem,  // 30px
    h3: 1.5rem,    // 24px
    body: 0.875rem // 14px
  ),
  md: (
    h1: 3rem,      // 48px
    h2: 2.125rem,  // 34px
    h3: 1.5rem,    // 24px
    body: 1rem     // 16px
  )
);
```

### Line Length Control
```scss
.text-content {
  max-width: 65ch; // Optimal reading length
  
  @media (min-width: 1280px) {
    column-count: 2;
    column-gap: 32px;
  }
}
```

---

## 📏 Spacing System

### Responsive Spacing Scale

```typescript
const spacing = {
  xs: {
    unit: 4,
    containerPadding: 16,
    sectionGap: 24,
    componentGap: 16
  },
  sm: {
    unit: 8,
    containerPadding: 24,
    sectionGap: 32,
    componentGap: 20
  },
  md: {
    unit: 8,
    containerPadding: 32,
    sectionGap: 48,
    componentGap: 24
  },
  lg: {
    unit: 8,
    containerPadding: 48,
    sectionGap: 64,
    componentGap: 32
  }
};
```

---

## 🖼️ Image & Media Handling

### Responsive Images
```html
<picture>
  <source 
    media="(max-width: 599px)" 
    srcset="image-mobile.webp 1x, image-mobile@2x.webp 2x"
  />
  <source 
    media="(max-width: 959px)" 
    srcset="image-tablet.webp 1x, image-tablet@2x.webp 2x"
  />
  <source 
    media="(min-width: 960px)" 
    srcset="image-desktop.webp 1x, image-desktop@2x.webp 2x"
  />
  <img 
    src="image-fallback.jpg" 
    alt="Platform overview"
    loading="lazy"
  />
</picture>
```

### Icon Sizing
```typescript
const getIconSize = (breakpoint: string): number => {
  switch(breakpoint) {
    case 'xs': return 20;
    case 'sm': return 24;
    case 'md': 
    default: return 28;
  }
};
```

---

## 🎯 Touch Target Sizing

### Minimum Touch Targets
```scss
// Mobile touch targets
@media (hover: none) and (pointer: coarse) {
  .clickable {
    min-height: 44px;
    min-width: 44px;
    
    // Add padding for small elements
    &.small {
      padding: 12px;
      margin: -12px;
    }
  }
}

// Desktop can be smaller
@media (hover: hover) and (pointer: fine) {
  .clickable {
    min-height: 32px;
    min-width: 32px;
  }
}
```

---

## 📋 Responsive Patterns Library

### 1. Hide/Show Pattern
```tsx
// Hide on mobile
<Box sx={{ display: { xs: 'none', sm: 'block' } }}>
  Desktop only content
</Box>

// Show only on mobile
<Box sx={{ display: { xs: 'block', sm: 'none' } }}>
  Mobile only content
</Box>
```

### 2. Stack/Inline Pattern
```tsx
<Stack 
  direction={{ xs: 'column', sm: 'row' }}
  spacing={{ xs: 2, sm: 3 }}
  alignItems={{ xs: 'stretch', sm: 'center' }}
>
  <Item>First</Item>
  <Item>Second</Item>
  <Item>Third</Item>
</Stack>
```

### 3. Drawer/Sidebar Pattern
```tsx
const isMobile = useMediaQuery('(max-width: 599px)');

return isMobile ? (
  <Drawer 
    variant="temporary"
    open={open}
    onClose={handleClose}
  >
    <NavigationContent />
  </Drawer>
) : (
  <Drawer
    variant="permanent"
    sx={{ width: drawerWidth }}
  >
    <NavigationContent />
  </Drawer>
);
```

### 4. Progressive Disclosure
```tsx
// Show more details on larger screens
<Card>
  <CardContent>
    <Typography variant="h6">{title}</Typography>
    <Typography 
      sx={{ 
        display: { xs: 'none', sm: 'block' } 
      }}
    >
      {description}
    </Typography>
    <Box 
      sx={{ 
        display: { xs: 'none', md: 'block' } 
      }}
    >
      <DetailedMetrics />
    </Box>
  </CardContent>
</Card>
```

---

## 🔄 Responsive State Management

### Viewport-Based Features
```typescript
const useResponsiveFeatures = () => {
  const isMobile = useMediaQuery('(max-width: 599px)');
  const isTablet = useMediaQuery('(min-width: 600px) and (max-width: 959px)');
  const isDesktop = useMediaQuery('(min-width: 960px)');
  
  return {
    navigation: isMobile ? 'drawer' : 'sidebar',
    dataDisplay: isMobile ? 'cards' : 'table',
    chartsSimplified: isMobile,
    showAdvancedFeatures: isDesktop,
    itemsPerPage: isMobile ? 10 : isTablet ? 25 : 50
  };
};
```

---

## 📱 Platform-Specific Considerations

### iOS Safari
```scss
// Handle safe areas
.container {
  padding-left: env(safe-area-inset-left);
  padding-right: env(safe-area-inset-right);
  padding-bottom: env(safe-area-inset-bottom);
}

// Prevent zoom on input focus
input, select, textarea {
  font-size: 16px; // Prevents zoom on iOS
}
```

### Android Chrome
```scss
// Address bar color
<meta name="theme-color" content="#1976D2">

// Smooth scrolling
.scrollable {
  -webkit-overflow-scrolling: touch;
  overscroll-behavior: contain;
}
```

---

## 🧪 Responsive Testing Checklist

### Devices to Test
- [ ] iPhone SE (375px)
- [ ] iPhone 12/13 (390px)
- [ ] iPhone 14 Pro Max (430px)
- [ ] iPad Mini (768px)
- [ ] iPad Pro 11" (834px)
- [ ] iPad Pro 12.9" (1024px)
- [ ] MacBook Air (1280px)
- [ ] Desktop (1920px)
- [ ] 4K Display (3840px)

### Orientations
- [ ] Portrait mode
- [ ] Landscape mode
- [ ] Split-screen (tablets)

### Key Test Scenarios
1. [ ] Navigation accessibility
2. [ ] Form input usability
3. [ ] Table/card transitions
4. [ ] Chart readability
5. [ ] Touch target sizes
6. [ ] Text readability
7. [ ] Image loading
8. [ ] Scroll performance
9. [ ] Gesture support
10. [ ] Keyboard navigation

---

This comprehensive responsive design specification ensures the Gunj Operator UI provides an optimal experience across all devices and screen sizes.
