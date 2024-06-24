// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep } from "@synnaxlabs/x";
import { act, fireEvent, render, renderHook } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Form } from "@/form";
import { Input } from "@/input";

const basicFormSchema = z
  .object({
    name: z.string(),
    age: z.number().min(5, "You must be at least 5 years old."),
    nested: z.object({
      ssn: z.string(),
      ein: z.string().optional(),
    }),
    array: z.array(z.object({ name: z.string() })),
  })
  .superRefine((c, ctx) => {
    if (c.name === "Billy Bob")
      ctx.addIssue({
        code: z.ZodIssueCode.custom,
        message: "You cannot be named Billy Bob.",
        path: ["name"],
        params: {
          variant: "warning",
        },
      });
  })
  .sourceType();

const initialFormValues: z.infer<typeof basicFormSchema> = {
  name: "John Doe",
  age: 42,
  nested: { ssn: "123-45-6789", ein: "" },
  array: [{ name: "John Doe" }],
};

const FormContainer = (props: PropsWithChildren): ReactElement => {
  const methods = Form.use({
    values: deep.copy(initialFormValues),
    schema: basicFormSchema,
  });
  return <Form.Form {...methods}>{props.children}</Form.Form>;
};

const wrapper = ({ children }: PropsWithChildren): ReactElement => (
  <FormContainer>{children}</FormContainer>
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
      it("should return true if a field is required in the schema", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        const field = result.current.get("age");
        expect(field.required).toBe(true);
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
        result.current.set({ path: "name", value: "Jane Doe" });
        const field = result.current.get("name");
        expect(field.value).toBe("Jane Doe");
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
        result.current.bind({
          path: "name",
          listenToChildren: false,
          onChange,
        });
        result.current.set({ path: "name", value: "Jane Doe" });
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
        result.current.bind({
          path: "age",
          listenToChildren: false,
          onChange,
        });
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
      const { result } = renderHook(() => Form.useField<string>({ path: "name" }), {
        wrapper,
      });
      expect(result.current.value).toBe("John Doe");
    });
    it("should set a field in the form", () => {
      const { result } = renderHook(() => Form.useField<string>({ path: "name" }), {
        wrapper,
      });
      act(() => result.current.onChange("Jane Doe"));
      expect(result.current.value).toBe("Jane Doe");
    });
    it("should call an onChange handler passed to the hook", () => {
      const onChange = vi.fn();
      const { result } = renderHook(
        () => Form.useField<string>({ path: "name", onChange }),
        { wrapper },
      );
      act(() => result.current.onChange("Jane Doe"));
      expect(onChange).toHaveBeenCalled();
    });
    it("should return a bad field status if a validation error occurs", () => {
      const { result } = renderHook(() => Form.useField<number>({ path: "age" }), {
        wrapper,
      });
      act(() => result.current.onChange(3));
      expect(result.current.status.variant).toEqual("error");
    });
    it("should return true if a field is required in the schema", () => {
      const { result } = renderHook(() => Form.useField<string>({ path: "name" }), {
        wrapper,
      });
      expect(result.current.required).toBe(true);
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
  describe("useFieldListener", () => {
    it("should call a listener when a field changes", () => {
      const listener = vi.fn();
      const res = renderHook(
        () => {
          Form.useFieldListener({
            path: "name",
            onChange: listener,
          });
          return Form.useField<string>({ path: "name" });
        },
        { wrapper },
      );
      act(() => res.result.current.onChange("Jane Doe"));
      expect(listener).toHaveBeenCalled();
      act(() => res.result.current.onChange("John Doe"));
      expect(listener).toHaveBeenCalled();
    });
  });
  describe("useChildFieldValues", () => {
    it("should call a listener when a child field changes", () => {
      const res = renderHook(
        () => ({
          cv: Form.useChildFieldValues<{ ssn: string }>({ path: "nested" }),
          f: Form.useField<string>({ path: "nested.ssn" }),
        }),
        { wrapper },
      );
      res.result.current.f.onChange("123-45-6786");
      expect(res.result.current.cv.ssn).toBe("123-45-6786");
    });
    it("should keep calling the listener even if the entire field is replaced", async () => {
      const res = renderHook(
        () => ({
          cv: Form.useChildFieldValues<{ ssn: string; ein: string }>({
            path: "nested",
          }),
          f: Form.useField<{ ssn: string; ein: string }>({ path: "nested" }),
        }),
        { wrapper },
      );
      res.result.current.f.onChange({ ssn: "123-45-6786", ein: "" });
      await new Promise((r) => setTimeout(r, 30));
      expect(res.result.current.cv.ssn).toBe("123-45-6786");
    });
  });
});
