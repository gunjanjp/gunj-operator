/** @type {import('next').NextConfig} */
const nextConfig = {
  // Static export for containerization
  output: 'export',
  
  // React configuration
  reactStrictMode: true,
  
  // Performance optimizations
  swcMinify: true,
  
  // Experimental features for optimization
  experimental: {
    optimizeCss: true,
    optimizePackageImports: [
      '@mui/material',
      '@mui/icons-material',
      'recharts',
      'd3',
      'lodash'
    ]
  },
  
  // Compiler options
  compiler: {
    // Remove console logs in production
    removeConsole: process.env.NODE_ENV === 'production' ? {
      exclude: ['error', 'warn']
    } : false,
    
    // Enable styled-components if we use it
    styledComponents: false
  },
  
  // Image optimization (even for static export)
  images: {
    unoptimized: true, // Required for static export
    formats: ['image/avif', 'image/webp']
  },
  
  // Security headers
  async headers() {
    return [
      {
        source: '/:path*',
        headers: [
          {
            key: 'X-Frame-Options',
            value: 'DENY'
          },
          {
            key: 'X-Content-Type-Options',
            value: 'nosniff'
          },
          {
            key: 'X-XSS-Protection',
            value: '1; mode=block'
          },
          {
            key: 'Referrer-Policy',
            value: 'strict-origin-when-cross-origin'
          }
        ]
      }
    ]
  },
  
  // Environment variables
  env: {
    NEXT_PUBLIC_API_URL: process.env.NEXT_PUBLIC_API_URL || 'http://localhost:8080',
    NEXT_PUBLIC_WS_URL: process.env.NEXT_PUBLIC_WS_URL || 'ws://localhost:8080',
    NEXT_PUBLIC_VERSION: process.env.npm_package_version || '2.0.0'
  },
  
  // Webpack configuration
  webpack: (config, { isServer, dev }) => {
    // Bundle analyzer
    if (process.env.ANALYZE === 'true') {
      const { BundleAnalyzerPlugin } = require('webpack-bundle-analyzer');
      config.plugins.push(
        new BundleAnalyzerPlugin({
          analyzerMode: 'static',
          openAnalyzer: true
        })
      );
    }
    
    // Custom webpack configs
    config.resolve.fallback = {
      ...config.resolve.fallback,
      fs: false,
      net: false,
      tls: false
    };
    
    return config;
  },
  
  // Trailing slash behavior
  trailingSlash: true,
  
  // Disable x-powered-by header
  poweredByHeader: false,
  
  // Generate build ID
  generateBuildId: async () => {
    // Use git commit hash if available
    if (process.env.GIT_COMMIT_SHA) {
      return process.env.GIT_COMMIT_SHA.substring(0, 8);
    }
    return `build-${Date.now()}`;
  }
};

module.exports = nextConfig;
