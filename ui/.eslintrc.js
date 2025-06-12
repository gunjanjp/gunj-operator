// Gunj Operator - ESLint Configuration
// Version: v2.0
// Purpose: TypeScript and React linting for the UI

module.exports = {
  root: true,
  
  // Parser for TypeScript
  parser: '@typescript-eslint/parser',
  
  // Parser options
  parserOptions: {
    ecmaVersion: 2022,
    sourceType: 'module',
    ecmaFeatures: {
      jsx: true,
    },
    project: './tsconfig.json',
    tsconfigRootDir: __dirname,
  },
  
  // Environment
  env: {
    browser: true,
    es2022: true,
    node: true,
    jest: true,
  },
  
  // Plugins
  plugins: [
    '@typescript-eslint',
    'react',
    'react-hooks',
    'jsx-a11y',
    'import',
    'prettier',
    'jest',
    'testing-library',
    'security',
    'promise',
    'unicorn',
  ],
  
  // Extends
  extends: [
    'eslint:recommended',
    'plugin:@typescript-eslint/recommended',
    'plugin:@typescript-eslint/recommended-requiring-type-checking',
    'plugin:react/recommended',
    'plugin:react-hooks/recommended',
    'plugin:jsx-a11y/recommended',
    'plugin:import/errors',
    'plugin:import/warnings',
    'plugin:import/typescript',
    'plugin:jest/recommended',
    'plugin:testing-library/react',
    'plugin:security/recommended',
    'plugin:promise/recommended',
    'plugin:unicorn/recommended',
    'prettier', // Must be last to override other configs
  ],
  
  // Settings
  settings: {
    react: {
      version: 'detect',
    },
    'import/resolver': {
      typescript: {
        alwaysTryTypes: true,
        project: './tsconfig.json',
      },
    },
  },
  
  // Rules
  rules: {
    // TypeScript specific
    '@typescript-eslint/explicit-function-return-type': 'off',
    '@typescript-eslint/explicit-module-boundary-types': 'off',
    '@typescript-eslint/no-explicit-any': 'error',
    '@typescript-eslint/no-unused-vars': ['error', { 
      argsIgnorePattern: '^_',
      varsIgnorePattern: '^_',
    }],
    '@typescript-eslint/consistent-type-imports': ['error', {
      prefer: 'type-imports',
    }],
    '@typescript-eslint/naming-convention': [
      'error',
      {
        selector: 'interface',
        format: ['PascalCase'],
        prefix: ['I'],
      },
      {
        selector: 'typeAlias',
        format: ['PascalCase'],
      },
      {
        selector: 'enum',
        format: ['PascalCase'],
      },
      {
        selector: 'enumMember',
        format: ['UPPER_CASE'],
      },
    ],
    
    // React specific
    'react/react-in-jsx-scope': 'off', // Not needed in React 18
    'react/prop-types': 'off', // We use TypeScript
    'react/display-name': 'error',
    'react/jsx-curly-brace-presence': ['error', { 
      props: 'never', 
      children: 'never' 
    }],
    'react/jsx-boolean-value': ['error', 'never'],
    'react/jsx-no-useless-fragment': 'error',
    'react/self-closing-comp': 'error',
    'react/jsx-sort-props': ['error', {
      callbacksLast: true,
      shorthandFirst: true,
      reservedFirst: true,
    }],
    
    // React Hooks
    'react-hooks/rules-of-hooks': 'error',
    'react-hooks/exhaustive-deps': 'error',
    
    // Import
    'import/order': ['error', {
      groups: [
        'builtin',
        'external',
        'internal',
        'parent',
        'sibling',
        'index',
        'object',
        'type',
      ],
      pathGroups: [
        {
          pattern: 'react',
          group: 'builtin',
          position: 'before',
        },
        {
          pattern: '@/**',
          group: 'internal',
          position: 'before',
        },
      ],
      pathGroupsExcludedImportTypes: ['react'],
      'newlines-between': 'always',
      alphabetize: {
        order: 'asc',
        caseInsensitive: true,
      },
    }],
    'import/no-duplicates': 'error',
    'import/no-cycle': 'error',
    'import/no-self-import': 'error',
    
    // General
    'no-console': ['warn', { allow: ['warn', 'error'] }],
    'no-debugger': 'error',
    'no-alert': 'warn',
    'prefer-const': 'error',
    'no-var': 'error',
    'object-shorthand': 'error',
    'prefer-template': 'error',
    'prefer-destructuring': ['error', {
      array: true,
      object: true,
    }],
    'no-nested-ternary': 'error',
    'max-lines': ['error', {
      max: 300,
      skipBlankLines: true,
      skipComments: true,
    }],
    'max-lines-per-function': ['error', {
      max: 80,
      skipBlankLines: true,
      skipComments: true,
    }],
    'complexity': ['error', 15],
    
    // Unicorn overrides
    'unicorn/filename-case': ['error', {
      cases: {
        kebabCase: true,
        pascalCase: true,
      },
    }],
    'unicorn/prevent-abbreviations': 'off',
    'unicorn/no-null': 'off',
    'unicorn/no-array-reduce': 'off',
    
    // Security
    'security/detect-object-injection': 'off', // Too many false positives
    
    // Accessibility
    'jsx-a11y/anchor-is-valid': ['error', {
      components: ['Link'],
      specialLink: ['to'],
    }],
  },
  
  // Overrides for specific files
  overrides: [
    // Test files
    {
      files: ['**/*.test.ts', '**/*.test.tsx', '**/*.spec.ts', '**/*.spec.tsx'],
      env: {
        jest: true,
      },
      rules: {
        '@typescript-eslint/no-explicit-any': 'off',
        'max-lines-per-function': 'off',
        'max-lines': 'off',
      },
    },
    
    // Configuration files
    {
      files: ['*.js', '*.config.js'],
      env: {
        node: true,
      },
      rules: {
        '@typescript-eslint/no-var-requires': 'off',
        'unicorn/prefer-module': 'off',
      },
    },
    
    // Storybook files
    {
      files: ['*.stories.tsx', '*.stories.ts'],
      rules: {
        'import/no-default-export': 'off',
      },
    },
  ],
};
