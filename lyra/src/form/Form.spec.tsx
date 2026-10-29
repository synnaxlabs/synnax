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
import { renderToString } from "react-dom/server";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Form } from "@/form";
import { createLocalStore } from "@/form/store";
import { Input } from "@/input";

const basicFormSchema = z
  .object({
    name: z.string().min(1, "You must enter a name."),
    optionalField: z.string().optional(),
    age: z.number().min(5, "You must be at least 5 years old."),
    nested: z.object({ ssn: z.string().min(1), ein: z.string().optional() }),
    array: z.array(z.object({ key: z.string(), name: z.string().min(1) })),
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

const verdictSchema = z.object({
  optionalField: z.string().optional(),
  defaulted: z.string().default("cat"),
});

const VerdictFormContainer = (props: PropsWithChildren): ReactElement => {
  const methods = Form.use<typeof verdictSchema>({
    values: {} as z.infer<typeof verdictSchema>,
    schema: verdictSchema,
  });
  return <Form.Form<typeof verdictSchema> {...methods}>{props.children}</Form.Form>;
};

const verdictWrapper = ({ children }: PropsWithChildren): ReactElement => (
  <VerdictFormContainer>{children}</VerdictFormContainer>
);

describe("Form", () => {
  describe("use", () => {
    describe("get", () => {
      it("should get a value from the form", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        const field = result.current.get("name");
        expect(field.value).toBe("John Doe");
        expect(field.status.variant).toEqual("success");
      });

      it("should return the correct nested values", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        const field = result.current.get("nested.ssn");
        expect(field.value).toBe("123-45-6789");
      });

      it("should throw an error if optional is false and the field is null", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        expect(() => result.current.get("ssn")).toThrow();
      });

      it("should return null if optional is true and the field is null", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        const field = result.current.get("ssn", { optional: true });
        expect(field).toBeNull();
      });

      it("should return true if the schema refuses the field empty", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        expect(result.current.get("name").required).toBe(true);
      });

      it("should return false for a number, which no input can empty", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        expect(result.current.get("age").required).toBe(false);
      });
    });

    describe("set", () => {
      it("should set a value in the form", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        result.current.set("name", "Jane Doe");
        const field = result.current.get("name");
        expect(field.value).toBe("Jane Doe");
      });

      it("should correctly set all values in the form at once", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
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
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
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

    it("should show the schema default for an absent field without writing it", () => {
      const { result } = renderHook(
        () => {
          const field = Form.useField<string>("defaulted");
          const { value } = Form.useContext();
          return { field, stored: "defaulted" in (value() as object) };
        },
        { wrapper: verdictWrapper },
      );
      expect(result.current.field.value).toBe("cat");
      expect(result.current.stored).toBe(false);
    });

    it("should read an absent optional field as undefined", () => {
      const { result } = renderHook(() => Form.useField<string>("optionalField"), {
        wrapper: verdictWrapper,
      });
      expect(result.current.value).toBeUndefined();
      expect(result.current.required).toBe(false);
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

    it("should use the schema default when the field is absent", () => {
      const { result } = renderHook(() => Form.useFieldState<string>("defaulted"), {
        wrapper: verdictWrapper,
      });
      expect(result.current?.value).toBe("cat");
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
      expect(optionalResult.current?.required).toBe(false);
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

    it("should use the schema default when the field is absent", () => {
      const { result } = renderHook(() => Form.useFieldValue<string>("defaulted"), {
        wrapper: verdictWrapper,
      });
      expect(result.current).toBe("cat");
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

  describe("useStatuses", () => {
    const renderStatuses = (path: string) =>
      renderHook(() => ({ ctx: Form.useContext(), statuses: Form.useStatuses(path) }), {
        wrapper,
      });

    it("should read the statuses at and under the path", () => {
      const { result } = renderStatuses("nested");
      act(() => {
        result.current.ctx.setStatus("nested.ssn", {
          variant: "error",
          message: "SSN is taken",
        });
        result.current.ctx.setStatus("name", { variant: "error", message: "Taken" });
      });
      expect(result.current.statuses).toEqual([
        { key: "nested.ssn", variant: "error", message: "SSN is taken" },
      ]);
    });

    it("should leave out success statuses", () => {
      const { result } = renderHook(
        () => {
          Form.useFieldState("nested.ssn");
          return Form.useStatuses("nested");
        },
        { wrapper },
      );
      expect(result.current).toEqual([]);
    });

    it("should keep its result when a status outside the path changes", () => {
      const { result } = renderStatuses("nested");
      act(() =>
        result.current.ctx.setStatus("nested.ssn", {
          variant: "error",
          message: "Bad",
        }),
      );
      const before = result.current.statuses;
      act(() =>
        result.current.ctx.setStatus("name", { variant: "error", message: "" }),
      );
      expect(result.current.statuses).toBe(before);
    });
  });

  describe("Field", () => {
    it("should render its value on the server", () => {
      const html = renderToString(
        <FormContainer>
          <Form.Field path="name" />
        </FormContainer>,
      );
      expect(html).toContain('value="John Doe"');
    });
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
    it("should render nothing when visible is false", () => {
      const c = render(<Form.Field path="name" visible={false} />, { wrapper });
      expect(c.queryByDisplayValue("John Doe")).toBeNull();
    });
    it("should allow a hidden field to bind an absent path", () => {
      const c = render(<Form.Field path="absent" visible={false} />, { wrapper });
      expect(c.container.innerHTML).toBe("");
    });
    it("should render help text if validation on the field fails", () => {
      const c = render(
        <Form.Field<number> path="age">{(p) => <Input.Numeric {...p} />}</Form.Field>,
        { wrapper },
      );
      const input = c.getByDisplayValue("42");
      fireEvent.change(input, { target: { value: 1 } });
      fireEvent.blur(input);
      // we're executing an async parse so we need to wait for the error to show up
      expect(c.findByText("You must be at least 5 years old.")).toBeTruthy();
    });
    describe("required mark", () => {
      it("should mark the default text input when the schema refuses it empty", () => {
        const c = render(<Form.Field path="name" />, { wrapper });
        expect(c.getByText("*")).toBeTruthy();
      });
      it("should not mark a custom input", () => {
        const c = render(
          <Form.Field<string> path="name">{(p) => <Input.Text {...p} />}</Form.Field>,
          { wrapper },
        );
        expect(c.queryByText("*")).toBeNull();
      });
      it("should mark a custom input that is emptiable", () => {
        const c = render(
          <Form.Field<string> path="name" emptiable>
            {(p) => <Input.Text {...p} />}
          </Form.Field>,
          { wrapper },
        );
        expect(c.getByText("*")).toBeTruthy();
      });
      it("should mark a text field", () => {
        const c = render(<Form.TextField path="name" />, { wrapper });
        expect(c.getByText("*")).toBeTruthy();
      });
      it("should not mark a numeric field", () => {
        const c = render(<Form.NumericField path="age" />, { wrapper });
        expect(c.queryByText("*")).toBeNull();
      });
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
        const c = render(<Form.Field path="array.0.key" />, { wrapper });
        expect(c.queryByText("*")).toBeNull();
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

  describe("fieldListUtils over a store", () => {
    const byName = (a: record.KeyedNamed, b: record.KeyedNamed) =>
      a.name.localeCompare(b.name);
    const renderOverStore = () => {
      const store = createLocalStore({
        ...deep.copy(initialFormValues),
        array: [
          { key: "a", name: "A" },
          { key: "b", name: "B" },
          { key: "c", name: "C" },
        ],
      });
      const calls = {
        insert: vi.spyOn(store, "insert"),
        move: vi.spyOn(store, "move"),
        remove: vi.spyOn(store, "remove"),
        set: vi.spyOn(store, "set"),
      };
      const { result } = renderHook(() =>
        Form.fieldListUtils<string, record.KeyedNamed>(Form.use({ store }), "array"),
      );
      return { utils: () => result.current, calls };
    };

    it("should insert a pushed entry after the last entry", () => {
      const { utils, calls } = renderOverStore();
      const d = { key: "d", name: "D" };
      act(() => utils().push(d));
      expect(calls.insert.mock.calls).toEqual([["array", d, "c"]]);
      expect(calls.set).not.toHaveBeenCalled();
    });

    it("should insert an added entry after the entry before its index", () => {
      const { utils, calls } = renderOverStore();
      const d = { key: "d", name: "D" };
      const e = { key: "e", name: "E" };
      act(() => utils().add(d, 1));
      act(() => utils().add(e, 0));
      expect(calls.insert.mock.calls).toEqual([
        ["array", d, "a"],
        ["array", e, undefined],
      ]);
      expect(calls.set).not.toHaveBeenCalled();
    });

    it("should move only the entries that a sort puts out of place", () => {
      const { utils, calls } = renderOverStore();
      act(() => utils().sort?.(byName));
      expect(calls.move).not.toHaveBeenCalled();
      act(() => utils().sort?.((a, b) => byName(b, a)));
      expect(calls.move.mock.calls).toEqual([
        ["array", "c", undefined],
        ["array", "b", "c"],
      ]);
      expect(calls.set).not.toHaveBeenCalled();
    });

    it("should move a pushed entry into place when the push sorts", () => {
      const { utils, calls } = renderOverStore();
      const first = { key: "0", name: "0" };
      act(() => utils().push(first, byName));
      expect(calls.insert.mock.calls).toEqual([["array", first, "c"]]);
      expect(calls.move.mock.calls).toEqual([["array", "0", undefined]]);
      expect(calls.set).not.toHaveBeenCalled();
    });

    it("should remove each dropped entry by its key", () => {
      const { utils, calls } = renderOverStore();
      act(() => {
        utils().remove(["a", "c"]);
      });
      expect(calls.remove.mock.calls).toEqual([["array.a"], ["array.c"]]);
      expect(calls.set).not.toHaveBeenCalled();
    });

    it("should remove each entry that keepOnly does not keep", () => {
      const { utils, calls } = renderOverStore();
      act(() => {
        utils().keepOnly("b");
      });
      expect(calls.remove.mock.calls).toEqual([["array.a"], ["array.c"]]);
      expect(calls.set).not.toHaveBeenCalled();
    });
  });

  describe("keyed list edits", () => {
    it("should call onChange with the list path for an insert", () => {
      const onChange = vi.fn();
      const { result } = renderHook(() =>
        Form.use<typeof basicFormSchema>({
          values: deep.copy(initialFormValues),
          onChange,
        }),
      );
      act(() => result.current.insert("array", { key: "key2", name: "Jane" }, "key1"));
      expect(result.current.value().array.map(({ key }) => key)).toEqual([
        "key1",
        "key2",
      ]);
      expect(onChange).toHaveBeenCalledWith(
        expect.objectContaining({ path: "array", prev: initialFormValues.array }),
      );
    });

    it("should move an entry", () => {
      const { result } = renderHook(() =>
        Form.use<typeof basicFormSchema>({ values: deep.copy(initialFormValues) }),
      );
      act(() => {
        result.current.insert("array", { key: "key2", name: "Jane" });
        result.current.move("array", "key1");
      });
      expect(result.current.value().array.map(({ key }) => key)).toEqual([
        "key1",
        "key2",
      ]);
    });

    it("should call onChange once for each entry that a list remove drops", () => {
      const onChange = vi.fn();
      const { result } = renderHook(() => {
        const form = Form.use<typeof basicFormSchema>({
          values: deep.copy(initialFormValues),
          onChange,
        });
        const utils = Form.fieldListUtils<string, record.KeyedNamed>(form, "array");
        return { form, utils };
      });
      act(() => result.current.utils.push({ key: "key2", name: "Jane" }));
      onChange.mockClear();
      act(() => {
        result.current.utils.remove(["key1", "key2"]);
      });
      expect(result.current.form.value().array).toEqual([]);
      expect(onChange.mock.calls.map(([{ path }]) => path)).toEqual([
        "array.key1",
        "array.key2",
      ]);
    });

    it("should validate once for all the edits in a batch", () => {
      let validations = 0;
      const schema = basicFormSchema.check(() => {
        validations++;
      });
      const { result } = renderHook(() =>
        Form.use({ values: deep.copy(initialFormValues), schema }),
      );
      validations = 0;
      act(() =>
        result.current.batch(() => {
          result.current.set("name", "Jane Doe");
          result.current.set("age", 50);
        }),
      );
      expect(validations).toBe(1);
      expect(result.current.get("age").value).toBe(50);
    });

    it("should notify once for a batch", () => {
      const { result } = renderHook(() =>
        Form.use({ values: deep.copy(initialFormValues) }),
      );
      const listener = vi.fn();
      result.current.bind(listener);
      act(() =>
        result.current.batch(() => {
          result.current.set("name", "Jane Doe");
          result.current.set("age", 50);
          result.current.remove("array.key1");
        }),
      );
      expect(listener).toHaveBeenCalledTimes(1);
    });

    it("should neither validate nor notify for a batch that changes nothing", () => {
      let validations = 0;
      const schema = basicFormSchema.check(() => {
        validations++;
      });
      const { result } = renderHook(() =>
        Form.use({ values: deep.copy(initialFormValues), schema }),
      );
      const listener = vi.fn();
      result.current.bind(listener);
      validations = 0;
      act(() => result.current.batch(() => {}));
      expect(validations).toBe(0);
      expect(listener).not.toHaveBeenCalled();
    });
  });

  describe("store", () => {
    const renderWithStore = () => {
      const store = createLocalStore(deep.copy(initialFormValues));
      let renders = 0;
      const res = renderHook(() => {
        renders++;
        const form = Form.use({ store, schema: basicFormSchema });
        const age = Form.useFieldValue<number, number, typeof basicFormSchema>("age", {
          ctx: form,
        });
        return { form, age };
      });
      return { store, res, renders: () => renders };
    };

    it("should read the values that the store holds", () => {
      const { res } = renderWithStore();
      expect(res.result.current.age).toBe(42);
    });

    it("should write edits to the store", () => {
      const { store, res } = renderWithStore();
      act(() => res.result.current.form.set("age", 50));
      expect(store.get().age).toBe(50);
    });

    it("should re-render a field that another writer changes", () => {
      const { store, res } = renderWithStore();
      act(() => store.set("age", 50));
      expect(res.result.current.age).toBe(50);
    });

    it("should read another store that replaces the first", () => {
      const first = createLocalStore(deep.copy(initialFormValues));
      const second = createLocalStore({ ...deep.copy(initialFormValues), age: 7 });
      const onHasTouched = vi.fn();
      const { result, rerender } = renderHook(
        ({ store }) => {
          const form = Form.use({ store, schema: basicFormSchema, onHasTouched });
          const age = Form.useFieldValue<number, number, typeof basicFormSchema>(
            "age",
            {
              ctx: form,
            },
          );
          return { form, age };
        },
        { initialProps: { store: first } },
      );
      act(() => result.current.form.set("name", "Jane Doe"));
      rerender({ store: second });
      expect(result.current.age).toBe(7);
      expect(onHasTouched).toHaveBeenLastCalledWith(false);
      act(() => second.set("age", 8));
      expect(result.current.age).toBe(8);
      act(() => first.set("age", 9));
      expect(result.current.age).toBe(8);
    });

    it("should not re-render for a change that no read field reaches", () => {
      const { store, renders } = renderWithStore();
      const before = renders();
      act(() => store.set("nested.ssn", "000-00-0000"));
      expect(renders()).toBe(before);
    });
  });

  describe("reset functionality", () => {
    describe("reset()", () => {
      it("should reset all form values to initial values", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
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
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
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
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );

        // Set invalid values to trigger validation errors
        result.current.set("name", "Billy Bob"); // Required field
        result.current.set("age", 3); // Below minimum

        expect(result.current.get("name").status.variant).toBe("warning");
        expect(result.current.get("age").status.variant).toBe("error");

        result.current.reset();

        expect(result.current.get("name").status.variant).toBe("success");
        expect(result.current.get("age").status.variant).toBe("success");
      });

      it("should clear all touched states when resetting", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
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
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );

        result.current.set("nested.ssn", "999-99-9999");
        result.current.set("nested.ein", "98-7654321");

        expect(result.current.get("nested.ssn").value).toBe("999-99-9999");
        expect(result.current.get("nested.ein").value).toBe("98-7654321");

        result.current.reset();

        expect(result.current.get("nested.ssn").value).toBe("123-45-6789");
        expect(result.current.get("nested.ein").value).toBe("");
      });

      it("should handle resetting array values correctly", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );

        result.current.set("array", [
          { key: "key1", name: "Modified Name" },
          { key: "key2", name: "New Item" },
        ]);

        expect(result.current.get("array").value).toHaveLength(2);
        expect(result.current.get("array.0.name").value).toBe("Modified Name");

        result.current.reset();

        expect(result.current.get("array").value).toHaveLength(1);
        expect(result.current.get("array.0.name").value).toBe("John Doe");
        expect(result.current.get("array.0.key").value).toBe("key1");
      });
    });

    describe("setCurrentStateAsInitialValues()", () => {
      it("should set current form state as new initial values", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );

        result.current.set("name", "Jane Doe");
        result.current.set("age", 25);

        result.current.setCurrentStateAsInitialValues();

        expect(result.current.get("name").value).toBe("Jane Doe");
        expect(result.current.get("age").value).toBe(25);

        // But touched states should be cleared
        expect(result.current.get("name").touched).toBe(false);
        expect(result.current.get("age").touched).toBe(false);

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

        result.current.set("name", "Jane Doe");
        expect(onHasTouched).toHaveBeenLastCalledWith(true);

        result.current.setCurrentStateAsInitialValues();

        expect(onHasTouched).toHaveBeenLastCalledWith(false);
      });

      it("should handle nested and array values when setting current state as initial", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );

        result.current.set("nested.ssn", "999-99-9999");
        result.current.set("array.0.name", "New Array Name");

        result.current.setCurrentStateAsInitialValues();

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
        Form.use({
          values: deep.copy(initialFormValues),
          schema: basicFormSchema,
        }),
      );
      result.current.set("name", "Jane Doe");
      const field = result.current.get("name");
      expect(field.touched).toBe(true);
    });

    describe("markTouched option", () => {
      it("should not mark field as touched when markTouched is false", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        result.current.set("name", "Jane Doe", { markTouched: false });
        const field = result.current.get("name");
        expect(field.touched).toBe(false);
      });

      it("should explicitly mark field as touched when markTouched is true", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
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
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        result.current.set("nested.ssn", "999-99-9999", { markTouched: false });
        expect(result.current.get("nested.ssn").touched).toBe(false);

        result.current.set("nested.ein", "12-3456789", { markTouched: true });
        expect(result.current.get("nested.ein").touched).toBe(true);
      });

      it("should work with array fields and markTouched option", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        result.current.set("array.0.name", "Modified Name", { markTouched: false });
        expect(result.current.get("array.0.name").touched).toBe(false);

        result.current.set("array.0.key", "modifiedKey", { markTouched: true });
        expect(result.current.get("array.0.key").touched).toBe(true);
      });

      it("should still clear touched when value equals initial regardless of markTouched", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        result.current.set("name", "Jane Doe", { markTouched: true });
        expect(result.current.get("name").touched).toBe(true);
        result.current.set("name", "John Doe", { markTouched: true });
        expect(result.current.get("name").touched).toBe(false);
      });

      it("should affect validation when markTouched is false", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        result.current.set("age", 3, { markTouched: false });
        const isValid = result.current.validate();
        expect(isValid).toBe(false);
        expect(result.current.get("age").status.variant).toBe("error");
      });

      it("should trigger validation when markTouched is true and value is invalid", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        result.current.set("age", 3, { markTouched: true });
        const isValid = result.current.validate();
        expect(isValid).toBe(false);
        expect(result.current.get("age").status.variant).toBe("error");
      });

      it("should handle multiple fields with different markTouched settings", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
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
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
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
        Form.use({
          values: deep.copy(initialFormValues),
          schema: basicFormSchema,
        }),
      );
      result.current.set("name", "John Doe");
      const field = result.current.get("name");
      expect(field.touched).toBe(false);
    });

    it("should mark a field as untouched when resetting to initial value", () => {
      const { result } = renderHook(() =>
        Form.use({
          values: deep.copy(initialFormValues),
          schema: basicFormSchema,
        }),
      );
      result.current.set("name", "Jane Doe");
      expect(result.current.get("name").touched).toBe(true);
      result.current.set("name", "John Doe");
      expect(result.current.get("name").touched).toBe(false);
    });

    it("should clear all touched states when resetting the form", () => {
      const { result } = renderHook(() =>
        Form.use({
          values: deep.copy(initialFormValues),
          schema: basicFormSchema,
        }),
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
        Form.use({
          values: deep.copy(initialFormValues),
          schema: basicFormSchema,
        }),
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
        Form.use({
          values: deep.copy(initialFormValues),
          schema: basicFormSchema,
        }),
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
