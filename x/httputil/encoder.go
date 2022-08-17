package httputil

import (
	"github.com/arya-analytics/x/binary"
	"github.com/cockroachdb/errors"
)

type EncoderDecoder interface {
	binary.EncoderDecoder
	ContentType() string
}

type JSONEncoderDecoder struct {
	binary.JSONEncoderDecoder
}

func (j *JSONEncoderDecoder) ContentType() string { return "application/json" }

type MsgPackEncoderDecoder struct {
	binary.MsgPackEncoderDecoder
}

func (m *MsgPackEncoderDecoder) ContentType() string { return "application/msgpack" }

var encoderDecoders = []EncoderDecoder{
	&JSONEncoderDecoder{},
	&MsgPackEncoderDecoder{},
}

func DetermineEncoderDecoder(contentType string) (EncoderDecoder, error) {
	for _, ecd := range encoderDecoders {
		if ecd.ContentType() == contentType {
			return ecd, nil
		}
	}
	return nil, errors.New("[encoding] - unable to determine encoding type")
}

func SupportedContentTypes() []string {
	var contentTypes []string
	for _, ecd := range encoderDecoders {
		contentTypes = append(contentTypes, ecd.ContentType())
	}
	return contentTypes
}
