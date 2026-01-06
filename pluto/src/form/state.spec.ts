// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { State } from "@/form/state";

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

describe("State", () => {
  describe("constructor", () => {
    it("should initialize with values and schema", () => {
      const state = new State(initialValues, basicSchema);
      expect(state.values).toEqual(initialValues);
      expect(state.initialValues).toEqual(initialValues);
    });

    it("should initialize without schema", () => {
      const state = new State(initialValues);
      expect(state.values).toEqual(initialValues);
      expect(state.initialValues).toEqual(initialValues);
    });

    it("should deep copy initial values", () => {
      const state = new State(initialValues, basicSchema);
      state.values.name = "Jane Doe";
      expect(state.initialValues.name).toBe("John Doe");
    });
  });

  describe("setValue", () => {
    it("should set a top-level value", () => {
      const state = new State(initialValues, basicSchema);
      state.setValue("name", "Jane Doe");
      expect(state.values.name).toBe("Jane Doe");
    });

    it("should set a nested value", () => {
      const state = new State(initialValues, basicSchema);
      state.setValue("profile.bio", "Updated bio");
      expect(state.values.profile.bio).toBe("Updated bio");
    });

    it("should set array values", () => {
      const state = new State(initialValues, basicSchema);
      state.setValue("tags.0", "engineer");
      expect(state.values.tags[0]).toBe("engineer");
    });

    it("should replace entire object when path is empty", () => {
      const state = new State(initialValues, basicSchema);
      const newValues = { ...initialValues, name: "Jane Doe" };
      state.setValue("", newValues);
      expect(state.values).toEqual(newValues);
    });

    it("should mark field as touched when value changes", () => {
      const state = new State(initialValues, basicSchema);
      state.setValue("name", "Jane Doe");
      const fieldState = state.getState("name");
      expect(fieldState.touched).toBe(true);
    });

    it("should not mark field as touched when value equals initial", () => {
      const state = new State(initialValues, basicSchema);
      state.setValue("name", "Jane Doe");
      state.setValue("name", "John Doe"); // Back to initial
      const fieldState = state.getState("name");
      expect(fieldState.touched).toBe(false);
    });

    describe("markTouched option", () => {
      it("should mark field as touched when markTouched is true", () => {
        const state = new State(initialValues, basicSchema);
        state.setValue("name", "Jane Doe", { markTouched: true });
        const fieldState = state.getState("name");
        expect(fieldState.touched).toBe(true);
      });

      it("should not mark field as touched when markTouched is false", () => {
        const state = new State(initialValues, basicSchema);
        state.setValue("name", "Jane Doe", { markTouched: false });
        const fieldState = state.getState("name");
        expect(fieldState.touched).toBe(false);
      });

      it("should mark field as touched by default when value changes", () => {
        const state = new State(initialValues, basicSchema);
        state.setValue("name", "Jane Doe");
        const fieldState = state.getState("name");
        expect(fieldState.touched).toBe(true);
      });

      it("should mark nested fields as touched when markTouched is true", () => {
        const state = new State(initialValues, basicSchema);
        state.setValue("profile.bio", "New bio", { markTouched: true });
        const fieldState = state.getState("profile.bio");
        expect(fieldState.touched).toBe(true);
      });

      it("should not mark nested fields as touched when markTouched is false", () => {
        const state = new State(initialValues, basicSchema);
        state.setValue("profile.bio", "New bio", { markTouched: false });
        const fieldState = state.getState("profile.bio");
        expect(fieldState.touched).toBe(false);
      });

      it("should mark array elements as touched when markTouched is true", () => {
        const state = new State(initialValues, basicSchema);
        state.setValue("tags.0", "new-tag", { markTouched: true });
        const fieldState = state.getState("tags.0");
        expect(fieldState.touched).toBe(true);
      });

      it("should not mark array elements as touched when markTouched is false", () => {
        const state = new State(initialValues, basicSchema);
        state.setValue("tags.0", "new-tag", { markTouched: false });
        const fieldState = state.getState("tags.0");
        expect(fieldState.touched).toBe(false);
      });

      it("should still clear touched when value equals initial regardless of markTouched", () => {
        const state = new State(initialValues, basicSchema);
        state.setValue("name", "Jane Doe", { markTouched: true });
        expect(state.getState("name").touched).toBe(true);

        // Setting back to initial should clear touched even with markTouched: true
        state.setValue("name", "John Doe", { markTouched: true });
        expect(state.getState("name").touched).toBe(false);
      });

      it("should handle markTouched for entire object replacement", () => {
        const state = new State(initialValues, basicSchema);
        const newValues = { ...initialValues, name: "Jane Doe" };
        state.setValue("", newValues, { markTouched: true });
        const fieldState = state.getState("");
        expect(fieldState.touched).toBe(true);
      });

      it("should respect markTouched when setting undefined values", () => {
        const state = new State(initialValues, basicSchema);
        state.setValue("optionalField", "some value", { markTouched: true });
        expect(state.getState("optionalField").touched).toBe(true);

        state.setValue("optionalField", undefined, { markTouched: false });
        expect(state.getState("optionalField", { optional: true })).toBeNull();
      });

      it("should work with validation when markTouched is used", () => {
        const state = new State(initialValues, basicSchema);
        state.setValue("email", "invalid-email", { markTouched: true });

        const isValid = state.validate();
        expect(isValid).toBe(false);

        const fieldState = state.getState("email");
        expect(fieldState.touched).toBe(true);
        expect(fieldState.status.variant).toBe("error");
      });

      it("should not trigger validation for untouched fields when markTouched is false", () => {
        const state = new State(initialValues, basicSchema);
        state.setValue("email", "invalid-email", { markTouched: false });

        const isValid = state.validate();
        expect(isValid).toBe(true);

        const fieldState = state.getState("email");
        expect(fieldState.touched).toBe(false);
        expect(fieldState.status.variant).toBe("success");
      });

      it("should handle complex nested updates with markTouched", () => {
        const state = new State(initialValues, basicSchema);
        const newProfile = {
          bio: "Updated bio",
          website: "https://newsite.com",
        };
        state.setValue("profile", newProfile, { markTouched: true });

        expect(state.getState("profile").touched).toBe(true);
        expect(state.getState("profile.bio").touched).toBe(false);
        expect(state.getState("profile.website").touched).toBe(false);
      });

      it("should clear orphaned touched paths even with markTouched option", () => {
        const state = new State(initialValues, basicSchema);

        // Add a new item and mark its child as touched
        state.setValue("tags", [...state.values.tags, "new-tag"], {
          markTouched: true,
        });
        state.setValue("tags.2", "modified-tag", { markTouched: true });
        expect(state.getState("tags.2").touched).toBe(true);

        // Remove the item - should clear the orphaned touched path
        state.setValue("tags", initialValues.tags, { markTouched: false });
        expect(state.hasBeenTouched).toBe(false);
      });
    });
  });

  describe("setStatus and clearStatus", () => {
    it("should set status for a field", () => {
      const state = new State(initialValues, basicSchema);
      const status = {
        key: "name",
        variant: "error" as const,
        message: "Error message",
      };
      state.setStatus("name", status);
      const fieldState = state.getState("name");
      expect(fieldState.status).toEqual(status);
    });

    it("should clear status for a field", () => {
      const state = new State(initialValues, basicSchema);
      const status = {
        key: "name",
        variant: "error" as const,
        message: "Error message",
      };
      state.setStatus("name", status);
      state.clearStatus("name");
      const fieldState = state.getState("name");
      expect(fieldState.status.variant).toBe("success");
    });
  });

  describe("setTouched and clearTouched", () => {
    it("should mark field as touched", () => {
      const state = new State(initialValues, basicSchema);
      state.setTouched("name");
      const fieldState = state.getState("name");
      expect(fieldState.touched).toBe(true);
    });

    it("should clear touched state for specific field", () => {
      const state = new State(initialValues, basicSchema);
      state.setTouched("name");
      state.clearTouched("name");
      const fieldState = state.getState("name");
      expect(fieldState.touched).toBe(false);
    });

    it("should clear all touched states when no path provided", () => {
      const state = new State(initialValues, basicSchema);
      state.setTouched("name");
      state.setTouched("email");
      state.clearTouched();
      expect(state.getState("name").touched).toBe(false);
      expect(state.getState("email").touched).toBe(false);
    });
  });

  describe("reset", () => {
    it("should reset to initial values", () => {
      const state = new State(initialValues, basicSchema);
      state.setValue("name", "Jane Doe");
      state.setStatus("name", { key: "name", variant: "error", message: "Error" });
      state.setTouched("name");

      state.reset();

      expect(state.values).toEqual(initialValues);
      expect(state.getState("name").status.variant).toBe("success");
      expect(state.getState("name").touched).toBe(false);
    });

    it("should reset to new initial values when provided", () => {
      const state = new State(initialValues, basicSchema);
      const newInitialValues = { ...initialValues, name: "Jane Doe" };

      state.reset(newInitialValues);

      expect(state.values).toEqual(newInitialValues);
      expect(state.initialValues).toEqual(newInitialValues);
    });

    it("should clear all statuses and touched states on reset", () => {
      const state = new State(initialValues, basicSchema);
      state.setValue("name", "Test");
      state.setStatus("name", { key: "name", variant: "error", message: "Error" });
      state.setTouched("name");

      state.reset();

      expect(state.getState("name").status.variant).toBe("success");
      expect(state.getState("name").touched).toBe(false);
    });
  });

  describe("setCurrentStateAsInitialValues", () => {
    it("should snapshot current state as initial values", () => {
      const state = new State(initialValues, basicSchema);
      state.setValue("name", "Jane Doe");
      state.setTouched("name");

      state.setCurrentStateAsInitialValues();

      expect(state.initialValues.name).toBe("Jane Doe");
      expect(state.getState("name").touched).toBe(false);
    });

    it("should make previously touched fields untouched", () => {
      const state = new State(initialValues, basicSchema);
      state.setValue("name", "Jane Doe");
      state.setValue("email", "jane@example.com");
      expect(state.getState("name").touched).toBe(true);
      expect(state.getState("email").touched).toBe(true);

      state.setCurrentStateAsInitialValues();

      expect(state.getState("name").touched).toBe(false);
      expect(state.getState("email").touched).toBe(false);
    });
  });

  describe("remove", () => {
    it("should remove field value, status, and touched state", () => {
      const state = new State(initialValues, basicSchema);
      state.setStatus("name", { key: "name", variant: "error", message: "Error" });
      state.setTouched("name");

      state.remove("name");

      expect(state.values.name).toBeUndefined();
      expect(state.getState("name", { optional: true })).toBeNull();
    });

    it("should remove nested field", () => {
      const state = new State(initialValues, basicSchema);
      state.remove("profile.bio");
      expect(state.values.profile.bio).toBeUndefined();
    });

    it("should remove array element", () => {
      const state = new State(initialValues, basicSchema);
      state.remove("tags.0");
      expect(state.values.tags).toEqual(["typescript"]);
    });
  });

  describe("validate", () => {
    it("should return true for valid data", () => {
      const state = new State(initialValues, basicSchema);
      const isValid = state.validate();
      expect(isValid).toBe(true);
    });

    it("should return true even if untouched fields are invalid", () => {
      const invalidValues = { ...initialValues, email: "invalid-email" };
      const state = new State(invalidValues, basicSchema);
      const isValid = state.validate();
      expect(isValid).toBe(true);
    });

    it("should set error status for invalid fields", () => {
      const invalidValues = { ...initialValues, email: "invalid-email" };
      const state = new State(invalidValues, basicSchema);
      state.setTouched("email");
      state.validate();
      const fieldState = state.getState("email");
      expect(fieldState.status.variant).toBe("error");
      expect(fieldState.status.message).toBe("Invalid email format");
    });

    it("should return true for warnings (non-error variants)", () => {
      const warningValues = { ...initialValues, name: "admin", age: 16 };
      const state = new State(warningValues, basicSchema);
      state.setTouched("name");
      const isValid = state.validate();
      expect(isValid).toBe(true);
      expect(state.getState("name").status.variant).toBe("warning");
    });

    it("should clear previous validation errors on successful validation", () => {
      const state = new State({ ...initialValues, email: "invalid" }, basicSchema);
      state.setTouched("email");
      state.validate();
      expect(state.getState("email").status.variant).toBe("error");

      state.setValue("email", "valid@example.com");
      state.validate();
      expect(state.getState("email").status.variant).toBe("success");
    });

    it("should return true when no schema is provided", () => {
      const state = new State(initialValues);
      const isValid = state.validate();
      expect(isValid).toBe(true);
    });
  });

  describe("validateAsync", () => {
    it("should return true for valid data", async () => {
      const state = new State(initialValues, basicSchema);
      const isValid = await state.validateAsync();
      expect(isValid).toBe(true);
    });

    it("should return false for invalid data", async () => {
      const invalidValues = { ...initialValues, email: "invalid-email" };
      const state = new State(invalidValues, basicSchema);
      state.setTouched("email");
      const isValid = await state.validateAsync();
      expect(isValid).toBe(false);
    });

    it("should set error status for invalid fields", async () => {
      const invalidValues = { ...initialValues, email: "invalid-email" };
      const state = new State(invalidValues, basicSchema);
      state.setTouched("email");
      await state.validateAsync();
      const fieldState = state.getState("email");
      expect(fieldState.status.variant).toBe("error");
      expect(fieldState.status.message).toBe("Invalid email format");
    });

    it("should return true when no schema is provided", async () => {
      const state = new State(initialValues);
      const isValid = await state.validateAsync();
      expect(isValid).toBe(true);
    });
  });

  describe("hasBeenTouched", () => {
    it("should return false initially", () => {
      const state = new State(initialValues, basicSchema);
      expect(state.hasBeenTouched).toBe(false);
    });

    it("should return true when any field is touched", () => {
      const state = new State(initialValues, basicSchema);
      state.setTouched("name");
      expect(state.hasBeenTouched).toBe(true);
    });

    it("should return false when all touched states are cleared", () => {
      const state = new State(initialValues, basicSchema);
      state.setTouched("name");
      state.setTouched("email");
      state.clearTouched();
      expect(state.hasBeenTouched).toBe(false);
    });
  });

  describe("getState", () => {
    it("should return field state for existing field", () => {
      const state = new State(initialValues, basicSchema);
      const fieldState = state.getState("name");
      expect(fieldState.value).toBe("John Doe");
      expect(fieldState.status.variant).toBe("success");
      expect(fieldState.touched).toBe(false);
      expect(fieldState.required).toBe(true);
    });

    it("should return field state for nested field", () => {
      const state = new State(initialValues, basicSchema);
      const fieldState = state.getState("profile.bio");
      expect(fieldState.value).toBe("Software developer");
    });

    it("should return field state for array element", () => {
      const state = new State(initialValues, basicSchema);
      const fieldState = state.getState("tags.0");
      expect(fieldState.value).toBe("developer");
    });

    it("should throw error for non-existent required field", () => {
      const state = new State(initialValues, basicSchema);
      expect(() => state.getState("nonExistent")).toThrow();
    });

    it("should return null for non-existent optional field", () => {
      const state = new State(initialValues, basicSchema);
      const fieldState = state.getState("nonExistent", { optional: true });
      expect(fieldState).toBeNull();
    });

    it("should set and return default value when field is null", () => {
      const state = new State(
        { ...initialValues, optionalField: undefined },
        basicSchema,
      );
      const fieldState = state.getState("optionalField", { defaultValue: "default" });
      expect(fieldState?.value).toBe("default");
      expect(state.values.optionalField).toBe("default");
    });

    it("should not override existing value with default value", () => {
      const state = new State(
        { ...initialValues, optionalField: "existing" },
        basicSchema,
      );
      const fieldState = state.getState("optionalField", { defaultValue: "default" });
      expect(fieldState?.value).toBe("existing");
    });

    it("should correctly determine required status from schema", () => {
      const state = new State(initialValues, basicSchema);
      expect(state.getState("name").required).toBe(true);
      expect(state.getState("optionalField", { optional: true })).toBeNull();
    });

    it("should return cached reference for performance", () => {
      const state = new State(initialValues, basicSchema);
      const fieldState1 = state.getState("name");
      const fieldState2 = state.getState("name");
      expect(fieldState1).toBe(fieldState2);
    });

    it("should update cached reference when field changes", () => {
      const state = new State(initialValues, basicSchema);
      const fieldState1 = state.getState("name");
      state.setValue("name", "Jane Doe");
      const fieldState2 = state.getState("name");
      expect(fieldState1).not.toBe(fieldState2);
      expect(fieldState2.value).toBe("Jane Doe");
    });
  });

  describe("observer functionality", () => {
    it("should notify observers when notify is called", () => {
      const invalidValues = { ...initialValues, email: "invalid" };
      const state = new State(invalidValues, basicSchema);
      const observer = vi.fn();
      state.onChange(observer);
      state.notify();
      expect(observer).toHaveBeenCalled();
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
      const state = new State(invalidValues, basicSchema);
      state.setTouched("name");
      state.setTouched("email");
      state.setTouched("profile.website");

      const isValid = state.validate(true, "profile");

      expect(isValid).toBe(false);
      expect(state.getState("profile.website").status.variant).toBe("error");
      expect(state.getState("name").status.variant).toBe("success");
      expect(state.getState("email").status.variant).toBe("success");
    });

    it("should validate specific field path", () => {
      const invalidValues = {
        ...initialValues,
        name: "",
        email: "invalid-email",
      };
      const state = new State(invalidValues, basicSchema);
      state.setTouched("name");
      state.setTouched("email");

      const isValid = state.validate(true, "email");

      expect(isValid).toBe(false);
      expect(state.getState("email").status.variant).toBe("error");
      expect(state.getState("name").status.variant).toBe("success");
    });

    it("should validate parent path and include child errors", () => {
      const invalidValues = {
        ...initialValues,
        name: "",
        profile: { ...initialValues.profile, website: "invalid-url" },
      };
      const state = new State(invalidValues, basicSchema);
      state.setTouched("name");
      state.setTouched("profile.website");

      const isValid = state.validate(true, "profile");

      expect(isValid).toBe(false);
      expect(state.getState("profile.website").status.variant).toBe("error");
      expect(state.getState("name").status.variant).toBe("success");
    });

    it("should validate child path when parent is specified", () => {
      const invalidValues = {
        ...initialValues,
        email: "invalid-email",
        profile: { ...initialValues.profile, website: "invalid-url" },
      };
      const state = new State(invalidValues, basicSchema);
      state.setTouched("email");
      state.setTouched("profile.website");

      const isValid = state.validate(true, "profile.website");

      expect(isValid).toBe(false);
      expect(state.getState("profile.website").status.variant).toBe("error");
      expect(state.getState("email").status.variant).toBe("success");
    });

    it("should return true when validating path with no errors", () => {
      const partiallyInvalidValues = {
        ...initialValues,
        email: "invalid-email",
      };
      const state = new State(partiallyInvalidValues, basicSchema);
      state.setTouched("email");
      state.setTouched("name");

      const isValid = state.validate(true, "name");

      expect(isValid).toBe(true);
      expect(state.getState("name").status.variant).toBe("success");
      expect(state.getState("email").status.variant).toBe("success");
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
      const state = new State(invalidValues, arraySchema);
      state.setTouched("tags.0");
      state.setTouched("tags.1");

      const isValid = state.validate(true, "tags.0");

      expect(isValid).toBe(false);
      expect(state.getState("tags.0").status.variant).toBe("error");
    });
  });

  describe("async path validation", () => {
    it("should validate only specified path asynchronously", async () => {
      const invalidValues = {
        ...initialValues,
        name: "",
        email: "invalid-email",
      };
      const state = new State(invalidValues, basicSchema);
      state.setTouched("name");
      state.setTouched("email");

      const isValid = await state.validateAsync(true, "email");

      expect(isValid).toBe(false);
      expect(state.getState("email").status.variant).toBe("error");
      expect(state.getState("name").status.variant).toBe("success");
    });

    it("should validate nested paths asynchronously", async () => {
      const invalidValues = {
        ...initialValues,
        name: "",
        profile: { ...initialValues.profile, website: "invalid-url" },
      };
      const state = new State(invalidValues, basicSchema);
      state.setTouched("name");
      state.setTouched("profile.website");

      const isValid = await state.validateAsync(true, "profile");

      expect(isValid).toBe(false);
      expect(state.getState("profile.website").status.variant).toBe("error");
      expect(state.getState("name").status.variant).toBe("success");
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
      const state = new State({ a: { type: "a", color: "red" } }, schema);
      state.setTouched("a");
      state.validate();
      expect(state.getState("a").status.variant).toBe("success");
    });

    it("should return an error for the most favored union field", () => {
      const state = new State({ a: { type: "a", color: "" } }, schema);
      state.setTouched("a.color");
      state.validate();
      expect(state.getState("a.color").status.variant).toBe("error");
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

      const state = new State(initialValues, arraySchema);
      expect(state.hasBeenTouched).toBe(false);

      const newItem = { name: "item3", value: 30 };
      const updatedItems = [...state.values.items, newItem];
      state.setValue("items", updatedItems);

      expect(state.hasBeenTouched).toBe(true);

      state.setValue("items.2.name", "modified_item3");

      expect(state.hasBeenTouched).toBe(true);
      expect(state.getState("items.2.name").touched).toBe(true);

      const finalItems = state.values.items.slice(0, 2);
      state.setValue("items", finalItems);

      expect(state.hasBeenTouched).toBe(false);
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

      const state = new State(initialValues, arraySchema);

      state.setValue("name", "modified");
      expect(state.hasBeenTouched).toBe(true);

      state.setValue("items", [...state.values.items, { title: "new item" }]);

      state.setValue("items.1.title", "modified new item");

      state.setValue("items", [{ title: "original" }]);

      expect(state.hasBeenTouched).toBe(true);
      expect(state.getState("name").touched).toBe(true);
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

      const state = new State(initialValues, nestedArraySchema);

      const newGroup = { name: "group2", items: [{ value: "item2" }] };
      state.setValue("groups", [...state.values.groups, newGroup]);

      state.setValue("groups.1.items.0.value", "modified_item2");

      expect(state.hasBeenTouched).toBe(true);

      state.setValue("groups", [{ name: "group1", items: [{ value: "item1" }] }]);

      expect(state.hasBeenTouched).toBe(false);
    });
  });

  describe("edge cases", () => {
    it("should handle empty string paths correctly", () => {
      const state = new State(initialValues, basicSchema);
      const newValues = { ...initialValues, name: "Jane Doe" };
      state.setValue("", newValues);
      expect(state.values).toEqual(newValues);
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
      const state = new State(deepValues, deepSchema);
      state.setValue("level1.level2.level3.value", "updated");
      expect(state.values.level1.level2.level3.value).toBe("updated");
    });

    it("should handle array mutations correctly", () => {
      const state = new State(initialValues, basicSchema);
      state.setValue("tags", ["new", "tags"]);
      expect(state.values.tags).toEqual(["new", "tags"]);
    });

    it("should handle null and undefined values", () => {
      const state = new State(
        { test: null },
        z.object({ test: z.string().nullable() }),
      );
      expect(state.getState("test", { optional: true })).toBeNull();
    });

    it("should handle schema without required fields", () => {
      const optionalSchema = z.object({
        optional: z.string().optional(),
      });
      const state = new State({ optional: undefined }, optionalSchema);
      expect(state.getState("optional", { optional: true })).toBeNull();
    });
  });
});
