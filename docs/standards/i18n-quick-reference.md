# i18n Quick Reference Card

## React Components

```typescript
// Basic usage
import { useTranslation } from 'react-i18next';

const Component = () => {
  const { t, i18n } = useTranslation();
  
  return (
    <div>
      {/* Simple translation */}
      <h1>{t('welcome')}</h1>
      
      {/* With variables */}
      <p>{t('greeting', { name: userName })}</p>
      
      {/* Pluralization */}
      <span>{t('items', { count: 5 })}</span>
      
      {/* Change language */}
      <button onClick={() => i18n.changeLanguage('ja')}>
        日本語
      </button>
    </div>
  );
};
```

## Translation Files

```json
// public/locales/en/common.json
{
  "welcome": "Welcome",
  "greeting": "Hello, {{name}}!",
  "items_one": "{{count}} item",
  "items_other": "{{count}} items"
}
```

## Go Backend

```go
// Get translator
p := c.MustGet("i18n").(*message.Printer)

// Use in response
msg := p.Sprintf("platform.created", platformName)
```

## Common Commands

```bash
# Extract new strings
npm run i18n:extract

# Check translation status
npm run i18n:status

# Validate translations
npm run i18n:validate

# Test specific language
npm run dev -- --lang=ja
```

## Checklist
- [ ] All strings use t() function
- [ ] Pluralization handled correctly
- [ ] No hardcoded dates/numbers
- [ ] RTL layout considered
- [ ] Variables are meaningful