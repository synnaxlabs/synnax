// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Middleware, Unreachable } from "@synnaxlabs/freighter";
import { array, errors } from "@synnaxlabs/x";
import { z } from "zod";

export class SynnaxError extends errors.createTyped("sy") {}

/**
 * Raised when a validation error occurs.
 */
export class ValidationError extends SynnaxError.sub("validation") {}

export class PathError extends ValidationError.sub("path") {
  readonly path: string[];
  readonly error: Error;
  static readonly encodedSchema = z.object({
    path: z.string().array(),
    error: errors.payloadZ,
  });

  constructor(path: string | string[], error: Error) {
    const arrPath = array.toArray(path);
    super(`${arrPath.join(".")}: ${error.message}`);
    this.path = arrPath.flatMap((p) => p.split("."));
    this.error = error;
  }

  static decode(payload: errors.Payload): PathError {
    const decoded = PathError.encodedSchema.parse(JSON.parse(payload.data));
    return new PathError(decoded.path, errors.decode(decoded.error) as Error);
  }
}

/**
 * AuthError is raised when an authentication error occurs.
 */
export class AuthError extends SynnaxError.sub("auth") {}

/**
 * InvalidTokenError is raised when an authentication token is invalid.
 */
export class InvalidTokenError extends AuthError.sub("invalid_token") {}

export class ExpiredTokenError extends AuthError.sub("expired_token") {}

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

export class DisconnectedError extends SynnaxError.sub("disconnected") {
  constructor(message: string = "Operation failed because no cluster is connected.") {
    super(message);
  }
}

/**
 * Raised when time-series data is not contiguous.
 */
export class ContiguityError extends SynnaxError.sub("contiguity") {}

const decode = (payload: errors.Payload): Error | null => {
  if (!payload.type.startsWith(SynnaxError.TYPE)) return null;
  if (payload.type.startsWith(ValidationError.TYPE)) {
    if (payload.type === PathError.TYPE) return PathError.decode(payload);
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
  throw new errors.NotImplemented();
};

errors.register({ encode, decode });

export const validateFieldNotNull = (
  key: string,
  value: unknown,
  message: string = "must be provided",
): void => {
  if (value == null) throw new PathError(key, new ValidationError(message));
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
