package api

import (
	"github.com/arya-analytics/delta/pkg/api/errors"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
	"reflect"
	"strings"
)

func newValidator() *validator.Validate {
	v := validator.New()
	v.RegisterTagNameFunc(tagNameFunc)
	return v
}

var tagNames = []string{"json", "msgpack"}

func tagNameFunc(fld reflect.StructField) string {
	if fld.Anonymous {
		return errors.EmbeddedFieldName
	}
	for _, tagName := range tagNames {
		if name, ok := getTagName(fld, tagName); ok {
			return name
		}
	}
	zap.S().Warnf("no tag name found for field %s", fld.Name)
	return strings.ToLower(fld.Name)
}

func getTagName(fld reflect.StructField, tagName string) (string, bool) {
	name := strings.SplitN(fld.Tag.Get(tagName), ",", 2)[0]
	return name, name != "-"
}
