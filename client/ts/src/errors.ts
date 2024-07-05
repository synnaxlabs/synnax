// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  BaseTypedError,
  errorMatcher,
  type ErrorPayload,
  type Middleware,
  registerError,
  Unreachable,
} from "@synnaxlabs/freighter";

const _FREIGHTER_EXCEPTION_PREFIX = "sy.";

export interface Field {
  field: string;
  message: string;
}

/**
 * Raised when a validation error occurs.
 */
export class ValidationError extends BaseTypedError {
  static readonly TYPE = _FREIGHTER_EXCEPTION_PREFIX + "validation";
  type = ValidationError.TYPE;
  static readonly matches = errorMatcher(ValidationError.TYPE);
}

export class FieldError extends ValidationError {
  static readonly TYPE = ValidationError.TYPE + ".field";
  type = FieldError.TYPE;
  static readonly matches = errorMatcher(FieldError.TYPE);
  readonly field: string;
  readonly message: string;

  constructor(field: string, message: string) {
    super(field + ": " + message);
    this.field = field;
    this.message = message;
  }
}

/**
 * AuthError is raised when an authentication error occurs.
 */
export class AuthError extends BaseTypedError {
  static readonly TYPE = _FREIGHTER_EXCEPTION_PREFIX + "auth";
  type = AuthError.TYPE;
  static readonly matches = errorMatcher(AuthError.TYPE);
}

/**
 * InvalidTokenError is raised when an authentication token is invalid.
 */
export class InvalidTokenError extends AuthError {
  static readonly TYPE = AuthError.TYPE + ".invalid-token";
  type = InvalidTokenError.TYPE;
  static readonly matches = errorMatcher(InvalidTokenError.TYPE);
}

/**
 * UnexpectedError is raised when an unexpected error occurs.
 */
export class UnexpectedError extends BaseTypedError {
  static readonly TYPE = _FREIGHTER_EXCEPTION_PREFIX + "unexpected";
  type = UnexpectedError.TYPE;
  static readonly matches = errorMatcher(UnexpectedError.TYPE);

  constructor(message: string) {
    super(`
    Unexpected error encountered:

    ${message}

    Please report this to the Synnax team.
    `);
  }
}

/**
 * QueryError is raised when a query error occurs.
 */
export class QueryError extends BaseTypedError {
  static readonly TYPE = _FREIGHTER_EXCEPTION_PREFIX + "query";
  type = QueryError.TYPE;
  static readonly matches = errorMatcher(QueryError.TYPE);
}

export class NotFoundError extends QueryError {
  static readonly TYPE = QueryError.TYPE + ".not_found";
  type = NotFoundError.TYPE;
  static readonly matches = errorMatcher(NotFoundError.TYPE);
}

export class MultipleFoundError extends QueryError {
  static readonly TYPE = QueryError.TYPE + ".multiple_results";
  type = MultipleFoundError.TYPE;
  static readonly matches = errorMatcher(MultipleFoundError.TYPE);
}

/**
 * RouteError is raised when a routing error occurs.
 */
export class RouteError extends BaseTypedError {
  static readonly TYPE = _FREIGHTER_EXCEPTION_PREFIX + "route";
  type = RouteError.TYPE;
  static readonly matches = errorMatcher(RouteError.TYPE);
  path: string;

  constructor(message: string, path: string) {
    super(message);
    this.path = path;
  }
}

export class ControlError extends BaseTypedError {
  static readonly TYPE = _FREIGHTER_EXCEPTION_PREFIX + "control";
  type = ControlError.TYPE;
  static readonly matches = errorMatcher(ControlError.TYPE);
}

export class UnauthorizedError extends ControlError {
  static readonly TYPE = ControlError.TYPE + ".unauthorized";
  type = UnauthorizedError.TYPE;
  static readonly matches = errorMatcher(UnauthorizedError.TYPE);
}

/**
 * Raised when time-series data is not contiguous.
 */
export class ContiguityError extends BaseTypedError {
  static readonly TYPE = _FREIGHTER_EXCEPTION_PREFIX + "contiguity";
  type = ContiguityError.TYPE;
  static readonly matches = errorMatcher(ContiguityError.TYPE);
}

const decode = (payload: ErrorPayload): Error | null => {
  if (!payload.type.startsWith(_FREIGHTER_EXCEPTION_PREFIX)) return null;
  if (payload.type.startsWith(ValidationError.TYPE)) {
    if (payload.type === FieldError.TYPE) {
      const values = payload.data.split(": ");
      if (values.length < 2) return new ValidationError(payload.data);
      return new FieldError(values[0], values[1]);
    }
    return new ValidationError(payload.data);
  }

  if (payload.type.startsWith(AuthError.TYPE)) {
    if (payload.type.startsWith(InvalidTokenError.TYPE))
      return new InvalidTokenError(payload.data);
    return new AuthError(payload.data);
  }

  if (payload.type.startsWith(UnexpectedError.TYPE)) {
    return new UnexpectedError(payload.data);
  }

  if (payload.type.startsWith(QueryError.TYPE)) {
    if (payload.type.startsWith(NotFoundError.TYPE))
      return new NotFoundError(payload.data);
    if (payload.type.startsWith(MultipleFoundError.TYPE))
      return new MultipleFoundError(payload.data);
    return new QueryError(payload.data);
  }

  if (payload.type.startsWith(ControlError.TYPE)) {
    if (payload.type.startsWith(UnauthorizedError.TYPE))
      return new UnauthorizedError(payload.data);
    return new ControlError(payload.data);
  }

  if (payload.type.startsWith(RouteError.TYPE))
    return new RouteError(payload.data, payload.data);

  return new UnexpectedError(payload.data);
};

const encode = (): ErrorPayload => {
  throw new Error("Not implemented");
};

registerError({ encode, decode });

export const validateFieldNotNull = (
  key: string,
  value: unknown,
  message: string = "must be provided",
): void => {
  if (value == null) throw new FieldError(key, message);
};

export const errorsMiddleware: Middleware = async (ctx, next) => {
  const [res, err] = await next(ctx);
  if (err == null) return [res, err];
  if (err instanceof Unreachable)
    return [
      res,
      new Unreachable({
        message: `Cannot reach cluster at ${err.url.host}:${err.url.port}`,
        url: err.url,
      }),
    ];
  return [res, err];
};
