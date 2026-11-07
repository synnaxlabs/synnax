// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { errors, id, uuid } from "@synnaxlabs/x";
import { describe, expect, test } from "vitest";

import {
  AccessDeniedError,
  AuthError,
  ContiguityError,
  ControlError,
  ExpiredLicenseError,
  InvalidLicenseError,
  LicenseError,
  LicenseFingerprintError,
  LicenseLimitError,
  MissingLicenseError,
  MultipleFoundError,
  NotFoundError,
  PathError,
  QueryError,
  RouteError,
  UnauthorizedError,
  UnexpectedError,
  ValidationError,
} from "@/index";
import { status } from "@/status";
import { createTestClient } from "@/testutil";

describe("error", () => {
  describe("type matching", () => {
    const ERRORS: [string, Error, errors.Matchable][] = [
      [ValidationError.TYPE, new ValidationError(), ValidationError],
      [
        PathError.TYPE,
        new PathError("field", new ValidationError("message")),
        PathError,
      ],
      [AuthError.TYPE, new AuthError(), AuthError],
      [UnexpectedError.TYPE, new UnexpectedError("message"), UnexpectedError],
      [QueryError.TYPE, new QueryError("message"), QueryError],
      [NotFoundError.TYPE, new NotFoundError("message"), NotFoundError],
      [MultipleFoundError.TYPE, new MultipleFoundError("message"), MultipleFoundError],
      [RouteError.TYPE, new RouteError("message", ""), RouteError],
      [ControlError.TYPE, new ControlError("message"), ControlError],
      [UnauthorizedError.TYPE, new UnauthorizedError("message"), UnauthorizedError],
      [ContiguityError.TYPE, new ContiguityError("message"), ContiguityError],
      [LicenseError.TYPE, new LicenseError("message"), LicenseError],
      [
        MissingLicenseError.TYPE,
        new MissingLicenseError("message"),
        MissingLicenseError,
      ],
      [
        ExpiredLicenseError.TYPE,
        new ExpiredLicenseError("message"),
        ExpiredLicenseError,
      ],
      [
        InvalidLicenseError.TYPE,
        new InvalidLicenseError("message"),
        InvalidLicenseError,
      ],
      [
        LicenseFingerprintError.TYPE,
        new LicenseFingerprintError("message"),
        LicenseFingerprintError,
      ],
      [LicenseLimitError.TYPE, new LicenseLimitError("message"), LicenseLimitError],
    ];
    ERRORS.forEach(([typeName, error, type]) =>
      test(`matches ${typeName}`, () => {
        expect(type.matches(error)).toBe(true);
      }),
    );
  });

  describe("decode", () => {
    const TOKEN_ERRORS = [
      "sy.auth.invalid_token",
      "sy.auth.expired_token",
      "sy.auth.session_ended",
    ];
    TOKEN_ERRORS.forEach((type) =>
      test(`decodes ${type} as an auth error`, () => {
        const decoded = errors.decode({ type, data: "message" });
        expect(decoded).toMatchObject({ type, message: "message" });
        expect(AuthError.matches(decoded)).toBe(true);
      }),
    );
  });

  describe("encode", () => {
    test("encodes synnax errors into a payload", () => {
      const payload = errors.encode(new NotFoundError("nope"));
      expect(payload.type).toBe(NotFoundError.TYPE);
      expect(payload.data).toBe("nope");
    });

    test("round trips a synnax error through the registry", () => {
      const decoded = errors.decode(errors.encode(new NotFoundError("nope")));
      expect(NotFoundError.matches(decoded)).toBe(true);
    });

    test("round trips a path error through the registry", () => {
      const original = new PathError("field", new ValidationError("bad"));
      const decoded = errors.decode(errors.encode(original));
      expect(PathError.matches(decoded)).toBe(true);
      expect((decoded as PathError).path).toEqual(["field"]);
    });

    test("returns null for foreign typed errors so the registry falls through", () => {
      class ForeignError extends errors.createTyped("foreign") {}
      const payload = errors.encode(new ForeignError("boom"));
      expect(payload.type).toBe(errors.UNKNOWN);
      expect(payload.data).toBe("boom");
    });
  });
});

describe("AccessDeniedError", () => {
  test("should lead the description with a plain denial", () => {
    const err = new AccessDeniedError("access denied: auth error");
    const stat = status.fromException(err, "Failed to delete range");
    expect(stat.variant).toBe("error");
    expect(stat.message).toBe("Failed to delete range");
    expect(stat.description).toBe(
      "You do not have permission to do that. access denied: auth error",
    );
  });

  test("should survive the wire as a denial that still renders", () => {
    const decoded = errors.decode(errors.encode(new AccessDeniedError("nope")));
    expect(AccessDeniedError.matches(decoded)).toBe(true);
    expect(status.fromException(decoded, "Failed to delete range").description).toBe(
      "You do not have permission to do that. nope",
    );
  });
});

const client = createTestClient();

test("client", async () => {
  expect.assertions(2);
  try {
    await client.channels.retrieve(id.create());
  } catch (e) {
    expect(NotFoundError.matches(e)).toBe(true);
  }
  try {
    await client.schematics.retrieve(uuid.create());
  } catch (e) {
    expect(NotFoundError.matches(e)).toBe(true);
  }
});
