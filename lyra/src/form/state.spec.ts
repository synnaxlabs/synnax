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
import { createLocalStore } from "@/form/store";

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

interface Tracked {
  channels: { key: string; port: number }[];
  config: { type: "a"; value: string } | { type: "b"; value?: string };
}

const trackedSchema = z.object({
  channels: z.array(z.object({ key: z.string(), port: z.number() })),
  config: z.discriminatedUnion("type", [
    z.object({ type: z.literal("a"), value: z.string().min(1) }),
    z.object({ type: z.literal("b"), value: z.string().optional() }),
  ]),
});

const buildTracked = () => {
  const store = createLocalStore<Tracked>({
    channels: [
      { key: "a", port: 1 },
      { key: "b", port: 2 },
    ],
    config: { type: "a", value: "v" },
  });
  const state = new State(store, trackedSchema);
  const handleChange = vi.fn();
  state.onChange(handleChange);
  return { store, state, handleChange };
};

describe("State", () => {
  describe("constructor", () => {
    it("should initialize with values and schema", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      expect(state.values).toEqual(initialValues);
      expect(state.initialValues).toEqual(initialValues);
    });

    it("should initialize without schema", () => {
      const state = new State(createLocalStore(initialValues));
      expect(state.values).toEqual(initialValues);
      expect(state.initialValues).toEqual(initialValues);
    });

    it("should deep copy initial values", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      state.values.name = "Jane Doe";
      expect(state.initialValues.name).toBe("John Doe");
    });
  });

  describe("setValue", () => {
    describe("materializing an absent ancestor", () => {
      it("should build a defaulted subtree before writing its leaf", () => {
        const state = new State(createLocalStore(verdictValues), verdictSchema);
        state.setValue("color.g", 5);
        expect(state.values.color).toEqual({ r: 1, g: 5 });
      });

      it("should build an optional subtree only when it can complete it", () => {
        const state = new State(createLocalStore(verdictValues), verdictSchema);
        state.setValue("redline.upper", 10);
        expect(state.values.redline).toEqual({ upper: 10 });
      });

      it("should build nested defaults below an optional ancestor", () => {
        const state = new State(createLocalStore(verdictValues), verdictSchema);
        state.setValue("layout.margin.x", 9);
        expect(state.values.layout).toEqual({ margin: { x: 9 } });
      });

      it("should leave a present ancestor alone", () => {
        const state = new State(
          createLocalStore({ ...verdictValues, color: { r: 3, g: 3 } }),
          verdictSchema,
        );
        state.setValue("color.g", 5);
        expect(state.values.color).toEqual({ r: 3, g: 5 });
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
        const state = new State(createLocalStore(initialValues), basicSchema);
        const { values } = state;
        const { profile } = values;
        state.setValue("profile.bio", "Updated bio");
        expect(state.values).not.toBe(values);
        expect(state.values.profile).not.toBe(profile);
        expect(profile.bio).toBe("Software developer");
      });

      it("should keep the identity of untouched siblings", () => {
        const state = new State(createLocalStore(initialValues), basicSchema);
        const { tags } = state.values;
        state.setValue("profile.bio", "Updated bio");
        expect(state.values.tags).toBe(tags);
      });

      it("should copy an array entry reached through its key", () => {
        const state = new State<z.ZodType<Keyed>>(createLocalStore(keyed()));
        const { channels, other } = state.values;
        state.setValue("channels.b.port", 3);
        expect(state.values.channels).not.toBe(channels);
        expect(state.values.channels[1]).not.toBe(channels[1]);
        expect(state.values.channels[1].port).toBe(3);
        expect(state.values.channels[0]).toBe(channels[0]);
        expect(state.values.other).toBe(other);
        expect(channels[1].port).toBe(2);
      });

      it("should give a field's state a new value after a write below it", () => {
        const state = new State<z.ZodType<Keyed>>(createLocalStore(keyed()));
        const before = state.getState<{ port: number }>("channels.0").value;
        state.setValue("channels.0.port", 3);
        const after = state.getState<{ port: number }>("channels.0").value;
        expect(after).not.toBe(before);
        expect(after.port).toBe(3);
        expect(before.port).toBe(1);
      });

      it("should keep the state object of a sibling the write left unchanged", () => {
        const state = new State(createLocalStore(initialValues), basicSchema);
        const sibling = state.getState("email");
        const written = state.getState("name");
        state.setValue("name", "Jane Doe");
        expect(state.getState("email")).toBe(sibling);
        expect(state.getState("name")).not.toBe(written);
        expect(state.getState("name").value).toBe("Jane Doe");
      });

      it("should give the ancestors of a removed path a new identity", () => {
        const state = new State<z.ZodType<Keyed>>(createLocalStore(keyed()));
        const { channels } = state.values;
        state.remove("channels.0");
        expect(state.values.channels).not.toBe(channels);
        expect(state.values.channels).toEqual([{ key: "b", port: 2 }]);
        expect(channels).toHaveLength(2);
      });

      it("should leave the older value unchanged when a shorter key also matches", () => {
        const state = new State<z.ZodType<Record<string, object>>>(
          createLocalStore({
            a: {},
            "a.b": { x: 1, y: 2 },
          }),
        );
        const { values } = state;
        state.remove("a.b.x");
        expect(state.values).toEqual({ a: {}, "a.b": { y: 2 } });
        expect(state.values["a.b"]).not.toBe(values["a.b"]);
        expect(values).toEqual({ a: {}, "a.b": { x: 1, y: 2 } });
      });
    });

    it("should set a top-level value", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      state.setValue("name", "Jane Doe");
      expect(state.values.name).toBe("Jane Doe");
    });

    it("should set a nested value", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      state.setValue("profile.bio", "Updated bio");
      expect(state.values.profile.bio).toBe("Updated bio");
    });

    it("should set array values", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      state.setValue("tags.0", "engineer");
      expect(state.values.tags[0]).toBe("engineer");
    });

    it("should replace entire object when path is empty", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      const newValues = { ...initialValues, name: "Jane Doe" };
      state.setValue("", newValues);
      expect(state.values).toEqual(newValues);
    });

    it("should mark field as touched when value changes", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      state.setValue("name", "Jane Doe");
      const fieldState = state.getState("name");
      expect(fieldState.touched).toBe(true);
    });

    it("should not mark field as touched when value equals initial", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      state.setValue("name", "Jane Doe");
      state.setValue("name", "John Doe"); // Back to initial
      const fieldState = state.getState("name");
      expect(fieldState.touched).toBe(false);
    });

    describe("markTouched option", () => {
      it("should mark field as touched when markTouched is true", () => {
        const state = new State(createLocalStore(initialValues), basicSchema);
        state.setValue("name", "Jane Doe", { markTouched: true });
        const fieldState = state.getState("name");
        expect(fieldState.touched).toBe(true);
      });

      it("should not mark field as touched when markTouched is false", () => {
        const state = new State(createLocalStore(initialValues), basicSchema);
        state.setValue("name", "Jane Doe", { markTouched: false });
        const fieldState = state.getState("name");
        expect(fieldState.touched).toBe(false);
      });

      it("should mark field as touched by default when value changes", () => {
        const state = new State(createLocalStore(initialValues), basicSchema);
        state.setValue("name", "Jane Doe");
        const fieldState = state.getState("name");
        expect(fieldState.touched).toBe(true);
      });

      it("should mark nested fields as touched when markTouched is true", () => {
        const state = new State(createLocalStore(initialValues), basicSchema);
        state.setValue("profile.bio", "New bio", { markTouched: true });
        const fieldState = state.getState("profile.bio");
        expect(fieldState.touched).toBe(true);
      });

      it("should not mark nested fields as touched when markTouched is false", () => {
        const state = new State(createLocalStore(initialValues), basicSchema);
        state.setValue("profile.bio", "New bio", { markTouched: false });
        const fieldState = state.getState("profile.bio");
        expect(fieldState.touched).toBe(false);
      });

      it("should mark array elements as touched when markTouched is true", () => {
        const state = new State(createLocalStore(initialValues), basicSchema);
        state.setValue("tags.0", "new-tag", { markTouched: true });
        const fieldState = state.getState("tags.0");
        expect(fieldState.touched).toBe(true);
      });

      it("should not mark array elements as touched when markTouched is false", () => {
        const state = new State(createLocalStore(initialValues), basicSchema);
        state.setValue("tags.0", "new-tag", { markTouched: false });
        const fieldState = state.getState("tags.0");
        expect(fieldState.touched).toBe(false);
      });

      it("should still clear touched when value equals initial regardless of markTouched", () => {
        const state = new State(createLocalStore(initialValues), basicSchema);
        state.setValue("name", "Jane Doe", { markTouched: true });
        expect(state.getState("name").touched).toBe(true);

        // Setting back to initial should clear touched even with markTouched: true
        state.setValue("name", "John Doe", { markTouched: true });
        expect(state.getState("name").touched).toBe(false);
      });

      it("should handle markTouched for entire object replacement", () => {
        const state = new State(createLocalStore(initialValues), basicSchema);
        const newValues = { ...initialValues, name: "Jane Doe" };
        state.setValue("", newValues, { markTouched: true });
        const fieldState = state.getState("");
        expect(fieldState.touched).toBe(true);
      });

      it("should respect markTouched when setting undefined values", () => {
        const state = new State(createLocalStore(initialValues), basicSchema);
        state.setValue("optionalField", "some value", { markTouched: true });
        expect(state.getState("optionalField").touched).toBe(true);

        state.setValue("optionalField", undefined, { markTouched: false });
        expect(state.getState("optionalField").value).toBeUndefined();
      });

      it("should work with validation when markTouched is used", () => {
        const state = new State(createLocalStore(initialValues), basicSchema);
        state.setValue("email", "invalid-email", { markTouched: true });

        const isValid = state.validate();
        expect(isValid).toBe(false);

        const fieldState = state.getState("email");
        expect(fieldState.touched).toBe(true);
        expect(fieldState.status.variant).toBe("error");
      });

      it("should not trigger validation for untouched fields when markTouched is false", () => {
        const state = new State(createLocalStore(initialValues), basicSchema);
        state.setValue("email", "invalid-email", { markTouched: false });

        const isValid = state.validate();
        expect(isValid).toBe(true);

        const fieldState = state.getState("email");
        expect(fieldState.touched).toBe(false);
        expect(fieldState.status.variant).toBe("success");
      });

      it("should handle complex nested updates with markTouched", () => {
        const state = new State(createLocalStore(initialValues), basicSchema);
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
        const state = new State(createLocalStore(initialValues), basicSchema);

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
      const state = new State(createLocalStore(initialValues), basicSchema);
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
      const state = new State(createLocalStore(initialValues), basicSchema);
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

    it("should clear every status when no path is given", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      state.setStatus("name", { key: "name", variant: "error", message: "e" });
      state.setStatus("email", { key: "email", variant: "error", message: "e" });
      state.clearStatus();
      expect(state.getState("name").status.variant).toBe("success");
      expect(state.getState("email").status.variant).toBe("success");
    });

    describe("external statuses", () => {
      const taken = { key: "name", variant: "error" as const, message: "Taken" };

      it("should survive validation after an edit to another field", () => {
        const state = new State(createLocalStore(initialValues), basicSchema);
        state.setStatus("name", taken);
        state.setValue("age", 30);
        state.validate();
        expect(state.getState("name").status).toEqual(taken);
      });

      it("should clear the status when the value at its path changes", () => {
        const state = new State(createLocalStore(initialValues), basicSchema);
        state.setStatus("name", taken);
        state.setValue("name", "Jane Doe");
        expect(state.getState("name").status.variant).toBe("success");
      });

      it("should keep the status when a write to its parent keeps its value", () => {
        const state = new State(createLocalStore(initialValues), basicSchema);
        const status = {
          key: "profile.bio",
          variant: "warning" as const,
          message: "Short",
        };
        state.setStatus("profile.bio", status);
        state.setValue("profile", {
          ...initialValues.profile,
          website: "https://x.com",
        });
        expect(state.getState("profile.bio").status).toEqual(status);
      });

      it("should clear the status when its path is removed", () => {
        const state = new State(createLocalStore(initialValues), basicSchema);
        state.setStatus("tags.1", { key: "tags.1", variant: "error", message: "Bad" });
        state.remove("tags.1");
        expect(state.getStatuses()).toEqual([]);
      });

      it("should survive the removal of a form-level validation error", () => {
        const schema = z
          .object({ a: z.string(), b: z.string() })
          .refine(({ a, b }) => a !== b, "Must differ");
        const state = new State(createLocalStore({ a: "x", b: "x" }), schema);
        state.validate(true);
        expect(state.getStatuses().map(({ message }) => message)).toEqual([
          "Must differ",
        ]);
        state.setStatus("a", { key: "a", variant: "error", message: "Taken" });
        state.setValue("b", "y");
        state.validate(true);
        expect(state.getStatuses().map(({ message }) => message)).toEqual(["Taken"]);
      });

      it("should yield to a status that validation sets at its path", () => {
        const state = new State(
          createLocalStore({ ...initialValues, email: "invalid" }),
          basicSchema,
        );
        state.setStatus("email", { key: "email", variant: "error", message: "Taken" });
        state.validate(true);
        expect(state.getState("email").status.message).toBe("Invalid email format");
        state.setValue("name", "Jane Doe");
        state.validate();
        expect(state.getState("email").status.variant).toBe("success");
      });
    });
  });

  describe("setTouched and clearTouched", () => {
    it("should mark field as touched", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      state.setTouched("name");
      const fieldState = state.getState("name");
      expect(fieldState.touched).toBe(true);
    });

    it("should clear touched state for specific field", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      state.setTouched("name");
      state.clearTouched("name");
      const fieldState = state.getState("name");
      expect(fieldState.touched).toBe(false);
    });

    it("should clear all touched states when no path provided", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      state.setTouched("name");
      state.setTouched("email");
      state.clearTouched();
      expect(state.getState("name").touched).toBe(false);
      expect(state.getState("email").touched).toBe(false);
    });
  });

  describe("reset", () => {
    it("should reset to initial values", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      state.setValue("name", "Jane Doe");
      state.setStatus("name", { key: "name", variant: "error", message: "Error" });
      state.setTouched("name");

      state.reset();

      expect(state.values).toEqual(initialValues);
      expect(state.getState("name").status.variant).toBe("success");
      expect(state.getState("name").touched).toBe(false);
    });

    it("should reset to new initial values when provided", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      const newInitialValues = { ...initialValues, name: "Jane Doe" };

      state.reset(newInitialValues);

      expect(state.values).toEqual(newInitialValues);
      expect(state.initialValues).toEqual(newInitialValues);
    });

    it("should clear all statuses and touched states on reset", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
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
      const state = new State(createLocalStore(initialValues), basicSchema);
      state.setValue("name", "Jane Doe");
      state.setTouched("name");

      state.setCurrentStateAsInitialValues();

      expect(state.initialValues.name).toBe("Jane Doe");
      expect(state.getState("name").touched).toBe(false);
    });

    it("should make previously touched fields untouched", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
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
      const state = new State(createLocalStore(initialValues), basicSchema);
      state.setStatus("name", { key: "name", variant: "error", message: "Error" });
      state.setTouched("name");

      state.remove("name");

      expect(state.values.name).toBeUndefined();
      expect(state.getState("name", { optional: true })).toBeNull();
    });

    it("should remove nested field", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      state.remove("profile.bio");
      expect(state.values.profile.bio).toBeUndefined();
    });

    it("should remove array element", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      state.remove("tags.0");
      expect(state.values.tags).toEqual(["typescript"]);
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
    const keysOf = (state: State<z.ZodType<Keyed>>): string[] =>
      state.values.channels.map(({ key }) => key);

    it("should insert an entry after the named entry", () => {
      const state = new State<z.ZodType<Keyed>>(createLocalStore(createKeyed()));
      state.insert("channels", { key: "c", port: 3 }, "a");
      expect(keysOf(state)).toEqual(["a", "c", "b"]);
      expect(state.getState("channels").touched).toBe(true);
    });

    it("should refresh a field whose index an insert shifts", () => {
      const state = new State<z.ZodType<Keyed>>(createLocalStore(createKeyed()));
      expect(state.getState("channels.0.port").value).toBe(1);
      state.insert("channels", { key: "c", port: 3 });
      expect(state.getState("channels.0.port").value).toBe(3);
    });

    it("should move an entry and refresh the fields it shifts", () => {
      const state = new State<z.ZodType<Keyed>>(createLocalStore(createKeyed()));
      expect(state.getState("channels.0.port").value).toBe(1);
      state.move("channels", "b");
      expect(keysOf(state)).toEqual(["b", "a"]);
      expect(state.getState("channels.0.port").value).toBe(2);
    });

    it("should clear the touched flag when a move restores the list", () => {
      const state = new State<z.ZodType<Keyed>>(createLocalStore(createKeyed()));
      state.move("channels", "b");
      state.move("channels", "a");
      expect(state.getState("channels").touched).toBe(false);
    });

    it("should mark the list touched when it removes an entry", () => {
      const state = new State<z.ZodType<Keyed>>(createLocalStore(createKeyed()));
      state.remove("channels.a");
      expect(keysOf(state)).toEqual(["b"]);
      expect(state.getState("channels").touched).toBe(true);
    });
  });

  describe("batch", () => {
    it("should make every write in the batch one store change", () => {
      const store = createLocalStore(initialValues);
      const state = new State(store, basicSchema);
      const handleChange = vi.fn();
      store.onChange(handleChange);
      state.batch(() => {
        state.setValue("name", "Jane");
        state.setValue("age", 30);
      });
      expect(handleChange).toHaveBeenCalledTimes(1);
      expect(state.getState("name").value).toBe("Jane");
      expect(state.getState("age").value).toBe(30);
    });
  });

  describe("listen", () => {
    it("should refresh a field that another writer changes", () => {
      const { store, state, handleChange } = buildTracked();
      state.listen();
      expect(state.getState("channels.b.port").value).toBe(2);
      store.set("channels.b.port", 3);
      expect(state.getState("channels.b.port").value).toBe(3);
      expect(handleChange).toHaveBeenCalledTimes(1);
    });

    it("should clear a status from setStatus when another writer changes its value", () => {
      const { store, state } = buildTracked();
      state.listen();
      const taken = {
        key: "channels.b.port",
        variant: "error" as const,
        message: "Taken",
      };
      state.setStatus("channels.b.port", taken);
      store.set("channels.b.port", 3);
      expect(state.getState("channels.b.port").status.variant).toBe("success");
    });

    it("should keep the state of a field that the change does not reach", () => {
      const { store, state } = buildTracked();
      state.listen();
      const prev = state.getState("channels.a.port");
      store.set("channels.b.port", 3);
      expect(state.getState("channels.a.port")).toBe(prev);
    });

    it("should refresh a field whose discriminator another writer changes", () => {
      const { store, state } = buildTracked();
      state.listen();
      expect(state.getState("config.value").required).toBe(true);
      store.set("config", { type: "b" });
      expect(state.getState("config.value").required).toBe(false);
    });

    it("should refresh a field that another writer removes", () => {
      const { store, state } = buildTracked();
      state.listen();
      expect(state.getState("channels.0.port").value).toBe(1);
      store.remove("channels.a");
      expect(state.getState("channels.0.port").value).toBe(2);
    });

    it("should not notify for its own writes", () => {
      const { state, handleChange } = buildTracked();
      state.listen();
      state.setValue("channels.a.port", 5);
      expect(handleChange).not.toHaveBeenCalled();
    });

    it("should catch up on a change made before it listens", () => {
      const { store, state, handleChange } = buildTracked();
      expect(state.getState("channels.b.port").value).toBe(2);
      store.set("channels.b.port", 3);
      state.listen();
      expect(handleChange).toHaveBeenCalledTimes(1);
      expect(state.getState("channels.b.port").value).toBe(3);
    });

    it("should stop following after the destructor runs", () => {
      const { store, state, handleChange } = buildTracked();
      const destroy = state.listen();
      destroy();
      store.set("channels.b.port", 3);
      expect(handleChange).not.toHaveBeenCalled();
    });
  });

  describe("setStore", () => {
    it("should read another store with its values as the initial ones", () => {
      const { state } = buildTracked();
      state.setValue("channels.a.port", 5);
      const next = createLocalStore<Tracked>({
        channels: [{ key: "c", port: 9 }],
        config: { type: "b" },
      });
      expect(state.setStore(next)).toBe(true);
      expect(state.getState("channels.0.port").value).toBe(9);
      expect(state.hasBeenTouched).toBe(false);
      expect(state.initialValues).toEqual(next.get());
    });

    it("should report no change for the store it already reads", () => {
      const { store, state } = buildTracked();
      expect(state.setStore(store)).toBe(false);
    });
  });

  describe("validate", () => {
    it("should return true for valid data", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      const isValid = state.validate();
      expect(isValid).toBe(true);
    });

    it("should return true even if untouched fields are invalid", () => {
      const invalidValues = { ...initialValues, email: "invalid-email" };
      const state = new State(createLocalStore(invalidValues), basicSchema);
      const isValid = state.validate();
      expect(isValid).toBe(true);
    });

    it("should set error status for invalid fields", () => {
      const invalidValues = { ...initialValues, email: "invalid-email" };
      const state = new State(createLocalStore(invalidValues), basicSchema);
      state.setTouched("email");
      state.validate();
      const fieldState = state.getState("email");
      expect(fieldState.status.variant).toBe("error");
      expect(fieldState.status.message).toBe("Invalid email format");
    });

    it("should return true for warnings (non-error variants)", () => {
      const warningValues = { ...initialValues, name: "admin", age: 16 };
      const state = new State(createLocalStore(warningValues), basicSchema);
      state.setTouched("name");
      const isValid = state.validate();
      expect(isValid).toBe(true);
      expect(state.getState("name").status.variant).toBe("warning");
    });

    it("should clear previous validation errors on successful validation", () => {
      const state = new State(
        createLocalStore({ ...initialValues, email: "invalid" }),
        basicSchema,
      );
      state.setTouched("email");
      state.validate();
      expect(state.getState("email").status.variant).toBe("error");

      state.setValue("email", "valid@example.com");
      state.validate();
      expect(state.getState("email").status.variant).toBe("success");
    });

    it("should keep the state object of a field whose error did not change", () => {
      const state = new State(
        createLocalStore({ ...initialValues, email: "invalid" }),
        basicSchema,
      );
      state.setTouched("email");
      state.validate();
      const before = state.getState("email");
      state.setValue("name", "Jane Doe");
      state.validate();
      expect(state.getState("email")).toBe(before);
    });

    it("should give a field a new state object when its error changes", () => {
      const state = new State(
        createLocalStore({ ...initialValues, age: -1 }),
        basicSchema,
      );
      state.setTouched("age");
      state.validate();
      expect(state.getState("age").status.message).toBe("Age must be positive");
      state.setValue("age", 200);
      state.validate();
      expect(state.getState("age").status.message).toBe("Age must be realistic");
    });

    it("should return true when no schema is provided", () => {
      const state = new State(createLocalStore(initialValues));
      const isValid = state.validate();
      expect(isValid).toBe(true);
    });
  });

  describe("validateAsync", () => {
    it("should return true for valid data", async () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      const isValid = await state.validateAsync();
      expect(isValid).toBe(true);
    });

    it("should return false for invalid data", async () => {
      const invalidValues = { ...initialValues, email: "invalid-email" };
      const state = new State(createLocalStore(invalidValues), basicSchema);
      state.setTouched("email");
      const isValid = await state.validateAsync();
      expect(isValid).toBe(false);
    });

    it("should set error status for invalid fields", async () => {
      const invalidValues = { ...initialValues, email: "invalid-email" };
      const state = new State(createLocalStore(invalidValues), basicSchema);
      state.setTouched("email");
      await state.validateAsync();
      const fieldState = state.getState("email");
      expect(fieldState.status.variant).toBe("error");
      expect(fieldState.status.message).toBe("Invalid email format");
    });

    it("should return true when no schema is provided", async () => {
      const state = new State(createLocalStore(initialValues));
      const isValid = await state.validateAsync();
      expect(isValid).toBe(true);
    });
  });

  describe("hasBeenTouched", () => {
    it("should return false initially", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      expect(state.hasBeenTouched).toBe(false);
    });

    it("should return true when any field is touched", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      state.setTouched("name");
      expect(state.hasBeenTouched).toBe(true);
    });

    it("should return false when all touched states are cleared", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      state.setTouched("name");
      state.setTouched("email");
      state.clearTouched();
      expect(state.hasBeenTouched).toBe(false);
    });
  });

  describe("getState", () => {
    it("should return field state for existing field", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      const fieldState = state.getState("name");
      expect(fieldState.value).toBe("John Doe");
      expect(fieldState.status.variant).toBe("success");
      expect(fieldState.touched).toBe(false);
      expect(fieldState.required).toBe(true);
    });

    it("should return field state for nested field", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      const fieldState = state.getState("profile.bio");
      expect(fieldState.value).toBe("Software developer");
    });

    it("should return field state for array element", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      const fieldState = state.getState("tags.0");
      expect(fieldState.value).toBe("developer");
    });

    it("should throw error for non-existent required field", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      expect(() => state.getState("nonExistent")).toThrow();
    });

    it("should return null for non-existent optional field", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      const fieldState = state.getState("nonExistent", { optional: true });
      expect(fieldState).toBeNull();
    });

    describe("schema verdict on an absent value", () => {
      it("should show the schema default without writing it", () => {
        const state = new State(createLocalStore(verdictValues), verdictSchema);
        const fieldState = state.getState("defaulted");
        expect(fieldState.value).toBe("cat");
        expect(fieldState.required).toBe(false);
        expect("defaulted" in state.values).toBe(false);
      });

      it("should not override an existing value with the default", () => {
        const state = new State(
          createLocalStore({ ...verdictValues, defaulted: "dog" }),
          verdictSchema,
        );
        expect(state.getState("defaulted").value).toBe("dog");
      });

      it("should read an optional field as undefined", () => {
        const state = new State(createLocalStore(verdictValues), verdictSchema);
        const fieldState = state.getState("optionalField");
        expect(fieldState.value).toBeUndefined();
        expect(fieldState.required).toBe(false);
      });

      it("should throw for a required field", () => {
        const state = new State(
          createLocalStore({
            ...verdictValues,
            name: undefined,
          } as unknown as VerdictValues),
          verdictSchema,
        );
        expect(() => state.getState("name")).toThrow(
          "Field name is required and holds no value",
        );
        expect(state.getState("name", { optional: true })).toBeNull();
      });

      it("should throw for a path the schema does not contain", () => {
        const state = new State(createLocalStore(verdictValues), verdictSchema);
        expect(() => state.getState("nested.typo")).toThrow(
          "Field nested.typo is not in the form",
        );
      });

      it("should treat a path under a record of unknowns as optional", () => {
        const state = new State(createLocalStore(verdictValues), verdictSchema);
        const fieldState = state.getState("extra.anything.deep");
        expect(fieldState.value).toBeUndefined();
        expect(fieldState.required).toBe(false);
      });

      it("should read a leaf of a defaulted subtree from the subtree default", () => {
        const state = new State(createLocalStore(verdictValues), verdictSchema);
        expect(state.getState("color.r").value).toBe(1);
      });

      it("should read a leaf below an optional ancestor from nested defaults", () => {
        const state = new State(createLocalStore(verdictValues), verdictSchema);
        expect(state.getState("layout.margin.x").value).toBe(2);
      });
    });

    it("should correctly determine required status from schema", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      expect(state.getState("name").required).toBe(true);
      expect(state.getState("optionalField").required).toBe(false);
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
        expect(
          new State(createLocalStore(values), schema).getState("free").required,
        ).toBe(false);
      });

      it("should not require a value no input can empty", () => {
        const state = new State(createLocalStore(values), schema);
        expect(state.getState("count").required).toBe(false);
        expect(state.getState("on").required).toBe(false);
      });

      it("should require a list the schema refuses empty", () => {
        expect(
          new State(createLocalStore(values), schema).getState("tags").required,
        ).toBe(true);
      });

      it("should not require a list the schema accepts empty", () => {
        expect(
          new State(createLocalStore(values), schema).getState("open").required,
        ).toBe(false);
      });

      it("should not require a field whose schema checks asynchronously", () => {
        const asyncSchema = z.object({
          name: z.string().refine(async (v) => v !== "taken"),
        });
        const state = new State(createLocalStore({ name: "a" }), asyncSchema);
        expect(state.getState("name").required).toBe(false);
      });
    });

    it("should return cached reference for performance", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      const fieldState1 = state.getState("name");
      const fieldState2 = state.getState("name");
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
        return {
          state: new State(createLocalStore(values), schema),
          checks: () => checks,
        };
      };

      it("should not consult the schema again for an unchanged field", () => {
        const { state, checks } = build();
        state.getState("tracked");
        state.getState("tracked");
        expect(checks()).toBe(1);
      });

      it("should consult the schema again after the field changes", () => {
        const { state, checks } = build();
        state.getState("tracked");
        state.setValue("tracked", "b");
        expect(state.getState("tracked").value).toBe("b");
        expect(checks()).toBe(2);
      });

      it("should refresh a field when a sibling discriminator changes", () => {
        const { state } = build();
        expect(state.getState("config.value").required).toBe(true);
        state.setValue("config.type", "b");
        expect(state.getState("config.value").required).toBe(false);
      });

      it("should refresh a child when its parent is replaced", () => {
        const { state } = build();
        state.getState("nested.leaf");
        state.setValue("nested", { leaf: "m" });
        expect(state.getState("nested.leaf").value).toBe("m");
      });

      describe("a change elsewhere in the form", () => {
        const buildNested = () => {
          let checks = 0;
          const schema = z.object({
            name: z.string(),
            config: z.object({
              tracked: z.string().check(() => {
                checks++;
              }),
            }),
          });
          const state = new State(
            createLocalStore({ name: "a", config: { tracked: "a" } }),
            schema,
          );
          state.getState("config.tracked");
          return { state, checks: () => checks };
        };

        it.each([
          ["a write", (s: State<z.ZodType>) => s.setValue("name", "b")],
          ["setTouched", (s: State<z.ZodType>) => s.setTouched("name")],
          ["clearTouched", (s: State<z.ZodType>) => s.clearTouched("name")],
          [
            "setStatus",
            (s: State<z.ZodType>) =>
              s.setStatus("name", { key: "name", variant: "error", message: "e" }),
          ],
          ["clearStatus", (s: State<z.ZodType>) => s.clearStatus("name")],
        ])("should not consult the schema for a field after %s", (_, change) => {
          const { state, checks } = buildNested();
          change(state);
          state.getState("config.tracked");
          expect(checks()).toBe(1);
        });

        it("should consult the schema again after clearing every touched flag", () => {
          const { state, checks } = buildNested();
          state.clearTouched();
          state.getState("config.tracked");
          expect(checks()).toBe(2);
        });
      });

      it("should refresh every field on reset", () => {
        const { state } = build();
        state.setValue("other", "y");
        state.setStatus("tracked", { key: "tracked", variant: "error", message: "e" });
        state.reset();
        expect(state.getState("other").touched).toBe(false);
        expect(state.getState("tracked").status.variant).toBe("success");
      });
    });

    it("should update cached reference when field changes", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
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
      const state = new State(createLocalStore(invalidValues), basicSchema);
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
      const state = new State(createLocalStore(invalidValues), basicSchema);
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
      const state = new State(createLocalStore(invalidValues), basicSchema);
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
      const state = new State(createLocalStore(invalidValues), basicSchema);
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
      const state = new State(createLocalStore(invalidValues), basicSchema);
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
      const state = new State(createLocalStore(partiallyInvalidValues), basicSchema);
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
      const state = new State(createLocalStore(invalidValues), arraySchema);
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
      const state = new State(createLocalStore(invalidValues), basicSchema);
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
      const state = new State(createLocalStore(invalidValues), basicSchema);
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
      const state = new State(
        createLocalStore<z.infer<typeof schema>>({ a: { type: "a", color: "red" } }),
        schema,
      );
      state.setTouched("a");
      state.validate();
      expect(state.getState("a").status.variant).toBe("success");
    });

    it("should return an error for the most favored union field", () => {
      const state = new State(
        createLocalStore<z.infer<typeof schema>>({ a: { type: "a", color: "" } }),
        schema,
      );
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

      const state = new State(createLocalStore(initialValues), arraySchema);
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

      const state = new State(createLocalStore(initialValues), arraySchema);

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

      const state = new State(createLocalStore(initialValues), nestedArraySchema);

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
      const state = new State(createLocalStore(initialValues), basicSchema);
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
      const state = new State(createLocalStore(deepValues), deepSchema);
      state.setValue("level1.level2.level3.value", "updated");
      expect(state.values.level1.level2.level3.value).toBe("updated");
    });

    it("should handle array mutations correctly", () => {
      const state = new State(createLocalStore(initialValues), basicSchema);
      state.setValue("tags", ["new", "tags"]);
      expect(state.values.tags).toEqual(["new", "tags"]);
    });

    it("should handle null and undefined values", () => {
      const state = new State(
        createLocalStore({ test: null }),
        z.object({ test: z.string().nullable() }),
      );
      expect(state.getState("test", { optional: true })).toBeNull();
    });

    it("should handle schema without required fields", () => {
      const optionalSchema = z.object({
        optional: z.string().optional(),
      });
      const state = new State(
        createLocalStore({ optional: undefined }),
        optionalSchema,
      );
      expect(state.getState("optional").value).toBeUndefined();
    });
  });
});
