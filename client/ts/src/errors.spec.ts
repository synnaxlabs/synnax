import { MatchableErrorType } from "@synnaxlabs/freighter/src/errors";
import { describe, expect, test } from "vitest";

import {
  AuthError,
  ContiguityError,
  ControlError,
  FieldError,
  InvalidTokenError,
  MultipleFoundError,
  NotFoundError,
  QueryError,
  RouteError,
  UnauthorizedError,
  UnexpectedError,
  ValidationError,
} from "@/errors";

describe("error", () => {
  describe("type matching", () => {
    const ERRORS: [string, Error, MatchableErrorType][] = [
      [ValidationError.TYPE, new ValidationError(), ValidationError],
      [FieldError.TYPE, new FieldError("field", "message"), FieldError],
      [AuthError.TYPE, new AuthError(), AuthError],
      [InvalidTokenError.TYPE, new InvalidTokenError(), InvalidTokenError],
      [UnexpectedError.TYPE, new UnexpectedError("message"), UnexpectedError],
      [QueryError.TYPE, new QueryError("message"), QueryError],
      [NotFoundError.TYPE, new NotFoundError("message"), NotFoundError],
      [MultipleFoundError.TYPE, new MultipleFoundError("message"), MultipleFoundError],
      [RouteError.TYPE, new RouteError("message", ""), RouteError],
      [ControlError.TYPE, new ControlError("message"), ControlError],
      [UnauthorizedError.TYPE, new UnauthorizedError("message"), UnauthorizedError],
      [ContiguityError.TYPE, new ContiguityError("message"), ContiguityError],
    ];
    ERRORS.forEach(([typeName, error, type]) =>
      test(`matches ${typeName}`, () => expect(type.matches(error)).toBeTruthy()),
    );
  });
});
