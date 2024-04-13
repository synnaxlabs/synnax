// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement } from "react";

import { act, fireEvent, render, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Form } from "@/form";
import { Input } from "@/input";

const basicFormSchema = z.object({
  name: z.string(),
  age: z.number().min(5, "You must be at least 5 years old."),
});

const FormContainer = (props: PropsWithChildren): ReactElement => {
  const methods = Form.use({
    values: { name: "John Doe", age: 42 },
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
            values: { name: "John Doe", age: 42 },
            schema: basicFormSchema,
          }),
        );
        const field = result.current.get({ path: "name" });
        expect(field.value).toBe("John Doe");
        expect(field.status.variant).toEqual("success");
      });
      it("should throw an error if optional is false and the field is null", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: { name: "John Doe", age: 42 },
            schema: basicFormSchema,
          }),
        );
        expect(() => result.current.get({ path: "ssn", optional: false })).toThrow();
      });
      it("should return null if optional is true and the field is null", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: { name: "John Doe", age: 42 },
            schema: basicFormSchema,
          }),
        );
        const field = result.current.get({ path: "ssn", optional: true });
        expect(field).toBeNull();
      });
      it("should return true if a field is required in the schema", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: { name: "John Doe", age: 42 },
            schema: basicFormSchema,
          }),
        );
        const field = result.current.get({ path: "age" });
        expect(field.required).toBe(true);
      });
    });
    describe("set", () => {
      it("should set a value in the form", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: { name: "John Doe", age: 42 },
            schema: basicFormSchema,
          }),
        );
        result.current.set({ path: "name", value: "Jane Doe" });
        const field = result.current.get({ path: "name" });
        expect(field.value).toBe("Jane Doe");
      });
    });
    describe("bind", () => {
      it("should bind a listener for form changes", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: { name: "John Doe", age: 42 },
            schema: basicFormSchema,
          }),
        );
        const listener = vi.fn();
        result.current.bind({
          path: "name",
          listenToChildren: false,
          listener,
        });
        result.current.set({ path: "name", value: "Jane Doe" });
        expect(listener).toHaveBeenCalled();
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
    it("should apply a default value if the field is null", () => {
      const { result } = renderHook(
        () => Form.useField<string>({ path: "ssn", defaultValue: "123-45-6789" }),
        { wrapper },
      );
      expect(result.current.value).toBe("123-45-6789");
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
      expect(c.getByText("You must be at least 5 years old.")).toBeTruthy();
    });
  });
});
