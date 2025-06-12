# Frontend React Setup Decision

**Date**: June 12, 2025  
**Status**: Accepted  
**Micro-task**: Phase 1.2.2 - Task 1 - Finalize React version and setup

## Decision

We will use **Next.js 14.2.x with React 18.3 LTS** for the Gunj Operator UI.

## Context

The Gunj Operator UI requires:
- High performance (< 3 seconds initial load)
- Real-time updates via WebSocket/SSE
- Complex data visualizations
- Offline support capability
- Enterprise-grade stability
- Containerized deployment separate from Go API

## Considered Options

### Option 1: React 19 + Next.js 15 (User Preference)
**Pros:**
- Latest features (Server Components, improved Suspense)
- Better concurrent rendering
- Enhanced performance potential

**Cons:**
- Still in RC/beta (as of June 2025)
- Limited ecosystem compatibility
- Potential stability issues for enterprise

### Option 2: React 18.3 + Next.js 14.2 (Recommended)
**Pros:**
- LTS stability for enterprise
- Mature ecosystem support
- Production-proven
- Smooth upgrade path to React 19

**Cons:**
- Missing newest React features
- Requires future migration

### Option 3: React 18.3 + Vite 5.x
**Pros:**
- Fastest development builds
- Simpler configuration
- Lightweight setup

**Cons:**
- Lacks production optimizations
- No built-in SSG/SSR
- More manual optimization needed

## Decision Rationale

We choose **React 18.3 + Next.js 14.2** because:

1. **Enterprise Stability**: React 18.3 is LTS and battle-tested
2. **Next.js Benefits**:
   - Superior production optimizations
   - Automatic code splitting
   - Built-in performance features
   - Image optimization
   - Static export capability for containerization
3. **Future Ready**: Easy migration path to React 19 when stable
4. **Developer Experience**: Fast refresh, great tooling
5. **CNCF Compliance**: Proven technology stack

## Implementation Details

### Configuration
```json
{
  "dependencies": {
    "react": "^18.3.1",
    "react-dom": "^18.3.1",
    "next": "^14.2.5"
  },
  "devDependencies": {
    "typescript": "^5.3.3",
    "@types/react": "^18.3.3",
    "@types/node": "^20.14.10"
  }
}
```

### Next.js Configuration
```javascript
// next.config.js
/** @type {import('next').NextConfig} */
const nextConfig = {
  output: 'export', // Static export for containerization
  reactStrictMode: true,
  swcMinify: true,
  experimental: {
    optimizeCss: true,
    optimizePackageImports: ['@mui/material', '@mui/icons-material']
  },
  compiler: {
    removeConsole: process.env.NODE_ENV === 'production'
  }
}

module.exports = nextConfig
```

### TypeScript Configuration
```json
{
  "compilerOptions": {
    "target": "ES2022",
    "lib": ["dom", "dom.iterable", "esnext"],
    "allowJs": false,
    "skipLibCheck": true,
    "strict": true,
    "forceConsistentCasingInFileNames": true,
    "noEmit": true,
    "esModuleInterop": true,
    "module": "esnext",
    "moduleResolution": "bundler",
    "resolveJsonModule": true,
    "isolatedModules": true,
    "jsx": "preserve",
    "incremental": true,
    "paths": {
      "@/*": ["./src/*"]
    }
  }
}
```

## Migration Strategy

### Phase 1: Current Implementation (2025)
- React 18.3 LTS + Next.js 14.2
- Monitor React 19 stability
- Prepare codebase for future features

### Phase 2: Future Migration (2026)
- Upgrade to React 19 when LTS
- Leverage new concurrent features
- Minimal code changes required

## Consequences

### Positive
- Enterprise-ready from day one
- Excellent performance out of the box
- Strong community support
- Clear upgrade path

### Negative
- Not using absolute latest React features
- Need future migration effort
- Slightly larger bundle than Vite SPA

## References
- [Next.js 14 Documentation](https://nextjs.org/docs)
- [React 18 LTS Announcement](https://react.dev/blog/2022/03/29/react-v18)
- [Next.js Static Exports](https://nextjs.org/docs/app/building-your-application/deploying/static-exports)
