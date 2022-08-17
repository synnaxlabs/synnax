package grpc

//
//// TranslateResponseForward translates an api error response into a grpc compatible error response.
//func TranslateResponseForward(res errors.Response) *errorsv1.Response {
//	grpcRes := &errorsv1.Response{}
//	for _, err := range res.Errors {
//		t := &errorsv1.Typed{Type: convertTypeForward(err.Type)}
//		if err.Type != errors.TypeValidation {
//			t.Message = parseErrorMessageForward(err)
//		} else {
//			// Do a best effort to parse the validation fields
//			// If we can't, we'll just return the message as a best effort.
//			flds, ok := parseValidationFieldsForward(err)
//			if !ok {
//				t.Message = parseErrorMessageForward(err)
//			} else {
//				t.Fields = flds
//			}
//		}
//		grpcRes.Errors = append(grpcRes.Errors, t)
//	}
//	return grpcRes
//}
//
//// TranslateResponseBackward translates a grpc error response into an api error response.
//func TranslateResponseBackward(res *errorsv1.Response) errors.Response {
//	resp := errors.Response{}
//	for _, err := range res.Errors {
//		errT := convertTypeBackward(err.Type)
//		t := errors.Typed{Type: convertTypeBackward(err.Type)}
//		if errT != errors.TypeValidation {
//			t.Err = errors.Message{Message: err.Message.Message}
//		} else {
//			// Do a best effort to parse the validation fields
//			// If we can't, we'll just return the message as a best effort.
//			flds := parseValidationFieldsBackward(err.Fields)
//			if len(flds) == 0 {
//				t.Err = errors.Message{Message: err.Message.Message}
//			} else {
//				t.Err = flds
//			}
//		}
//	}
//	return resp
//}
//
//var (
//	forwardTypeMap = map[errors.Type]errorsv1.Type{
//		errors.TypeUnexpected: errorsv1.Type_UNEXPECTED,
//		errors.TypeGeneral:    errorsv1.Type_GENERAL,
//		errors.TypeNil:        errorsv1.Type_NIL,
//		errors.TypeValidation: errorsv1.Type_VALIDATION,
//		errors.TypeParse:      errorsv1.Type_PARSE,
//		errors.TypeAuth:       errorsv1.Type_AUTH,
//	}
//	backwardTypeMap = map[errorsv1.Type]errors.Type{
//		errorsv1.Type_UNEXPECTED: errors.TypeUnexpected,
//		errorsv1.Type_GENERAL:    errors.TypeGeneral,
//		errorsv1.Type_NIL:        errors.TypeNil,
//		errorsv1.Type_VALIDATION: errors.TypeValidation,
//		errorsv1.Type_PARSE:      errors.TypeParse,
//		errorsv1.Type_AUTH:       errors.TypeAuth,
//	}
//)
//
//func convertTypeForward(t errors.Type) errorsv1.Type {
//	grpcT, ok := forwardTypeMap[t]
//	if !ok {
//		return errorsv1.Type_UNEXPECTED
//	}
//	return grpcT
//}
//
//func convertTypeBackward(t errorsv1.Type) errors.Type {
//	apiT, ok := backwardTypeMap[t]
//	if !ok {
//		return errors.TypeUnexpected
//	}
//	return apiT
//}
//
//func parseErrorMessageForward(err errors.Typed) *errorsv1.Message {
//	return &errorsv1.Message{Message: err.Error()}
//}
//
//func parseValidationFieldsForward(err errors.Typed) ([]*errorsv1.Field, bool) {
//	fields, ok := err.Err.(errors.Fields)
//	if !ok {
//		return nil, false
//	}
//	res := make([]*errorsv1.Field, len(fields))
//	for i, f := range fields {
//		res[i] = &errorsv1.Field{Field: f.Field, Message: f.Message}
//	}
//	return res, true
//}
//
//func parseValidationFieldsBackward(fields []*errorsv1.Field) errors.Fields {
//	res := make(errors.Fields, len(fields))
//	for i, f := range fields {
//		res[i] = errors.Field{Field: f.Field, Message: f.Message}
//	}
//	return res
//}
