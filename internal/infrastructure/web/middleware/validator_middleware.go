package middleware

import (
	"encoding/json"
	"net/http"

	"github.com/go-playground/validator/v10"
)

// Validate is a middleware that validates request payloads using go-playground/validator.
// Usage: http.HandlerFunc(Validate(w, r, &myStruct))
func Validate(payload interface{}) func(http.Handler) http.Handler {
	validate := validator.New()

	// Register custom validators if needed
	// validate.RegisterValidation("custom_rule", customValidator)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if err := validate.Struct(payload); err != nil {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusBadRequest)

				// Build validation error response
				var errors []FieldError
				for _, err := range err.(validator.ValidationErrors) {
					errors = append(errors, FieldError{
						Field:   err.Field(),
						Tag:     err.Tag(),
						Value:   err.Value(),
						Message: formatValidationError(err),
					})
				}

				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":        "validation failed",
					"field_errors": errors,
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// FieldError represents a single field validation error
type FieldError struct {
	Field   string      `json:"field"`
	Tag     string      `json:"tag"`
	Value   interface{} `json:"value,omitempty"`
	Message string      `json:"message"`
}

// formatValidationError creates a human-readable error message
func formatValidationError(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return err.Field() + " is required"
	case "min":
		return err.Field() + " must be at least " + err.Param() + " characters"
	case "max":
		return err.Field() + " must be at most " + err.Param() + " characters"
	case "email":
		return err.Field() + " must be a valid email address"
	case "oneof":
		return err.Field() + " must be one of: " + err.Param()
	case "uuid":
		return err.Field() + " must be a valid UUID"
	case "gte":
		return err.Field() + " must be greater than or equal to " + err.Param()
	case "lte":
		return err.Field() + " must be less than or equal to " + err.Param()
	default:
		return err.Field() + " failed validation on " + err.Tag()
	}
}
