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
} from "@synnaxlabs/freighter";
import { z } from "zod";

const _FREIGHTER_EXCEPTION_TYPE = "synnax.api.errors";

const APIErrorPayloadSchema = z.object({
  type: z.string(),
  error: z.record(z.unknown()),
});

type APIErrorPayload = z.infer<typeof APIErrorPayloadSchema>;

enum APIErrorType {
  General = "general",
  Nil = "nil",
  Parse = "parse",
  Auth = "auth",
  Unexpected = "unexpected",
  Validation = "validation",
  Query = "query",
  Route = "route",
}

export interface Field {
  field: string;
  message: string;
}

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

const parsePayload = (payload: APIErrorPayload): Error | null => {
  switch (payload.type) {
    case APIErrorType.General:
      return new GeneralError(payload.error.message as string);
    case APIErrorType.Parse:
      return new ParseError(payload.error.message as string);
    case APIErrorType.Auth:
      return new AuthError(payload.error.message as string);
    case APIErrorType.Unexpected:
      return new UnexpectedError(JSON.stringify(payload.error));
    case APIErrorType.Validation:
      return new ValidationError(payload.error.fields as string | Field[]);
    case APIErrorType.Query:
      return new QueryError(payload.error.message as string);
    case APIErrorType.Route:
      return new RouteError(
        payload.error.path as string,
        payload.error.message as string,
      );
    default:
      return null;
  }
};

const decode = (encoded: string): Error | null => {
  return parsePayload(APIErrorPayloadSchema.parse(JSON.parse(encoded)));
};

const encode = (): string => {
  throw new Error("Not implemented");
};

registerError({ type: _FREIGHTER_EXCEPTION_TYPE, encode, decode });

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
