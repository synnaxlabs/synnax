// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Middleware, Unreachable } from "@synnaxlabs/freighter";
import { errors } from "@synnaxlabs/x";

export class SynnaxError extends errors.createTyped("sy") {}

/**
 * Raised when a validation error occurs.
 */
export class ValidationError extends SynnaxError.sub("validation") {}

export class FieldError extends ValidationError.sub("field") {
  readonly field: string;
  readonly message: string;

  constructor(field: string, message: string) {
    super(`${field}: ${message}`);
    this.field = field;
    this.message = message;
  }
}

/**
 * AuthError is raised when an authentication error occurs.
 */
export class AuthError extends SynnaxError.sub("auth") {}

/**
 * InvalidTokenError is raised when an authentication token is invalid.
 */
export class InvalidTokenError extends AuthError.sub("invalid-token") {}

export class ExpiredTokenError extends AuthError.sub("expired-token") {}

/**
 * UnexpectedError is raised when an unexpected error occurs.
 */
export class UnexpectedError extends SynnaxError.sub("unexpected") {
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
export class QueryError extends SynnaxError.sub("query") {}

export class NotFoundError extends QueryError.sub("not_found") {}

export class MultipleFoundError extends QueryError.sub("multiple_results") {}

/**
 * RouteError is raised when a routing error occurs.
 */
export class RouteError extends SynnaxError.sub("route") {
  path: string;

  constructor(message: string, path: string) {
    super(message);
    this.path = path;
  }
}

export class ControlError extends SynnaxError.sub("control") {}

export class UnauthorizedError extends ControlError.sub("unauthorized") {}

/**
 * Raised when time-series data is not contiguous.
 */
export class ContiguityError extends SynnaxError.sub("contiguity") {}

const decode = (payload: errors.Payload): Error | null => {
  if (!payload.type.startsWith(SynnaxError.TYPE)) return null;
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
    if (payload.type.startsWith(ExpiredTokenError.TYPE))
      return new ExpiredTokenError(payload.data);
    return new AuthError(payload.data);
  }

  if (payload.type.startsWith(UnexpectedError.TYPE))
    return new UnexpectedError(payload.data);

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

const encode = (): errors.Payload => {
  throw new Error("Not implemented");
};

errors.register({ encode, decode });

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
