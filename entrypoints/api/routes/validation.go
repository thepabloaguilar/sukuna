package routes

import (
	"github.com/go-playground/validator/v10"
)

var structValidator = validator.New()

type errorResponse struct {
	FailedField string
	Tag         string
	Value       string
}

func validateStruct(s interface{}) []errorResponse {
	errors := make([]errorResponse, 0)

	err := structValidator.Struct(s)
	if err != nil {
		for _, err := range err.(validator.ValidationErrors) {
			errorResponse := errorResponse{
				FailedField: err.StructNamespace(),
				Tag:         err.Tag(),
				Value:       err.Param(),
			}
			errors = append(errors, errorResponse)
		}
	}

	return errors
}
