package validation

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator"
	"github.com/meghashyamc/wheresthat/logger"
)

type Validator struct {
	validator                *validator.Validate
	logger                   logger.Logger
	tagValidationDetailsOnce sync.Once
	tagValidationDetailsMap  map[string]tagValidationDetails
}

type tagValidationDetails struct {
	validatorFunc validator.Func
	err           error
}

func New(logger logger.Logger) (*Validator, error) {
	validator := &Validator{validator: validator.New(), logger: logger}
	validator.validator.RegisterTagNameFunc(useJSONFieldNames)
	if err := validator.registerCustomValidatorsForTags(); err != nil {
		return nil, err
	}

	return validator, nil
}

func (v *Validator) Validate(i any) error {

	if err := v.validator.Struct(i); err != nil {
		v.logger.Warn("validation failed", "err", err.Error())
		var validationErrs validator.ValidationErrors
		if errors.As(err, &validationErrs) && len(validationErrs) > 0 {

			tagValidationDetails, ok := v.getTagValidationDetails()[validationErrs[0].Tag()]
			if ok {
				return tagValidationDetails.err
			}

			switch validationErrs[0].Tag() {
			case "required":
				return fmt.Errorf("missing required field '%s'", validationErrs[0].Field())

			case "min", "max":
				return fmt.Errorf("value or length of field '%s' is not in the expected range", validationErrs[0].Field())

			}
		}
		return err
	}
	return nil
}
func (v *Validator) getTagValidationDetails() map[string]tagValidationDetails {
	v.tagValidationDetailsOnce.Do(func() {
		v.tagValidationDetailsMap = map[string]tagValidationDetails{
			"valid_path":  {validatorFunc: v.isValidPath, err: errors.New("invalid path")},
			"valid_query": {validatorFunc: v.isValidQuery, err: errors.New("invalid query")},
		}
	})
	return v.tagValidationDetailsMap
}

func (v *Validator) registerCustomValidatorsForTags() error {

	tagValidationDetailsMap := v.getTagValidationDetails()

	for tag, tagValidationDetails := range tagValidationDetailsMap {
		if err := v.validator.RegisterValidation(tag, tagValidationDetails.validatorFunc); err != nil {
			v.logger.Error("failed to register customer validator function", "err", err.Error())
			return err
		}
	}
	return nil
}

func useJSONFieldNames(fld reflect.StructField) string {
	name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
	if name == "-" {
		return ""
	}
	return name
}

func (v *Validator) isValidPath(fl validator.FieldLevel) bool {
	inputPath := fl.Field().String()
	if len(inputPath) == 0 {
		return true
	}
	if strings.TrimSpace(inputPath) == "" {
		v.logger.Warn("validation path is empty", "path", inputPath)
		return false
	}

	if strings.Contains(inputPath, "\x00") {
		v.logger.Warn("validation path has null byte", "path", inputPath)
		return false
	}

	if !strings.HasPrefix(inputPath, "/") {
		v.logger.Warn("validation path does not start with /", "path", inputPath)
		return false
	}

	if _, err := os.Stat(inputPath); err != nil {
		v.logger.Info("path does not exist", "path", inputPath)
		return false
	}

	return true
}

func (v *Validator) isValidQuery(fl validator.FieldLevel) bool {
	query := fl.Field().String()
	if len(query) == 0 {
		return false
	}
	if strings.TrimSpace(query) == "" {
		v.logger.Warn("query is empty", "query", query)
		return false
	}

	return true
}
