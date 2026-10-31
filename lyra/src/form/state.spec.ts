// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Form } from "@/form";

const create = <Z extends z.ZodType>(values: z.infer<Z>, schema?: Z) => {
  const touched = { current: false };
  const onHasTouched = (value: boolean) => {
    touched.current = value;
  };
  const { result } = renderHook(() => Form.use<Z>({ values, schema, onHasTouched }));
  return Object.assign(result.current, { touched });
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

const initialValues = {
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
type VerdictValues = z.infer<typeof verdictSchema>;
const verdictValues = {
  name: "John Doe",
  extra: {},
  nested: { ssn: "1" },
} as VerdictValues;

describe("use", () => {
  describe("constructor", () => {
    it("should initialize with values and schema", () => {
      const state = create(initialValues, basicSchema);
      expect(state.value()).toEqual(initialValues);
    });

    it("should initialize without schema", () => {
      const state = create(initialValues);
      expect(state.value()).toEqual(initialValues);
    });

    it("should keep the initial values when the state changes", () => {
      const values = { ...initialValues, profile: { ...initialValues.profile } };
      const state = create(values, basicSchema);
      state.set("profile.bio", "Updated bio");
      expect(values.profile.bio).toBe("Software developer");
    });
  });

  describe("setValue", () => {
    describe("materializing an absent ancestor", () => {
      it("should build a defaulted subtree before writing its leaf", () => {
        const state = create(verdictValues, verdictSchema);
        state.set("color.g", 5);
        expect(state.value().color).toEqual({ r: 1, g: 5 });
      });

      it("should build an optional subtree only when it can complete it", () => {
        const state = create(verdictValues, verdictSchema);
        state.set("redline.upper", 10);
        expect(state.value().redline).toEqual({ upper: 10 });
      });

      it("should build nested defaults below an optional ancestor", () => {
        const state = create(verdictValues, verdictSchema);
        state.set("layout.margin.x", 9);
        expect(state.value().layout).toEqual({ margin: { x: 9 } });
      });

      it("should leave a present ancestor alone", () => {
        const state = create(
          { ...verdictValues, color: { r: 3, g: 3 } },
          verdictSchema,
        );
        state.set("color.g", 5);
        expect(state.value().color).toEqual({ r: 3, g: 5 });
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

      it("should give every ancestor of the written path a new identity", () => {
        const state = create(initialValues, basicSchema);
        const values = state.value();
        const { profile } = values;
        state.set("profile.bio", "Updated bio");
        expect(state.value()).not.toBe(values);
        expect(state.value().profile).not.toBe(profile);
        expect(profile.bio).toBe("Software developer");
      });

      it("should keep the identity of untouched siblings", () => {
        const state = create(initialValues, basicSchema);
        const { tags } = state.value();
        state.set("profile.bio", "Updated bio");
        expect(state.value().tags).toBe(tags);
      });

      it("should copy an array entry reached through its key", () => {
        const state = create<z.ZodType<Keyed>>(keyed());
        const { channels, other } = state.value();
        state.set("channels.b.port", 3);
        expect(state.value().channels).not.toBe(channels);
        expect(state.value().channels[1]).not.toBe(channels[1]);
        expect(state.value().channels[1].port).toBe(3);
        expect(state.value().channels[0]).toBe(channels[0]);
        expect(state.value().other).toBe(other);
        expect(channels[1].port).toBe(2);
      });

      it("should give a field's state a new value after a write below it", () => {
        const state = create<z.ZodType<Keyed>>(keyed());
        const before = state.get<{ port: number }>("channels.0").value;
        state.set("channels.0.port", 3);
        const after = state.get<{ port: number }>("channels.0").value;
        expect(after).not.toBe(before);
        expect(after.port).toBe(3);
        expect(before.port).toBe(1);
      });

      it("should keep the state object of a sibling the write left unchanged", () => {
        const state = create(initialValues, basicSchema);
        const sibling = state.get("email");
        const written = state.get("name");
        state.set("name", "Jane Doe");
        expect(state.get("email")).toBe(sibling);
        expect(state.get("name")).not.toBe(written);
        expect(state.get("name").value).toBe("Jane Doe");
      });

      it("should give the ancestors of a removed path a new identity", () => {
        const state = create<z.ZodType<Keyed>>(keyed());
        const { channels } = state.value();
        state.remove("channels.0");
        expect(state.value().channels).not.toBe(channels);
        expect(state.value().channels).toEqual([{ key: "b", port: 2 }]);
        expect(channels).toHaveLength(2);
      });

      it("should leave the older value unchanged when a shorter key also matches", () => {
        const state = create<z.ZodType<Record<string, object>>>({
          a: {},
          "a.b": { x: 1, y: 2 },
        });
        const values = state.value();
        state.remove("a.b.x");
        expect(state.value()).toEqual({ a: {}, "a.b": { y: 2 } });
        expect(state.value()["a.b"]).not.toBe(values["a.b"]);
        expect(values).toEqual({ a: {}, "a.b": { x: 1, y: 2 } });
      });
    });

    it("should set a top-level value", () => {
      const state = create(initialValues, basicSchema);
      state.set("name", "Jane Doe");
      expect(state.value().name).toBe("Jane Doe");
    });

    it("should set a nested value", () => {
      const state = create(initialValues, basicSchema);
      state.set("profile.bio", "Updated bio");
      expect(state.value().profile.bio).toBe("Updated bio");
    });

    it("should set array values", () => {
      const state = create(initialValues, basicSchema);
      state.set("tags.0", "engineer");
      expect(state.value().tags[0]).toBe("engineer");
    });

    it("should replace entire object when path is empty", () => {
      const state = create(initialValues, basicSchema);
      const newValues = { ...initialValues, name: "Jane Doe" };
      state.set("", newValues);
      expect(state.value()).toEqual(newValues);
    });

    it("should mark field as touched when value changes", () => {
      const state = create(initialValues, basicSchema);
      state.set("name", "Jane Doe");
      const fieldState = state.get("name");
      expect(fieldState.touched).toBe(true);
    });

    it("should not mark field as touched when value equals initial", () => {
      const state = create(initialValues, basicSchema);
      state.set("name", "Jane Doe");
      state.set("name", "John Doe"); // Back to initial
      const fieldState = state.get("name");
      expect(fieldState.touched).toBe(false);
    });

    describe("markTouched option", () => {
      it("should mark field as touched when markTouched is true", () => {
        const state = create(initialValues, basicSchema);
        state.set("name", "Jane Doe", { markTouched: true });
        const fieldState = state.get("name");
        expect(fieldState.touched).toBe(true);
      });

      it("should not mark field as touched when markTouched is false", () => {
        const state = create(initialValues, basicSchema);
        state.set("name", "Jane Doe", { markTouched: false });
        const fieldState = state.get("name");
        expect(fieldState.touched).toBe(false);
      });

      it("should mark field as touched by default when value changes", () => {
        const state = create(initialValues, basicSchema);
        state.set("name", "Jane Doe");
        const fieldState = state.get("name");
        expect(fieldState.touched).toBe(true);
      });

      it("should mark nested fields as touched when markTouched is true", () => {
        const state = create(initialValues, basicSchema);
        state.set("profile.bio", "New bio", { markTouched: true });
        const fieldState = state.get("profile.bio");
        expect(fieldState.touched).toBe(true);
      });

      it("should not mark nested fields as touched when markTouched is false", () => {
        const state = create(initialValues, basicSchema);
        state.set("profile.bio", "New bio", { markTouched: false });
        const fieldState = state.get("profile.bio");
        expect(fieldState.touched).toBe(false);
      });

      it("should mark array elements as touched when markTouched is true", () => {
        const state = create(initialValues, basicSchema);
        state.set("tags.0", "new-tag", { markTouched: true });
        const fieldState = state.get("tags.0");
        expect(fieldState.touched).toBe(true);
      });

      it("should not mark array elements as touched when markTouched is false", () => {
        const state = create(initialValues, basicSchema);
        state.set("tags.0", "new-tag", { markTouched: false });
        const fieldState = state.get("tags.0");
        expect(fieldState.touched).toBe(false);
      });

      it("should still clear touched when value equals initial regardless of markTouched", () => {
        const state = create(initialValues, basicSchema);
        state.set("name", "Jane Doe", { markTouched: true });
        expect(state.get("name").touched).toBe(true);

        // Setting back to initial should clear touched even with markTouched: true
        state.set("name", "John Doe", { markTouched: true });
        expect(state.get("name").touched).toBe(false);
      });

      it("should handle markTouched for entire object replacement", () => {
        const state = create(initialValues, basicSchema);
        const newValues = { ...initialValues, name: "Jane Doe" };
        state.set("", newValues, { markTouched: true });
        const fieldState = state.get("");
        expect(fieldState.touched).toBe(true);
      });

      it("should respect markTouched when setting undefined values", () => {
        const state = create(initialValues, basicSchema);
        state.set("optionalField", "some value", { markTouched: true });
        expect(state.get("optionalField").touched).toBe(true);

        state.set("optionalField", undefined, { markTouched: false });
        expect(state.get("optionalField").value).toBeUndefined();
      });

      it("should work with validation when markTouched is used", () => {
        const state = create(initialValues, basicSchema);
        state.set("email", "invalid-email", { markTouched: true });

        const isValid = state.validate();
        expect(isValid).toBe(false);

        const fieldState = state.get("email");
        expect(fieldState.touched).toBe(true);
        expect(fieldState.status.variant).toBe("error");
      });

      it("should not trigger validation for untouched fields when markTouched is false", () => {
        const state = create(initialValues, basicSchema);
        state.set("email", "invalid-email", { markTouched: false });

        const fieldState = state.get("email");
        expect(fieldState.touched).toBe(false);
        expect(fieldState.status.variant).toBe("success");
      });

      it("should handle complex nested updates with markTouched", () => {
        const state = create(initialValues, basicSchema);
        const newProfile = {
          bio: "Updated bio",
          website: "https://newsite.com",
        };
        state.set("profile", newProfile, { markTouched: true });

        expect(state.get("profile").touched).toBe(true);
        expect(state.get("profile.bio").touched).toBe(false);
        expect(state.get("profile.website").touched).toBe(false);
      });

      it("should clear orphaned touched paths even with markTouched option", () => {
        const state = create(initialValues, basicSchema);

        state.set("tags", [...state.value().tags, "new-tag"], {
          markTouched: true,
        });
        state.set("tags.2", "modified-tag", { markTouched: true });
        expect(state.get("tags.2").touched).toBe(true);

        // Remove the item - should clear the orphaned touched path
        state.set("tags", initialValues.tags, { markTouched: false });
        expect(state.touched.current).toBe(false);
      });
    });
  });

  describe("setStatus and clearStatuses", () => {
    it("should set status for a field", () => {
      const state = create(initialValues, basicSchema);
      const status = {
        key: "name",
        variant: "error" as const,
        message: "Error message",
      };
      state.setStatus("name", status);
      const fieldState = state.get("name");
      expect(fieldState.status).toEqual(status);
    });

    it("should key a status by its path", () => {
      const state = create(initialValues, basicSchema);
      state.setStatus("name", { variant: "error", message: "Taken" });
      expect(state.getStatuses()).toEqual([
        { key: "name", variant: "error", message: "Taken" },
      ]);
    });

    it("should clear every status when no path is given", () => {
      const state = create(initialValues, basicSchema);
      state.setStatus("name", { key: "name", variant: "error", message: "Taken" });
      state.setStatus("age", { key: "age", variant: "warning", message: "Old" });
      state.clearStatuses();
      expect(state.getStatuses()).toEqual([]);
    });

    describe("external statuses", () => {
      const taken = { key: "name", variant: "error" as const, message: "Taken" };

      it("should survive validation after an edit to another field", () => {
        const state = create(initialValues, basicSchema);
        state.setStatus("name", taken);
        state.set("age", 30);
        state.validate();
        expect(state.get("name").status).toEqual(taken);
      });

      it("should clear the status when the value at its path changes", () => {
        const state = create(initialValues, basicSchema);
        state.setStatus("name", taken);
        state.set("name", "Jane Doe");
        expect(state.get("name").status.variant).toBe("success");
      });

      it("should keep the status when a write to its parent keeps its value", () => {
        const state = create(initialValues, basicSchema);
        const status = {
          key: "profile.bio",
          variant: "warning" as const,
          message: "Short",
        };
        state.setStatus("profile.bio", status);
        state.set("profile", {
          ...initialValues.profile,
          website: "https://x.com",
        });
        expect(state.get("profile.bio").status).toEqual(status);
      });

      it("should clear the status when its path is removed", () => {
        const state = create(initialValues, basicSchema);
        state.setStatus("tags.1", { key: "tags.1", variant: "error", message: "Bad" });
        state.remove("tags.1");
        expect(state.getStatuses()).toEqual([]);
      });

      it("should survive the removal of a form-level validation error", () => {
        const schema = z
          .object({ a: z.string(), b: z.string() })
          .refine(({ a, b }) => a !== b, "Must differ");
        const state = create({ a: "x", b: "x" }, schema);
        state.validate();
        expect(state.getStatuses().map(({ message }) => message)).toEqual([
          "Must differ",
        ]);
        state.setStatus("a", { key: "a", variant: "error", message: "Taken" });
        state.set("b", "y");
        state.validate();
        expect(state.getStatuses().map(({ message }) => message)).toEqual(["Taken"]);
      });

      it("should yield to a status that validation sets at its path", () => {
        const state = create({ ...initialValues, email: "invalid" }, basicSchema);
        state.setStatus("email", { key: "email", variant: "error", message: "Taken" });
        state.validate();
        expect(state.get("email").status.message).toBe("Invalid email format");
        state.set("name", "Jane Doe");
        expect(state.get("email").status.variant).toBe("success");
      });
    });
  });

  describe("reset", () => {
    it("should reset to initial values", () => {
      const state = create(initialValues, basicSchema);
      state.set("name", "Jane Doe");
      state.setStatus("name", { key: "name", variant: "error", message: "Error" });

      state.reset();

      expect(state.value()).toEqual(initialValues);
      expect(state.get("name").status.variant).toBe("success");
      expect(state.get("name").touched).toBe(false);
    });

    it("should reset to new initial values when provided", () => {
      const state = create(initialValues, basicSchema);
      const newInitialValues = { ...initialValues, name: "Jane Doe" };

      state.reset(newInitialValues);

      expect(state.value()).toEqual(newInitialValues);
      state.set("name", "John Doe");
      state.reset();
      expect(state.value()).toEqual(newInitialValues);
    });

    it("should clear all statuses and touched states on reset", () => {
      const state = create(initialValues, basicSchema);
      state.set("name", "Test");
      state.setStatus("name", { key: "name", variant: "error", message: "Error" });

      state.reset();

      expect(state.get("name").status.variant).toBe("success");
      expect(state.get("name").touched).toBe(false);
    });
  });

  describe("setCurrentStateAsInitialValues", () => {
    it("should snapshot current state as initial values", () => {
      const state = create(initialValues, basicSchema);
      state.set("name", "Jane Doe");

      state.setCurrentStateAsInitialValues();

      expect(state.get("name").touched).toBe(false);
      state.set("name", "John Doe");
      state.reset();
      expect(state.get("name").value).toBe("Jane Doe");
    });

    it("should make previously touched fields untouched", () => {
      const state = create(initialValues, basicSchema);
      state.set("name", "Jane Doe");
      state.set("email", "jane@example.com");
      expect(state.get("name").touched).toBe(true);
      expect(state.get("email").touched).toBe(true);

      state.setCurrentStateAsInitialValues();

      expect(state.get("name").touched).toBe(false);
      expect(state.get("email").touched).toBe(false);
    });
  });

  describe("remove", () => {
    it("should remove field value, status, and touched state", () => {
      const state = create(initialValues, basicSchema);
      state.setStatus("name", { key: "name", variant: "error", message: "Error" });

      state.remove("name");

      expect(state.value().name).toBeUndefined();
      expect(state.get("name", { optional: true })).toBeNull();
    });

    it("should remove nested field", () => {
      const state = create(initialValues, basicSchema);
      state.remove("profile.bio");
      expect(state.value().profile.bio).toBeUndefined();
    });

    it("should remove array element", () => {
      const state = create(initialValues, basicSchema);
      state.remove("tags.0");
      expect(state.value().tags).toEqual(["typescript"]);
    });
  });

  describe("validate", () => {
    it("should return true for valid data", () => {
      const state = create(initialValues, basicSchema);
      const isValid = state.validate();
      expect(isValid).toBe(true);
    });

    it("should not flag an untouched invalid field when another field changes", () => {
      const invalidValues = { ...initialValues, email: "invalid-email" };
      const state = create(invalidValues, basicSchema);
      state.set("name", "Jane Doe");
      expect(state.get("email").status.variant).toBe("success");
    });

    it("should set error status for invalid fields", () => {
      const invalidValues = { ...initialValues, email: "invalid-email" };
      const state = create(invalidValues, basicSchema);
      state.validate();
      const fieldState = state.get("email");
      expect(fieldState.status.variant).toBe("error");
      expect(fieldState.status.message).toBe("Invalid email format");
    });

    it("should return true for warnings (non-error variants)", () => {
      const warningValues = { ...initialValues, name: "admin", age: 16 };
      const state = create(warningValues, basicSchema);
      const isValid = state.validate();
      expect(isValid).toBe(true);
      expect(state.get("name").status.variant).toBe("warning");
    });

    it("should clear previous validation errors on successful validation", () => {
      const state = create({ ...initialValues, email: "invalid" }, basicSchema);
      state.validate();
      expect(state.get("email").status.variant).toBe("error");

      state.set("email", "valid@example.com");
      state.validate();
      expect(state.get("email").status.variant).toBe("success");
    });

    it("should keep the status object of a field whose result did not change", () => {
      const state = create(initialValues, basicSchema);
      state.set("email", "invalid");
      const status = state.get("email").status;
      expect(status.variant).toBe("error");
      state.set("name", "Jane Doe");
      expect(state.get("email").status).toBe(status);
    });

    it("should return true when no schema is provided", () => {
      const state = create(initialValues);
      const isValid = state.validate();
      expect(isValid).toBe(true);
    });
  });

  describe("validateAsync", () => {
    it("should return true for valid data", async () => {
      const state = create(initialValues, basicSchema);
      const isValid = await state.validateAsync();
      expect(isValid).toBe(true);
    });

    it("should return false for invalid data", async () => {
      const invalidValues = { ...initialValues, email: "invalid-email" };
      const state = create(invalidValues, basicSchema);
      const isValid = await state.validateAsync();
      expect(isValid).toBe(false);
    });

    it("should set error status for invalid fields", async () => {
      const invalidValues = { ...initialValues, email: "invalid-email" };
      const state = create(invalidValues, basicSchema);
      await state.validateAsync();
      const fieldState = state.get("email");
      expect(fieldState.status.variant).toBe("error");
      expect(fieldState.status.message).toBe("Invalid email format");
    });

    it("should return true when no schema is provided", async () => {
      const state = create(initialValues);
      const isValid = await state.validateAsync();
      expect(isValid).toBe(true);
    });
  });

  describe("getState", () => {
    it("should return field state for existing field", () => {
      const state = create(initialValues, basicSchema);
      const fieldState = state.get("name");
      expect(fieldState.value).toBe("John Doe");
      expect(fieldState.status.variant).toBe("success");
      expect(fieldState.touched).toBe(false);
      expect(fieldState.required).toBe(true);
    });

    it("should return field state for nested field", () => {
      const state = create(initialValues, basicSchema);
      const fieldState = state.get("profile.bio");
      expect(fieldState.value).toBe("Software developer");
    });

    it("should return field state for array element", () => {
      const state = create(initialValues, basicSchema);
      const fieldState = state.get("tags.0");
      expect(fieldState.value).toBe("developer");
    });

    it("should throw error for non-existent required field", () => {
      const state = create(initialValues, basicSchema);
      expect(() => state.get("nonExistent")).toThrow();
    });

    it("should return null for non-existent optional field", () => {
      const state = create(initialValues, basicSchema);
      const fieldState = state.get("nonExistent", { optional: true });
      expect(fieldState).toBeNull();
    });

    describe("schema verdict on an absent value", () => {
      it("should show the schema default without writing it", () => {
        const state = create(verdictValues, verdictSchema);
        const fieldState = state.get("defaulted");
        expect(fieldState.value).toBe("cat");
        expect(fieldState.required).toBe(false);
        expect("defaulted" in state.value()).toBe(false);
      });

      it("should not override an existing value with the default", () => {
        const state = create({ ...verdictValues, defaulted: "dog" }, verdictSchema);
        expect(state.get("defaulted").value).toBe("dog");
      });

      it("should read an optional field as undefined", () => {
        const state = create(verdictValues, verdictSchema);
        const fieldState = state.get("optionalField");
        expect(fieldState.value).toBeUndefined();
        expect(fieldState.required).toBe(false);
      });

      it("should throw for a required field", () => {
        const state = create(
          { ...verdictValues, name: undefined } as unknown as VerdictValues,
          verdictSchema,
        );
        expect(() => state.get("name")).toThrow(
          "Field name is required and holds no value",
        );
        expect(state.get("name", { optional: true })).toBeNull();
      });

      it("should throw for a path the schema does not contain", () => {
        const state = create(verdictValues, verdictSchema);
        expect(() => state.get("nested.typo")).toThrow(
          "Field nested.typo is not in the form",
        );
      });

      it("should treat a path under a record of unknowns as optional", () => {
        const state = create(verdictValues, verdictSchema);
        const fieldState = state.get("extra.anything.deep");
        expect(fieldState.value).toBeUndefined();
        expect(fieldState.required).toBe(false);
      });

      it("should read a leaf of a defaulted subtree from the subtree default", () => {
        const state = create(verdictValues, verdictSchema);
        expect(state.get("color.r").value).toBe(1);
      });

      it("should read a leaf below an optional ancestor from nested defaults", () => {
        const state = create(verdictValues, verdictSchema);
        expect(state.get("layout.margin.x").value).toBe(2);
      });
    });

    it("should correctly determine required status from schema", () => {
      const state = create(initialValues, basicSchema);
      expect(state.get("name").required).toBe(true);
      expect(state.get("optionalField").required).toBe(false);
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

      it("should not require a string the schema accepts empty", () => {
        expect(create(values, schema).get("free").required).toBe(false);
      });

      it("should not require a value no input can empty", () => {
        const state = create(values, schema);
        expect(state.get("count").required).toBe(false);
        expect(state.get("on").required).toBe(false);
      });

      it("should require a list the schema refuses empty", () => {
        expect(create(values, schema).get("tags").required).toBe(true);
      });

      it("should not require a list the schema accepts empty", () => {
        expect(create(values, schema).get("open").required).toBe(false);
      });

      it("should not require a field whose schema checks asynchronously", () => {
        const asyncSchema = z.object({
          name: z.string().refine(async (v) => v !== "taken"),
        });
        const state = create({ name: "a" }, asyncSchema);
        expect(state.get("name").required).toBe(false);
      });
    });

    it("should return cached reference for performance", () => {
      const state = create(initialValues, basicSchema);
      const fieldState1 = state.get("name");
      const fieldState2 = state.get("name");
      expect(fieldState1).toBe(fieldState2);
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
        return { state: create(values, schema), checks: () => checks };
      };

      it("should not consult the schema again for an unchanged field", () => {
        const { state, checks } = build();
        state.get("tracked");
        state.get("tracked");
        expect(checks()).toBe(1);
      });

      it("should consult the schema again after the field changes", () => {
        const { state, checks } = build();
        state.get("tracked");
        state.set("tracked", "b");
        const before = checks();
        expect(state.get("tracked").value).toBe("b");
        expect(checks()).toBe(before + 1);
      });

      it("should refresh a field when a sibling discriminator changes", () => {
        const { state } = build();
        expect(state.get("config.value").required).toBe(true);
        state.set("config.type", "b");
        expect(state.get("config.value").required).toBe(false);
      });

      it("should refresh a child when its parent is replaced", () => {
        const { state } = build();
        state.get("nested.leaf");
        state.set("nested", { leaf: "m" });
        expect(state.get("nested.leaf").value).toBe("m");
      });

      it("should refresh every field on reset", () => {
        const { state } = build();
        state.set("other", "y");
        state.setStatus("tracked", { key: "tracked", variant: "error", message: "e" });
        state.reset();
        expect(state.get("other").touched).toBe(false);
        expect(state.get("tracked").status.variant).toBe("success");
      });
    });

    it("should update cached reference when field changes", () => {
      const state = create(initialValues, basicSchema);
      const fieldState1 = state.get("name");
      state.set("name", "Jane Doe");
      const fieldState2 = state.get("name");
      expect(fieldState1).not.toBe(fieldState2);
      expect(fieldState2.value).toBe("Jane Doe");
    });
  });

  describe("path validation", () => {
    it("should validate only specified path and its children", () => {
      const invalidValues = {
        ...initialValues,
        name: "",
        email: "invalid-email",
        profile: { ...initialValues.profile, website: "invalid-url" },
      };
      const state = create(invalidValues, basicSchema);

      const isValid = state.validate("profile");

      expect(isValid).toBe(false);
      expect(state.get("profile.website").status.variant).toBe("error");
      expect(state.get("name").status.variant).toBe("success");
      expect(state.get("email").status.variant).toBe("success");
    });

    it("should validate specific field path", () => {
      const invalidValues = {
        ...initialValues,
        name: "",
        email: "invalid-email",
      };
      const state = create(invalidValues, basicSchema);

      const isValid = state.validate("email");

      expect(isValid).toBe(false);
      expect(state.get("email").status.variant).toBe("error");
      expect(state.get("name").status.variant).toBe("success");
    });

    it("should validate parent path and include child errors", () => {
      const invalidValues = {
        ...initialValues,
        name: "",
        profile: { ...initialValues.profile, website: "invalid-url" },
      };
      const state = create(invalidValues, basicSchema);

      const isValid = state.validate("profile");

      expect(isValid).toBe(false);
      expect(state.get("profile.website").status.variant).toBe("error");
      expect(state.get("name").status.variant).toBe("success");
    });

    it("should validate child path when parent is specified", () => {
      const invalidValues = {
        ...initialValues,
        email: "invalid-email",
        profile: { ...initialValues.profile, website: "invalid-url" },
      };
      const state = create(invalidValues, basicSchema);

      const isValid = state.validate("profile.website");

      expect(isValid).toBe(false);
      expect(state.get("profile.website").status.variant).toBe("error");
      expect(state.get("email").status.variant).toBe("success");
    });

    it("should return true when validating path with no errors", () => {
      const partiallyInvalidValues = {
        ...initialValues,
        email: "invalid-email",
      };
      const state = create(partiallyInvalidValues, basicSchema);

      const isValid = state.validate("name");

      expect(isValid).toBe(true);
      expect(state.get("name").status.variant).toBe("success");
      expect(state.get("email").status.variant).toBe("success");
    });

    it("should handle array paths correctly", () => {
      const invalidValues = {
        ...initialValues,
        tags: ["", "valid-tag"],
      };
      const arraySchema = z.object({
        ...basicSchema.shape,
        tags: z.array(z.string().min(1, "Tag cannot be empty")),
      });
      const state = create(invalidValues, arraySchema);

      const isValid = state.validate("tags.0");

      expect(isValid).toBe(false);
      expect(state.get("tags.0").status.variant).toBe("error");
    });
  });

  describe("async path validation", () => {
    it("should validate only specified path asynchronously", async () => {
      const invalidValues = {
        ...initialValues,
        name: "",
        email: "invalid-email",
      };
      const state = create(invalidValues, basicSchema);

      const isValid = await state.validateAsync("email");

      expect(isValid).toBe(false);
      expect(state.get("email").status.variant).toBe("error");
      expect(state.get("name").status.variant).toBe("success");
    });

    it("should validate nested paths asynchronously", async () => {
      const invalidValues = {
        ...initialValues,
        name: "",
        profile: { ...initialValues.profile, website: "invalid-url" },
      };
      const state = create(invalidValues, basicSchema);

      const isValid = await state.validateAsync("profile");

      expect(isValid).toBe(false);
      expect(state.get("profile.website").status.variant).toBe("error");
      expect(state.get("name").status.variant).toBe("success");
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
    it("should return success when all fields are valid", () => {
      const state = create({ a: { type: "a", color: "red" } }, schema);
      state.validate();
      expect(state.get("a").status.variant).toBe("success");
    });

    it("should return an error for the most favored union field", () => {
      const state = create({ a: { type: "a", color: "" } }, schema);
      state.validate();
      expect(state.get("a.color").status.variant).toBe("error");
    });
  });

  describe("field array touched tracking", () => {
    it("should reset touched status when field array item is removed after modification", () => {
      const arraySchema = z.object({
        items: z.array(
          z.object({
            name: z.string(),
            value: z.number(),
          }),
        ),
      });

      const initialValues = {
        items: [
          { name: "item1", value: 10 },
          { name: "item2", value: 20 },
        ],
      };

      const state = create(initialValues, arraySchema);
      expect(state.touched.current).toBe(false);

      const newItem = { name: "item3", value: 30 };
      const updatedItems = [...state.value().items, newItem];
      state.set("items", updatedItems);

      expect(state.touched.current).toBe(true);

      state.set("items.2.name", "modified_item3");

      expect(state.touched.current).toBe(true);
      expect(state.get("items.2.name").touched).toBe(true);

      const finalItems = state.value().items.slice(0, 2);
      state.set("items", finalItems);

      expect(state.touched.current).toBe(false);
    });

    it("should maintain touched status for other modified fields when array item is removed", () => {
      const arraySchema = z.object({
        name: z.string(),
        items: z.array(
          z.object({
            title: z.string(),
          }),
        ),
      });

      const initialValues = {
        name: "test",
        items: [{ title: "original" }],
      };

      const state = create(initialValues, arraySchema);

      state.set("name", "modified");
      expect(state.touched.current).toBe(true);

      state.set("items", [...state.value().items, { title: "new item" }]);

      state.set("items.1.title", "modified new item");

      state.set("items", [{ title: "original" }]);

      expect(state.touched.current).toBe(true);
      expect(state.get("name").touched).toBe(true);
    });

    it("should handle nested array operations correctly", () => {
      const nestedArraySchema = z.object({
        groups: z.array(
          z.object({
            name: z.string(),
            items: z.array(
              z.object({
                value: z.string(),
              }),
            ),
          }),
        ),
      });

      const initialValues = {
        groups: [
          {
            name: "group1",
            items: [{ value: "item1" }],
          },
        ],
      };

      const state = create(initialValues, nestedArraySchema);

      const newGroup = { name: "group2", items: [{ value: "item2" }] };
      state.set("groups", [...state.value().groups, newGroup]);

      state.set("groups.1.items.0.value", "modified_item2");

      expect(state.touched.current).toBe(true);

      state.set("groups", [{ name: "group1", items: [{ value: "item1" }] }]);

      expect(state.touched.current).toBe(false);
    });
  });

  describe("edge cases", () => {
    it("should handle empty string paths correctly", () => {
      const state = create(initialValues, basicSchema);
      const newValues = { ...initialValues, name: "Jane Doe" };
      state.set("", newValues);
      expect(state.value()).toEqual(newValues);
    });

    it("should handle deeply nested paths", () => {
      const deepValues = {
        level1: {
          level2: {
            level3: {
              value: "deep",
            },
          },
        },
      };
      const deepSchema = z.object({
        level1: z.object({
          level2: z.object({
            level3: z.object({
              value: z.string(),
            }),
          }),
        }),
      });
      const state = create(deepValues, deepSchema);
      state.set("level1.level2.level3.value", "updated");
      expect(state.value().level1.level2.level3.value).toBe("updated");
    });

    it("should handle array mutations correctly", () => {
      const state = create(initialValues, basicSchema);
      state.set("tags", ["new", "tags"]);
      expect(state.value().tags).toEqual(["new", "tags"]);
    });

    it("should handle null and undefined values", () => {
      const state = create({ test: null }, z.object({ test: z.string().nullable() }));
      expect(state.get("test", { optional: true })).toBeNull();
    });

    it("should handle schema without required fields", () => {
      const optionalSchema = z.object({
        optional: z.string().optional(),
      });
      const state = create({ optional: undefined }, optionalSchema);
      expect(state.get("optional").value).toBeUndefined();
    });
  });
});
