# Gunj Operator Internationalization (i18n) Plan

**Version**: 1.0  
**Date**: June 12, 2025  
**Phase**: 1.4.2 - CNCF Compliance Planning  
**Micro-task**: MT-1.4.2.5  
**Status**: Planning Document  
**Author**: Gunjan Patel (gunjanjp@gmail.com)  

---

## 📋 Executive Summary

This document outlines the comprehensive internationalization (i18n) strategy for the Gunj Operator project. As part of our commitment to CNCF compliance and global accessibility, we aim to make the operator accessible to users worldwide by supporting multiple languages and regional formats.

### 🎯 i18n Goals

1. **Multi-language Support**: Enable the operator to support at least 10 major languages
2. **Cultural Adaptation**: Respect regional formats for dates, numbers, and currencies
3. **RTL Support**: Full support for right-to-left languages
4. **Accessibility**: Ensure i18n doesn't compromise accessibility standards
5. **Developer Experience**: Make it easy for contributors to add new translations
6. **Performance**: Minimal impact on application performance

---

## 🌍 Supported Languages

### Phase 1 Languages (Launch)
1. **English (en)** - Primary language
2. **Spanish (es)** - 559M speakers
3. **Chinese Simplified (zh-CN)** - 1.1B speakers
4. **Japanese (ja)** - 125M speakers
5. **German (de)** - 100M speakers

### Phase 2 Languages (6 months post-launch)
6. **French (fr)** - 280M speakers
7. **Portuguese (pt)** - 258M speakers
8. **Hindi (hi)** - 602M speakers
9. **Arabic (ar)** - 422M speakers (RTL)
10. **Korean (ko)** - 81M speakers

### Phase 3 Languages (Community-driven)
- Russian (ru)
- Italian (it)
- Dutch (nl)
- Polish (pl)
- Turkish (tr)

---

## 🏗️ Technical Architecture

### Backend i18n Strategy

#### 1. Go Backend Implementation
```go
// pkg/i18n/i18n.go
package i18n

import (
    "github.com/nicksnyder/go-i18n/v2/i18n"
    "golang.org/x/text/language"
)

type Localizer struct {
    bundle       *i18n.Bundle
    defaultLang  language.Tag
    supportedLangs []language.Tag
}

// Message structure for consistent formatting
type Message struct {
    ID          string
    Description string
    Other       string
}

// Error messages with i18n support
var (
    ErrPlatformNotFound = Message{
        ID:          "error.platform.notFound",
        Description: "Platform resource not found",
        Other:       "Platform '{{.Name}}' not found in namespace '{{.Namespace}}'",
    }
    
    ErrInsufficientResources = Message{
        ID:          "error.resources.insufficient",
        Description: "Insufficient cluster resources",
        Other:       "Insufficient {{.ResourceType}} available. Required: {{.Required}}, Available: {{.Available}}",
    }
)
```

#### 2. Translation File Structure
```yaml
# locales/en/messages.yaml
error:
  platform:
    notFound: "Platform '{{.Name}}' not found in namespace '{{.Namespace}}'"
    invalidSpec: "Invalid platform specification: {{.Error}}"
  resources:
    insufficient: "Insufficient {{.ResourceType}} available. Required: {{.Required}}, Available: {{.Available}}"
    
status:
  phase:
    pending: "Pending"
    installing: "Installing"
    ready: "Ready"
    failed: "Failed"
    upgrading: "Upgrading"
    
component:
  prometheus:
    name: "Prometheus"
    description: "Time series database for metrics"
  grafana:
    name: "Grafana"
    description: "Visualization and dashboards"
```

### Frontend i18n Strategy

#### 1. React i18n Setup
```typescript
// ui/src/i18n/config.ts
import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import LanguageDetector from 'i18next-browser-languagedetector';

export const supportedLanguages = [
  { code: 'en', name: 'English', nativeName: 'English' },
  { code: 'es', name: 'Spanish', nativeName: 'Español' },
  { code: 'zh-CN', name: 'Chinese', nativeName: '中文' },
  { code: 'ja', name: 'Japanese', nativeName: '日本語' },
  { code: 'de', name: 'German', nativeName: 'Deutsch' },
];

i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    fallbackLng: 'en',
    debug: process.env.NODE_ENV === 'development',
    
    interpolation: {
      escapeValue: false,
    },
    
    detection: {
      order: ['querystring', 'cookie', 'localStorage', 'navigator'],
      caches: ['localStorage', 'cookie'],
    },
  });
```

#### 2. Component Translation Pattern
```typescript
// ui/src/components/PlatformCard.tsx
import { useTranslation } from 'react-i18next';

export const PlatformCard: React.FC<Props> = ({ platform }) => {
  const { t, i18n } = useTranslation();
  
  return (
    <Card>
      <CardHeader>
        <Typography variant="h6">
          {platform.metadata.name}
        </Typography>
        <Chip 
          label={t(`status.phase.${platform.status.phase.toLowerCase()}`)}
          color={getStatusColor(platform.status.phase)}
        />
      </CardHeader>
      <CardContent>
        <Typography variant="body2">
          {t('platform.lastUpdated', {
            time: formatRelativeTime(platform.status.lastUpdateTime, i18n.language)
          })}
        </Typography>
      </CardContent>
    </Card>
  );
};
```

### CLI i18n Support

```go
// cmd/cli/commands/root.go
package commands

import (
    "github.com/spf13/cobra"
    "github.com/gunjanjp/gunj-operator/pkg/i18n"
)

var rootCmd = &cobra.Command{
    Use:   "gunj",
    Short: i18n.T("cli.description.short"),
    Long:  i18n.T("cli.description.long"),
}

func init() {
    rootCmd.PersistentFlags().String("language", "en", i18n.T("cli.flag.language"))
    rootCmd.PersistentFlags().String("output", "table", i18n.T("cli.flag.output"))
}
```

---

## 📝 Content Guidelines

### 1. Translation Standards

#### Text Expansion Rules
- Allow 30% expansion for European languages
- Allow 50% expansion for German and Russian
- Consider 15% contraction for Asian languages

#### UI Text Guidelines
```yaml
# Good - Clear, concise, translatable
button:
  save: "Save"
  cancel: "Cancel"
  createPlatform: "Create Platform"
  
# Bad - Concatenated strings
message:
  # DON'T DO THIS
  error: "Error: " + errorType + " occurred"
  
  # DO THIS INSTEAD
  error: "An error of type {{.Type}} occurred"
```

### 2. Cultural Considerations

#### Date/Time Formatting
```typescript
// ui/src/utils/formatting.ts
export const formatDate = (date: Date, locale: string): string => {
  return new Intl.DateTimeFormat(locale, {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  }).format(date);
};

// Examples:
// en-US: June 12, 2025, 2:30 PM
// de-DE: 12. Juni 2025, 14:30
// ja-JP: 2025年6月12日 14:30
```

#### Number Formatting
```typescript
export const formatNumber = (num: number, locale: string): string => {
  return new Intl.NumberFormat(locale).format(num);
};

// Examples:
// en-US: 1,234,567.89
// de-DE: 1.234.567,89
// hi-IN: 12,34,567.89
```

### 3. RTL Support

```css
/* ui/src/styles/rtl.css */
[dir="rtl"] {
  /* Flip layout direction */
  .sidebar {
    right: 0;
    left: auto;
  }
  
  /* Flip margins and paddings */
  .platform-card {
    margin-left: 0;
    margin-right: 1rem;
  }
  
  /* Flip icons */
  .arrow-icon {
    transform: scaleX(-1);
  }
}
```

---

## 🔧 Implementation Process

### Phase 1: Infrastructure Setup (Week 1)

1. **Backend i18n Framework**
   - [ ] Install go-i18n/v2 package
   - [ ] Create i18n middleware
   - [ ] Set up translation file structure
   - [ ] Implement language detection

2. **Frontend i18n Framework**
   - [ ] Install react-i18next
   - [ ] Configure language detection
   - [ ] Set up translation loading
   - [ ] Create language switcher component

3. **Build Pipeline Integration**
   - [ ] Add translation validation
   - [ ] Create extraction scripts
   - [ ] Set up translation sync
   - [ ] Configure CI/CD checks

### Phase 2: Content Extraction (Week 2)

1. **String Externalization**
   - [ ] Extract hardcoded strings from Go code
   - [ ] Extract strings from React components
   - [ ] Extract strings from documentation
   - [ ] Create translation keys

2. **Translation File Generation**
   - [ ] Generate base English files
   - [ ] Create translation templates
   - [ ] Set up pluralization rules
   - [ ] Document context for translators

### Phase 3: Translation Process (Week 3-4)

1. **Translation Workflow**
   - [ ] Set up Crowdin/Weblate integration
   - [ ] Create translator guidelines
   - [ ] Implement review process
   - [ ] Set up automated testing

2. **Quality Assurance**
   - [ ] Linguistic review process
   - [ ] Technical validation
   - [ ] Context verification
   - [ ] UI/UX testing

---

## 🧪 Testing Strategy

### 1. Unit Tests
```go
func TestLocalizedErrors(t *testing.T) {
    tests := []struct {
        name     string
        lang     string
        errorKey string
        params   map[string]interface{}
        expected string
    }{
        {
            name:     "English platform not found",
            lang:     "en",
            errorKey: "error.platform.notFound",
            params:   map[string]interface{}{"Name": "prod", "Namespace": "default"},
            expected: "Platform 'prod' not found in namespace 'default'",
        },
        {
            name:     "Spanish platform not found",
            lang:     "es",
            errorKey: "error.platform.notFound",
            params:   map[string]interface{}{"Name": "prod", "Namespace": "default"},
            expected: "Plataforma 'prod' no encontrada en el espacio de nombres 'default'",
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            localizer := i18n.NewLocalizer(tt.lang)
            result := localizer.Localize(tt.errorKey, tt.params)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### 2. E2E Tests
```typescript
describe('Language Switching', () => {
  it('should switch UI language', () => {
    cy.visit('/');
    cy.get('[data-testid=language-selector]').click();
    cy.get('[data-value=ja]').click();
    
    // Verify Japanese text appears
    cy.contains('プラットフォーム一覧');
    cy.contains('新規作成');
    
    // Verify persistence
    cy.reload();
    cy.contains('プラットフォーム一覧');
  });
});
```

---

## 📊 Success Metrics

### Quantitative Metrics
- **Translation Coverage**: >95% of UI strings translated
- **Language Support**: 10 languages within first year
- **Performance Impact**: <50ms additional load time
- **Bundle Size**: <100KB per language pack
- **User Adoption**: 40% non-English usage

### Qualitative Metrics
- **Translation Quality**: Native speaker approval >90%
- **Cultural Appropriateness**: Zero cultural issues reported
- **Developer Experience**: <1 hour to add new language
- **User Satisfaction**: Positive feedback from global users

---

## 🔄 Maintenance Plan

### 1. Continuous Updates
- Weekly translation syncs
- Monthly quality reviews
- Quarterly language additions
- Annual comprehensive audit

### 2. Community Involvement
- Open translation platform
- Contributor recognition
- Language-specific channels
- Regional meetups

### 3. Automation
- Automated string extraction
- Translation memory
- Quality checks
- Performance monitoring

---

## 📋 Checklist

### Pre-implementation
- [x] Define supported languages
- [x] Choose i18n frameworks
- [x] Design file structure
- [x] Create coding standards
- [ ] Set up translation platform
- [ ] Recruit initial translators

### Implementation
- [ ] Backend i18n setup
- [ ] Frontend i18n setup
- [ ] CLI i18n support
- [ ] String extraction
- [ ] Translation process
- [ ] Testing framework

### Post-implementation
- [ ] Documentation translation
- [ ] Marketing materials
- [ ] Community outreach
- [ ] Feedback collection
- [ ] Continuous improvement

---

## 🔗 References

### Standards
- [Unicode CLDR](http://cldr.unicode.org/)
- [W3C Internationalization](https://www.w3.org/International/)
- [IETF Language Tags](https://tools.ietf.org/html/bcp47)

### Tools
- [go-i18n](https://github.com/nicksnyder/go-i18n)
- [react-i18next](https://react.i18next.com/)
- [Crowdin](https://crowdin.com/)
- [Weblate](https://weblate.org/)

### Best Practices
- [CNCF i18n Guidelines](https://github.com/cncf/foundation/blob/main/i18n.md)
- [Kubernetes i18n](https://kubernetes.io/docs/contribute/localization/)
- [Mozilla L10n](https://mozilla-l10n.github.io/documentation/)

---

**Document Status**: This internationalization plan is part of Phase 1.4.2 CNCF Compliance Planning. It will be refined during implementation based on community feedback and technical constraints.