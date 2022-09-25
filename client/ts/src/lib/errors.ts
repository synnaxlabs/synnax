import { BaseTypedError, registerError } from '@synnaxlabs/freighter';
import { z } from 'zod';

const _FREIGHTER_EXCEPTION_TYPE = 'synnax.api.errors';

const APIErrorPayloadSchema = z.object({
  type: z.string(),
  error: z.record(z.unknown()),
});

type APIErrorPayload = z.infer<typeof APIErrorPayloadSchema>;

enum APIErrorType {
  GENERAL = 'general',
  NIL = 'nil',
  PARSE = 'parse',
  AUTH = 'auth',
  UNEXPECTED = 'unexpected',
  VALIDATION = 'validation',
  QUERY = 'query',
  ROUTE = 'route',
}

export type Field = {
  field: string;
  message: string;
};

class BaseError extends BaseTypedError {
  constructor(message: string) {
    super(message, _FREIGHTER_EXCEPTION_TYPE);
  }
}

/**
 * Raised when a validation error occurs.
 */
export class ValidationError extends BaseError {
  fields: Field[];

  constructor(fieldsOrMessage: string | Field[]) {
    if (typeof fieldsOrMessage === 'string') {
      super(fieldsOrMessage);
      this.fields = [];
    } else {
      super(
        fieldsOrMessage
          .map((field) => `${field.field}: ${field.message}`)
          .join('\n')
      );
      this.fields = fieldsOrMessage;
    }
  }
}

/**
 * GeneralError is raised when a general error occurs.
 */
export class GeneralError extends BaseError {}

/**
 * ParseError is raised when a parse error occurs.
 */
export class ParseError extends BaseError {}

/**
 * AuthError is raised when an authentication error occurs.
 */
export class AuthError extends BaseError {}

/**
 * UnexpectedError is raised when an unexpected error occurs.
 */
export class UnexpectedError extends BaseError {}

/**
 * QueryError is raised when a query error occurs.
 */
export class QueryError extends BaseError {}

/**
 * RouteError is raised when a routing error occurs.
 */
export class RouteError extends BaseError {
  path: string;

  constructor(message: string, path: string) {
    super(message);
    this.path = path;
  }
}

/**
 * Raised when time-series data is not contiguous.
 */
export class ContiguityError extends BaseError {}

const parsePayload = (payload: APIErrorPayload): Error | undefined => {
  switch (payload.type) {
    case APIErrorType.GENERAL:
      return new GeneralError(payload.error.message as string);
    case APIErrorType.PARSE:
      return new ParseError(payload.error.message as string);
    case APIErrorType.AUTH:
      return new AuthError(payload.error.message as string);
    case APIErrorType.UNEXPECTED:
      return new UnexpectedError(payload.error as unknown as string);
    case APIErrorType.VALIDATION:
      return new ValidationError(payload.error.fields as string | Field[]);
    case APIErrorType.QUERY:
      return new QueryError(payload.error.message as string);
    case APIErrorType.ROUTE:
      return new RouteError(
        payload.error.path as string,
        payload.error.message as string
      );
    default:
      return undefined;
  }
};

const decode = (encoded: string): Error | undefined => {
  return parsePayload(APIErrorPayloadSchema.parse(JSON.parse(encoded)));
};

const encode = (): string => {
  throw new Error('Not implemented');
};

registerError({ type: _FREIGHTER_EXCEPTION_TYPE, encode, decode });
