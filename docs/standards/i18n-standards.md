# Gunj Operator Internationalization (i18n) Standards v1.0

**Version**: 1.0  
**Last Updated**: June 12, 2025  
**Project**: Gunj Operator - Enterprise Observability Platform  
**Contact**: gunjanjp@gmail.com  
**Status**: Official i18n Guidelines  

---

## 📋 Executive Summary

This document defines comprehensive internationalization (i18n) and localization (l10n) standards for the Gunj Operator project. Our goal is to make the platform accessible to users worldwide by supporting multiple languages, regional formats, and cultural preferences.

### 🎯 Internationalization Goals

1. **Global Reach**: Support 15+ languages in the first year
2. **Complete Coverage**: 100% of UI strings externalized
3. **Cultural Sensitivity**: Adapt to regional preferences
4. **Seamless Experience**: Native feel for each locale
5. **Maintainability**: Efficient translation workflow

---

## 🏗️ Architecture Overview

### Technology Stack

#### Frontend (React)
- **Primary Framework**: react-i18next v13.x
- **Formatting Library**: FormatJS (react-intl) v6.x
- **Date Library**: date-fns v3.x with locale support
- **Build Tools**: i18next-scanner for extraction

#### Backend (Go)
- **Primary Library**: golang.org/x/text
- **Message Format**: ICU MessageFormat
- **Locale Detection**: Accept-Language header parsing
- **Resource Format**: JSON with namespace support

### Architecture Diagram

```
┌─────────────────────────────────────────────────┐
│                   Frontend (React)               │
├─────────────────────────────────────────────────┤
│  Components → useTranslation() → i18next        │
│                                    ↓             │
│              Language Detection    ↓             │
│              Browser/User Pref  → JSON bundles  │
└─────────────────────────────────────────────────┘
                         ↕ API ↕
┌─────────────────────────────────────────────────┐
│                   Backend (Go)                   │
├─────────────────────────────────────────────────┤
│  Handlers → i18n.T() → Message Catalog          │
│                           ↓                      │
│         Locale Negotiation → JSON/Go files      │
└─────────────────────────────────────────────────┘
```

---

## 🌍 Supported Languages

### Phase 1 (Launch)
1. **English (en)** - Default
2. **Spanish (es)**
3. **French (fr)**
4. **German (de)**
5. **Japanese (ja)**
6. **Simplified Chinese (zh-CN)**

### Phase 2 (6 months)
7. **Portuguese (pt-BR)**
8. **Italian (it)**
9. **Korean (ko)**
10. **Russian (ru)**
11. **Arabic (ar)** - RTL
12. **Hebrew (he)** - RTL

### Phase 3 (1 year)
13. **Hindi (hi)**
14. **Dutch (nl)**
15. **Polish (pl)**
16. **Turkish (tr)**
17. **Indonesian (id)**
18. **Vietnamese (vi)**

---

## 💻 Frontend Implementation

### React i18next Configuration

```typescript
// i18n/config.ts
import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';
import Backend from 'i18next-http-backend';
import { format as formatDate, formatDistance, formatRelative } from 'date-fns';
import { enUS, es, fr, de, ja, zhCN } from 'date-fns/locale';

const dateLocales = { en: enUS, es, fr, de, ja, 'zh-CN': zhCN };

i18n
  .use(Backend)
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    fallbackLng: 'en',
    debug: process.env.NODE_ENV === 'development',
    
    interpolation: {
      escapeValue: false, // React already escapes
      format: (value, format, lng) => {
        if (format === 'date') {
          return formatDate(value, 'PP', { locale: dateLocales[lng] });
        }
        if (format === 'relative') {
          return formatRelative(value, new Date(), { locale: dateLocales[lng] });
        }
        return value;
      },
    },
    
    backend: {
      loadPath: '/locales/{{lng}}/{{ns}}.json',
    },
    
    ns: ['common', 'platform', 'errors', 'validation'],
    defaultNS: 'common',
    
    detection: {
      order: ['localStorage', 'cookie', 'navigator', 'htmlTag'],
      caches: ['localStorage', 'cookie'],
    },
    
    react: {
      useSuspense: true,
      bindI18n: 'languageChanged loaded',
      bindI18nStore: 'added removed',
      transEmptyNodeValue: '',
      transSupportBasicHtmlNodes: true,
      transKeepBasicHtmlNodesFor: ['br', 'strong', 'i', 'p'],
    },
  });

export default i18n;
```

### Component Usage Patterns

```typescript
// Basic Translation
import { useTranslation } from 'react-i18next';

export const PlatformCard: React.FC<Props> = ({ platform }) => {
  const { t } = useTranslation('platform');
  
  return (
    <Card>
      <CardHeader>
        <Typography variant="h2">
          {t('card.title', { name: platform.name })}
        </Typography>
      </CardHeader>
      <CardContent>
        <StatusBadge 
          status={platform.status}
          label={t(`status.${platform.status}`)}
        />
      </CardContent>
    </Card>
  );
};

// Pluralization
const ComponentCount: React.FC<{ count: number }> = ({ count }) => {
  const { t } = useTranslation();
  
  return (
    <Typography>
      {t('component.count', { count })}
      {/* Output: "1 component" or "5 components" */}
    </Typography>
  );
};

// Formatted Values
const MetricsDisplay: React.FC<{ value: number; timestamp: Date }> = ({ value, timestamp }) => {
  const { t, i18n } = useTranslation();
  
  return (
    <div>
      <Typography>
        {t('metrics.value', { 
          value: new Intl.NumberFormat(i18n.language).format(value) 
        })}
      </Typography>
      <Typography variant="caption">
        {t('metrics.lastUpdated', { 
          time: formatDistance(timestamp, new Date(), { 
            locale: dateLocales[i18n.language],
            addSuffix: true 
          })
        })}
      </Typography>
    </div>
  );
};
```

### Translation File Structure

```json
// locales/en/platform.json
{
  "card": {
    "title": "Platform: {{name}}",
    "description": "Manage your observability platform"
  },
  "status": {
    "ready": "Ready",
    "installing": "Installing",
    "failed": "Failed",
    "upgrading": "Upgrading"
  },
  "component": {
    "count_one": "{{count}} component",
    "count_other": "{{count}} components"
  },
  "actions": {
    "create": "Create Platform",
    "edit": "Edit",
    "delete": "Delete",
    "view": "View Details"
  },
  "messages": {
    "createSuccess": "Platform {{name}} created successfully",
    "deleteConfirm": "Are you sure you want to delete {{name}}? This action cannot be undone."
  }
}

// locales/ja/platform.json
{
  "card": {
    "title": "プラットフォーム: {{name}}",
    "description": "可観測性プラットフォームを管理"
  },
  "status": {
    "ready": "準備完了",
    "installing": "インストール中",
    "failed": "失敗",
    "upgrading": "アップグレード中"
  },
  "component": {
    "count": "{{count}}個のコンポーネント"
  },
  "actions": {
    "create": "プラットフォームを作成",
    "edit": "編集",
    "delete": "削除",
    "view": "詳細を表示"
  }
}
```

---

## 🔧 Backend Implementation

### Go i18n Setup

```go
// pkg/i18n/i18n.go
package i18n

import (
    "embed"
    "encoding/json"
    "fmt"
    
    "golang.org/x/text/language"
    "golang.org/x/text/message"
    "golang.org/x/text/message/catalog"
)

//go:embed locales/*.json
var localesFS embed.FS

var (
    matcher  language.Matcher
    catalogs map[language.Tag]*catalog.Builder
)

func init() {
    // Initialize supported languages
    supported := []language.Tag{
        language.English,
        language.Spanish,
        language.French,
        language.German,
        language.Japanese,
        language.SimplifiedChinese,
    }
    
    matcher = language.NewMatcher(supported)
    catalogs = make(map[language.Tag]*catalog.Builder)
    
    // Load translations
    for _, lang := range supported {
        if err := loadTranslations(lang); err != nil {
            panic(fmt.Sprintf("failed to load translations for %s: %v", lang, err))
        }
    }
}

// T returns a translator for the given language tags
func T(langs ...string) *message.Printer {
    tags := make([]language.Tag, len(langs))
    for i, lang := range langs {
        tags[i] = language.Make(lang)
    }
    
    tag, _ := language.MatchStrings(matcher, langs...)
    return message.NewPrinter(tag)
}

// Middleware for extracting language preference
func LocaleMiddleware() gin.HandlerFunc {
    return func(c *gin.Context) {
        // Try Accept-Language header
        acceptLang := c.GetHeader("Accept-Language")
        
        // Try user preference from auth
        if user, exists := c.Get("user"); exists {
            if u, ok := user.(*User); ok && u.Language != "" {
                acceptLang = u.Language
            }
        }
        
        // Store printer in context
        printer := T(acceptLang)
        c.Set("i18n", printer)
        
        c.Next()
    }
}
```

### API Response Localization

```go
// internal/api/responses.go
package api

import (
    "github.com/gin-gonic/gin"
    "golang.org/x/text/message"
)

type LocalizedError struct {
    Code    string            `json:"code"`
    Message string            `json:"message"`
    Details map[string]string `json:"details,omitempty"`
}

func RespondWithError(c *gin.Context, code int, errKey string, args ...interface{}) {
    p := c.MustGet("i18n").(*message.Printer)
    
    // Translate error message
    msg := p.Sprintf(errKey, args...)
    
    c.JSON(code, gin.H{
        "error": LocalizedError{
            Code:    errKey,
            Message: msg,
            Details: extractDetails(c, errKey),
        },
    })
}

// Usage in handlers
func (h *Handler) CreatePlatform(c *gin.Context) {
    var req CreatePlatformRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        RespondWithError(c, 400, "error.validation.invalid_json")
        return
    }
    
    // Validation
    if len(req.Name) > 63 {
        RespondWithError(c, 400, "error.validation.name_too_long", 63)
        return
    }
    
    // Success response
    p := c.MustGet("i18n").(*message.Printer)
    c.JSON(201, gin.H{
        "message": p.Sprintf("platform.created", req.Name),
        "data": platform,
    })
}
```

---

## 🔄 RTL (Right-to-Left) Support

### CSS Architecture

```scss
// styles/base/_rtl.scss
// Logical properties for automatic RTL support
.card {
  margin-inline-start: var(--spacing-md);
  padding-inline: var(--spacing-lg);
  border-inline-start: 4px solid var(--primary-color);
}

// RTL-specific overrides
[dir="rtl"] {
  // Fix icons that shouldn't flip
  .no-flip {
    transform: scaleX(1) !important;
  }
  
  // Adjust specific components
  .timeline-item {
    &::before {
      inset-inline-end: -8px;
      inset-inline-start: auto;
    }
  }
}

// Direction-aware utilities
@mixin margin-start($value) {
  margin-inline-start: $value;
}

@mixin padding-end($value) {
  padding-inline-end: $value;
}

@mixin text-align-start {
  text-align: start;
}
```

### React RTL Detection

```typescript
// hooks/useRTL.ts
import { useTranslation } from 'react-i18next';
import { useEffect } from 'react';

const RTL_LANGUAGES = ['ar', 'he', 'fa', 'ur'];

export const useRTL = () => {
  const { i18n } = useTranslation();
  const isRTL = RTL_LANGUAGES.includes(i18n.language);
  
  useEffect(() => {
    // Update document direction
    document.documentElement.dir = isRTL ? 'rtl' : 'ltr';
    document.documentElement.lang = i18n.language;
    
    // Update Material-UI theme direction
    const theme = isRTL ? rtlTheme : ltrTheme;
    // Apply theme...
  }, [i18n.language, isRTL]);
  
  return { isRTL, direction: isRTL ? 'rtl' : 'ltr' };
};
```

---

## 📅 Date, Time, and Number Formatting

### Date/Time Formatting Standards

```typescript
// utils/formatters.ts
import { format, formatDistance, formatRelative, parseISO } from 'date-fns';
import { enUS, es, fr, de, ja, zhCN, ar, he } from 'date-fns/locale';

const locales = {
  en: enUS,
  es: es,
  fr: fr,
  de: de,
  ja: ja,
  'zh-CN': zhCN,
  ar: ar,
  he: he,
};

export const formatters = {
  // Short date: Jan 1, 2025
  shortDate: (date: Date | string, locale: string) => {
    const dateObj = typeof date === 'string' ? parseISO(date) : date;
    return format(dateObj, 'PP', { locale: locales[locale] || enUS });
  },
  
  // Long date: January 1, 2025 at 3:30 PM
  longDate: (date: Date | string, locale: string) => {
    const dateObj = typeof date === 'string' ? parseISO(date) : date;
    return format(dateObj, 'PPPp', { locale: locales[locale] || enUS });
  },
  
  // Relative time: 2 hours ago
  relative: (date: Date | string, locale: string) => {
    const dateObj = typeof date === 'string' ? parseISO(date) : date;
    return formatDistance(dateObj, new Date(), { 
      addSuffix: true,
      locale: locales[locale] || enUS 
    });
  },
  
  // Duration: 2h 30m
  duration: (seconds: number, locale: string) => {
    const hours = Math.floor(seconds / 3600);
    const minutes = Math.floor((seconds % 3600) / 60);
    const t = i18n.getFixedT(locale);
    
    if (hours > 0) {
      return t('time.duration.hoursMinutes', { hours, minutes });
    }
    return t('time.duration.minutes', { minutes });
  },
};
```

### Number Formatting

```typescript
// utils/numberFormatters.ts
export const numberFormatters = {
  // Standard number: 1,234.56
  decimal: (value: number, locale: string, decimals = 2) => {
    return new Intl.NumberFormat(locale, {
      minimumFractionDigits: decimals,
      maximumFractionDigits: decimals,
    }).format(value);
  },
  
  // Percentage: 98.5%
  percentage: (value: number, locale: string, decimals = 1) => {
    return new Intl.NumberFormat(locale, {
      style: 'percent',
      minimumFractionDigits: decimals,
      maximumFractionDigits: decimals,
    }).format(value / 100);
  },
  
  // Currency: $1,234.56
  currency: (value: number, locale: string, currency: string) => {
    return new Intl.NumberFormat(locale, {
      style: 'currency',
      currency: currency,
    }).format(value);
  },
  
  // File size: 1.2 GB
  fileSize: (bytes: number, locale: string) => {
    const units = ['B', 'KB', 'MB', 'GB', 'TB'];
    const index = Math.floor(Math.log(bytes) / Math.log(1024));
    const value = bytes / Math.pow(1024, index);
    
    return `${numberFormatters.decimal(value, locale, 1)} ${units[index]}`;
  },
  
  // Compact numbers: 1.2K, 3.4M
  compact: (value: number, locale: string) => {
    return new Intl.NumberFormat(locale, {
      notation: 'compact',
      compactDisplay: 'short',
    }).format(value);
  },
};
```

---

## 🔄 Translation Workflow

### 1. Development Phase

```bash
# Extract strings from source code
npm run i18n:extract

# This runs:
i18next-scanner --config i18next-scanner.config.js

# Validate all keys are translated
npm run i18n:validate

# Check for unused translations
npm run i18n:clean
```

### 2. String Extraction Configuration

```javascript
// i18next-scanner.config.js
module.exports = {
  input: [
    'src/**/*.{ts,tsx}',
    '!src/**/*.test.{ts,tsx}',
    '!**/node_modules/**',
  ],
  
  output: './public/locales',
  
  options: {
    debug: true,
    
    func: {
      list: ['t', 'i18next.t', 'i18n.t'],
      extensions: ['.ts', '.tsx'],
    },
    
    trans: {
      component: 'Trans',
      i18nKey: 'i18nKey',
      extensions: ['.tsx'],
      fallbackKey: (ns, value) => value,
    },
    
    lngs: ['en', 'es', 'fr', 'de', 'ja', 'zh-CN'],
    defaultLng: 'en',
    defaultNs: 'common',
    
    resource: {
      loadPath: '{{lng}}/{{ns}}.json',
      savePath: '{{lng}}/{{ns}}.json',
      jsonIndent: 2,
      lineEnding: '\n',
    },
    
    nsSeparator: ':',
    keySeparator: '.',
    pluralSeparator: '_',
    contextSeparator: '_',
    
    interpolation: {
      prefix: '{{',
      suffix: '}}',
    },
  },
};
```

### 3. Translation Management

```yaml
# .github/workflows/i18n.yml
name: i18n Management

on:
  push:
    paths:
      - 'src/**/*.tsx'
      - 'src/**/*.ts'
  
jobs:
  extract-strings:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      
      - name: Setup Node.js
        uses: actions/setup-node@v4
        with:
          node-version: '20'
      
      - name: Extract strings
        run: |
          npm ci
          npm run i18n:extract
      
      - name: Check for new strings
        id: check
        run: |
          if [ -n "$(git status --porcelain public/locales)" ]; then
            echo "new_strings=true" >> $GITHUB_OUTPUT
          fi
      
      - name: Create PR for new strings
        if: steps.check.outputs.new_strings == 'true'
        uses: peter-evans/create-pull-request@v5
        with:
          title: '[i18n] New strings to translate'
          body: |
            New translatable strings have been detected.
            Please review and translate for all supported languages.
          branch: i18n/new-strings
          commit-message: 'feat(i18n): add new translatable strings'
```

### 4. Translation Service Integration

```typescript
// scripts/sync-translations.ts
import { Translator } from '@crowdin/crowdin-api-client';

const CROWDIN_TOKEN = process.env.CROWDIN_TOKEN;
const PROJECT_ID = process.env.CROWDIN_PROJECT_ID;

async function syncTranslations() {
  const client = new Translator({
    token: CROWDIN_TOKEN,
  });
  
  // Upload source files
  await client.uploadTranslations({
    projectId: PROJECT_ID,
    files: {
      'en/common.json': './public/locales/en/common.json',
      'en/platform.json': './public/locales/en/platform.json',
    },
  });
  
  // Download completed translations
  const languages = ['es', 'fr', 'de', 'ja', 'zh-CN'];
  for (const lang of languages) {
    await client.downloadTranslations({
      projectId: PROJECT_ID,
      language: lang,
      outputDir: `./public/locales/${lang}`,
    });
  }
}
```

---

## 🌏 Cultural Adaptation

### Regional Preferences

```typescript
// config/regional.ts
interface RegionalConfig {
  dateFormat: string;
  timeFormat: '12h' | '24h';
  firstDayOfWeek: 0 | 1 | 6; // Sunday, Monday, Saturday
  numberFormat: {
    decimal: string;
    thousands: string;
  };
  currency: string;
  timezone: string;
}

export const regionalConfigs: Record<string, RegionalConfig> = {
  'en-US': {
    dateFormat: 'MM/dd/yyyy',
    timeFormat: '12h',
    firstDayOfWeek: 0,
    numberFormat: { decimal: '.', thousands: ',' },
    currency: 'USD',
    timezone: 'America/New_York',
  },
  'en-GB': {
    dateFormat: 'dd/MM/yyyy',
    timeFormat: '24h',
    firstDayOfWeek: 1,
    numberFormat: { decimal: '.', thousands: ',' },
    currency: 'GBP',
    timezone: 'Europe/London',
  },
  'de-DE': {
    dateFormat: 'dd.MM.yyyy',
    timeFormat: '24h',
    firstDayOfWeek: 1,
    numberFormat: { decimal: ',', thousands: '.' },
    currency: 'EUR',
    timezone: 'Europe/Berlin',
  },
  'ja-JP': {
    dateFormat: 'yyyy/MM/dd',
    timeFormat: '24h',
    firstDayOfWeek: 0,
    numberFormat: { decimal: '.', thousands: ',' },
    currency: 'JPY',
    timezone: 'Asia/Tokyo',
  },
};
```

### Cultural UI Adaptations

```typescript
// components/CulturalAdaptations.tsx
const PlatformStatusIcon: React.FC<{ status: string }> = ({ status }) => {
  const { i18n } = useTranslation();
  
  // Avoid red/green for certain cultures
  const useColorBlindFriendly = ['ja', 'ko'].includes(i18n.language);
  
  const getStatusColor = () => {
    if (useColorBlindFriendly) {
      return {
        ready: '#0066CC',     // Blue
        failed: '#CC6600',    // Orange
        installing: '#6633CC', // Purple
      }[status];
    }
    
    return {
      ready: '#4CAF50',      // Green
      failed: '#F44336',     // Red
      installing: '#2196F3', // Blue
    }[status];
  };
  
  return <StatusIcon color={getStatusColor()} />;
};
```

---

## 🧪 Testing Strategy

### Unit Tests for i18n

```typescript
// __tests__/i18n.test.tsx
import { render, screen } from '@testing-library/react';
import { I18nextProvider } from 'react-i18next';
import i18n from '../i18n/config';

describe('i18n', () => {
  beforeEach(() => {
    i18n.changeLanguage('en');
  });
  
  test('renders in different languages', async () => {
    const Component = () => {
      const { t } = useTranslation();
      return <div>{t('welcome')}</div>;
    };
    
    const { rerender } = render(
      <I18nextProvider i18n={i18n}>
        <Component />
      </I18nextProvider>
    );
    
    expect(screen.getByText('Welcome')).toBeInTheDocument();
    
    // Change language
    await i18n.changeLanguage('ja');
    rerender(
      <I18nextProvider i18n={i18n}>
        <Component />
      </I18nextProvider>
    );
    
    expect(screen.getByText('ようこそ')).toBeInTheDocument();
  });
  
  test('handles pluralization', () => {
    const Component = ({ count }: { count: number }) => {
      const { t } = useTranslation();
      return <div>{t('item_count', { count })}</div>;
    };
    
    const { rerender } = render(
      <I18nextProvider i18n={i18n}>
        <Component count={1} />
      </I18nextProvider>
    );
    
    expect(screen.getByText('1 item')).toBeInTheDocument();
    
    rerender(
      <I18nextProvider i18n={i18n}>
        <Component count={5} />
      </I18nextProvider>
    );
    
    expect(screen.getByText('5 items')).toBeInTheDocument();
  });
});
```

### E2E Tests for Localization

```typescript
// e2e/i18n.spec.ts
import { test, expect } from '@playwright/test';

test.describe('Internationalization', () => {
  test('language switcher works', async ({ page }) => {
    await page.goto('/');
    
    // Check default language
    await expect(page.locator('h1')).toContainText('Observability Platforms');
    
    // Switch to Japanese
    await page.click('[data-testid="language-switcher"]');
    await page.click('[data-value="ja"]');
    
    // Verify language changed
    await expect(page.locator('h1')).toContainText('可観測性プラットフォーム');
    
    // Verify persistence
    await page.reload();
    await expect(page.locator('h1')).toContainText('可観測性プラットフォーム');
  });
  
  test('RTL layout for Arabic', async ({ page }) => {
    await page.goto('/?lng=ar');
    
    // Check document direction
    const html = page.locator('html');
    await expect(html).toHaveAttribute('dir', 'rtl');
    await expect(html).toHaveAttribute('lang', 'ar');
    
    // Check layout direction
    const header = page.locator('header');
    const styles = await header.evaluate((el) => 
      window.getComputedStyle(el)
    );
    expect(styles.direction).toBe('rtl');
  });
});
```

### Visual Regression Tests

```typescript
// visual-tests/i18n-visual.spec.ts
import { test } from '@playwright/test';
import { argosScreenshot } from '@argos-ci/playwright';

const languages = ['en', 'ja', 'ar', 'de'];
const viewports = [
  { width: 1920, height: 1080, name: 'desktop' },
  { width: 768, height: 1024, name: 'tablet' },
  { width: 375, height: 667, name: 'mobile' },
];

for (const lang of languages) {
  for (const viewport of viewports) {
    test(`Platform list - ${lang} - ${viewport.name}`, async ({ page }) => {
      await page.setViewportSize(viewport);
      await page.goto(`/?lng=${lang}`);
      await page.waitForLoadState('networkidle');
      
      await argosScreenshot(page, `platform-list-${lang}-${viewport.name}`);
    });
  }
}
```

---

## 📊 Performance Optimization

### Lazy Loading Translations

```typescript
// i18n/lazy-config.ts
import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import ChainedBackend from 'i18next-chained-backend';
import LocalStorageBackend from 'i18next-localstorage-backend';
import HttpBackend from 'i18next-http-backend';

i18n
  .use(ChainedBackend)
  .use(initReactI18next)
  .init({
    backend: {
      backends: [
        LocalStorageBackend,  // Primary cache
        HttpBackend,          // Fallback
      ],
      backendOptions: [{
        expirationTime: 7 * 24 * 60 * 60 * 1000, // 7 days
        store: window.localStorage,
      }, {
        loadPath: '/locales/{{lng}}/{{ns}}.json',
        crossDomain: true,
        withCredentials: false,
      }],
    },
    
    // Load only needed namespaces
    ns: ['common'],
    defaultNS: 'common',
    
    // Lazy load namespaces
    partialBundledLanguages: true,
    
    react: {
      useSuspense: true,
      bindI18n: 'languageChanged loaded',
    },
  });
```

### Bundle Size Optimization

```javascript
// webpack.config.js
module.exports = {
  resolve: {
    alias: {
      // Use smaller date-fns locales
      'date-fns/locale': 'date-fns/locale/index.js',
    },
  },
  
  plugins: [
    // Extract translations to separate chunks
    new webpack.optimize.SplitChunksPlugin({
      cacheGroups: {
        translations: {
          test: /[\\/]locales[\\/]/,
          name: (module, chunks, cacheGroupKey) => {
            const match = module.resource.match(/locales[\\/](\w+)[\\/]/);
            return `locale-${match[1]}`;
          },
          chunks: 'all',
        },
      },
    }),
  ],
};
```

---

## 🚀 Implementation Roadmap

### Phase 1: Foundation (Weeks 1-2)
- [x] Technology selection
- [ ] i18n architecture setup
- [ ] Build pipeline configuration
- [ ] Basic English externalization

### Phase 2: Core Languages (Weeks 3-4)
- [ ] Spanish translation
- [ ] Japanese translation
- [ ] Simplified Chinese translation
- [ ] RTL support (Arabic prep)

### Phase 3: Extended Support (Weeks 5-6)
- [ ] Additional European languages
- [ ] Arabic and Hebrew (RTL)
- [ ] Cultural adaptations
- [ ] Regional formats

### Phase 4: Automation (Week 7)
- [ ] CI/CD integration
- [ ] Translation service setup
- [ ] Automated testing
- [ ] Performance optimization

### Phase 5: Polish (Week 8)
- [ ] Visual QA
- [ ] Documentation
- [ ] Team training
- [ ] Launch preparation

---

## 📋 Checklist for Developers

### When Adding New Features

- [ ] All user-facing strings use `t()` function
- [ ] Pluralization handled correctly
- [ ] Context provided for translators
- [ ] No hardcoded dates/numbers/currency
- [ ] RTL layout considered
- [ ] Accessibility maintained
- [ ] Screenshots updated for docs

### Code Review Checklist

- [ ] No English strings in code
- [ ] Translation keys are descriptive
- [ ] Proper namespace used
- [ ] Variables in translations are meaningful
- [ ] No concatenated translations
- [ ] Loading states have translations

---

## 📚 Resources

### Documentation
- [react-i18next Documentation](https://react.i18next.com/)
- [FormatJS Guide](https://formatjs.io/docs/getting-started/installation)
- [CLDR Data](http://cldr.unicode.org/)
- [ICU Message Format](https://unicode-org.github.io/icu/userguide/format_parse/messages/)

### Tools
- [i18next Scanner](https://github.com/i18next/i18next-scanner)
- [Crowdin](https://crowdin.com/)
- [Pontoon](https://pontoon.mozilla.org/)
- [BabelEdit](https://www.codeandweb.com/babeledit)

### Best Practices
- [Mozilla L10n Guide](https://mozilla-l10n.github.io/documentation/)
- [Google i18n Guide](https://developers.google.com/international/)
- [W3C i18n Guidelines](https://www.w3.org/International/)

---

**This document ensures the Gunj Operator provides a native experience for users worldwide.**

*For questions about internationalization, contact gunjanjp@gmail.com or join our i18n channel.*