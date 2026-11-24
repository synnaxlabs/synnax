// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Middleware, Timeout, Unreachable } from "@synnaxlabs/freighter";
import { array, errors } from "@synnaxlabs/x";
import { z } from "zod";

export class SynnaxError extends errors.createTyped("sy") {}

/** Raised when a validation error occurs. */
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

  static override matchExact(e: unknown): e is PathError {
    return (
      e != null &&
      typeof e === "object" &&
      "path" in e &&
      "error" in e &&
      "type" in e &&
      typeof e.type === "string" &&
      e.type === PathError.TYPE
    );
  }
}

/** AuthError is raised when an authentication error occurs. */
export class AuthError extends SynnaxError.sub("auth") {}

/** Raised when the Core matches no user to a username and password. */
export class InvalidCredentialsError extends AuthError.sub("invalid-credentials") {}

/** Raised when a request or login has no credentials, API key, token, or cookie. */
export class MissingCredentialsError extends AuthError.sub("missing_credentials") {}

/** Raised when the client cannot load an API key, such as from Synnax Desktop. */
export class APIKeyUnavailableError extends AuthError.sub("api_key_unavailable") {}

/** InvalidTokenError is raised when an authentication token is invalid. */
export class InvalidTokenError extends AuthError.sub("invalid_token") {}

export class ExpiredTokenError extends AuthError.sub("expired_token") {}

/** SessionEndedError is raised when the session a stream or request used has ended. */
export class SessionEndedError extends AuthError.sub("session_ended") {}

/** Raised when the caller's policies do not permit the action. */
export class AccessDeniedError extends AuthError.sub("access_denied") {
  toStatus() {
    return {
      message: "You do not have permission to do that",
      description: this.message,
    };
  }
}

/** UnexpectedError is raised when an unexpected error occurs. */
export class UnexpectedError extends SynnaxError.sub("unexpected") {
  constructor(message: string) {
    super(`
    Unexpected error encountered:

    ${message}

    Please report this to the Synnax team.
    `);
  }
}

/** QueryError is raised when a query error occurs. */
export class QueryError extends SynnaxError.sub("query") {}

export class NotFoundError extends QueryError.sub("not_found") {}

export class MultipleFoundError extends QueryError.sub("multiple_results") {}

/** UnsupportedError is raised when a query targets an entity that cannot answer it. */
export class UnsupportedError extends QueryError.sub("unsupported") {}

/** RouteError is raised when a routing error occurs. */
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
  constructor(message: string = "Operation failed because no Core is connected.") {
    super(message);
  }
}

/**
 * Whether the error means the Core could not be reached: a request that failed on the
 * wire or got no response in time, or one short-circuited while the connection is known
 * unreachable.
 */
export const isConnectionError = (err: unknown): boolean =>
  Unreachable.matches(err) || Timeout.matches(err) || DisconnectedError.matches(err);

const DOCUMENT_REFETCHES =
  "The document fetches its latest state from the Core. No action is needed.";

// The status text of each kind of InternalError: what failed, then what the user loses
// and what to do now.
const INTERNAL_STATUSES = {
  connectionCheck: {
    message: "Stopped checking the connection to the Core",
    description:
      "The connection status no longer updates, and a lost connection does not " +
      "recover on its own. Reload the window to check again.",
  },
  streamDenied: {
    message: "Live updates unavailable",
    description:
      "This user cannot read the stream of changes, so lists and values update " +
      "only when they are fetched again. Ask an administrator for access.",
  },
  previousCore: {
    message: "Failed to clear data from the previous Core",
    description:
      "A different Core now runs at this address, so some values may come from the " +
      "old one. Reload the window to clear them.",
  },
  liveUpdates: {
    message: "Failed to start live updates",
    description:
      "Lists and values do not update until live updates start. The connection " +
      "check tries again on its own.",
  },
  cacheRefresh: {
    message: "Failed to refresh cached data",
    description:
      "Some lists and values may be out of date until they change again. Reload " +
      "the window to fetch them again.",
  },
  notify: {
    message: "Failed to show a change",
    description:
      "A view may not show the latest data. Reload the window to refresh it.",
  },
  viewRefresh: {
    message: "Failed to keep a view current",
    description:
      "A view may show old data until it changes again. Reload the window to fetch " +
      "it again.",
  },
  recordLoad: {
    message: "Failed to load some records",
    description:
      "A view may leave out records that it should show. Reload the window to " +
      "fetch them again.",
  },
  documentChange: {
    message: "Failed to apply a change to a document",
    description: DOCUMENT_REFETCHES,
  },
  remoteChange: {
    message: "Failed to apply a change from another session",
    description: DOCUMENT_REFETCHES,
  },
  documentFetch: {
    message: "Failed to fetch a document",
    description:
      "The document may show old content until it changes again. Reload the window " +
      "to fetch it again.",
  },
} as const;

/**
 * Raised for a failure of work that the client runs on its own, with no caller to
 * return it to, such as keeping its caches current. Its message names what failed, its
 * description says what the user loses and what to do, and its cause is the raw error.
 */
export class InternalError extends SynnaxError.sub("internal") {
  readonly description: string;

  constructor(kind: keyof typeof INTERNAL_STATUSES, options: ErrorOptions) {
    const { message, description } = INTERNAL_STATUSES[kind];
    super(message, options);
    this.description = description;
  }

  static override matchExact(e: unknown): e is InternalError {
    return super.matchExact(e) && "description" in e;
  }
}

/** Raised when an edit targets a document that is not in the client's cache. */
export class NotCachedError extends SynnaxError.sub("not_cached") {}

/** Raised when time-series data is not contiguous. */
export class ContiguityError extends SynnaxError.sub("contiguity") {}

/** Raised when the Core refuses a request over its license. */
export class LicenseError extends SynnaxError.sub("license") {}

/** Raised when no license is active on the Core. */
export class MissingLicenseError extends LicenseError.sub("missing") {}

/** Raised when the license on the Core no longer applies. */
export class ExpiredLicenseError extends LicenseError.sub("expired") {}

/** Raised when a license key cannot be verified or is malformed. */
export class InvalidLicenseError extends LicenseError.sub("invalid") {}

/** Raised when a license is bound to a different machine. */
export class LicenseFingerprintError extends LicenseError.sub("fingerprint") {}

/** Raised when a channel would exceed the license's channel cap. */
export class LicenseLimitError extends LicenseError.sub("too_many") {}

const decode = (payload: errors.Payload): Error | null => {
  if (!payload.type.startsWith(SynnaxError.TYPE)) return null;
  if (payload.type.startsWith(ValidationError.TYPE)) {
    if (payload.type === PathError.TYPE) return PathError.decode(payload);
    return new ValidationError(payload.data);
  }

  if (payload.type.startsWith(AuthError.TYPE)) {
    if (payload.type.startsWith(InvalidCredentialsError.TYPE))
      return new InvalidCredentialsError(payload.data);
    if (payload.type.startsWith(MissingCredentialsError.TYPE))
      return new MissingCredentialsError(payload.data);
    if (payload.type.startsWith(APIKeyUnavailableError.TYPE))
      return new APIKeyUnavailableError(payload.data);
    if (payload.type.startsWith(InvalidTokenError.TYPE))
      return new InvalidTokenError(payload.data);
    if (payload.type.startsWith(ExpiredTokenError.TYPE))
      return new ExpiredTokenError(payload.data);
    if (payload.type.startsWith(SessionEndedError.TYPE))
      return new SessionEndedError(payload.data);
    if (payload.type.startsWith(AccessDeniedError.TYPE))
      return new AccessDeniedError(payload.data);
    return new AuthError(payload.data);
  }

  if (payload.type.startsWith(UnexpectedError.TYPE))
    return new UnexpectedError(payload.data);

  if (payload.type.startsWith(QueryError.TYPE)) {
    if (payload.type.startsWith(NotFoundError.TYPE))
      return new NotFoundError(payload.data);
    if (payload.type.startsWith(MultipleFoundError.TYPE))
      return new MultipleFoundError(payload.data);
    if (payload.type.startsWith(UnsupportedError.TYPE))
      return new UnsupportedError(payload.data);
    return new QueryError(payload.data);
  }

  if (payload.type.startsWith(ControlError.TYPE)) {
    if (payload.type.startsWith(UnauthorizedError.TYPE))
      return new UnauthorizedError(payload.data);
    return new ControlError(payload.data);
  }

  if (payload.type.startsWith(NotCachedError.TYPE))
    return new NotCachedError(payload.data);

  if (payload.type.startsWith(RouteError.TYPE))
    return new RouteError(payload.data, payload.data);

  if (payload.type.startsWith(LicenseError.TYPE)) {
    if (payload.type.startsWith(MissingLicenseError.TYPE))
      return new MissingLicenseError(payload.data);
    if (payload.type.startsWith(ExpiredLicenseError.TYPE))
      return new ExpiredLicenseError(payload.data);
    if (payload.type.startsWith(InvalidLicenseError.TYPE))
      return new InvalidLicenseError(payload.data);
    if (payload.type.startsWith(LicenseFingerprintError.TYPE))
      return new LicenseFingerprintError(payload.data);
    if (payload.type.startsWith(LicenseLimitError.TYPE))
      return new LicenseLimitError(payload.data);
    return new LicenseError(payload.data);
  }

  return new UnexpectedError(payload.data);
};

const encode: errors.Encoder = (error) => {
  if (!error.type.startsWith(SynnaxError.TYPE)) return null;
  if (PathError.matchExact(error)) {
    const { path, error: cause } = error;
    return {
      type: PathError.TYPE,
      data: JSON.stringify({ path, error: errors.encode(cause) }),
    };
  }
  return { type: error.type, data: error.message };
};

errors.register({ encode, decode });

export const errorsMiddleware: Middleware = async (ctx, next) => {
  try {
    return await next(ctx);
  } catch (err) {
    if (err instanceof Unreachable) {
      const message = `Cannot reach Core at ${err.url.host}:${err.url.port}`;
      err.message = err.hint == null ? message : `${message}. ${err.hint}`;
    }
    throw errors.fromUnknown(err);
  }
};
