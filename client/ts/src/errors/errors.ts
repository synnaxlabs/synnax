enum ErrorType {
  Unexpected = "unexpected",
  General = "general",
  Validation = "validation",
  Syntax = "syntax",
  Auth = "auth",
  Parse = "parse",
}

interface TypedError {
  Type: ErrorType;
  Error: unknown;
}

interface FieldError {
  Field: string;
  Message: string;
}