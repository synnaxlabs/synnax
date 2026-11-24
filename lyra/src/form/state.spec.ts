// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep } from "@synnaxlabs/x";
import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Form } from "@/form";
import { createLocalStore } from "@/form/testutil";

const create = <Z extends z.ZodType>(params: Form.UseParams<Z>) => {
  let touched = false;
  const onHasTouched = (value: boolean) => {
    touched = value;
  };
  const { result, unmount } = renderHook(() =>
    Form.use<Z>({ ...params, onHasTouched }),
  );
  return { form: result.current, hasTouched: () => touched, unmount };
};

const basicSchema = z
  .object({
    name: z.string().min(1, "Name is required"),
    email: z.email("Invalid email format"),
    age: z.number().min(0, "Age must be positive").max(120, "Age must be realistic"),
    profile: z.object({
      bio: z.string().optional(),
      website: z.url("Invalid URL").optional(),
    }),
    tags: z.array(z.string()),
    isActive: z.boolean(),
    optionalField: z.string().optional(),
  })
  .check((ctx) => {
    if (ctx.value.name === "admin" && ctx.value.age < 18)
      ctx.issues.push({
        input: ctx.value.name,
        code: "custom",
        message: "Admin users must be 18 or older",
        path: ["name"],
        params: { variant: "warning" },
      });
  });

type BasicValues = z.infer<typeof basicSchema>;

const initialValues: BasicValues = {
  name: "John Doe",
  email: "john@example.com",
  age: 25,
  profile: {
    bio: "Software developer",
    website: "https://johndoe.com",
  },
  tags: ["developer", "typescript"],
  isActive: true,
  optionalField: undefined,
};

const createBasic = (values: BasicValues = initialValues) =>
  create({ values, schema: basicSchema });

const basic = deep.path<BasicValues>();

const verdictSchema = z.object({
  name: z.string(),
  optionalField: z.string().optional(),
  defaulted: z.string().default("cat"),
  extra: z.record(z.string(), z.unknown()),
  nested: z.object({ ssn: z.string() }),
  redline: z.object({ lower: z.number().default(0), upper: z.number() }).optional(),
  color: z
    .object({ r: z.number().default(0), g: z.number().default(0) })
    .default({ r: 1, g: 1 }),
  layout: z
    .object({ margin: z.object({ x: z.number().default(1) }).default({ x: 2 }) })
    .optional(),
});
type VerdictValues = z.input<typeof verdictSchema>;
const verdictValues: VerdictValues = {
  name: "John Doe",
  extra: {},
  nested: { ssn: "1" },
};

const createVerdict = (values: VerdictValues = verdictValues) =>
  create<z.ZodType<VerdictValues>>({ values, schema: verdictSchema });

const verdict = deep.path<VerdictValues>();

const trackedSchema = z.object({
  channels: z.array(z.object({ key: z.string(), port: z.number() })),
  config: z.discriminatedUnion("type", [
    z.object({ type: z.literal("a"), value: z.string().min(1) }),
    z.object({ type: z.literal("b"), value: z.string().optional() }),
  ]),
});
type Tracked = z.infer<typeof trackedSchema>;

const createTrackedStore = () =>
  createLocalStore<Tracked>({
    channels: [
      { key: "a", port: 1 },
      { key: "b", port: 2 },
    ],
    config: { type: "a", value: "v" },
  });

const buildTracked = () => {
  const store = createTrackedStore();
  const { form, unmount } = create({ store, schema: trackedSchema });
  const handleChange = vi.fn();
  form.bind(handleChange);
  return { store, form, handleChange, unmount };
};

const tracked = deep.path<Tracked>();

const loose = deep.path<unknown>();

describe("State", () => {
  describe("constructor", () => {
    it("should initialize with values and schema", () => {
      const { form } = createBasic();
      expect(form.value()).toEqual(initialValues);
      form.set(basic.name, "Jane Doe");
      form.reset();
      expect(form.value()).toEqual(initialValues);
    });

    it("should initialize without schema", () => {
      const { form } = create({ values: initialValues });
      expect(form.value()).toEqual(initialValues);
      form.set(basic.name, "Jane Doe");
      form.reset();
      expect(form.value()).toEqual(initialValues);
    });

    it("should deep copy initial values", () => {
      const { form } = createBasic();
      form.value().name = "Jane Doe";
      form.reset();
      expect(form.value().name).toBe("John Doe");
    });
  });

  describe("setValue", () => {
    describe("materializing an absent ancestor", () => {
      it("should build a defaulted subtree before writing its leaf", () => {
        const { form } = createVerdict();
        form.set(deep.present(verdict.color).g, 5);
        expect(form.value().color).toEqual({ r: 1, g: 5 });
      });

      it("should build an optional subtree only when it can complete it", () => {
        const { form } = createVerdict();
        form.set(deep.present(verdict.redline).upper, 10);
        expect(form.value().redline).toEqual({ upper: 10 });
      });

      it("should build nested defaults below an optional ancestor", () => {
        const { form } = createVerdict();
        form.set(deep.present(deep.present(verdict.layout).margin).x, 9);
        expect(form.value().layout).toEqual({ margin: { x: 9 } });
      });

      it("should leave a present ancestor alone", () => {
        const { form } = createVerdict({ ...verdictValues, color: { r: 3, g: 3 } });
        form.set(deep.present(verdict.color).g, 5);
        expect(form.value().color).toEqual({ r: 3, g: 5 });
      });
    });

    describe("identity", () => {
      interface Keyed {
        channels: { key: string; port: number }[];
        other: { x: number };
      }
      const keyed = (): Keyed => ({
        channels: [
          { key: "a", port: 1 },
          { key: "b", port: 2 },
        ],
        other: { x: 1 },
      });
      const createKeyed = (values: Keyed = keyed()) =>
        create<z.ZodType<Keyed>>({ values });
      const path = deep.path<Keyed>();

      it("should give every ancestor of the written path a new identity", () => {
        const { form } = createBasic();
        const values = form.value();
        const { profile } = values;
        form.set(basic.profile.bio, "Updated bio");
        expect(form.value()).not.toBe(values);
        expect(form.value().profile).not.toBe(profile);
        expect(profile.bio).toBe("Software developer");
      });

      it("should keep the identity of untouched siblings", () => {
        const { form } = createBasic();
        const { tags } = form.value();
        form.set(basic.profile.bio, "Updated bio");
        expect(form.value().tags).toBe(tags);
      });

      it("should copy an array entry reached through its key", () => {
        const { form } = createKeyed();
        const { channels, other } = form.value();
        form.set(path.channels.b.port, 3);
        expect(form.value().channels).not.toBe(channels);
        expect(form.value().channels[1]).not.toBe(channels[1]);
        expect(form.value().channels[1].port).toBe(3);
        expect(form.value().channels[0]).toBe(channels[0]);
        expect(form.value().other).toBe(other);
        expect(channels[1].port).toBe(2);
      });

      it("should give a field's state a new value after a write below it", () => {
        const { form } = createKeyed();
        const before = form.get(path.channels[0]).value;
        form.set(path.channels[0].port, 3);
        const after = form.get(path.channels[0]).value;
        expect(after).not.toBe(before);
        expect(after.port).toBe(3);
        expect(before.port).toBe(1);
      });

      it("should give an entry's state a new value when its key holds a dot", () => {
        const { form } = createKeyed({
          channels: [{ key: "ns=1;s=a.b", port: 1 }],
          other: { x: 1 },
        });
        const entry = path.channels["ns=1;s=a.b"];
        const before = form.get(entry).value;
        form.set(entry.port, 3);
        const after = form.get(entry).value;
        expect(after).not.toBe(before);
        expect(after.port).toBe(3);
      });

      it("should keep the state object of a sibling the write left unchanged", () => {
        const { form } = createBasic();
        const sibling = form.get(basic.email);
        const written = form.get(basic.name);
        form.set(basic.name, "Jane Doe");
        expect(form.get(basic.email)).toBe(sibling);
        expect(form.get(basic.name)).not.toBe(written);
        expect(form.get(basic.name).value).toBe("Jane Doe");
      });

      it("should give the ancestors of a removed path a new identity", () => {
        const { form } = createKeyed();
        const { channels } = form.value();
        form.remove(path.channels[0]);
        expect(form.value().channels).not.toBe(channels);
        expect(form.value().channels).toEqual([{ key: "b", port: 2 }]);
        expect(channels).toHaveLength(2);
      });

      it("should leave the older value unchanged when a shorter key also matches", () => {
        type Nested = Record<string, Record<string, number>>;
        const { form } = create<z.ZodType<Nested>>({
          values: { a: {}, "a.b": { x: 1, y: 2 } },
        });
        const values = form.value();
        form.remove(deep.path<Nested>()["a.b"].x);
        expect(form.value()).toEqual({ a: {}, "a.b": { y: 2 } });
        expect(form.value()["a.b"]).not.toBe(values["a.b"]);
        expect(values).toEqual({ a: {}, "a.b": { x: 1, y: 2 } });
      });
    });

    it("should set a top-level value", () => {
      const { form } = createBasic();
      form.set(basic.name, "Jane Doe");
      expect(form.value().name).toBe("Jane Doe");
    });

    it("should set a nested value", () => {
      const { form } = createBasic();
      form.set(basic.profile.bio, "Updated bio");
      expect(form.value().profile.bio).toBe("Updated bio");
    });

    it("should set array values", () => {
      const { form } = createBasic();
      form.set(basic.tags[0], "engineer");
      expect(form.value().tags[0]).toBe("engineer");
    });

    it("should replace entire object when path is empty", () => {
      const { form } = createBasic();
      const newValues = { ...initialValues, name: "Jane Doe" };
      form.set(basic, newValues);
      expect(form.value()).toEqual(newValues);
    });

    it("should mark field as touched when value changes", () => {
      const { form } = createBasic();
      form.set(basic.name, "Jane Doe");
      expect(form.get(basic.name).touched).toBe(true);
    });

    it("should not mark field as touched when value equals initial", () => {
      const { form } = createBasic();
      form.set(basic.name, "Jane Doe");
      form.set(basic.name, "John Doe");
      expect(form.get(basic.name).touched).toBe(false);
    });

    describe("markTouched option", () => {
      it("should mark field as touched when markTouched is true", () => {
        const { form } = createBasic();
        form.set(basic.name, "Jane Doe", { markTouched: true });
        expect(form.get(basic.name).touched).toBe(true);
      });

      it("should not mark field as touched when markTouched is false", () => {
        const { form } = createBasic();
        form.set(basic.name, "Jane Doe", { markTouched: false });
        expect(form.get(basic.name).touched).toBe(false);
      });

      it("should mark field as touched by default when value changes", () => {
        const { form } = createBasic();
        form.set(basic.name, "Jane Doe");
        expect(form.get(basic.name).touched).toBe(true);
      });

      it("should mark nested fields as touched when markTouched is true", () => {
        const { form } = createBasic();
        form.set(basic.profile.bio, "New bio", { markTouched: true });
        expect(form.get(basic.profile.bio).touched).toBe(true);
      });

      it("should not mark nested fields as touched when markTouched is false", () => {
        const { form } = createBasic();
        form.set(basic.profile.bio, "New bio", { markTouched: false });
        expect(form.get(basic.profile.bio).touched).toBe(false);
      });

      it("should mark array elements as touched when markTouched is true", () => {
        const { form } = createBasic();
        form.set(basic.tags[0], "new-tag", { markTouched: true });
        expect(form.get(basic.tags[0]).touched).toBe(true);
      });

      it("should not mark array elements as touched when markTouched is false", () => {
        const { form } = createBasic();
        form.set(basic.tags[0], "new-tag", { markTouched: false });
        expect(form.get(basic.tags[0]).touched).toBe(false);
      });

      it("should still clear touched when value equals initial regardless of markTouched", () => {
        const { form } = createBasic();
        form.set(basic.name, "Jane Doe", { markTouched: true });
        expect(form.get(basic.name).touched).toBe(true);
        form.set(basic.name, "John Doe", { markTouched: true });
        expect(form.get(basic.name).touched).toBe(false);
      });

      it("should handle markTouched for entire object replacement", () => {
        const { form } = createBasic();
        form.set(basic, { ...initialValues, name: "Jane Doe" }, { markTouched: true });
        expect(form.get(basic).touched).toBe(true);
      });

      it("should respect markTouched when setting undefined values", () => {
        const { form } = createBasic();
        form.set(basic.optionalField, "some value", { markTouched: true });
        expect(form.get(basic.optionalField).touched).toBe(true);
        form.set(basic.optionalField, undefined, { markTouched: false });
        expect(form.get(basic.optionalField).value).toBeUndefined();
      });

      it("should work with validation when markTouched is used", () => {
        const { form } = createBasic();
        form.set(basic.email, "invalid-email", { markTouched: true });
        const fieldState = form.get(basic.email);
        expect(fieldState.touched).toBe(true);
        expect(fieldState.status.variant).toBe("error");
        expect(form.validate()).toBe(false);
      });

      it("should not trigger validation for untouched fields when markTouched is false", () => {
        const { form } = createBasic();
        form.set(basic.email, "invalid-email", { markTouched: false });
        const fieldState = form.get(basic.email);
        expect(fieldState.touched).toBe(false);
        expect(fieldState.status.variant).toBe("success");
      });

      it("should handle complex nested updates with markTouched", () => {
        const { form } = createBasic();
        form.set(
          basic.profile,
          { bio: "Updated bio", website: "https://newsite.com" },
          { markTouched: true },
        );
        expect(form.get(basic.profile).touched).toBe(true);
        expect(form.get(basic.profile.bio).touched).toBe(false);
        expect(form.get(basic.profile.website).touched).toBe(false);
      });

      it("should clear orphaned touched paths even with markTouched option", () => {
        const { form, hasTouched } = createBasic();
        form.set(basic.tags, [...form.value().tags, "new-tag"], { markTouched: true });
        form.set(basic.tags[2], "modified-tag", { markTouched: true });
        expect(form.get(basic.tags[2]).touched).toBe(true);
        form.set(basic.tags, initialValues.tags, { markTouched: false });
        expect(hasTouched()).toBe(false);
      });
    });
  });

  describe("setStatus and clearStatus", () => {
    it("should set status for a field", () => {
      const { form } = createBasic();
      const status = {
        key: "name",
        variant: "error" as const,
        message: "Error message",
      };
      form.setStatus(basic.name, status);
      expect(form.get(basic.name).status).toEqual(status);
    });

    it("should key a status by its path", () => {
      const { form } = createBasic();
      form.setStatus(basic.name, { variant: "error", message: "Taken" });
      expect(form.getStatuses()).toEqual([
        { key: "name", variant: "error", message: "Taken" },
      ]);
    });

    it("should clear status for a field", () => {
      const { form } = createBasic();
      form.setStatus(basic.name, { key: "name", variant: "error", message: "Error" });
      form.clearStatus(basic.name);
      expect(form.get(basic.name).status.variant).toBe("success");
    });

    it("should clear every status when no path is given", () => {
      const { form } = createBasic();
      form.setStatus(basic.name, { key: "name", variant: "error", message: "e" });
      form.setStatus(basic.email, { key: "email", variant: "error", message: "e" });
      form.clearStatuses();
      expect(form.get(basic.name).status.variant).toBe("success");
      expect(form.get(basic.email).status.variant).toBe("success");
    });

    it("should clear only the root status when the root path is given", () => {
      const { form } = createBasic();
      form.setStatus(basic, { key: "", variant: "error", message: "e" });
      form.setStatus(basic.name, { key: "name", variant: "error", message: "e" });
      form.clearStatus(basic);
      expect(form.getStatuses()).toEqual([
        { key: "name", variant: "error", message: "e" },
      ]);
    });

    describe("external statuses", () => {
      const taken = { key: "name", variant: "error" as const, message: "Taken" };

      it("should survive validation after an edit to another field", () => {
        const { form } = createBasic();
        form.setStatus(basic.name, taken);
        form.set(basic.age, 30);
        expect(form.get(basic.name).status).toEqual(taken);
      });

      it("should clear the status when the value at its path changes", () => {
        const { form } = createBasic();
        form.setStatus(basic.name, taken);
        form.set(basic.name, "Jane Doe");
        expect(form.get(basic.name).status.variant).toBe("success");
      });

      it("should keep the status when a write to its parent keeps its value", () => {
        const { form } = createBasic();
        const status = {
          key: "profile.bio",
          variant: "warning" as const,
          message: "Short",
        };
        form.setStatus(basic.profile.bio, status);
        form.set(basic.profile, { ...initialValues.profile, website: "https://x.com" });
        expect(form.get(basic.profile.bio).status).toEqual(status);
      });

      it("should clear the status when its path is removed", () => {
        const { form } = createBasic();
        form.setStatus(basic.tags[1], {
          key: "tags.1",
          variant: "error",
          message: "Bad",
        });
        form.remove(basic.tags[1]);
        expect(form.getStatuses()).toEqual([]);
      });

      it("should survive the removal of a form-level validation error", () => {
        const schema = z
          .object({ a: z.string(), b: z.string() })
          .refine(({ a, b }) => a !== b, "Must differ");
        const { form } = create({ values: { a: "x", b: "x" }, schema });
        const path = deep.path<z.infer<typeof schema>>();
        form.validate();
        expect(form.getStatuses().map(({ message }) => message)).toEqual([
          "Must differ",
        ]);
        form.setStatus(path.a, { key: "a", variant: "error", message: "Taken" });
        form.set(path.b, "y");
        form.validate();
        expect(form.getStatuses().map(({ message }) => message)).toEqual(["Taken"]);
      });

      it("should yield to a status that validation sets at its path", () => {
        const { form } = createBasic({ ...initialValues, email: "invalid" });
        form.setStatus(basic.email, {
          key: "email",
          variant: "error",
          message: "Taken",
        });
        form.validate();
        expect(form.get(basic.email).status.message).toBe("Invalid email format");
        form.set(basic.name, "Jane Doe");
        expect(form.get(basic.email).status.variant).toBe("success");
      });
    });
  });

  describe("touched", () => {
    it("should mark a field touched when a write changes it", () => {
      const { form, hasTouched } = createBasic();
      expect(hasTouched()).toBe(false);
      form.set(basic.name, "Jane Doe");
      expect(form.get(basic.name).touched).toBe(true);
      expect(hasTouched()).toBe(true);
    });

    it("should clear the touched flag of a field a write restores", () => {
      const { form } = createBasic();
      form.set(basic.name, "Jane Doe");
      form.set(basic.email, "jane@example.com");
      form.set(basic.name, "John Doe");
      expect(form.get(basic.name).touched).toBe(false);
      expect(form.get(basic.email).touched).toBe(true);
    });

    it("should report the form untouched when every field is restored", () => {
      const { form, hasTouched } = createBasic();
      form.set(basic.name, "Jane Doe");
      form.set(basic.email, "jane@example.com");
      form.set(basic.name, "John Doe");
      expect(hasTouched()).toBe(true);
      form.set(basic.email, "john@example.com");
      expect(hasTouched()).toBe(false);
    });
  });

  describe("reset", () => {
    it("should reset to initial values", () => {
      const { form } = createBasic();
      form.set(basic.name, "Jane Doe");
      form.setStatus(basic.name, { key: "name", variant: "error", message: "Error" });
      form.reset();
      expect(form.value()).toEqual(initialValues);
      expect(form.get(basic.name).status.variant).toBe("success");
      expect(form.get(basic.name).touched).toBe(false);
    });

    it("should reset to new initial values when provided", () => {
      const { form } = createBasic();
      const newInitialValues = { ...initialValues, name: "Jane Doe" };
      form.reset(newInitialValues);
      expect(form.value()).toEqual(newInitialValues);
      form.set(basic.name, "John Doe");
      form.reset();
      expect(form.value()).toEqual(newInitialValues);
    });

    it("should clear all statuses and touched states on reset", () => {
      const { form, hasTouched } = createBasic();
      form.set(basic.name, "Test");
      form.setStatus(basic.name, { key: "name", variant: "error", message: "Error" });
      form.reset();
      expect(form.get(basic.name).status.variant).toBe("success");
      expect(form.get(basic.name).touched).toBe(false);
      expect(hasTouched()).toBe(false);
    });

    it("should take the given values as initial when they equal the current ones", () => {
      const { form, hasTouched } = createBasic();
      form.set(basic.name, "Jane Doe");
      form.setStatus(basic.name, { key: "name", variant: "error", message: "Error" });
      form.reset({ ...initialValues, name: "Jane Doe" });
      expect(hasTouched()).toBe(false);
      expect(form.get(basic.name).status.variant).toBe("success");
      form.set(basic.name, "John Doe");
      expect(form.get(basic.name).touched).toBe(true);
    });
  });

  describe("setCurrentStateAsInitialValues", () => {
    it("should snapshot current state as initial values", () => {
      const { form } = createBasic();
      form.set(basic.name, "Jane Doe");
      form.setCurrentStateAsInitialValues();
      expect(form.get(basic.name).touched).toBe(false);
      form.set(basic.name, "John Doe");
      form.reset();
      expect(form.get(basic.name).value).toBe("Jane Doe");
    });

    it("should make previously touched fields untouched", () => {
      const { form, hasTouched } = createBasic();
      form.set(basic.name, "Jane Doe");
      form.set(basic.email, "jane@example.com");
      expect(form.get(basic.name).touched).toBe(true);
      expect(form.get(basic.email).touched).toBe(true);
      form.setCurrentStateAsInitialValues();
      expect(form.get(basic.name).touched).toBe(false);
      expect(form.get(basic.email).touched).toBe(false);
      expect(hasTouched()).toBe(false);
    });
  });

  describe("remove", () => {
    it("should remove field value, status, and touched state", () => {
      const { form } = createBasic();
      form.set(basic.name, "Jane Doe");
      form.setStatus(basic.name, { key: "name", variant: "error", message: "Error" });
      form.remove(basic.name);
      expect(form.value().name).toBeUndefined();
      expect(form.get(basic.name, { optional: true })).toBeNull();
    });

    it("should remove nested field", () => {
      const { form } = createBasic();
      form.remove(basic.profile.bio);
      expect(form.value().profile.bio).toBeUndefined();
    });

    it("should remove array element", () => {
      const { form } = createBasic();
      form.remove(basic.tags[0]);
      expect(form.value().tags).toEqual(["typescript"]);
    });
  });

  describe("keyed lists", () => {
    interface Keyed {
      channels: { key: string; port: number }[];
    }
    const createKeyed = (): Keyed => ({
      channels: [
        { key: "a", port: 1 },
        { key: "b", port: 2 },
      ],
    });
    const path = deep.path<Keyed>();
    const keysOf = (values: Keyed): string[] => values.channels.map(({ key }) => key);

    it("should insert an entry after the named entry", () => {
      const { form } = create<z.ZodType<Keyed>>({ values: createKeyed() });
      form.insert(path.channels, { key: "c", port: 3 }, "a");
      expect(keysOf(form.value())).toEqual(["a", "c", "b"]);
      expect(form.get(path.channels).touched).toBe(true);
    });

    it("should refresh a field whose index an insert shifts", () => {
      const { form } = create<z.ZodType<Keyed>>({ values: createKeyed() });
      expect(form.get(path.channels[0].port).value).toBe(1);
      form.insert(path.channels, { key: "c", port: 3 });
      expect(form.get(path.channels[0].port).value).toBe(3);
    });

    it("should move an entry and refresh the fields it shifts", () => {
      const { form } = create<z.ZodType<Keyed>>({ values: createKeyed() });
      expect(form.get(path.channels[0].port).value).toBe(1);
      form.move(path.channels, "b");
      expect(keysOf(form.value())).toEqual(["b", "a"]);
      expect(form.get(path.channels[0].port).value).toBe(2);
    });

    it("should clear the touched flag when a move restores the list", () => {
      const { form } = create<z.ZodType<Keyed>>({ values: createKeyed() });
      form.move(path.channels, "b");
      form.move(path.channels, "a");
      expect(form.get(path.channels).touched).toBe(false);
    });

    it("should mark the list touched when it removes an entry", () => {
      const { form } = create<z.ZodType<Keyed>>({ values: createKeyed() });
      form.remove(path.channels.a);
      expect(keysOf(form.value())).toEqual(["b"]);
      expect(form.get(path.channels).touched).toBe(true);
    });

    it("should clear the touched flag when a remove of a dotted key restores the list", () => {
      const { form } = create<z.ZodType<Keyed>>({ values: createKeyed() });
      form.insert(path.channels, { key: "ns=1;s=a.b", port: 3 });
      form.remove(path.channels["ns=1;s=a.b"]);
      expect(keysOf(form.value())).toEqual(["a", "b"]);
      expect(form.get(path.channels).touched).toBe(false);
    });

    it("should leave the values unchanged for a write to an entry another client removed", () => {
      const store = createLocalStore(createKeyed());
      const { form } = create<z.ZodType<Keyed>>({ store });
      store.remove("channels.a");
      const prev = form.value();
      expect(() => form.set(path.channels.a.port, 5)).not.toThrow();
      expect(form.value()).toBe(prev);
      expect(keysOf(form.value())).toEqual(["b"]);
    });

    it("should do nothing when it removes an entry the list does not hold", () => {
      const { form } = create<z.ZodType<Keyed>>({ values: createKeyed() });
      expect(() => form.remove(path.channels.c)).not.toThrow();
      expect(form.value()).toEqual(createKeyed());
      expect(form.get(path.channels).touched).toBe(false);
    });

    it("should refresh a field of an entry with a dot in its key when another client writes it", () => {
      const store = createLocalStore(createKeyed());
      const { form } = create<z.ZodType<Keyed>>({ store });
      form.insert(path.channels, { key: "ns=1;s=a.b", port: 3 });
      expect(form.get(path.channels["ns=1;s=a.b"].port).value).toBe(3);
      store.set("channels.ns=1;s=a.b.port", 4);
      expect(form.get(path.channels["ns=1;s=a.b"].port).value).toBe(4);
    });

    it("should keep the state of a field whose entry another client did not change", () => {
      const store = createLocalStore(createKeyed());
      const { form } = create<z.ZodType<Keyed>>({ store });
      const before = form.get(path.channels.b.port);
      store.set("channels.a.port", 9);
      expect(form.get(path.channels.b.port)).toBe(before);
      expect(form.get(path.channels.a.port).value).toBe(9);
    });
  });

  describe("batch", () => {
    it("should make every write in the batch one store change", () => {
      const store = createLocalStore(initialValues);
      const { form } = create({ store, schema: basicSchema });
      const handleChange = vi.fn();
      store.onChange(handleChange);
      form.batch(() => {
        form.set(basic.name, "Jane");
        form.set(basic.age, 30);
      });
      expect(handleChange).toHaveBeenCalledTimes(1);
      expect(form.get(basic.name).value).toBe("Jane");
      expect(form.get(basic.age).value).toBe(30);
    });

    it("should read a value that the batch wrote earlier", () => {
      const { form } = createBasic();
      expect(form.get(basic.profile.bio).value).toBe("Software developer");
      let read: unknown;
      form.batch(() => {
        form.set(basic.profile, { ...form.value().profile, bio: "Updated bio" });
        read = form.get(basic.profile.bio).value;
      });
      expect(read).toBe("Updated bio");
    });

    it("should keep a status set after a write in the same batch", () => {
      const { form } = createBasic();
      form.batch(() => {
        form.set(basic.name, "Jane Doe");
        form.setStatus(basic.name, { variant: "error", message: "Name is taken" });
      });
      expect(form.get(basic.name).status.message).toBe("Name is taken");
    });

    it("should drop a status that a write earlier in the batch cleared", () => {
      const { form } = createBasic();
      form.setStatus(basic.name, { variant: "error", message: "Name is taken" });
      let statuses: unknown;
      form.batch(() => {
        form.set(basic.name, "Jane Doe");
        statuses = form.getStatuses();
      });
      expect(statuses).toEqual([]);
    });
  });

  describe("listen", () => {
    it("should refresh a field that another writer changes", () => {
      const { store, form, handleChange } = buildTracked();
      expect(form.get(tracked.channels.b.port).value).toBe(2);
      store.set("channels.b.port", 3);
      expect(form.get(tracked.channels.b.port).value).toBe(3);
      expect(handleChange).toHaveBeenCalledTimes(1);
    });

    it("should clear a status from setStatus when another writer changes its value", () => {
      const { store, form } = buildTracked();
      form.setStatus(tracked.channels.b.port, {
        key: "channels.b.port",
        variant: "error",
        message: "Taken",
      });
      store.set("channels.b.port", 3);
      expect(form.get(tracked.channels.b.port).status.variant).toBe("success");
    });

    it("should keep the state of a field that the change does not reach", () => {
      const { store, form } = buildTracked();
      const prev = form.get(tracked.channels.a.port);
      store.set("channels.b.port", 3);
      expect(form.get(tracked.channels.a.port)).toBe(prev);
    });

    it("should refresh a field whose discriminator another writer changes", () => {
      const { store, form } = buildTracked();
      expect(form.get(tracked.config.value).required).toBe(true);
      store.set("config", { type: "b" });
      expect(form.get(tracked.config.value).required).toBe(false);
    });

    it("should refresh a field that another writer removes", () => {
      const { store, form } = buildTracked();
      expect(form.get(tracked.channels[0].port).value).toBe(1);
      store.remove("channels.a");
      expect(form.get(tracked.channels[0].port).value).toBe(2);
    });

    it("should notify once for its own write", () => {
      const { form, handleChange } = buildTracked();
      form.set(tracked.channels.a.port, 5);
      expect(handleChange).toHaveBeenCalledTimes(1);
    });

    it("should catch up on a change made before it listens", () => {
      const store = createTrackedStore();
      const handleChange = vi.fn();
      let rendered = false;
      const { result } = renderHook(() => {
        const form = Form.use({ store, schema: trackedSchema });
        if (!rendered) {
          rendered = true;
          form.bind(handleChange);
          expect(form.get(tracked.channels.b.port).value).toBe(2);
          store.set("channels.b.port", 3);
        }
        return form;
      });
      expect(handleChange).toHaveBeenCalledTimes(1);
      expect(result.current.get(tracked.channels.b.port).value).toBe(3);
    });

    it("should stop following after the form unmounts", () => {
      const { store, handleChange, unmount } = buildTracked();
      unmount();
      store.set("channels.b.port", 3);
      expect(handleChange).not.toHaveBeenCalled();
    });
  });

  describe("store", () => {
    it("should read another store with its values as the initial ones", () => {
      const store = createTrackedStore();
      const onHasTouched = vi.fn();
      const { result, rerender } = renderHook(
        (props) =>
          Form.use({ store: props.store, schema: trackedSchema, onHasTouched }),
        { initialProps: { store } },
      );
      result.current.set(tracked.channels.a.port, 5);
      const next = createLocalStore<Tracked>({
        channels: [{ key: "c", port: 9 }],
        config: { type: "b" },
      });
      rerender({ store: next });
      const form = result.current;
      expect(form.get(tracked.channels[0].port).value).toBe(9);
      expect(onHasTouched).toHaveBeenLastCalledWith(false);
      form.set(tracked.channels[0].port, 1);
      form.reset();
      expect(form.value()).toEqual(next.get());
      expect(form.value()).toEqual({
        channels: [{ key: "c", port: 9 }],
        config: { type: "b" },
      });
    });

    it("should keep its state for the store it already reads", () => {
      const store = createTrackedStore();
      const onHasTouched = vi.fn();
      const { result, rerender } = renderHook(
        (props) =>
          Form.use({ store: props.store, schema: trackedSchema, onHasTouched }),
        { initialProps: { store } },
      );
      result.current.set(tracked.channels.a.port, 5);
      rerender({ store });
      expect(result.current.get(tracked.channels.a.port).touched).toBe(true);
      expect(onHasTouched).toHaveBeenCalledExactlyOnceWith(true);
    });
  });

  describe("validate", () => {
    it("should return true for valid data", () => {
      const { form } = createBasic();
      expect(form.validate()).toBe(true);
    });

    it("should not flag an untouched invalid field when another field changes", () => {
      const { form } = createBasic({ ...initialValues, email: "invalid-email" });
      form.set(basic.name, "Jane Doe");
      expect(form.get(basic.email).status.variant).toBe("success");
      expect(form.getStatuses()).toEqual([]);
    });

    it("should set error status for invalid fields", () => {
      const { form } = createBasic();
      form.set(basic.email, "invalid-email");
      const fieldState = form.get(basic.email);
      expect(fieldState.status.variant).toBe("error");
      expect(fieldState.status.message).toBe("Invalid email format");
    });

    it("should return true for warnings (non-error variants)", () => {
      const { form } = createBasic({ ...initialValues, name: "admin", age: 16 });
      expect(form.validate()).toBe(true);
      expect(form.get(basic.name).status.variant).toBe("warning");
    });

    it("should clear previous validation errors on successful validation", () => {
      const { form } = createBasic();
      form.set(basic.email, "invalid");
      expect(form.get(basic.email).status.variant).toBe("error");
      form.set(basic.email, "valid@example.com");
      expect(form.get(basic.email).status.variant).toBe("success");
    });

    it("should keep the state object of a field whose error did not change", () => {
      const { form } = createBasic();
      form.set(basic.email, "invalid");
      const before = form.get(basic.email);
      form.set(basic.name, "Jane Doe");
      expect(form.get(basic.email)).toBe(before);
    });

    it("should give a field a new state object when its error changes", () => {
      const { form } = createBasic();
      form.set(basic.age, -1);
      const before = form.get(basic.age);
      expect(before.status.message).toBe("Age must be positive");
      form.set(basic.age, 200);
      expect(form.get(basic.age)).not.toBe(before);
      expect(form.get(basic.age).status.message).toBe("Age must be realistic");
    });

    it("should return true when no schema is provided", () => {
      const { form } = create({ values: initialValues });
      expect(form.validate()).toBe(true);
    });
  });

  describe("validateAsync", () => {
    it("should return true for valid data", async () => {
      const { form } = createBasic();
      expect(await form.validateAsync()).toBe(true);
    });

    it("should return false for invalid data", async () => {
      const { form } = createBasic({ ...initialValues, email: "invalid-email" });
      expect(await form.validateAsync()).toBe(false);
    });

    it("should set error status for invalid fields", async () => {
      const { form } = createBasic({ ...initialValues, email: "invalid-email" });
      await form.validateAsync();
      const fieldState = form.get(basic.email);
      expect(fieldState.status.variant).toBe("error");
      expect(fieldState.status.message).toBe("Invalid email format");
    });

    it("should return true when no schema is provided", async () => {
      const { form } = create({ values: initialValues });
      expect(await form.validateAsync()).toBe(true);
    });
  });

  describe("getState", () => {
    it("should return field state for existing field", () => {
      const { form } = createBasic();
      const fieldState = form.get(basic.name);
      expect(fieldState.value).toBe("John Doe");
      expect(fieldState.status.variant).toBe("success");
      expect(fieldState.touched).toBe(false);
      expect(fieldState.required).toBe(true);
    });

    it("should return field state for nested field", () => {
      const { form } = createBasic();
      expect(form.get(basic.profile.bio).value).toBe("Software developer");
    });

    it("should return field state for array element", () => {
      const { form } = createBasic();
      expect(form.get(basic.tags[0]).value).toBe("developer");
    });

    it("should throw error for non-existent required field", () => {
      const { form } = createBasic();
      expect(() => form.get(loose.nonExistent)).toThrow();
    });

    it("should return null for non-existent optional field", () => {
      const { form } = createBasic();
      expect(form.get(loose.nonExistent, { optional: true })).toBeNull();
    });

    describe("schema verdict on an absent value", () => {
      it("should show the schema default without writing it", () => {
        const { form } = createVerdict();
        const fieldState = form.get(verdict.defaulted);
        expect(fieldState.value).toBe("cat");
        expect(fieldState.required).toBe(false);
        expect("defaulted" in form.value()).toBe(false);
      });

      it("should not override an existing value with the default", () => {
        const { form } = createVerdict({ ...verdictValues, defaulted: "dog" });
        expect(form.get(verdict.defaulted).value).toBe("dog");
      });

      it("should read an optional field as undefined", () => {
        const { form } = createVerdict();
        const fieldState = form.get(verdict.optionalField);
        expect(fieldState.value).toBeUndefined();
        expect(fieldState.required).toBe(false);
      });

      it("should throw for a required field", () => {
        const { form } = createVerdict();
        form.remove(verdict.name);
        expect(() => form.get(verdict.name)).toThrow(
          "Field name is required and holds no value",
        );
        expect(form.get(verdict.name, { optional: true })).toBeNull();
      });

      it("should throw for a path the schema does not contain", () => {
        const { form } = createVerdict();
        expect(() => form.get(loose.nested.typo)).toThrow(
          "Field nested.typo is not in the form",
        );
      });

      it("should treat a path under a record of unknowns as optional", () => {
        const { form } = createVerdict();
        const fieldState = form.get(verdict.extra.anything.deep);
        expect(fieldState.value).toBeUndefined();
        expect(fieldState.required).toBe(false);
      });

      it("should read a leaf of a defaulted subtree from the subtree default", () => {
        const { form } = createVerdict();
        expect(form.get(deep.present(verdict.color).r).value).toBe(1);
      });

      it("should read a leaf below an optional ancestor from nested defaults", () => {
        const { form } = createVerdict();
        const margin = deep.present(deep.present(verdict.layout).margin);
        expect(form.get(margin.x).value).toBe(2);
      });
    });

    it("should correctly determine required status from schema", () => {
      const { form } = createBasic();
      expect(form.get(basic.name).required).toBe(true);
      expect(form.get(basic.optionalField).required).toBe(false);
    });

    describe("required", () => {
      const schema = z.object({
        free: z.string(),
        count: z.number(),
        on: z.boolean(),
        tags: z.array(z.string()).min(1),
        open: z.array(z.string()),
      });
      const values = { free: "a", count: 1, on: true, tags: ["t"], open: [] };
      const path = deep.path<z.infer<typeof schema>>();
      const createRequired = () => create({ values, schema }).form;

      it("should not require a string the schema accepts empty", () => {
        expect(createRequired().get(path.free).required).toBe(false);
      });

      it("should not require a value no input can empty", () => {
        const form = createRequired();
        expect(form.get(path.count).required).toBe(false);
        expect(form.get(path.on).required).toBe(false);
      });

      it("should require a list the schema refuses empty", () => {
        expect(createRequired().get(path.tags).required).toBe(true);
      });

      it("should not require a list the schema accepts empty", () => {
        expect(createRequired().get(path.open).required).toBe(false);
      });

      it("should not require a field whose schema checks asynchronously", () => {
        const asyncSchema = z.object({
          name: z.string().refine(async (v) => v !== "taken"),
        });
        const { form } = create({ values: { name: "a" }, schema: asyncSchema });
        const asyncPath = deep.path<z.infer<typeof asyncSchema>>();
        expect(form.get(asyncPath.name).required).toBe(false);
      });
    });

    it("should return cached reference for performance", () => {
      const { form } = createBasic();
      expect(form.get(basic.name)).toBe(form.get(basic.name));
    });

    describe("memoization", () => {
      const build = () => {
        let checks = 0;
        const schema = z.object({
          tracked: z.string().check(() => {
            checks++;
          }),
          other: z.string(),
          config: z.discriminatedUnion("type", [
            z.object({ type: z.literal("a"), value: z.string().min(1) }),
            z.object({ type: z.literal("b"), value: z.string().optional() }),
          ]),
          nested: z.object({ leaf: z.string() }),
        });
        const values: z.infer<typeof schema> = {
          tracked: "a",
          other: "x",
          config: { type: "a", value: "v" },
          nested: { leaf: "l" },
        };
        return {
          form: create({ values, schema }).form,
          path: deep.path<z.infer<typeof schema>>(),
          checks: () => checks,
        };
      };

      it("should not consult the schema again for an unchanged field", () => {
        const { form, path, checks } = build();
        form.get(path.tracked);
        form.get(path.tracked);
        expect(checks()).toBe(1);
      });

      it("should consult the schema again after the field changes", () => {
        const { form, path, checks } = build();
        form.get(path.tracked);
        form.set(path.tracked, "b");
        const before = checks();
        expect(form.get(path.tracked).value).toBe("b");
        expect(checks()).toBe(before + 1);
      });

      it("should refresh a field when a sibling discriminator changes", () => {
        const { form, path } = build();
        expect(form.get(path.config.value).required).toBe(true);
        form.set(path.config.type, "b");
        expect(form.get(path.config.value).required).toBe(false);
      });

      it("should refresh a child when its parent is replaced", () => {
        const { form, path } = build();
        form.get(path.nested.leaf);
        form.set(path.nested, { leaf: "m" });
        expect(form.get(path.nested.leaf).value).toBe("m");
      });

      describe("a change elsewhere in the form", () => {
        interface Values {
          name: string;
          config: { tracked: string };
        }
        type Return = Form.UseReturn<z.ZodType<Values>>;
        const path = deep.path<Values>();
        const buildNested = () => {
          let checks = 0;
          const counted = z.object({
            name: z.string(),
            config: z.object({
              tracked: z.string().check(() => {
                checks++;
              }),
            }),
          });
          const { form } = create<z.ZodType<Values>>({
            values: { name: "a", config: { tracked: "a" } },
            schema: counted,
          });
          form.get(path.config.tracked);
          return { form, checks: () => checks };
        };

        it.each([
          ["a write", (f: Return) => f.set(path.name, "b")],
          [
            "a write that restores the initial value",
            (f: Return) => {
              f.set(path.name, "b");
              f.set(path.name, "a");
            },
          ],
          [
            "setStatus",
            (f: Return) =>
              f.setStatus(path.name, { key: "name", variant: "error", message: "e" }),
          ],
          ["clearStatus", (f: Return) => f.clearStatus(path.name)],
        ])("should not consult the schema for a field after %s", (_, change) => {
          const { form, checks } = buildNested();
          change(form);
          const before = checks();
          form.get(path.config.tracked);
          expect(checks()).toBe(before);
        });

        it("should consult the schema again after clearing every touched flag", () => {
          const { form, checks } = buildNested();
          form.set(path.name, "b");
          form.setCurrentStateAsInitialValues();
          const before = checks();
          form.get(path.config.tracked);
          expect(checks()).toBe(before + 1);
        });
      });

      it("should refresh every field on reset", () => {
        const { form, path } = build();
        form.set(path.other, "y");
        form.setStatus(path.tracked, {
          key: "tracked",
          variant: "error",
          message: "e",
        });
        form.reset();
        expect(form.get(path.other).touched).toBe(false);
        expect(form.get(path.tracked).status.variant).toBe("success");
      });
    });

    it("should update cached reference when field changes", () => {
      const { form } = createBasic();
      const fieldState1 = form.get(basic.name);
      form.set(basic.name, "Jane Doe");
      const fieldState2 = form.get(basic.name);
      expect(fieldState1).not.toBe(fieldState2);
      expect(fieldState2.value).toBe("Jane Doe");
    });
  });

  describe("observer functionality", () => {
    it("should notify observers when the form changes", () => {
      const { form } = createBasic({ ...initialValues, email: "invalid" });
      const observer = vi.fn();
      form.bind(observer);
      form.validate();
      expect(observer).toHaveBeenCalled();
    });
  });

  describe("path validation", () => {
    it("should validate only specified path and its children", () => {
      const { form } = createBasic({
        ...initialValues,
        name: "",
        email: "invalid-email",
        profile: { ...initialValues.profile, website: "invalid-url" },
      });
      expect(form.validate(basic.profile)).toBe(false);
      expect(form.get(basic.profile.website).status.variant).toBe("error");
      expect(form.get(basic.name).status.variant).toBe("success");
      expect(form.get(basic.email).status.variant).toBe("success");
    });

    it("should validate specific field path", () => {
      const { form } = createBasic({
        ...initialValues,
        name: "",
        email: "invalid-email",
      });
      expect(form.validate(basic.email)).toBe(false);
      expect(form.get(basic.email).status.variant).toBe("error");
      expect(form.get(basic.name).status.variant).toBe("success");
    });

    it("should validate parent path and include child errors", () => {
      const { form } = createBasic({
        ...initialValues,
        name: "",
        profile: { ...initialValues.profile, website: "invalid-url" },
      });
      expect(form.validate(basic.profile)).toBe(false);
      expect(form.get(basic.profile.website).status.variant).toBe("error");
      expect(form.get(basic.name).status.variant).toBe("success");
    });

    it("should validate child path when parent is specified", () => {
      const { form } = createBasic({
        ...initialValues,
        email: "invalid-email",
        profile: { ...initialValues.profile, website: "invalid-url" },
      });
      expect(form.validate(basic.profile.website)).toBe(false);
      expect(form.get(basic.profile.website).status.variant).toBe("error");
      expect(form.get(basic.email).status.variant).toBe("success");
    });

    it("should return true when validating path with no errors", () => {
      const { form } = createBasic({ ...initialValues, email: "invalid-email" });
      expect(form.validate(basic.name)).toBe(true);
      expect(form.get(basic.name).status.variant).toBe("success");
      expect(form.get(basic.email).status.variant).toBe("success");
    });

    it("should handle array paths correctly", () => {
      const arraySchema = z.object({
        ...basicSchema.shape,
        tags: z.array(z.string().min(1, "Tag cannot be empty")),
      });
      const { form } = create({
        values: { ...initialValues, tags: ["", "valid-tag"] },
        schema: arraySchema,
      });
      expect(form.validate(basic.tags[0])).toBe(false);
      expect(form.get(basic.tags[0]).status.variant).toBe("error");
    });
  });

  describe("async path validation", () => {
    it("should validate only specified path asynchronously", async () => {
      const { form } = createBasic({
        ...initialValues,
        name: "",
        email: "invalid-email",
      });
      expect(await form.validateAsync(basic.email)).toBe(false);
      expect(form.get(basic.email).status.variant).toBe("error");
      expect(form.get(basic.name).status.variant).toBe("success");
    });

    it("should validate nested paths asynchronously", async () => {
      const { form } = createBasic({
        ...initialValues,
        name: "",
        profile: { ...initialValues.profile, website: "invalid-url" },
      });
      expect(await form.validateAsync(basic.profile)).toBe(false);
      expect(form.get(basic.profile.website).status.variant).toBe("error");
      expect(form.get(basic.name).status.variant).toBe("success");
    });
  });

  describe("union validation", () => {
    const schema = z.object({
      a: z.union([
        z.object({
          type: z.literal("a"),
          color: z.string().min(1, "Color is required"),
        }),
        z.object({
          type: z.literal("b"),
          height: z.number(),
        }),
      ]),
    });
    const path = deep.path<z.infer<typeof schema>>();
    const color = deep.variant(path.a, "type", "a").color;

    it("should return success when all fields are valid", () => {
      const { form } = create({ values: { a: { type: "a", color: "" } }, schema });
      form.set(color, "red");
      expect(form.get(path.a).status.variant).toBe("success");
      expect(form.get(color).status.variant).toBe("success");
    });

    it("should return an error for the most favored union field", () => {
      const { form } = create({ values: { a: { type: "a", color: "red" } }, schema });
      form.set(color, "");
      expect(form.get(color).status.variant).toBe("error");
    });
  });

  describe("field array touched tracking", () => {
    it("should reset touched status when field array item is removed after modification", () => {
      const arraySchema = z.object({
        items: z.array(z.object({ name: z.string(), value: z.number() })),
      });
      const path = deep.path<z.infer<typeof arraySchema>>();
      const { form, hasTouched } = create({
        values: {
          items: [
            { name: "item1", value: 10 },
            { name: "item2", value: 20 },
          ],
        },
        schema: arraySchema,
      });
      expect(hasTouched()).toBe(false);
      form.set(path.items, [...form.value().items, { name: "item3", value: 30 }]);
      expect(hasTouched()).toBe(true);
      form.set(path.items[2].name, "modified_item3");
      expect(hasTouched()).toBe(true);
      expect(form.get(path.items[2].name).touched).toBe(true);
      form.set(path.items, form.value().items.slice(0, 2));
      expect(hasTouched()).toBe(false);
    });

    it("should maintain touched status for other modified fields when array item is removed", () => {
      const arraySchema = z.object({
        name: z.string(),
        items: z.array(z.object({ title: z.string() })),
      });
      const path = deep.path<z.infer<typeof arraySchema>>();
      const { form, hasTouched } = create({
        values: { name: "test", items: [{ title: "original" }] },
        schema: arraySchema,
      });
      form.set(path.name, "modified");
      expect(hasTouched()).toBe(true);
      form.set(path.items, [...form.value().items, { title: "new item" }]);
      form.set(path.items[1].title, "modified new item");
      form.set(path.items, [{ title: "original" }]);
      expect(hasTouched()).toBe(true);
      expect(form.get(path.name).touched).toBe(true);
    });

    it("should handle nested array operations correctly", () => {
      const nestedArraySchema = z.object({
        groups: z.array(
          z.object({
            name: z.string(),
            items: z.array(z.object({ value: z.string() })),
          }),
        ),
      });
      const path = deep.path<z.infer<typeof nestedArraySchema>>();
      const { form, hasTouched } = create({
        values: { groups: [{ name: "group1", items: [{ value: "item1" }] }] },
        schema: nestedArraySchema,
      });
      form.set(path.groups, [
        ...form.value().groups,
        { name: "group2", items: [{ value: "item2" }] },
      ]);
      form.set(path.groups[1].items[0].value, "modified_item2");
      expect(hasTouched()).toBe(true);
      form.set(path.groups, [{ name: "group1", items: [{ value: "item1" }] }]);
      expect(hasTouched()).toBe(false);
    });
  });

  describe("edge cases", () => {
    it("should handle empty string paths correctly", () => {
      const { form } = createBasic();
      const newValues = { ...initialValues, name: "Jane Doe" };
      form.set(basic, newValues);
      expect(form.value()).toEqual(newValues);
    });

    it("should handle deeply nested paths", () => {
      const deepSchema = z.object({
        level1: z.object({
          level2: z.object({ level3: z.object({ value: z.string() }) }),
        }),
      });
      const path = deep.path<z.infer<typeof deepSchema>>();
      const { form } = create({
        values: { level1: { level2: { level3: { value: "deep" } } } },
        schema: deepSchema,
      });
      form.set(path.level1.level2.level3.value, "updated");
      expect(form.value().level1.level2.level3.value).toBe("updated");
    });

    it("should handle array mutations correctly", () => {
      const { form } = createBasic();
      form.set(basic.tags, ["new", "tags"]);
      expect(form.value().tags).toEqual(["new", "tags"]);
    });

    it("should handle null and undefined values", () => {
      const schema = z.object({ test: z.string().nullable() });
      const { form } = create({ values: { test: null }, schema });
      const path = deep.path<z.infer<typeof schema>>();
      expect(form.get(path.test, { optional: true })).toBeNull();
    });

    it("should handle schema without required fields", () => {
      const schema = z.object({ optional: z.string().optional() });
      const { form } = create({ values: { optional: undefined }, schema });
      const path = deep.path<z.infer<typeof schema>>();
      expect(form.get(path.optional).value).toBeUndefined();
    });
  });
});
