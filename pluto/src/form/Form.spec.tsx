// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep, type record } from "@synnaxlabs/x";
import { act, fireEvent, render, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Form } from "@/form";
import { Input } from "@/input";

const basicFormSchema = z
  .object({
    name: z.string().min(1, "You must enter a name."),
    optionalField: z.string().optional(),
    age: z.number().min(5, "You must be at least 5 years old."),
    nested: z.object({ ssn: z.string(), ein: z.string().optional() }),
    array: z.array(z.object({ key: z.string(), name: z.string() })),
  })
  .check((ctx) => {
    if (ctx.value.name === "Billy Bob")
      ctx.issues.push({
        input: ctx.value.name,
        code: "custom",
        message: "You cannot be named Billy Bob.",
        path: ["name"],
        params: { variant: "warning" },
      });
  });

const initialFormValues: z.infer<typeof basicFormSchema> = {
  name: "John Doe",
  age: 42,
  nested: { ssn: "123-45-6789", ein: "" },
  array: [{ key: "key1", name: "John Doe" }],
};

const FormContainer = (props: PropsWithChildren): ReactElement => {
  const methods = Form.use<typeof basicFormSchema>({
    values: deep.copy(initialFormValues),
    schema: basicFormSchema,
  });
  return <Form.Form<typeof basicFormSchema> {...methods}>{props.children}</Form.Form>;
};

const wrapper = ({ children }: PropsWithChildren): ReactElement => (
  <FormContainer>{children}</FormContainer>
);

describe("Form", () => {
  describe("use", () => {
    describe("get", () => {
      it("should get a value from the form", () => {
        const { result } = renderHook(() =>
          Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
        );
        const field = result.current.get("name");
        expect(field.value).toBe("John Doe");
        expect(field.status.variant).toEqual("success");
      });

      it("should return the correct nested values", () => {
        const { result } = renderHook(() =>
          Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
        );
        const field = result.current.get("nested.ssn");
        expect(field.value).toBe("123-45-6789");
      });

      it("should throw an error if optional is false and the field is null", () => {
        const { result } = renderHook(() =>
          Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
        );
        expect(() => result.current.get("ssn")).toThrow();
      });

      it("should return null if optional is true and the field is null", () => {
        const { result } = renderHook(() =>
          Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
        );
        const field = result.current.get("ssn", { optional: true });
        expect(field).toBeNull();
      });

      it("should return true if a field is required in the schema", () => {
        const { result } = renderHook(() =>
          Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
        );
        const field = result.current.get("age");
        expect(field.required).toBe(true);
      });
    });

    describe("set", () => {
      it("should set a value in the form", () => {
        const { result } = renderHook(() =>
          Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
        );
        result.current.set("name", "Jane Doe");
        const field = result.current.get("name");
        expect(field.value).toBe("Jane Doe");
      });

      it("should correctly set all values in the form at once", () => {
        const { result } = renderHook(() =>
          Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
        );
        result.current.set("", {
          name: "Jane Doe",
          age: 43,
          nested: { ssn: "123-45-6786", ein: "" },
          array: [{ name: "Jane Doe" }],
        });
        expect(result.current.get("name").value).toBe("Jane Doe");
        expect(result.current.get("age").value).toBe(43);
        expect(result.current.get("nested.ssn").value).toBe("123-45-6786");
        expect(result.current.get("array.0.name").value).toBe("Jane Doe");
      });
    });

    describe("bind", () => {
      it("should bind a listener for form changes", () => {
        const { result } = renderHook(() =>
          Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
        );
        const onChange = vi.fn();
        result.current.bind(onChange);
        result.current.set("name", "Jane Doe");
        expect(onChange).toHaveBeenCalled();
      });
    });
    describe("validate", () => {
      it("should return false if a validation error occurs", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy({ ...initialFormValues, age: 3 }),
            schema: basicFormSchema,
          }),
        );
        expect(result.current.validate()).toBe(false);
        expect(result.current.get("age").status.variant).toEqual("error");
      });
      it("should call a bound listener if a validation error occurs", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy({ ...initialFormValues, age: 3 }),
            schema: basicFormSchema,
          }),
        );
        const onChange = vi.fn();
        result.current.bind(onChange);
        result.current.validate();
        expect(onChange).toHaveBeenCalled();
      });
      it("should return true if all validation errors are just warnings", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy({ ...initialFormValues, name: "Billy Bob" }),
            schema: basicFormSchema,
          }),
        );
        expect(result.current.validate()).toBe(true);
      });
    });
  });

  describe("useField", () => {
    it("should get a field from the form", () => {
      const { result } = renderHook(() => Form.useField<string>("name"), {
        wrapper,
      });
      expect(result.current.value).toBe("John Doe");
    });
    it("should set a field in the form", () => {
      const { result } = renderHook(() => Form.useField<string>("name"), {
        wrapper,
      });
      act(() => result.current.onChange("Jane Doe"));
      expect(result.current.value).toBe("Jane Doe");
    });
    it("should call an onChange handler passed to the hook", () => {
      const onChange = vi.fn();
      const { result } = renderHook(() => Form.useField<string>("name", { onChange }), {
        wrapper,
      });
      act(() => result.current.onChange("Jane Doe"));
      expect(onChange).toHaveBeenCalled();
    });

    it("should return a bad field status if a validation error occurs", () => {
      const { result, rerender } = renderHook(() => Form.useField<number>("age"), {
        wrapper,
      });
      act(() => result.current.onChange(3));
      rerender();
      expect(result.current.status.variant).toEqual("error");
    });

    it("should still allow the caller to set the field value even if a validation error occurs", () => {
      const { result } = renderHook(() => Form.useField<string>("name"), { wrapper });
      act(() => result.current.onChange(""));
      expect(result.current.value).toBe("");
    });

    it("should return true if a field is required in the schema", () => {
      const { result } = renderHook(() => Form.useField<string>("name"), { wrapper });
      expect(result.current.required).toBe(true);
    });

    it("should set the default value if the field is null", () => {
      const { result } = renderHook(
        () => Form.useField<string>("optionalField", { defaultValue: "cat" }),
        { wrapper },
      );
      expect(result.current.value).toBe("cat");
    });

    it("should respect the initial value if it is provided", () => {
      const { result } = renderHook(
        () => Form.useField<string>("name", { defaultValue: "Federico" }),
        { wrapper },
      );
      expect(result.current.value).toBe("John Doe");
    });
  });

  describe("useFieldState", () => {
    it("should get the full field state from the form", () => {
      const { result } = renderHook(() => Form.useFieldState<string>("name"), {
        wrapper,
      });
      expect(result.current?.value).toBe("John Doe");
      expect(result.current?.status.variant).toBe("success");
      expect(result.current?.touched).toBe(false);
      expect(result.current?.required).toBe(true);
    });

    it("should return the correct nested field state", () => {
      const { result } = renderHook(() => Form.useFieldState<string>("nested.ssn"), {
        wrapper,
      });
      expect(result.current?.value).toBe("123-45-6789");
      expect(result.current?.status.variant).toBe("success");
      expect(result.current?.required).toBe(true);
    });

    it("should return null for optional fields when they don't exist", () => {
      const { result } = renderHook(
        () => Form.useFieldState<string>("nonExistentField", { optional: true }),
        { wrapper },
      );
      expect(result.current).toBeNull();
    });

    it("should use default value when field is null", () => {
      const { result } = renderHook(
        () => Form.useFieldState<string>("optionalField", { defaultValue: "default" }),
        { wrapper },
      );
      expect(result.current?.value).toBe("default");
    });

    it("should correctly identify required vs optional fields", () => {
      const { result: requiredResult } = renderHook(
        () => Form.useFieldState<string>("name"),
        { wrapper },
      );
      const { result: optionalResult } = renderHook(
        () => Form.useFieldState<string>("optionalField", { optional: true }),
        { wrapper },
      );

      expect(requiredResult.current?.required).toBe(true);
      expect(optionalResult.current?.required).toBeUndefined();
    });
  });

  describe("useFieldValue", () => {
    it("should get just the value from a field", () => {
      const { result } = renderHook(() => Form.useFieldValue<string>("name"), {
        wrapper,
      });
      expect(result.current).toBe("John Doe");
    });

    it("should return the correct nested field value", () => {
      const { result } = renderHook(() => Form.useFieldValue<string>("nested.ssn"), {
        wrapper,
      });
      expect(result.current).toBe("123-45-6789");
    });

    it("should return null for optional fields when they don't exist", () => {
      const { result } = renderHook(
        () => Form.useFieldValue<string>("nonExistentField", { optional: true }),
        { wrapper },
      );
      expect(result.current).toBeNull();
    });

    it("should use default value when field is null", () => {
      const { result } = renderHook(
        () => Form.useFieldValue<string>("optionalField", { defaultValue: "default" }),
        { wrapper },
      );
      expect(result.current).toBe("default");
    });

    it("should return array values correctly", () => {
      const { result } = renderHook(() => Form.useFieldValue("array"), {
        wrapper,
      });
      expect(result.current).toEqual([{ key: "key1", name: "John Doe" }]);
    });

    it("should return array element values correctly", () => {
      const { result } = renderHook(() => Form.useFieldValue<string>("array.0.name"), {
        wrapper,
      });
      expect(result.current).toBe("John Doe");
    });

    it("should return complex nested object values correctly", () => {
      const { result } = renderHook(() => Form.useFieldValue("nested"), {
        wrapper,
      });
      expect(result.current).toEqual({ ssn: "123-45-6789", ein: "" });
    });

    it("should update a parent value state when a child value changes", async () => {
      const { result } = renderHook(
        () => {
          const child = Form.useField("nested.ssn");
          const parent = Form.useField<{ ssn: string }>("nested");
          return { child, parent };
        },
        {
          wrapper,
        },
      );
      act(() => result.current.child.onChange("123-45-6786"));
      expect(result.current.parent.value?.ssn).toEqual("123-45-6786");
    });

    it("should update an array parent when a child in the array changes", async () => {
      const { result } = renderHook(
        () => {
          const ctx = Form.useContext();
          const parent = Form.useFieldValue<string[]>("array");
          return { ctx, parent };
        },
        {
          wrapper,
        },
      );
      act(() => result.current.ctx.set("array.key1.name", "Cat"));
      expect(result.current.parent).toEqual([{ key: "key1", name: "Cat" }]);
    });
  });

  describe("useFieldValid", () => {
    it("should return true for valid fields", () => {
      const { result } = renderHook(() => Form.useFieldValid("name"), {
        wrapper,
      });
      expect(result.current).toBe(true);
    });

    it("should return false for non-existent optional fields", () => {
      const { result } = renderHook(() => Form.useFieldValid("nonExistentField"), {
        wrapper,
      });
      expect(result.current).toBe(false);
    });

    it("should work with nested fields", () => {
      const { result } = renderHook(() => Form.useFieldValid("nested.ssn"), {
        wrapper,
      });
      expect(result.current).toBe(true);
    });

    it("should work with array fields", () => {
      const { result } = renderHook(() => Form.useFieldValid("array.0.name"), {
        wrapper,
      });
      expect(result.current).toBe(true);
    });
  });

  describe("Field", () => {
    it("should return a text field with the correct value", () => {
      const c = render(<Form.Field path="name" />, { wrapper });
      expect(c.getByDisplayValue("John Doe")).toBeTruthy();
    });
    it("should render the correct label", () => {
      const c = render(<Form.Field path="name" label="Full Name" />, { wrapper });
      expect(c.getByText("Full Name")).toBeTruthy();
    });
    it("should capitalize the last part of the path if the label is not provided", () => {
      const c = render(<Form.Field path="name" />, { wrapper });
      expect(c.getByText("Name")).toBeTruthy();
    });
    it("should render a required indicator if the field is required", () => {
      const c = render(<Form.Field path="name" />, { wrapper });
      expect(c.getByText("*")).toBeTruthy();
    });
    it("should set the value of the field when the input changes", () => {
      const c = render(<Form.Field path="name" />, { wrapper });
      const input = c.getByDisplayValue("John Doe");
      fireEvent.change(input, { target: { value: "Jane Doe" } });
      expect(c.getByDisplayValue("Jane Doe")).toBeTruthy();
    });
    it("should render help text if validation on the field fails", () => {
      const c = render(<Form.Field<number> path="age">{Input.Numeric}</Form.Field>, {
        wrapper,
      });
      const input = c.getByDisplayValue("42");
      fireEvent.change(input, { target: { value: 1 } });
      fireEvent.blur(input);
      // we're executing an async parse so we need to wait for the error to show up
      expect(c.findByText("You must be at least 5 years old.")).toBeTruthy();
    });
    describe("Nested Field", () => {
      it("should return a text field with the correct value", () => {
        const c = render(<Form.Field path="nested.ssn" />, { wrapper });
        expect(c.getByDisplayValue("123-45-6789")).toBeTruthy();
      });
      it("should mark the nested field as required if it is required in the schema", () => {
        const c = render(<Form.Field path="nested.ssn" />, { wrapper });
        expect(c.getByText("*")).toBeTruthy();
      });
      it("should not mark the nested field as required if it is not required in the schema", () => {
        const c = render(<Form.Field path="nested.ein" />, { wrapper });
        expect(c.queryByText("*")).toBeNull();
      });
    });
    describe("Array Field", () => {
      it("should return a text field with the correct value", () => {
        const c = render(<Form.Field path="array.0.name" />, { wrapper });
        expect(c.getByDisplayValue("John Doe")).toBeTruthy();
      });
      it("should mark the array field as required if it is required in the schema", () => {
        const c = render(<Form.Field path="array.0.name" />, { wrapper });
        expect(c.getByText("*")).toBeTruthy();
      });
      it("should not mark the array field as required if it is not required in the schema", () => {
        const c = render(<Form.Field path="array.0.name" />, { wrapper });
        expect(c.getByText("*")).toBeTruthy();
      });
    });
  });

  describe("NumericField", () => {
    it("should return a numeric field with the correct value", () => {
      const c = render(<Form.NumericField path="age" />, { wrapper });
      expect(c.getByDisplayValue("42")).toBeTruthy();
    });
    it("should set the value of the field when the input changes", () => {
      const c = render(<Form.NumericField path="age" />, { wrapper });
      const input = c.getByDisplayValue("42");
      fireEvent.change(input, { target: { value: 43 } });
      expect(c.getByDisplayValue("43")).toBeTruthy();
    });
    it("should render help text if validation on the field fails", () => {
      const c = render(<Form.NumericField path="age" />, { wrapper });
      const input = c.getByDisplayValue("42");
      fireEvent.change(input, { target: { value: 1 } });
      fireEvent.blur(input);
      // we're executing an async parse so we need to wait for the error to show up
      expect(c.findByText("You must be at least 5 years old.")).toBeTruthy();
    });
  });

  describe("useFieldLIst", () => {
    it("should return the array as the value", () => {
      const res = renderHook(() => Form.useFieldList("array"), { wrapper });
      expect(res.result.current.data).toEqual(["key1"]);
    });
    it("should correctly push a value onto the start of the array", () => {
      const res = renderHook(
        () => Form.useFieldList<string, record.KeyedNamed>("array"),
        { wrapper },
      );
      res.result.current.push({ key: "key2", name: "Jane Doe" });
      res.rerender();
      expect(res.result.current.data).toEqual(["key1", "key2"]);
    });

    it("should correctly remove the given index from the array", () => {
      const res = renderHook(() => Form.useFieldList("array"), { wrapper });
      act(() => {
        res.result.current.remove("key1");
      });
      expect(res.result.current.data).toEqual([]);
    });

    it("should correctly keep only the given index in the array", () => {
      const res = renderHook(
        () => Form.useFieldList<string, record.KeyedNamed>("array"),
        { wrapper },
      );
      res.result.current.push({ key: "key2", name: "Jane Doe" });
      res.rerender();
      res.result.current.keepOnly("key2");
      res.rerender();
      expect(res.result.current.data).toEqual(["key2"]);
    });
  });

  describe("useFieldListUtils", () => {
    it("should return current array values", () => {
      const { result } = renderHook(
        () => Form.useFieldListUtils<string, record.KeyedNamed>("array"),
        { wrapper },
      );
      expect(result.current.value()).toEqual([{ key: "key1", name: "John Doe" }]);
    });
    it("should push items to array", () => {
      const { result, rerender } = renderHook(
        () => {
          const utils = Form.useFieldListUtils<string, record.KeyedNamed>("array");
          const field = Form.useFieldValue<record.KeyedNamed[]>("array");
          return { utils, field };
        },
        { wrapper },
      );
      act(() => result.current.utils.push({ key: "key2", name: "Jane Doe" }));
      rerender();
      expect(result.current.field).toHaveLength(2);
      expect(result.current.field?.[1]).toEqual({ key: "key2", name: "Jane Doe" });
    });
    it("should add items at specific index", () => {
      const { result, rerender } = renderHook(
        () => {
          const utils = Form.useFieldListUtils<string, record.KeyedNamed>("array");
          const field = Form.useFieldValue<record.KeyedNamed[]>("array");
          return { utils, field };
        },
        { wrapper },
      );
      act(() => {
        result.current.utils.push({ key: "key2", name: "Jane Doe" });
        result.current.utils.add({ key: "key3", name: "Bob Smith" }, 1);
      });
      rerender();
      expect(result.current.field).toHaveLength(3);
      expect(result.current.field?.[1]).toEqual({ key: "key3", name: "Bob Smith" });
    });
    it("should remove items by key", () => {
      const { result, rerender } = renderHook(
        () => {
          const utils = Form.useFieldListUtils<string, record.KeyedNamed>("array");
          const field = Form.useFieldValue<record.KeyedNamed[]>("array");
          return { utils, field };
        },
        { wrapper },
      );
      act(() => {
        result.current.utils.push([
          { key: "key2", name: "Jane Doe" },
          { key: "key3", name: "Bob Smith" },
        ]);
      });
      rerender();
      act(() => {
        result.current.utils.remove(["key1", "key3"]);
      });
      rerender();
      expect(result.current.field).toHaveLength(1);
      expect(result.current.field?.[0]).toEqual({ key: "key2", name: "Jane Doe" });
    });
    it("should keep only specified items", () => {
      const { result, rerender } = renderHook(
        () => {
          const utils = Form.useFieldListUtils<string, record.KeyedNamed>("array");
          const field = Form.useFieldValue<record.KeyedNamed[]>("array");
          return { utils, field };
        },
        { wrapper },
      );
      act(() => {
        result.current.utils.push([
          { key: "key2", name: "Jane Doe" },
          { key: "key3", name: "Bob Smith" },
        ]);
      });
      rerender();
      act(() => {
        result.current.utils.keepOnly("key2");
      });
      rerender();
      expect(result.current.field).toHaveLength(1);
      expect(result.current.field?.[0]).toEqual({ key: "key2", name: "Jane Doe" });
    });
    it("should sort array items", () => {
      const { result, rerender } = renderHook(
        () => {
          const utils = Form.useFieldListUtils<string, record.KeyedNamed>("array");
          const field = Form.useFieldValue<record.KeyedNamed[]>("array");
          return { utils, field };
        },
        { wrapper },
      );
      act(() => {
        result.current.utils.push([
          { key: "key2", name: "Zara" },
          { key: "key3", name: "Alice" },
        ]);
      });
      rerender();
      act(() => result.current.utils.sort?.((a, b) => a.name.localeCompare(b.name)));
      rerender();
      expect(result.current.field?.[0].name).toBe("Alice");
      expect(result.current.field?.[1].name).toBe("John Doe");
      expect(result.current.field?.[2].name).toBe("Zara");
    });
  });

  describe("reset functionality", () => {
    describe("reset()", () => {
      it("should reset all form values to initial values", () => {
        const { result } = renderHook(() =>
          Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
        );

        result.current.set("name", "Jane Doe");
        result.current.set("age", 25);
        result.current.set("nested.ssn", "987-65-4321");
        result.current.set("array.0.name", "Changed Name");

        expect(result.current.get("name").value).toBe("Jane Doe");
        expect(result.current.get("age").value).toBe(25);
        expect(result.current.get("nested.ssn").value).toBe("987-65-4321");
        expect(result.current.get("array.0.name").value).toBe("Changed Name");

        result.current.reset();

        expect(result.current.get("name").value).toBe("John Doe");
        expect(result.current.get("age").value).toBe(42);
        expect(result.current.get("nested.ssn").value).toBe("123-45-6789");
        expect(result.current.get("array.0.name").value).toBe("John Doe");
      });

      it("should reset form values to new provided values", () => {
        const { result } = renderHook(() =>
          Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
        );

        const newValues = {
          name: "New Name",
          age: 30,
          nested: { ssn: "555-55-5555", ein: "12-3456789" },
          array: [{ key: "newKey", name: "New Array Name" }],
        };

        result.current.set("name", "Temporary Name");
        expect(result.current.get("name").value).toBe("Temporary Name");

        result.current.reset(newValues);

        expect(result.current.get("name").value).toBe("New Name");
        expect(result.current.get("age").value).toBe(30);
        expect(result.current.get("nested.ssn").value).toBe("555-55-5555");
        expect(result.current.get("nested.ein").value).toBe("12-3456789");
        expect(result.current.get("array.0.name").value).toBe("New Array Name");
      });

      it("should clear all validation errors when resetting", () => {
        const { result } = renderHook(() =>
          Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
        );

        // Set invalid values to trigger validation errors
        result.current.set("name", "Billy Bob"); // Required field
        result.current.set("age", 3); // Below minimum

        // Verify errors are present
        expect(result.current.get("name").status.variant).toBe("warning");
        expect(result.current.get("age").status.variant).toBe("error");

        // Reset form
        result.current.reset();

        // Verify errors are cleared
        expect(result.current.get("name").status.variant).toBe("success");
        expect(result.current.get("age").status.variant).toBe("success");
      });

      it("should clear all touched states when resetting", () => {
        const { result } = renderHook(() =>
          Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
        );
        result.current.set("name", "Jane Doe");
        result.current.set("age", 25);
        result.current.set("nested.ssn", "987-65-4321");
        expect(result.current.get("name").touched).toBe(true);
        expect(result.current.get("age").touched).toBe(true);
        expect(result.current.get("nested.ssn").touched).toBe(true);
        result.current.reset();
        expect(result.current.get("name").touched).toBe(false);
        expect(result.current.get("age").touched).toBe(false);
        expect(result.current.get("nested.ssn").touched).toBe(false);
      });

      it("should call onChange handler when resetting", () => {
        const onChange = vi.fn();
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
            onChange,
          }),
        );

        result.current.set("name", "Jane Doe");
        onChange.mockClear();

        result.current.reset();

        expect(onChange).toHaveBeenCalled();
      });

      it("should call onHasTouched with false when resetting from touched state", () => {
        const onHasTouched = vi.fn();
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
            onHasTouched,
          }),
        );
        result.current.set("name", "Jane Doe");
        expect(onHasTouched).toHaveBeenLastCalledWith(true);
        result.current.reset();
        expect(onHasTouched).toHaveBeenLastCalledWith(false);
      });

      it("should handle resetting nested objects correctly", () => {
        const { result } = renderHook(() =>
          Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
        );

        // Change nested values
        result.current.set("nested.ssn", "999-99-9999");
        result.current.set("nested.ein", "98-7654321");

        // Verify changes
        expect(result.current.get("nested.ssn").value).toBe("999-99-9999");
        expect(result.current.get("nested.ein").value).toBe("98-7654321");

        // Reset form
        result.current.reset();

        // Verify nested values are reset
        expect(result.current.get("nested.ssn").value).toBe("123-45-6789");
        expect(result.current.get("nested.ein").value).toBe("");
      });

      it("should handle resetting array values correctly", () => {
        const { result } = renderHook(() =>
          Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
        );

        // Modify array
        result.current.set("array", [
          { key: "key1", name: "Modified Name" },
          { key: "key2", name: "New Item" },
        ]);

        // Verify changes
        expect(result.current.get("array").value).toHaveLength(2);
        expect(result.current.get("array.0.name").value).toBe("Modified Name");

        // Reset form
        result.current.reset();

        // Verify array is reset
        expect(result.current.get("array").value).toHaveLength(1);
        expect(result.current.get("array.0.name").value).toBe("John Doe");
        expect(result.current.get("array.0.key").value).toBe("key1");
      });
    });

    describe("setCurrentStateAsInitialValues()", () => {
      it("should set current form state as new initial values", () => {
        const { result } = renderHook(() =>
          Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
        );

        // Change values
        result.current.set("name", "Jane Doe");
        result.current.set("age", 25);

        // Set current state as initial
        result.current.setCurrentStateAsInitialValues();

        // Values should remain the same
        expect(result.current.get("name").value).toBe("Jane Doe");
        expect(result.current.get("age").value).toBe(25);

        // But touched states should be cleared
        expect(result.current.get("name").touched).toBe(false);
        expect(result.current.get("age").touched).toBe(false);

        // Now resetting should go to the new "initial" values
        result.current.set("name", "Another Name");
        result.current.reset();
        expect(result.current.get("name").value).toBe("Jane Doe"); // New initial value
        expect(result.current.get("age").value).toBe(25); // New initial value
      });

      it("should call onHasTouched with false when setting current state as initial", () => {
        const onHasTouched = vi.fn();
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
            onHasTouched,
          }),
        );

        // Touch a field
        result.current.set("name", "Jane Doe");
        expect(onHasTouched).toHaveBeenLastCalledWith(true);

        // Set current state as initial
        result.current.setCurrentStateAsInitialValues();

        // Should call onHasTouched with false
        expect(onHasTouched).toHaveBeenLastCalledWith(false);
      });

      it("should handle nested and array values when setting current state as initial", () => {
        const { result } = renderHook(() =>
          Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
        );

        // Change nested and array values
        result.current.set("nested.ssn", "999-99-9999");
        result.current.set("array.0.name", "New Array Name");

        // Set current state as initial
        result.current.setCurrentStateAsInitialValues();

        // Values should remain the same but not be touched
        expect(result.current.get("nested.ssn").value).toBe("999-99-9999");
        expect(result.current.get("nested.ssn").touched).toBe(false);
        expect(result.current.get("array.0.name").value).toBe("New Array Name");
        expect(result.current.get("array.0.name").touched).toBe(false);

        // Changing back to original values should now mark as touched
        result.current.set("nested.ssn", "123-45-6789");
        expect(result.current.get("nested.ssn").touched).toBe(true);
      });
    });
  });

  describe("touched state", () => {
    it("should mark a field as touched when its value changes", () => {
      const { result } = renderHook(() =>
        Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
      );
      result.current.set("name", "Jane Doe");
      const field = result.current.get("name");
      expect(field.touched).toBe(true);
    });

    describe("markTouched option", () => {
      it("should not mark field as touched when markTouched is false", () => {
        const { result } = renderHook(() =>
          Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
        );
        result.current.set("name", "Jane Doe", { markTouched: false });
        const field = result.current.get("name");
        expect(field.touched).toBe(false);
      });

      it("should explicitly mark field as touched when markTouched is true", () => {
        const { result } = renderHook(() =>
          Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
        );
        result.current.set("name", "Jane Doe", { markTouched: true });
        const field = result.current.get("name");
        expect(field.touched).toBe(true);
      });

      it("should not trigger onHasTouched when markTouched is false", () => {
        const onHasTouched = vi.fn();
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
            onHasTouched,
          }),
        );
        result.current.set("name", "Jane Doe", { markTouched: false });
        expect(onHasTouched).not.toHaveBeenCalled();
      });

      it("should trigger onHasTouched when markTouched is true", () => {
        const onHasTouched = vi.fn();
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
            onHasTouched,
          }),
        );
        result.current.set("name", "Jane Doe", { markTouched: true });
        expect(onHasTouched).toHaveBeenCalledWith(true);
      });

      it("should work with nested fields and markTouched option", () => {
        const { result } = renderHook(() =>
          Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
        );
        result.current.set("nested.ssn", "999-99-9999", { markTouched: false });
        expect(result.current.get("nested.ssn").touched).toBe(false);

        result.current.set("nested.ein", "12-3456789", { markTouched: true });
        expect(result.current.get("nested.ein").touched).toBe(true);
      });

      it("should work with array fields and markTouched option", () => {
        const { result } = renderHook(() =>
          Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
        );
        result.current.set("array.0.name", "Modified Name", { markTouched: false });
        expect(result.current.get("array.0.name").touched).toBe(false);

        result.current.set("array.0.key", "modifiedKey", { markTouched: true });
        expect(result.current.get("array.0.key").touched).toBe(true);
      });

      it("should still clear touched when value equals initial regardless of markTouched", () => {
        const { result } = renderHook(() =>
          Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
        );
        result.current.set("name", "Jane Doe", { markTouched: true });
        expect(result.current.get("name").touched).toBe(true);
        result.current.set("name", "John Doe", { markTouched: true });
        expect(result.current.get("name").touched).toBe(false);
      });

      it("should affect validation when markTouched is false", () => {
        const { result } = renderHook(() =>
          Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
        );
        result.current.set("age", 3, { markTouched: false });
        const isValid = result.current.validate();
        expect(isValid).toBe(false);
        expect(result.current.get("age").status.variant).toBe("error");
      });

      it("should trigger validation when markTouched is true and value is invalid", () => {
        const { result } = renderHook(() =>
          Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
        );
        result.current.set("age", 3, { markTouched: true });
        const isValid = result.current.validate();
        expect(isValid).toBe(false);
        expect(result.current.get("age").status.variant).toBe("error");
      });

      it("should handle multiple fields with different markTouched settings", () => {
        const { result } = renderHook(() =>
          Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
        );
        result.current.set("name", "Jane Doe", { markTouched: false });
        result.current.set("age", 30, { markTouched: true });
        result.current.set("nested.ssn", "111-11-1111", { markTouched: false });

        expect(result.current.get("name").touched).toBe(false);
        expect(result.current.get("age").touched).toBe(true);
        expect(result.current.get("nested.ssn").touched).toBe(false);
      });

      it("should preserve markTouched behavior through form operations", () => {
        const { result } = renderHook(() =>
          Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
        );
        result.current.set("name", "Jane Doe", { markTouched: false });
        expect(result.current.get("name").touched).toBe(false);
        result.current.set("age", 30);
        expect(result.current.get("age").touched).toBe(true);
        expect(result.current.get("name").touched).toBe(false);
      });
    });

    it("should not mark a field as touched when setting it to its initial value", () => {
      const { result } = renderHook(() =>
        Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
      );
      result.current.set("name", "John Doe");
      const field = result.current.get("name");
      expect(field.touched).toBe(false);
    });

    it("should mark a field as untouched when resetting to initial value", () => {
      const { result } = renderHook(() =>
        Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
      );
      result.current.set("name", "Jane Doe");
      expect(result.current.get("name").touched).toBe(true);
      result.current.set("name", "John Doe");
      expect(result.current.get("name").touched).toBe(false);
    });

    it("should clear all touched states when resetting the form", () => {
      const { result } = renderHook(() =>
        Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
      );
      result.current.set("name", "Jane Doe");
      result.current.set("age", 25);
      expect(result.current.get("name").touched).toBe(true);
      expect(result.current.get("age").touched).toBe(true);
      result.current.reset(deep.copy(initialFormValues));
      expect(result.current.get("name").touched).toBe(false);
      expect(result.current.get("age").touched).toBe(false);
    });

    it("should call onHasTouched when form touched state changes", () => {
      const onHasTouched = vi.fn();
      const { result } = renderHook(() =>
        Form.use({
          values: deep.copy(initialFormValues),
          schema: basicFormSchema,
          onHasTouched,
        }),
      );
      result.current.set("name", "Jane Doe");
      expect(onHasTouched).toHaveBeenLastCalledWith(true);
      result.current.set("age", 25);
      expect(onHasTouched).toHaveBeenCalledTimes(1);
      result.current.set("name", "John Doe");
      result.current.set("age", 42);
      expect(onHasTouched).toHaveBeenLastCalledWith(false);
    });

    it("should snapshot touched state when requested", () => {
      const { result } = renderHook(() =>
        Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
      );
      result.current.set("name", "Jane Doe");
      result.current.set("age", 25);
      expect(result.current.get("name").touched).toBe(true);
      expect(result.current.get("age").touched).toBe(true);
      result.current.setCurrentStateAsInitialValues();
      expect(result.current.get("name").touched).toBe(false);
      expect(result.current.get("age").touched).toBe(false);
      result.current.set("name", "John Doe");
      expect(result.current.get("name").touched).toBe(true);
    });

    it("should properly track touched state through reset and setCurrentStateAsInitialValues", () => {
      const { result } = renderHook(() =>
        Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema }),
      );
      result.current.set("name", "Jane Doe");
      expect(result.current.get("name").touched).toBe(true);
      result.current.reset();
      expect(result.current.get("name").touched).toBe(false);
      result.current.set("name", "Jane Doe");
      expect(result.current.get("name").touched).toBe(true);
      result.current.setCurrentStateAsInitialValues();
      expect(result.current.get("name").touched).toBe(false);
      result.current.set("name", "John Doe");
      expect(result.current.get("name").touched).toBe(true);
      result.current.set("name", "Jane Doe");
      expect(result.current.get("name").touched).toBe(false);
    });
  });
});
