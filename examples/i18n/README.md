# Internationalization Examples

This directory contains examples of how to implement and use internationalization (i18n) in the Gunj Operator.

## Directory Structure

```
i18n/
├── locales/              # Example translation files
│   ├── en/              # English (source)
│   ├── ja/              # Japanese
│   └── es/              # Spanish
├── react-component.tsx   # Example React component with i18n
├── go-handler.go        # Example Go handler with i18n
└── config.yaml          # Example platform config with i18n metadata
```

## Quick Start

1. **React Components**: See `react-component.tsx` for how to use translations in UI
2. **Go Backend**: See `go-handler.go` for server-side translations
3. **Translation Files**: Check `locales/` for translation file structure
4. **Configuration**: See `config.yaml` for i18n metadata in CRDs

## Key Concepts

- All user-facing strings must be externalized
- Use meaningful translation keys
- Support pluralization from the start
- Consider text expansion (German ~30% longer than English)
- Test with RTL languages (Arabic, Hebrew)

## Testing

```bash
# Test with different languages
npm run dev -- --lang=ja
npm run dev -- --lang=es

# Run i18n tests
npm run test:i18n
```