package validator

import (
	"strings"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func Validate(s any) map[string]string {
	err := validate.Struct(s)
	if err == nil {
		return nil
	}

	errors := make(map[string]string)
	for _, err := range err.(validator.ValidationErrors) {
		field := strings.ToLower(err.Field())
		errors[field] = validationMessage(err)
	}
	return errors
}


func validationMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "this field is required"

	case "email":
		return "invalid email format"
    
	case "min":
		return "this field must be at least " + err.Param()
	
    case "max":
		return "this field must be at most " + err.Param()
		
	default:
		return "Invalid value"
	}
}