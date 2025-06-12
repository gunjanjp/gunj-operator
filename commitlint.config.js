// Commitlint configuration for Gunj Operator
// Enforces Conventional Commits specification

module.exports = {
  extends: ['@commitlint/config-conventional'],
  rules: {
    // Type enum
    'type-enum': [
      2,
      'always',
      [
        'feat',     // New feature
        'fix',      // Bug fix
        'docs',     // Documentation changes
        'style',    // Code style changes (formatting, etc)
        'refactor', // Code refactoring
        'perf',     // Performance improvements
        'test',     // Adding or updating tests
        'chore',    // Maintenance tasks
        'build',    // Build system changes
        'ci',       // CI/CD changes
        'revert',   // Revert previous commit
        'wip',      // Work in progress (for draft PRs)
      ],
    ],
    
    // Scope enum - components of the project
    'scope-enum': [
      2,
      'always',
      [
        // Core components
        'operator',
        'api',
        'ui',
        'cli',
        
        // Operator specifics
        'controller',
        'webhook',
        'crd',
        'rbac',
        
        // API specifics
        'rest',
        'graphql',
        'auth',
        
        // UI specifics
        'components',
        'pages',
        'hooks',
        'store',
        
        // Infrastructure
        'docker',
        'k8s',
        'helm',
        'ci',
        
        // Documentation
        'docs',
        'examples',
        
        // Testing
        'test',
        'e2e',
        
        // Dependencies
        'deps',
        
        // General (when no specific scope)
        '*',
      ],
    ],
    
    // Case rules
    'type-case': [2, 'always', 'lower-case'],
    'scope-case': [2, 'always', 'lower-case'],
    'subject-case': [2, 'never', ['sentence-case', 'start-case', 'pascal-case', 'upper-case']],
    
    // Length rules
    'header-max-length': [2, 'always', 72],
    'subject-min-length': [2, 'always', 10],
    'body-max-line-length': [2, 'always', 100],
    
    // Subject rules
    'subject-empty': [2, 'never'],
    'subject-full-stop': [2, 'never', '.'],
    
    // Body rules
    'body-leading-blank': [2, 'always'],
    'body-min-length': [1, 'always', 20], // Warning for short bodies
    
    // Footer rules
    'footer-leading-blank': [2, 'always'],
    
    // Custom rules
    'signed-off-by': [0, 'always', 'Signed-off-by:'], // Disabled, using DCO
  },
  
  // Parser presets
  parserPreset: {
    parserOpts: {
      noteKeywords: ['BREAKING CHANGE', 'BREAKING-CHANGE'],
      issuePrefixes: ['#', 'GH-', 'JIRA-'],
    },
  },
  
  // Help URL
  helpUrl: 'https://www.conventionalcommits.org/en/v1.0.0/',
};
