// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement } from "react";

import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Form } from "@/form";

const basicFormSchema = z.object({
  name: z.string(),
  age: z.number(),
});

const FormContainer = (props: PropsWithChildren): ReactElement => {
  const methods = Form.use<typeof basicFormSchema>({
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
          Form.use<typeof basicFormSchema>({
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
          Form.use<typeof basicFormSchema>({
            values: { name: "John Doe", age: 42 },
            schema: basicFormSchema,
          }),
        );
        expect(() => result.current.get({ path: "ssn", optional: false })).toThrow();
      });
      it("should return null if optional is true and the field is null", () => {
        const { result } = renderHook(() =>
          Form.use<typeof basicFormSchema>({
            values: { name: "John Doe", age: 42 },
            schema: basicFormSchema,
          }),
        );
        const field = result.current.get({ path: "ssn", optional: true });
        expect(field).toBeNull();
      });
    });
    describe("set", () => {
      it("should set a value in the form", () => {
        const { result } = renderHook(() =>
          Form.use<typeof basicFormSchema>({
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
          Form.use<typeof basicFormSchema>({
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
  });
});
