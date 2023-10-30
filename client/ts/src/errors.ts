// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  BaseTypedError,
  type Middleware,
  registerError,
  Unreachable,
  type ErrorPayload,
} from "@synnaxlabs/freighter";

const _FREIGHTER_EXCEPTION_PREFIX = "sy.api.";

enum APIErrorType {
  General = _FREIGHTER_EXCEPTION_PREFIX + "general",
  Parse = _FREIGHTER_EXCEPTION_PREFIX + "parse",
  Auth = _FREIGHTER_EXCEPTION_PREFIX + "auth",
  Unexpected = _FREIGHTER_EXCEPTION_PREFIX + "unexpected",
  Validation = _FREIGHTER_EXCEPTION_PREFIX + "validation",
  Query = _FREIGHTER_EXCEPTION_PREFIX + "query",
  Route = _FREIGHTER_EXCEPTION_PREFIX + "route",
}

export interface Field {
  field: string;
  message: string;
}

class BaseError extends BaseTypedError {
  constructor(message: string) {
    super(message, _FREIGHTER_EXCEPTION_PREFIX);
  }
}

/**
 * Raised when a validation error occurs.
 */
export class ValidationError extends BaseError {
  fields: Field[];

  constructor(fieldsOrMessage: string | Field[] | Field) {
    if (typeof fieldsOrMessage === "string") {
      super(fieldsOrMessage);
      this.fields = [];
    } else if (Array.isArray(fieldsOrMessage)) {
      super(
        fieldsOrMessage.map((field) => `${field.field}: ${field.message}`).join("\n"),
      );
      this.fields = fieldsOrMessage;
    } else {
      super(`${fieldsOrMessage.field}: ${fieldsOrMessage.message}`);
      this.fields = [fieldsOrMessage];
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

const decode = (payload: ErrorPayload): Error | null => {
  if (!payload.type.startsWith(_FREIGHTER_EXCEPTION_PREFIX)) return null;
  switch (payload.type) {
    case APIErrorType.General:
      return new GeneralError(payload.data);
    case APIErrorType.Parse:
      return new ParseError(payload.data);
    case APIErrorType.Auth:
      return new AuthError(payload.data);
    case APIErrorType.Unexpected:
      return new UnexpectedError(payload.data);
    case APIErrorType.Validation:
      return new ValidationError(payload.data);
    case APIErrorType.Query:
      return new QueryError(payload.data);
    case APIErrorType.Route:
      return new RouteError(payload.data, payload.data);
    default:
      return new UnexpectedError(payload.data);
  }
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
  if (value == null) throw new ValidationError({ field: key, message });
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
