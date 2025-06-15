// Example Go Handler with i18n
// This shows best practices for using internationalization in Gunj Operator API

package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gunjanjp/gunj-operator/api/v1beta1"
	"github.com/gunjanjp/gunj-operator/pkg/i18n"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// PlatformHandler handles platform-related API requests
type PlatformHandler struct {
	client client.Client
}

// LocalizedResponse represents a standard API response with i18n support
type LocalizedResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message,omitempty"`
	Data    interface{}            `json:"data,omitempty"`
	Error   *LocalizedError        `json:"error,omitempty"`
	Meta    map[string]interface{} `json:"meta,omitempty"`
}

// LocalizedError represents an error with localized message
type LocalizedError struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Details map[string]string `json:"details,omitempty"`
}

// CreatePlatformRequest represents the request body for creating a platform
type CreatePlatformRequest struct {
	Name      string                            `json:"name" binding:"required,min=1,max=63,alphanum"`
	Namespace string                            `json:"namespace" binding:"required,min=1,max=63"`
	Spec      v1beta1.ObservabilityPlatformSpec `json:"spec" binding:"required"`
}

// i18n message keys
const (
	// Success messages
	MsgPlatformCreated = "platform.messages.created"
	MsgPlatformUpdated = "platform.messages.updated"
	MsgPlatformDeleted = "platform.messages.deleted"

	// Error messages
	ErrInvalidJSON        = "error.validation.invalid_json"
	ErrPlatformNotFound   = "error.platform.not_found"
	ErrPlatformExists     = "error.platform.already_exists"
	ErrInternalServer     = "error.internal_server"
	ErrUnauthorized       = "error.unauthorized"
	ErrForbidden          = "error.forbidden"
	ErrValidationFailed   = "error.validation.failed"
	ErrNameTooLong        = "error.validation.name_too_long"
	ErrInvalidNameFormat  = "error.validation.invalid_name_format"
	ErrResourceQuotaLimit = "error.resource.quota_exceeded"
)

// LocaleMiddleware extracts the preferred language from the request
func LocaleMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Priority order for language detection:
		// 1. Query parameter (?lang=ja)
		// 2. Accept-Language header
		// 3. User preference (from auth token)
		// 4. Default to English

		var preferredLang string

		// Check query parameter
		if lang := c.Query("lang"); lang != "" {
			preferredLang = lang
		} else if acceptLang := c.GetHeader("Accept-Language"); acceptLang != "" {
			// Parse Accept-Language header
			tags, _, err := language.ParseAcceptLanguage(acceptLang)
			if err == nil && len(tags) > 0 {
				preferredLang = tags[0].String()
			}
		}

		// Check user preference from auth context
		if user, exists := c.Get("user"); exists {
			if u, ok := user.(map[string]interface{}); ok {
				if userLang, ok := u["language"].(string); ok && userLang != "" {
					preferredLang = userLang
				}
			}
		}

		// Create message printer for the preferred language
		printer := i18n.T(preferredLang)
		c.Set("i18n", printer)
		c.Set("locale", preferredLang)

		c.Next()
	}
}

// CreatePlatform handles POST /api/v1/platforms
func (h *PlatformHandler) CreatePlatform(c *gin.Context) {
	printer := c.MustGet("i18n").(*message.Printer)

	// Parse request body
	var req CreatePlatformRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, http.StatusBadRequest, printer, ErrInvalidJSON)
		return
	}

	// Additional validation
	if len(req.Name) > 63 {
		respondWithError(c, http.StatusBadRequest, printer, ErrNameTooLong,
			map[string]string{"max": "63", "actual": fmt.Sprintf("%d", len(req.Name))})
		return
	}

	// Create platform resource
	platform := &v1beta1.ObservabilityPlatform{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: req.Namespace,
		},
		Spec: req.Spec,
	}

	// Set creation timestamp for i18n formatting
	platform.CreationTimestamp = metav1.NewTime(time.Now())

	// Attempt to create the platform
	ctx := c.Request.Context()
	if err := h.client.Create(ctx, platform); err != nil {
		if errors.IsAlreadyExists(err) {
			respondWithError(c, http.StatusConflict, printer, ErrPlatformExists,
				map[string]string{"name": req.Name, "namespace": req.Namespace})
			return
		}

		if errors.IsForbidden(err) {
			respondWithError(c, http.StatusForbidden, printer, ErrForbidden)
			return
		}

		// Check for quota exceeded
		if errors.IsResourceExpired(err) || errors.IsTooManyRequests(err) {
			respondWithError(c, http.StatusTooManyRequests, printer, ErrResourceQuotaLimit)
			return
		}

		// Generic internal error
		respondWithError(c, http.StatusInternalServerError, printer, ErrInternalServer)
		return
	}

	// Success response with localized message
	msg := printer.Sprintf(MsgPlatformCreated, req.Name)
	c.JSON(http.StatusCreated, LocalizedResponse{
		Success: true,
		Message: msg,
		Data:    platform,
		Meta: map[string]interface{}{
			"created_at": formatTimeForLocale(platform.CreationTimestamp.Time, c.GetString("locale")),
			"api_version": platform.APIVersion,
			"kind":       platform.Kind,
		},
	})
}

// GetPlatform handles GET /api/v1/platforms/:name
func (h *PlatformHandler) GetPlatform(c *gin.Context) {
	printer := c.MustGet("i18n").(*message.Printer)
	
	name := c.Param("name")
	namespace := c.Query("namespace")
	if namespace == "" {
		namespace = "default"
	}

	// Fetch platform
	platform := &v1beta1.ObservabilityPlatform{}
	key := client.ObjectKey{Name: name, Namespace: namespace}
	
	ctx := c.Request.Context()
	if err := h.client.Get(ctx, key, platform); err != nil {
		if errors.IsNotFound(err) {
			respondWithError(c, http.StatusNotFound, printer, ErrPlatformNotFound,
				map[string]string{"name": name, "namespace": namespace})
			return
		}
		
		respondWithError(c, http.StatusInternalServerError, printer, ErrInternalServer)
		return
	}

	// Enrich response with localized status
	localizedStatus := printer.Sprintf("platform.status.%s", platform.Status.Phase)
	
	c.JSON(http.StatusOK, LocalizedResponse{
		Success: true,
		Data: map[string]interface{}{
			"platform": platform,
			"status_text": localizedStatus,
		},
		Meta: map[string]interface{}{
			"last_updated": formatTimeForLocale(platform.Status.LastUpdateTime.Time, c.GetString("locale")),
			"age": formatDurationForLocale(time.Since(platform.CreationTimestamp.Time), printer),
		},
	})
}

// Helper function to respond with localized error
func respondWithError(c *gin.Context, code int, printer *message.Printer, msgKey string, details ...map[string]string) {
	msg := printer.Sprintf(msgKey)
	
	var detailMap map[string]string
	if len(details) > 0 {
		detailMap = details[0]
	}
	
	c.JSON(code, LocalizedResponse{
		Success: false,
		Error: &LocalizedError{
			Code:    msgKey,
			Message: msg,
			Details: detailMap,
		},
	})
}

// Helper function to format time based on locale
func formatTimeForLocale(t time.Time, locale string) string {
	// Map locale to time format
	formats := map[string]string{
		"en":    "Jan 2, 2006 3:04 PM",
		"en-US": "1/2/2006 3:04 PM",
		"en-GB": "02/01/2006 15:04",
		"ja":    "2006年1月2日 15:04",
		"es":    "2/1/2006 15:04",
		"de":    "2.1.2006 15:04",
		"fr":    "2/1/2006 15:04",
		"zh-CN": "2006年1月2日 15:04",
	}
	
	format, ok := formats[locale]
	if !ok {
		// Default to RFC3339
		return t.Format(time.RFC3339)
	}
	
	return t.Format(format)
}

// Helper function to format duration with i18n
func formatDurationForLocale(d time.Duration, printer *message.Printer) string {
	if d < time.Minute {
		seconds := int(d.Seconds())
		return printer.Sprintf("time.duration.seconds", seconds)
	} else if d < time.Hour {
		minutes := int(d.Minutes())
		return printer.Sprintf("time.duration.minutes", minutes)
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		return printer.Sprintf("time.duration.hours", hours)
	} else {
		days := int(d.Hours() / 24)
		return printer.Sprintf("time.duration.days", days)
	}
}

// Example of batch operation with localized progress messages
func (h *PlatformHandler) DeletePlatforms(c *gin.Context) {
	printer := c.MustGet("i18n").(*message.Printer)
	
	var req struct {
		Platforms []string `json:"platforms" binding:"required,min=1"`
		Namespace string   `json:"namespace" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		respondWithError(c, http.StatusBadRequest, printer, ErrInvalidJSON)
		return
	}
	
	// Track results
	results := make([]map[string]interface{}, 0, len(req.Platforms))
	successCount := 0
	
	for _, platformName := range req.Platforms {
		platform := &v1beta1.ObservabilityPlatform{}
		key := client.ObjectKey{Name: platformName, Namespace: req.Namespace}
		
		result := map[string]interface{}{
			"name": platformName,
		}
		
		ctx := c.Request.Context()
		if err := h.client.Get(ctx, key, platform); err != nil {
			if errors.IsNotFound(err) {
				result["success"] = false
				result["error"] = printer.Sprintf(ErrPlatformNotFound)
			} else {
				result["success"] = false
				result["error"] = printer.Sprintf(ErrInternalServer)
			}
		} else if err := h.client.Delete(ctx, platform); err != nil {
			result["success"] = false
			result["error"] = printer.Sprintf(ErrInternalServer)
		} else {
			result["success"] = true
			result["message"] = printer.Sprintf(MsgPlatformDeleted, platformName)
			successCount++
		}
		
		results = append(results, result)
	}
	
	// Summary message
	summaryMsg := printer.Sprintf("platform.batch.deleted_summary", 
		map[string]interface{}{
			"success": successCount,
			"total":   len(req.Platforms),
		})
	
	c.JSON(http.StatusOK, LocalizedResponse{
		Success: successCount == len(req.Platforms),
		Message: summaryMsg,
		Data: map[string]interface{}{
			"results": results,
		},
		Meta: map[string]interface{}{
			"total":   len(req.Platforms),
			"success": successCount,
			"failed":  len(req.Platforms) - successCount,
		},
	})
}