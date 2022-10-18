package logutil

import (
	"fmt"
	"github.com/kr/pretty"
	"github.com/sirupsen/logrus"
)

func Pretty(v ...any) {
	formats := make([]interface{}, len(v))
	for i, x := range v {
		formats[i] = fmt.Sprintf("%# s", pretty.Formatter(x))
	}
	logrus.Info(formats...)
}
