// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it } from "vitest";
import { z } from "zod";

import { errors } from "@/errors";
import { status } from "@/status";
import { zod } from "@/zod";

const asParseError = (e: unknown): zod.ParseError => {
  if (!zod.ParseError.matches(e)) throw new Error("expected ParseError");
  return e as zod.ParseError;
};

const parseExpectingError = (
  schema: z.ZodType,
  value: unknown,
  opts?: zod.ParseOptions,
): zod.ParseError => {
  try {
    zod.parse(schema, value, opts);
    throw new Error("expected parse to throw");
  } catch (e) {
    return asParseError(e);
  }
};

describe("zod.parse", () => {
  describe("success", () => {
    it("should return the parsed value when parsing succeeds", () => {
      const schema = z.object({ name: z.string(), age: z.number() });
      expect(zod.parse(schema, { name: "Alice", age: 30 })).toEqual({
        name: "Alice",
        age: 30,
      });
    });

    it("should apply transforms and coercions like schema.parse", () => {
      const schema = z.object({ count: z.coerce.number() });
      expect(zod.parse(schema, { count: "42" })).toEqual({ count: 42 });
    });

    it("should not throw on the success path", () => {
      expect(() => zod.parse(z.string(), "hello")).not.toThrow();
    });
  });

  describe("ParseError shape", () => {
    it("should carry the typed error discriminator", () => {
      const err = parseExpectingError(z.string(), 42);
      expect(err.name).toBe("zod.parse");
      expect(err.type).toBe("zod.parse");
      expect(zod.ParseError.matches(err)).toBe(true);
    });

    it("should not match unrelated errors via ParseError.matches", () => {
      expect(zod.ParseError.matches(new Error("unrelated"))).toBe(false);
      expect(zod.ParseError.matches("string")).toBe(false);
      expect(zod.ParseError.matches(null)).toBe(false);
    });

    it("should preserve the original input on the error", () => {
      const input = { port: "8080" };
      expect(parseExpectingError(z.object({ port: z.number() }), input).input).toBe(
        input,
      );
    });

    it("should retain the zod error as cause", () => {
      expect(parseExpectingError(z.string(), 42).cause).toBeInstanceOf(z.ZodError);
    });

    it("should use the default label when none is provided", () => {
      expect(parseExpectingError(z.string(), 42).label).toBe("value");
    });

    it("should store the label when provided", () => {
      expect(parseExpectingError(z.string(), 42, { label: "task config" }).label).toBe(
        "task config",
      );
    });

    it("should store the context when provided", () => {
      expect(
        parseExpectingError(
          z.object({ port: z.number() }),
          { port: "x" },
          { context: { taskKey: "tk-1" } },
        ).context,
      ).toEqual({ taskKey: "tk-1" });
    });
  });

  describe("message formatting", () => {
    it("should render a root-level scalar mismatch", () => {
      expect(parseExpectingError(z.string(), 42).message).toBe(
        `Failed to parse value (1 issue)

  × expected string, received 42`,
      );
    });

    it("should render a single top-level field failure with parent view", () => {
      expect(
        parseExpectingError(
          z.object({ port: z.number() }),
          { port: "8080" },
          { label: "thing" },
        ).message,
      ).toBe(
        `Failed to parse thing (1 issue)

  {
  ✗ "port": "8080"  × expected number
  }`,
      );
    });

    it("should render multiple top-level sibling failures inline", () => {
      expect(
        parseExpectingError(z.object({ name: z.string(), port: z.number() }), {
          name: 42,
          port: "8080",
        }).message,
      ).toBe(
        `Failed to parse value (2 issues)

  {
  ✗ "name": 42,     × expected string
  ✗ "port": "8080"  × expected number
  }`,
      );
    });

    it("should group siblings under a shared nested parent with context keys", () => {
      expect(
        parseExpectingError(
          z.object({
            config: z.object({
              autoStart: z.boolean(),
              port: z.number(),
              host: z.string(),
              sampleRate: z.number(),
            }),
          }),
          {
            config: {
              autoStart: false,
              port: "8080",
              host: 42,
              sampleRate: 10,
            },
          },
          { label: "task" },
        ).message,
      ).toBe(
        `Failed to parse task (2 issues)

  at config
    {
      "autoStart": false,
    ✗ "port": "8080",  × expected number
    ✗ "host": 42,      × expected string
      "sampleRate": 10
    }`,
      );
    });

    it("should render a nested array element with sibling context", () => {
      expect(
        parseExpectingError(
          z.object({
            channels: z.array(z.object({ name: z.string(), port: z.number() })),
          }),
          {
            channels: [
              { name: "temp", port: 8080 },
              { name: "pressure", port: "oops" },
            ],
          },
          { label: "task config" },
        ).message,
      ).toBe(
        `Failed to parse task config (1 issue)

  at channels[1]
    {
      "name": "pressure",
    ✗ "port": "oops"  × expected number
    }`,
      );
    });

    it("should render a top-level array element failure with array parent view", () => {
      expect(parseExpectingError(z.array(z.number()), [1, 2, "three"]).message).toBe(
        `Failed to parse value (1 issue)

  [
    1,
    2,
  ✗ "three"  × expected number
  ]`,
      );
    });

    it("should render a missing required field as a synthetic entry", () => {
      expect(
        parseExpectingError(
          z.object({
            task: z.number(),
            running: z.boolean(),
            data: z.object({}),
          }),
          { task: 1, running: true },
          { label: "status" },
        ).message,
      ).toBe(
        `Failed to parse status (1 issue)

  {
    "task": 1,
    "running": true,
  ✗ "data": <missing>  × expected object
  }`,
      );
    });

    it("should render unrecognized keys as per-key marks", () => {
      expect(
        parseExpectingError(z.strictObject({ name: z.string() }), {
          name: "Alice",
          extra: 1,
          another: 2,
        }).message,
      ).toBe(
        `Failed to parse value (2 issues)

  {
    "name": "Alice",
  ✗ "extra": 1,   × unexpected key
  ✗ "another": 2  × unexpected key
  }`,
      );
    });

    it("should render enum mismatches at the root flat", () => {
      expect(parseExpectingError(z.enum(["a", "b", "c"]), "d").message).toBe(
        `Failed to parse value (1 issue)

  × expected one of ["a","b","c"], received "d"`,
      );
    });

    it("should render too_small at the root flat", () => {
      expect(parseExpectingError(z.number().min(10), 5).message).toBe(
        `Failed to parse value (1 issue)

  × number too small: expected >=10, received 5`,
      );
    });

    it("should render too_big at the root flat", () => {
      expect(parseExpectingError(z.string().max(3), "hello").message).toBe(
        `Failed to parse value (1 issue)

  × string too large: expected <=3, received "hello"`,
      );
    });

    it("should respect exclusive lower bound for .gt()", () => {
      expect(parseExpectingError(z.number().gt(10), 10).message).toBe(
        `Failed to parse value (1 issue)

  × number too small: expected >10, received 10`,
      );
    });

    it("should respect exclusive upper bound for .lt()", () => {
      expect(parseExpectingError(z.number().lt(10), 10).message).toBe(
        `Failed to parse value (1 issue)

  × number too large: expected <10, received 10`,
      );
    });

    it("should render invalid_format for email", () => {
      expect(parseExpectingError(z.email(), "not-an-email").message).toBe(
        `Failed to parse value (1 issue)

  × expected email format, received "not-an-email"`,
      );
    });

    it("should render invalid_format for url", () => {
      expect(parseExpectingError(z.url(), "not-a-url").message).toBe(
        `Failed to parse value (1 issue)

  × expected url format, received "not-a-url"`,
      );
    });

    it("should render invalid_format for uuid", () => {
      expect(parseExpectingError(z.uuid(), "not-a-uuid").message).toBe(
        `Failed to parse value (1 issue)

  × expected uuid format, received "not-a-uuid"`,
      );
    });

    it("should render not_multiple_of", () => {
      expect(parseExpectingError(z.number().multipleOf(5), 7).message).toBe(
        `Failed to parse value (1 issue)

  × expected multiple of 5, received 7`,
      );
    });

    it("should render custom refinement messages", () => {
      const schema = z.number().refine((n) => n > 0, { message: "must be positive" });
      expect(parseExpectingError(schema, -5).message).toBe(
        `Failed to parse value (1 issue)

  × must be positive, received -5`,
      );
    });

    it("should render a nested invalid_format with the parent view", () => {
      expect(
        parseExpectingError(
          z.object({ email: z.email() }),
          { email: "foo" },
          { label: "contact" },
        ).message,
      ).toBe(
        `Failed to parse contact (1 issue)

  {
  ✗ "email": "foo"  × expected email format
  }`,
      );
    });

    it("should render a nested not_multiple_of with the parent view", () => {
      expect(
        parseExpectingError(
          z.object({ count: z.number().multipleOf(10) }),
          { count: 7 },
          { label: "spec" },
        ).message,
      ).toBe(
        `Failed to parse spec (1 issue)

  {
  ✗ "count": 7  × expected multiple of 10
  }`,
      );
    });

    it("should render a record field with a bad value via invalid_element recursion", () => {
      expect(
        parseExpectingError(
          z.record(z.string(), z.number()),
          { a: 1, b: "bad", c: 3 },
          { label: "scores" },
        ).message,
      ).toBe(
        `Failed to parse scores (1 issue)

  {
    "a": 1,
  ✗ "b": "bad",  × expected number
    "c": 3
  }`,
      );
    });

    it("should keep a marked array index visible past the default truncation", () => {
      expect(
        parseExpectingError(z.array(z.number()), [
          0,
          1,
          2,
          3,
          4,
          5,
          "oops",
          7,
          8,
          9,
          10,
          11,
          12,
        ]).message,
      ).toBe(
        `Failed to parse value (1 issue)

  [
    0,
    1,
    2,
    3,
    4,
    5,
  ✗ "oops",  × expected number
    7,
    "…(+5 more)"
  ]`,
      );
    });

    it("should render a context footer when context is provided", () => {
      expect(
        parseExpectingError(
          z.object({ port: z.number() }),
          { port: "8080" },
          {
            label: "device",
            context: { deviceKey: "dev-1", make: "labjack" },
          },
        ).message,
      ).toBe(
        `Failed to parse device (1 issue)

  {
  ✗ "port": "8080"  × expected number
  }

  context: deviceKey=dev-1, make=labjack`,
      );
    });

    it("should omit the context footer when context is empty", () => {
      expect(
        parseExpectingError(z.string(), 42, { label: "thing", context: {} }).message,
      ).toBe(
        `Failed to parse thing (1 issue)

  × expected string, received 42`,
      );
    });

    it("should pluralize issues correctly: one says '1 issue'", () => {
      const firstLine = parseExpectingError(z.string(), 42).message.split("\n")[0];
      expect(firstLine).toBe("Failed to parse value (1 issue)");
    });

    it("should pluralize issues correctly: two says '2 issues'", () => {
      const firstLine = parseExpectingError(
        z.object({ a: z.string(), b: z.string() }),
        { a: 1, b: 2 },
      ).message.split("\n")[0];
      expect(firstLine).toBe("Failed to parse value (2 issues)");
    });
  });

  describe("union handling", () => {
    it("should flatten invalid_union to the deepest-reaching branch", () => {
      const schema = z.union([
        z.object({ kind: z.literal("x"), x: z.object({ deep: z.string() }) }),
        z.object({ kind: z.literal("y"), y: z.number() }),
      ]);
      expect(parseExpectingError(schema, { kind: "x", x: { deep: 42 } }).message).toBe(
        `Failed to parse value (1 issue)

  at x
    {
    ✗ "deep": 42  × expected string
    }`,
      );
    });

    it("should flatten a deeply nested discriminated union", () => {
      const schema = z.discriminatedUnion("kind", [
        z.object({
          kind: z.literal("a"),
          payload: z.object({ value: z.number() }),
        }),
        z.object({
          kind: z.literal("b"),
          payload: z.object({ value: z.string() }),
        }),
      ]);
      expect(
        parseExpectingError(schema, {
          kind: "a",
          payload: { value: "not-a-number" },
        }).message,
      ).toBe(
        `Failed to parse value (1 issue)

  at payload
    {
    ✗ "value": "not-a-number"  × expected number
    }`,
      );
    });

    it("should flatten a real-world nested optional union to the exact failing leaf", () => {
      const statusZ = z.object({
        details: z.object({ data: z.object({}).nullable() }),
      });
      const taskZ = z.object({ key: z.number(), status: statusZ });
      const responseZ = z.object({
        tasks: z.array(taskZ).nullable().optional(),
      });
      expect(
        parseExpectingError(
          responseZ,
          {
            tasks: [
              {
                key: 1,
                status: { details: { task: 1, running: true } },
              },
            ],
          },
          { label: "task" },
        ).message,
      ).toBe(
        `Failed to parse task (1 issue)

  at tasks[0].status.details
    {
      "task": 1,
      "running": true,
    ✗ "data": <missing>  × expected object
    }`,
      );
    });
  });

  describe("redaction", () => {
    it("should redact sensitive sibling values in the parent view", () => {
      expect(
        parseExpectingError(z.object({ name: z.string(), password: z.string() }), {
          name: 42,
          password: "hunter2",
        }).message,
      ).toBe(
        `Failed to parse value (1 issue)

  {
  ✗ "name": 42,  × expected string
    "password": "[REDACTED]"
  }`,
      );
    });

    it("should redact secrets at arbitrary depth in the parent view", () => {
      expect(
        parseExpectingError(
          z.object({
            outer: z.object({
              middle: z.object({
                inner: z.object({
                  name: z.string(),
                  password: z.string(),
                }),
              }),
            }),
          }),
          {
            outer: {
              middle: {
                inner: { name: 42, password: "hunter2" },
              },
            },
          },
        ).message,
      ).toBe(
        `Failed to parse value (1 issue)

  at outer.middle.inner
    {
    ✗ "name": 42,  × expected string
      "password": "[REDACTED]"
    }`,
      );
    });

    it("should redact secrets in the details input regardless of depth", () => {
      const err = parseExpectingError(
        z.object({
          creds: z.object({ token: z.string() }),
          name: z.string(),
        }),
        { creds: { token: "secret-token" }, name: 42 },
      );
      const details = err.toStatus().details as Record<string, unknown>;
      expect(JSON.stringify(details.input)).not.toContain("secret-token");
      expect(JSON.stringify(details.input)).toContain("[REDACTED]");
    });

    it("should not touch values under keys that are not in the default redact list", () => {
      const err = parseExpectingError(
        z.object({ notes: z.string(), ssn: z.string() }),
        { notes: 42, ssn: "123-45-6789" },
      );
      const details = err.toStatus().details as Record<string, unknown>;
      expect(JSON.stringify(details.input)).toContain("123-45-6789");
    });
  });

  describe("truncation", () => {
    it("should bound the error message length for huge failing values", () => {
      const huge = Array.from({ length: 10_000 }, (_, i) => i);
      const err = parseExpectingError(z.string(), huge);
      expect(err.message.length).toBeLessThan(2_000);
    });
  });

  describe("non-ZodError exceptions", () => {
    it("should not wrap non-zod errors", () => {
      const schema = z.string().transform(() => {
        throw new Error("boom");
      });
      expect(() => zod.parse(schema, "hi")).toThrow();
    });
  });

  describe("toStatus", () => {
    const makeError = () =>
      parseExpectingError(
        z.object({ config: z.object({ port: z.number(), password: z.string() }) }),
        { config: { port: "8080", password: "hunter2" } },
        { label: "task config", context: { taskKey: "tk-1" } },
      );

    it("should return a concise headline as the message", () => {
      expect(makeError().toStatus().message).toBe("Failed to parse task config");
    });

    it("should return the full formatted breakdown as the description", () => {
      expect(makeError().toStatus().description).toBe(
        `Failed to parse task config (1 issue)

  at config
    {
    ✗ "port": "8080",  × expected number
      "password": "[REDACTED]"
    }

  context: taskKey=tk-1`,
      );
    });

    it("should return the redacted input, issues, and context in the details", () => {
      const details = makeError().toStatus().details as Record<string, unknown>;
      expect(details.input).toEqual({
        config: {
          port: "8080",
          password: "[REDACTED]",
        },
      });
      expect(details.context).toEqual({ taskKey: "tk-1" });
      expect(Array.isArray(details.issues)).toBe(true);
      expect((details.issues as unknown[]).length).toBe(1);
    });

    it("should include the raw zod issues for programmatic inspection", () => {
      const err = makeError();
      const details = err.toStatus().details as Record<string, unknown>;
      expect(details.issues).toBe(err.issues);
    });

    it("should omit the context key when no context is provided", () => {
      const err = parseExpectingError(z.string(), 42);
      const details = err.toStatus().details as Record<string, unknown>;
      expect(details.input).toBe(42);
      expect(details.context).toBeUndefined();
      expect(Array.isArray(details.issues)).toBe(true);
    });
  });

  describe("status.fromException integration", () => {
    const makeError = () =>
      parseExpectingError(
        z.object({
          channels: z.array(z.object({ name: z.string(), port: z.number() })),
          password: z.string(),
        }),
        {
          channels: [{ name: "temp", port: "oops" }],
          password: "hunter2",
        },
        { label: "task config", context: { taskKey: "tk-1" } },
      );

    const expectedDescription = `Failed to parse task config (1 issue)

  at channels[0]
    {
      "name": "temp",
    ✗ "port": "oops"  × expected number
    }

  context: taskKey=tk-1`;

    it("should set the status message from toStatus.message", () => {
      expect(status.fromException(makeError()).message).toBe(
        "Failed to parse task config",
      );
    });

    it("should set the status description from toStatus.description", () => {
      expect(status.fromException(makeError()).description).toBe(expectedDescription);
    });

    it("should merge the redacted input, issues, and context into status.details", () => {
      const s = status.fromException(makeError());
      const details = s.details as Record<string, unknown>;
      expect(details.context).toEqual({ taskKey: "tk-1" });
      expect(details.input).toEqual({
        channels: [{ name: "temp", port: "oops" }],
        password: "[REDACTED]",
      });
      expect(Array.isArray(details.issues)).toBe(true);
      expect((details.issues as unknown[]).length).toBe(1);
    });

    it("should still expose issues on the underlying ParseError", () => {
      const err = makeError();
      expect(err.issues.length).toBeGreaterThan(0);
    });

    it("should prefix a caller-provided message with the parse headline", () => {
      expect(status.fromException(makeError(), "Saving failed").message).toBe(
        "Saving failed: Failed to parse task config",
      );
    });

    it("should include a populated stack and the original error in details", () => {
      const details = status.fromException(makeError()).details as Record<
        string,
        unknown
      >;
      expect(typeof details.stack).toBe("string");
      expect(details.error).toBeInstanceOf(Error);
    });
  });

  describe("errors registry round-trip", () => {
    const makeError = () =>
      parseExpectingError(
        z.object({ port: z.number(), name: z.string(), password: z.string() }),
        { port: "8080", name: 42, password: "hunter2" },
        { label: "task config", context: { taskKey: "tk-1" } },
      );

    it("should encode a ParseError with the correct type discriminator", () => {
      expect(errors.encode(makeError()).type).toBe("zod.parse");
    });

    it("should not leak secrets in the encoded payload", () => {
      expect(errors.encode(makeError()).data).not.toContain("hunter2");
    });

    it("should decode a ParseError payload back into a matching instance", () => {
      const decoded = asParseError(errors.decode(errors.encode(makeError())));
      expect(decoded.label).toBe("task config");
      expect(decoded.context).toEqual({ taskKey: "tk-1" });
      expect(decoded.issues).toHaveLength(2);
    });

    it("should round-trip through the registry without leaking secrets", () => {
      const decoded = asParseError(errors.decode(errors.encode(makeError())));
      expect(JSON.stringify(decoded.input)).not.toContain("hunter2");
      expect(JSON.stringify(decoded.issues)).not.toContain("hunter2");
    });
  });
});
