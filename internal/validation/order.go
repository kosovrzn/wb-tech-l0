package validation

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/kosovrzn/wb-tech-l0/internal/domain"
)

type Validator struct {
	validate *validator.Validate
}

func New() *Validator {
	return &Validator{validate: validator.New()}
}

func (v *Validator) ValidateOrder(o *domain.Order) error {
	if o == nil {
		return fmt.Errorf("order is nil")
	}

	if err := v.validate.Struct(o); err != nil {
		if invalid, ok := err.(*validator.InvalidValidationError); ok {
			return invalid
		}
		if verrs, ok := err.(validator.ValidationErrors); ok {
			var b strings.Builder
			b.WriteString("order validation failed: ")
			for i, fe := range verrs {
				if i > 0 {
					b.WriteString("; ")
				}
				b.WriteString(fieldError(fe))
			}
			return errors.New(b.String())
		}
		return err
	}
	return nil
}

func fieldError(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s is required", fe.Namespace())
	case "alpha":
		return fmt.Sprintf("%s must contain alphabetic characters only", fe.Namespace())
	case "alphanumunicode":
		return fmt.Sprintf("%s must contain letters or numbers only", fe.Namespace())
	case "printascii":
		return fmt.Sprintf("%s must contain printable ASCII characters only", fe.Namespace())
	case "uppercase":
		return fmt.Sprintf("%s must be uppercase", fe.Namespace())
	case "numeric":
		return fmt.Sprintf("%s must contain digits only", fe.Namespace())
	case "len":
		return fmt.Sprintf("%s must be %s characters", fe.Namespace(), fe.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters", fe.Namespace(), fe.Param())
	case "gt":
		return fmt.Sprintf("%s must be greater than %s", fe.Namespace(), fe.Param())
	case "gte":
		return fmt.Sprintf("%s must be greater than or equal to %s", fe.Namespace(), fe.Param())
	case "e164":
		return fmt.Sprintf("%s must be a valid phone in E.164 format", fe.Namespace())
	case "email":
		return fmt.Sprintf("%s must be a valid email", fe.Namespace())
	case "min":
		return fmt.Sprintf("%s must have at least %s items", fe.Namespace(), fe.Param())
	default:
		return fmt.Sprintf("%s failed on %s validation", fe.Namespace(), fe.Tag())
	}
}
