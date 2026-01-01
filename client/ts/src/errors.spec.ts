// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type errors, id, uuid } from "@synnaxlabs/x";
import { describe, expect, test } from "vitest";

import {
  AuthError,
  ContiguityError,
  ControlError,
  InvalidTokenError,
  MultipleFoundError,
  NotFoundError,
  PathError,
  QueryError,
  RouteError,
  UnauthorizedError,
  UnexpectedError,
  ValidationError,
} from "@/errors";
import { createTestClient } from "@/testutil/client";

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
      test(`matches ${typeName}`, () => {
        expect(type.matches(error)).toBe(true);
      }),
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
    await client.workspaces.schematics.retrieve({ key: uuid.create() });
  } catch (e) {
    expect(NotFoundError.matches(e)).toBe(true);
  }
});
