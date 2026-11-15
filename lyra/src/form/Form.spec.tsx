// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep, type record, TimeSpan } from "@synnaxlabs/x";
import { act, fireEvent, render, renderHook, screen } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { renderToString } from "react-dom/server";
import { beforeAll, describe, expect, it, vi } from "vitest";
import { z } from "zod";

import { Form } from "@/form";
import { createLocalStore } from "@/form/store";
import { Input } from "@/input";
import { loadLanguage } from "@/input/time/suggest";
import { mockBoundingClientRect } from "@/testutil/dom";

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

const path = deep.path<z.infer<typeof basicFormSchema>>();
const verdict = deep.path<z.infer<typeof verdictSchema>>();
// Reaches fields that the schema does not declare.
const loose = deep.path<unknown>();

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
        const field = result.current.get(path.name);
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
        const field = result.current.get(path.nested.ssn);
        expect(field.value).toBe("123-45-6789");
      });

      it("should throw an error if optional is false and the field is null", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        expect(() => result.current.get(loose.ssn)).toThrow(
          "Field ssn is not in the form",
        );
      });

      it("should return null if optional is true and the field is null", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        const field = result.current.get(loose.ssn, { optional: true });
        expect(field).toBeNull();
      });

      it("should return true if the schema refuses the field empty", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        expect(result.current.get(path.name).required).toBe(true);
      });

      it("should return false for a number, which no input can empty", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        expect(result.current.get(path.age).required).toBe(false);
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
        result.current.set(path.name, "Jane Doe");
        const field = result.current.get(path.name);
        expect(field.value).toBe("Jane Doe");
      });

      it("should correctly set all values in the form at once", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        result.current.set(path, {
          name: "Jane Doe",
          age: 43,
          nested: { ssn: "123-45-6786", ein: "" },
          array: [{ key: "key1", name: "Jane Doe" }],
        });
        expect(result.current.get(path.name).value).toBe("Jane Doe");
        expect(result.current.get(path.age).value).toBe(43);
        expect(result.current.get(path.nested.ssn).value).toBe("123-45-6786");
        expect(result.current.get(path.array[0].name).value).toBe("Jane Doe");
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
        result.current.set(path.name, "Jane Doe");
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
        expect(result.current.get(path.age).status.variant).toEqual("error");
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
    describe("addCheck", () => {
      const notJohn = z.string().refine((v) => v !== "John Doe", "Not John");
      const useBasicForm = () =>
        Form.use({ values: deep.copy(initialFormValues), schema: basicFormSchema });

      it("should show a check's issue on an untouched field", () => {
        const { result } = renderHook(useBasicForm);
        act(() => {
          result.current.addCheck(path.name, notJohn);
        });
        expect(result.current.get(path.name).status).toMatchObject({
          variant: "error",
          message: "Not John",
        });
      });
      it("should not make the form invalid", () => {
        const { result } = renderHook(useBasicForm);
        act(() => {
          result.current.addCheck(path.name, notJohn);
        });
        expect(result.current.validate()).toBe(true);
      });
      it("should check again on every set", () => {
        const { result } = renderHook(useBasicForm);
        act(() => {
          result.current.addCheck(path.name, notJohn);
        });
        act(() => result.current.set(path.name, "Jane Doe"));
        expect(result.current.get(path.name).status.variant).toBe("success");
      });
      it("should check again when another client changes the value", () => {
        const store = createLocalStore(deep.copy(initialFormValues));
        const { result } = renderHook(() =>
          Form.use({ store, schema: basicFormSchema }),
        );
        act(() => {
          result.current.addCheck(path.name, notJohn);
        });
        act(() => store.set("name", "Jane Doe"));
        expect(result.current.get(path.name).status.variant).toBe("success");
        act(() => store.set("name", "John Doe"));
        expect(result.current.get(path.name).status.message).toBe("Not John");
      });
      it("should check the values of a new store", () => {
        const first = createLocalStore(
          deep.copy({ ...initialFormValues, name: "Jane Doe" }),
        );
        const second = createLocalStore(deep.copy(initialFormValues));
        const { result, rerender } = renderHook(
          ({ store }) => Form.use({ store, schema: basicFormSchema }),
          { initialProps: { store: first } },
        );
        act(() => {
          result.current.addCheck(path.name, notJohn);
        });
        expect(result.current.get(path.name).status.variant).toBe("success");
        rerender({ store: second });
        expect(result.current.get(path.name).status.message).toBe("Not John");
      });
      it("should check the values a reset gives", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy({ ...initialFormValues, name: "Jane Doe" }),
            schema: basicFormSchema,
          }),
        );
        act(() => {
          result.current.addCheck(path.name, notJohn);
        });
        act(() => result.current.reset(deep.copy(initialFormValues)));
        expect(result.current.get(path.name).status.message).toBe("Not John");
      });
      it("should check again after a remove", () => {
        const { result } = renderHook(useBasicForm);
        act(() => {
          result.current.addCheck(
            path,
            z.object({ optionalField: z.literal("").optional() }),
          );
        });
        act(() => result.current.set(path.optionalField, "set"));
        expect(result.current.get(path.optionalField).status.variant).toBe("error");
        act(() => result.current.remove(path.optionalField));
        expect(result.current.getStatuses()).toEqual([]);
      });
      it("should clear its issues once removed", () => {
        const { result } = renderHook(useBasicForm);
        let remove = () => {};
        act(() => {
          remove = result.current.addCheck(path.name, notJohn);
        });
        act(() => remove());
        expect(result.current.get(path.name).status.variant).toBe("success");
      });
      it("should keep an error over a later warning on the same field", () => {
        const { result } = renderHook(useBasicForm);
        const warning = z.string().check(({ value, issues }) => {
          issues.push({
            code: "custom",
            message: "Odd name",
            input: value,
            params: { variant: "warning" },
          });
        });
        act(() => {
          result.current.addCheck(path.name, notJohn);
          result.current.addCheck(path.name, warning);
        });
        expect(result.current.get(path.name).status.message).toBe("Not John");
      });
    });
  });

  describe("useCheck", () => {
    const notJohn = z.string().refine((v) => v !== "John Doe", "Not John");

    it("should check the field while mounted", () => {
      const { result, unmount } = renderHook(
        () => {
          Form.useCheck(path.name, notJohn);
          return Form.useContext();
        },
        { wrapper },
      );
      expect(result.current.get(path.name).status.message).toBe("Not John");
      const ctx = result.current;
      unmount();
      expect(ctx.get(path.name).status.variant).toBe("success");
    });
    it("should replace the check when the schema changes", () => {
      const notJane = z.string().refine((v) => v !== "John Doe", "Not Jane");
      const { result, rerender } = renderHook(
        ({ schema }: { schema: z.ZodType }) => {
          Form.useCheck(path.name, schema);
          return Form.useContext();
        },
        { wrapper, initialProps: { schema: notJohn } },
      );
      rerender({ schema: notJane });
      expect(result.current.get(path.name).status.message).toBe("Not Jane");
    });
  });

  describe("focus", () => {
    const NO_FOCUS = "Field name does not take focus";

    it("should run the handler bound to the path", () => {
      const handler = vi.fn();
      const { result } = renderHook(
        () => {
          Form.useFocusHandler(path.name, handler);
          return Form.useContext();
        },
        { wrapper },
      );
      result.current.focus(path.name);
      expect(handler).toHaveBeenCalledTimes(1);
    });
    it("should run the newest handler after a render", () => {
      const first = vi.fn();
      const second = vi.fn();
      const { result, rerender } = renderHook(
        ({ handler }: { handler: () => void }) => {
          Form.useFocusHandler(path.name, handler);
          return Form.useContext();
        },
        { wrapper, initialProps: { handler: first } },
      );
      rerender({ handler: second });
      result.current.focus(path.name);
      expect(first).not.toHaveBeenCalled();
      expect(second).toHaveBeenCalledTimes(1);
    });
    it("should throw when no field at the path takes focus", () => {
      const { result } = renderHook(() => Form.useContext(), { wrapper });
      expect(() => result.current.focus(path.name)).toThrow(NO_FOCUS);
    });
    it("should stop running the handler once unmounted", () => {
      const handler = vi.fn();
      const { result, unmount } = renderHook(
        () => {
          Form.useFocusHandler(path.name, handler);
          return Form.useContext();
        },
        { wrapper },
      );
      const ctx = result.current;
      unmount();
      expect(() => ctx.focus(path.name)).toThrow(NO_FOCUS);
      expect(handler).not.toHaveBeenCalled();
    });
    it("should keep a newer handler when an older one is removed", () => {
      const { result } = renderHook(() => Form.useContext(), { wrapper });
      const older = vi.fn();
      const newer = vi.fn();
      const removeOlder = result.current.addFocusHandler(path.name, older);
      result.current.addFocusHandler(path.name, newer);
      removeOlder();
      result.current.focus(path.name);
      expect(older).not.toHaveBeenCalled();
      expect(newer).toHaveBeenCalledTimes(1);
    });
  });

  describe("useField", () => {
    it("should get a field from the form", () => {
      const { result } = renderHook(() => Form.useField(path.name), {
        wrapper,
      });
      expect(result.current.value).toBe("John Doe");
    });
    it("should set a field in the form", () => {
      const { result } = renderHook(() => Form.useField(path.name), {
        wrapper,
      });
      act(() => result.current.onChange("Jane Doe"));
      expect(result.current.value).toBe("Jane Doe");
    });
    it("should call an onChange handler passed to the hook", () => {
      const onChange = vi.fn();
      const { result } = renderHook(() => Form.useField(path.name, { onChange }), {
        wrapper,
      });
      act(() => result.current.onChange("Jane Doe"));
      expect(onChange).toHaveBeenCalled();
    });

    it("should return a bad field status if a validation error occurs", () => {
      const { result, rerender } = renderHook(() => Form.useField(path.age), {
        wrapper,
      });
      act(() => result.current.onChange(3));
      rerender();
      expect(result.current.status.variant).toEqual("error");
    });

    it("should still allow the caller to set the field value even if a validation error occurs", () => {
      const { result } = renderHook(() => Form.useField(path.name), { wrapper });
      act(() => result.current.onChange(""));
      expect(result.current.value).toBe("");
    });

    it("should return true if a field is required in the schema", () => {
      const { result } = renderHook(() => Form.useField(path.name), { wrapper });
      expect(result.current.required).toBe(true);
    });

    it("should show the schema default for an absent field without writing it", () => {
      const { result } = renderHook(
        () => {
          const field = Form.useField(verdict.defaulted);
          const { value } = Form.useContext();
          return { field, stored: "defaulted" in (value() as object) };
        },
        { wrapper: verdictWrapper },
      );
      expect(result.current.field.value).toBe("cat");
      expect(result.current.stored).toBe(false);
    });

    it("should read an absent optional field as undefined", () => {
      const { result } = renderHook(() => Form.useField(verdict.optionalField), {
        wrapper: verdictWrapper,
      });
      expect(result.current.value).toBeUndefined();
      expect(result.current.required).toBe(false);
    });
  });

  describe("useFieldState", () => {
    it("should get the full field state from the form", () => {
      const { result } = renderHook(() => Form.useFieldState(path.name), {
        wrapper,
      });
      expect(result.current?.value).toBe("John Doe");
      expect(result.current?.status.variant).toBe("success");
      expect(result.current?.touched).toBe(false);
      expect(result.current?.required).toBe(true);
    });

    it("should return the correct nested field state", () => {
      const { result } = renderHook(() => Form.useFieldState(path.nested.ssn), {
        wrapper,
      });
      expect(result.current?.value).toBe("123-45-6789");
      expect(result.current?.status.variant).toBe("success");
      expect(result.current?.required).toBe(true);
    });

    it("should return null for optional fields when they don't exist", () => {
      const { result } = renderHook(
        () => Form.useFieldState(loose.nonExistentField, { optional: true }),
        { wrapper },
      );
      expect(result.current).toBeNull();
    });

    it("should use the schema default when the field is absent", () => {
      const { result } = renderHook(() => Form.useFieldState(verdict.defaulted), {
        wrapper: verdictWrapper,
      });
      expect(result.current?.value).toBe("cat");
    });

    it("should correctly identify required vs optional fields", () => {
      const { result: requiredResult } = renderHook(
        () => Form.useFieldState(path.name),
        { wrapper },
      );
      const { result: optionalResult } = renderHook(
        () => Form.useFieldState(path.optionalField, { optional: true }),
        { wrapper },
      );

      expect(requiredResult.current?.required).toBe(true);
      expect(optionalResult.current?.required).toBe(false);
    });
  });

  describe("useFieldValue", () => {
    it("should get just the value from a field", () => {
      const { result } = renderHook(() => Form.useFieldValue(path.name), {
        wrapper,
      });
      expect(result.current).toBe("John Doe");
    });

    it("should return the correct nested field value", () => {
      const { result } = renderHook(() => Form.useFieldValue(path.nested.ssn), {
        wrapper,
      });
      expect(result.current).toBe("123-45-6789");
    });

    it("should return null for optional fields when they don't exist", () => {
      const { result } = renderHook(
        () => Form.useFieldValue(loose.nonExistentField, { optional: true }),
        { wrapper },
      );
      expect(result.current).toBeNull();
    });

    it("should use the schema default when the field is absent", () => {
      const { result } = renderHook(() => Form.useFieldValue(verdict.defaulted), {
        wrapper: verdictWrapper,
      });
      expect(result.current).toBe("cat");
    });

    it("should return array values correctly", () => {
      const { result } = renderHook(() => Form.useFieldValue(path.array), {
        wrapper,
      });
      expect(result.current).toEqual([{ key: "key1", name: "John Doe" }]);
    });

    it("should return array element values correctly", () => {
      const { result } = renderHook(() => Form.useFieldValue(path.array[0].name), {
        wrapper,
      });
      expect(result.current).toBe("John Doe");
    });

    it("should return complex nested object values correctly", () => {
      const { result } = renderHook(() => Form.useFieldValue(path.nested), {
        wrapper,
      });
      expect(result.current).toEqual({ ssn: "123-45-6789", ein: "" });
    });

    it("should update a parent value state when a child value changes", async () => {
      const { result } = renderHook(
        () => {
          const child = Form.useField(path.nested.ssn);
          const parent = Form.useField(path.nested);
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
          const parent = Form.useFieldValue(path.array);
          return { ctx, parent };
        },
        {
          wrapper,
        },
      );
      act(() => result.current.ctx.set(path.array.key1.name, "Cat"));
      expect(result.current.parent).toEqual([{ key: "key1", name: "Cat" }]);
    });
  });

  describe("useFieldValid", () => {
    it("should return true for valid fields", () => {
      const { result } = renderHook(() => Form.useFieldValid(path.name), {
        wrapper,
      });
      expect(result.current).toBe(true);
    });

    it("should return false for non-existent optional fields", () => {
      const { result } = renderHook(() => Form.useFieldValid(loose.nonExistentField), {
        wrapper,
      });
      expect(result.current).toBe(false);
    });

    it("should work with nested fields", () => {
      const { result } = renderHook(() => Form.useFieldValid(path.nested.ssn), {
        wrapper,
      });
      expect(result.current).toBe(true);
    });

    it("should work with array fields", () => {
      const { result } = renderHook(() => Form.useFieldValid(path.array[0].name), {
        wrapper,
      });
      expect(result.current).toBe(true);
    });
  });

  describe("useStatuses", () => {
    const renderStatuses = (path: deep.Target<unknown>) =>
      renderHook(() => ({ ctx: Form.useContext(), statuses: Form.useStatuses(path) }), {
        wrapper,
      });

    it("should read the statuses at and under the path", () => {
      const { result } = renderStatuses(path.nested);
      act(() => {
        result.current.ctx.setStatus(path.nested.ssn, {
          variant: "error",
          message: "SSN is taken",
        });
        result.current.ctx.setStatus(path.name, { variant: "error", message: "Taken" });
      });
      expect(result.current.statuses).toEqual([
        { key: "nested.ssn", variant: "error", message: "SSN is taken" },
      ]);
    });

    it("should clear the status at one path and keep the others", () => {
      const { result } = renderStatuses(path.nested);
      act(() => {
        result.current.ctx.setStatus(path.nested.ssn, {
          variant: "error",
          message: "A",
        });
        result.current.ctx.setStatus(path.nested.ein, {
          variant: "error",
          message: "B",
        });
      });
      act(() => result.current.ctx.clearStatus(path.nested.ssn));
      expect(result.current.statuses).toEqual([
        { key: "nested.ein", variant: "error", message: "B" },
      ]);
    });

    it("should leave out success statuses", () => {
      const { result } = renderHook(
        () => {
          Form.useFieldState(path.nested.ssn);
          return Form.useStatuses(path.nested);
        },
        { wrapper },
      );
      expect(result.current).toEqual([]);
    });

    it("should keep its result when a status outside the path changes", () => {
      const { result } = renderStatuses(path.nested);
      act(() =>
        result.current.ctx.setStatus(path.nested.ssn, {
          variant: "error",
          message: "Bad",
        }),
      );
      const before = result.current.statuses;
      act(() =>
        result.current.ctx.setStatus(path.name, { variant: "error", message: "" }),
      );
      expect(result.current.statuses).toBe(before);
    });
  });

  describe("Field", () => {
    it("should render its value on the server", () => {
      const html = renderToString(
        <FormContainer>
          <Form.Field path={path.name} />
        </FormContainer>,
      );
      expect(html).toContain('value="John Doe"');
    });
    it("should return a text field with the correct value", () => {
      const c = render(<Form.Field path={path.name} />, { wrapper });
      expect(c.getByDisplayValue("John Doe")).toBeTruthy();
    });
    it("should render the correct label", () => {
      const c = render(<Form.Field path={path.name} label="Full Name" />, { wrapper });
      expect(c.getByText("Full Name")).toBeTruthy();
    });
    it("should capitalize the last part of the path if the label is not provided", () => {
      const c = render(<Form.Field path={path.name} />, { wrapper });
      expect(c.getByText("Name")).toBeTruthy();
    });
    it("should render a required indicator if the field is required", () => {
      const c = render(<Form.Field path={path.name} />, { wrapper });
      expect(c.getByText("*")).toBeTruthy();
    });
    it("should set the value of the field when the input changes", () => {
      const c = render(<Form.Field path={path.name} />, { wrapper });
      const input = c.getByDisplayValue("John Doe");
      fireEvent.change(input, { target: { value: "Jane Doe" } });
      expect(c.getByDisplayValue("Jane Doe")).toBeTruthy();
    });
    it("should render nothing when visible is false", () => {
      const c = render(<Form.Field path={path.name} visible={false} />, { wrapper });
      expect(c.queryByDisplayValue("John Doe")).toBeNull();
    });
    it("should allow a hidden field to bind an absent path", () => {
      const c = render(<Form.Field path={loose.absent} visible={false} />, { wrapper });
      expect(c.container.innerHTML).toBe("");
    });
    it("should render help text if validation on the field fails", () => {
      const c = render(
        <Form.Field path={path.age}>{(p) => <Input.Numeric {...p} />}</Form.Field>,
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
        const c = render(<Form.Field path={path.name} />, { wrapper });
        expect(c.getByText("*")).toBeTruthy();
      });
      it("should not mark a custom input", () => {
        const c = render(
          <Form.Field path={path.name}>{(p) => <Input.Text {...p} />}</Form.Field>,
          { wrapper },
        );
        expect(c.queryByText("*")).toBeNull();
      });
      it("should mark a custom input that is emptiable", () => {
        const c = render(
          <Form.Field path={path.name} emptiable>
            {(p) => <Input.Text {...p} />}
          </Form.Field>,
          { wrapper },
        );
        expect(c.getByText("*")).toBeTruthy();
      });
      it("should mark a text field", () => {
        const c = render(<Form.TextField path={path.name} />, { wrapper });
        expect(c.getByText("*")).toBeTruthy();
      });
      it("should not mark a numeric field", () => {
        const c = render(<Form.NumericField path={path.age} />, { wrapper });
        expect(c.queryByText("*")).toBeNull();
      });
    });
    describe("Nested Field", () => {
      it("should return a text field with the correct value", () => {
        const c = render(<Form.Field path={path.nested.ssn} />, { wrapper });
        expect(c.getByDisplayValue("123-45-6789")).toBeTruthy();
      });
      it("should mark the nested field as required if it is required in the schema", () => {
        const c = render(<Form.Field path={path.nested.ssn} />, { wrapper });
        expect(c.getByText("*")).toBeTruthy();
      });
      it("should not mark the nested field as required if it is not required in the schema", () => {
        const c = render(<Form.Field path={path.nested.ein} />, { wrapper });
        expect(c.queryByText("*")).toBeNull();
      });
    });
    describe("Array Field", () => {
      it("should return a text field with the correct value", () => {
        const c = render(<Form.Field path={path.array[0].name} />, { wrapper });
        expect(c.getByDisplayValue("John Doe")).toBeTruthy();
      });
      it("should mark the array field as required if it is required in the schema", () => {
        const c = render(<Form.Field path={path.array[0].name} />, { wrapper });
        expect(c.getByText("*")).toBeTruthy();
      });
      it("should not mark the array field as required if it is not required in the schema", () => {
        const c = render(<Form.Field path={path.array[0].key} />, { wrapper });
        expect(c.queryByText("*")).toBeNull();
      });
    });
  });

  describe("NumericField", () => {
    it("should return a numeric field with the correct value", () => {
      const c = render(<Form.NumericField path={path.age} />, { wrapper });
      expect(c.getByDisplayValue("42")).toBeTruthy();
    });
    it("should set the value of the field when the input changes", () => {
      const c = render(<Form.NumericField path={path.age} />, { wrapper });
      const input = c.getByDisplayValue("42");
      fireEvent.change(input, { target: { value: 43 } });
      expect(c.getByDisplayValue("43")).toBeTruthy();
    });
    it("should render help text if validation on the field fails", () => {
      const c = render(<Form.NumericField path={path.age} />, { wrapper });
      const input = c.getByDisplayValue("42");
      fireEvent.change(input, { target: { value: 1 } });
      fireEvent.blur(input);
      // we're executing an async parse so we need to wait for the error to show up
      expect(c.findByText("You must be at least 5 years old.")).toBeTruthy();
    });
  });

  describe("TimeSpanField", () => {
    beforeAll(loadLanguage);
    beforeAll(() => {
      Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
    });
    const spanSchema = z.object({
      timeoutMs: z.number(),
      keepAlive: z.number().optional(),
    });
    const span = deep.path<z.infer<typeof spanSchema>>();
    const SpanForm = (props: PropsWithChildren): ReactElement => {
      const methods = Form.use<typeof spanSchema>({
        values: { timeoutMs: 1500, keepAlive: 30 },
        schema: spanSchema,
      });
      return <Form.Form<typeof spanSchema> {...methods}>{props.children}</Form.Form>;
    };
    const edit = (input: HTMLElement, value: string): void => {
      fireEvent.focus(input);
      fireEvent.change(input, { target: { value } });
      fireEvent.blur(input);
    };

    it("should show and store the number in its unit", () => {
      const c = render(
        <Form.TimeSpanField path={span.timeoutMs} unit={TimeSpan.MILLISECOND} />,
        { wrapper: SpanForm },
      );
      const input = c.getByDisplayValue("1s 500ms");
      edit(input, "250");
      expect(c.getByDisplayValue("250ms")).toBeTruthy();
    });

    it("should name the bound a span crosses", () => {
      const c = render(
        <Form.TimeSpanField
          path={span.timeoutMs}
          label="Timeout"
          unit={TimeSpan.MILLISECOND}
          bounds={{ upper: TimeSpan.MINUTE }}
        />,
        { wrapper: SpanForm },
      );
      const input = c.getByDisplayValue("1s 500ms");
      fireEvent.focus(input);
      fireEvent.change(input, { target: { value: "5min" } });
      expect(c.getByText("Timeout must be 1min or less")).toBeTruthy();
    });

    it("should store undefined when an emptiable field is emptied", () => {
      const c = render(
        <Form.TimeSpanField
          path={span.keepAlive}
          unit={TimeSpan.SECOND}
          emptiable
          inputProps={{ placeholder: "30s" }}
        />,
        { wrapper: SpanForm },
      );
      edit(c.getByDisplayValue("30s"), "");
      expect(c.getByPlaceholderText("30s")).toHaveProperty("value", "");
    });
  });

  describe("CheckboxField", () => {
    const flagsSchema = z.object({
      enabled: z.boolean(),
      disabled: z.boolean(),
    });

    const flags = deep.path<z.infer<typeof flagsSchema>>();

    const flagsWrapper =
      (field?: ReactElement) =>
      ({ children }: PropsWithChildren): ReactElement => {
        const methods = Form.use<typeof flagsSchema>({
          values: { enabled: true, disabled: false },
          schema: flagsSchema,
        });
        return (
          <Form.Form<typeof flagsSchema> {...methods}>
            {field}
            {children}
          </Form.Form>
        );
      };

    it("should check the box when the stored value is true", () => {
      render(<Form.CheckboxField path={flags.enabled} />, { wrapper: flagsWrapper() });
      expect(
        screen.getByRole<HTMLInputElement>("checkbox", { name: "Enabled" }).checked,
      ).toBe(true);
    });

    it("should never show a required mark", () => {
      const c = render(<Form.CheckboxField path={flags.enabled} />, {
        wrapper: flagsWrapper(),
      });
      expect(c.queryByText("*")).toBeNull();
    });

    it("should check the box when an inverted flag is false", () => {
      render(<Form.CheckboxField path={flags.disabled} label="Saving" inverted />, {
        wrapper: flagsWrapper(),
      });
      expect(
        screen.getByRole<HTMLInputElement>("checkbox", { name: "Saving" }).checked,
      ).toBe(true);
    });

    it("should store the inverse of the box for an inverted flag", () => {
      const { result } = renderHook(() => Form.useFieldValue(flags.disabled), {
        wrapper: flagsWrapper(
          <Form.CheckboxField path={flags.disabled} label="Saving" inverted />,
        ),
      });
      expect(result.current).toBe(false);
      act(() => {
        fireEvent.click(screen.getByRole("checkbox", { name: "Saving" }));
      });
      expect(result.current).toBe(true);
    });
  });

  describe("useFieldLIst", () => {
    it("should return the array as the value", () => {
      const res = renderHook(() => Form.useFieldList(path.array), { wrapper });
      expect(res.result.current.data).toEqual(["key1"]);
    });
    it("should correctly push a value onto the start of the array", () => {
      const res = renderHook(() => Form.useFieldList(path.array), { wrapper });
      act(() => {
        res.result.current.push({ key: "key2", name: "Jane Doe" });
      });
      expect(res.result.current.data).toEqual(["key1", "key2"]);
    });

    it("should correctly remove the given index from the array", () => {
      const res = renderHook(() => Form.useFieldList(path.array), { wrapper });
      act(() => {
        res.result.current.remove("key1");
      });
      expect(res.result.current.data).toEqual([]);
    });

    it("should correctly keep only the given index in the array", () => {
      const res = renderHook(() => Form.useFieldList(path.array), { wrapper });
      act(() => {
        res.result.current.push({ key: "key2", name: "Jane Doe" });
      });
      act(() => {
        res.result.current.keepOnly("key2");
      });
      expect(res.result.current.data).toEqual(["key2"]);
    });
  });

  describe("useFieldListUtils", () => {
    it("should return current array values", () => {
      const { result } = renderHook(() => Form.useFieldListUtils(path.array), {
        wrapper,
      });
      expect(result.current.value()).toEqual([{ key: "key1", name: "John Doe" }]);
    });
    it("should push items to array", () => {
      const { result, rerender } = renderHook(
        () => {
          const utils = Form.useFieldListUtils(path.array);
          const field = Form.useFieldValue(path.array);
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
          const utils = Form.useFieldListUtils(path.array);
          const field = Form.useFieldValue(path.array);
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
          const utils = Form.useFieldListUtils(path.array);
          const field = Form.useFieldValue(path.array);
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
          const utils = Form.useFieldListUtils(path.array);
          const field = Form.useFieldValue(path.array);
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
          const utils = Form.useFieldListUtils(path.array);
          const field = Form.useFieldValue(path.array);
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
        Form.fieldListUtils(Form.use({ store }), path.array),
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

    it.each<[string, (utils: Form.FieldListUtils<record.KeyedNamed>) => void]>([
      ["remove", (utils) => utils.remove("1")],
      ["keepOnly", (utils) => utils.keepOnly("0")],
    ])("should %s by key when a key reads as an index", (_, edit) => {
      const store = createLocalStore({
        array: [
          { key: "1", name: "first" },
          { key: "0", name: "second" },
        ],
      });
      const { result } = renderHook(() =>
        Form.fieldListUtils(Form.use({ store }), path.array),
      );
      act(() => edit(result.current));
      expect(store.get().array).toEqual([{ key: "0", name: "second" }]);
    });

    it("should remove each dropped entry from the end of the list", () => {
      const { utils, calls } = renderOverStore();
      act(() => {
        utils().remove(["a", "c"]);
      });
      expect(calls.remove.mock.calls).toEqual([["array.2"], ["array.0"]]);
      expect(calls.set).not.toHaveBeenCalled();
    });

    it("should remove each entry that keepOnly does not keep", () => {
      const { utils, calls } = renderOverStore();
      act(() => {
        utils().keepOnly("b");
      });
      expect(calls.remove.mock.calls).toEqual([["array.2"], ["array.0"]]);
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
      act(() =>
        result.current.insert(path.array, { key: "key2", name: "Jane" }, "key1"),
      );
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
        result.current.insert(path.array, { key: "key2", name: "Jane" });
        result.current.move(path.array, "key1");
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
        const utils = Form.fieldListUtils(form, path.array);
        return { form, utils };
      });
      act(() => result.current.utils.push({ key: "key2", name: "Jane" }));
      onChange.mockClear();
      act(() => {
        result.current.utils.remove(["key1", "key2"]);
      });
      expect(result.current.form.value().array).toEqual([]);
      expect(onChange.mock.calls.map(([{ path }]) => path)).toEqual([
        "array.1",
        "array.0",
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
          result.current.set(path.name, "Jane Doe");
          result.current.set(path.age, 50);
        }),
      );
      expect(validations).toBe(1);
      expect(result.current.get(path.age).value).toBe(50);
    });

    it("should notify once for a batch", () => {
      const { result } = renderHook(() =>
        Form.use({ values: deep.copy(initialFormValues) }),
      );
      const listener = vi.fn();
      result.current.bind(listener);
      act(() =>
        result.current.batch(() => {
          result.current.set(path.name, "Jane Doe");
          result.current.set(path.age, 50);
          result.current.remove(path.array.key1);
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
        const age = Form.useFieldValue(path.age, { ctx: form });
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
      act(() => res.result.current.form.set(path.age, 50));
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
          const age = Form.useFieldValue(path.age, { ctx: form });
          return { form, age };
        },
        { initialProps: { store: first } },
      );
      act(() => result.current.form.set(path.name, "Jane Doe"));
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

        result.current.set(path.name, "Jane Doe");
        result.current.set(path.age, 25);
        result.current.set(path.nested.ssn, "987-65-4321");
        result.current.set(path.array[0].name, "Changed Name");

        expect(result.current.get(path.name).value).toBe("Jane Doe");
        expect(result.current.get(path.age).value).toBe(25);
        expect(result.current.get(path.nested.ssn).value).toBe("987-65-4321");
        expect(result.current.get(path.array[0].name).value).toBe("Changed Name");

        result.current.reset();

        expect(result.current.get(path.name).value).toBe("John Doe");
        expect(result.current.get(path.age).value).toBe(42);
        expect(result.current.get(path.nested.ssn).value).toBe("123-45-6789");
        expect(result.current.get(path.array[0].name).value).toBe("John Doe");
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

        result.current.set(path.name, "Temporary Name");
        expect(result.current.get(path.name).value).toBe("Temporary Name");

        result.current.reset(newValues);

        expect(result.current.get(path.name).value).toBe("New Name");
        expect(result.current.get(path.age).value).toBe(30);
        expect(result.current.get(path.nested.ssn).value).toBe("555-55-5555");
        expect(result.current.get(path.nested.ein).value).toBe("12-3456789");
        expect(result.current.get(path.array[0].name).value).toBe("New Array Name");
      });

      it("should clear all validation errors when resetting", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );

        // Set invalid values to trigger validation errors
        result.current.set(path.name, "Billy Bob"); // Required field
        result.current.set(path.age, 3); // Below minimum

        expect(result.current.get(path.name).status.variant).toBe("warning");
        expect(result.current.get(path.age).status.variant).toBe("error");

        result.current.reset();

        expect(result.current.get(path.name).status.variant).toBe("success");
        expect(result.current.get(path.age).status.variant).toBe("success");
      });

      it("should clear all touched states when resetting", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        result.current.set(path.name, "Jane Doe");
        result.current.set(path.age, 25);
        result.current.set(path.nested.ssn, "987-65-4321");
        expect(result.current.get(path.name).touched).toBe(true);
        expect(result.current.get(path.age).touched).toBe(true);
        expect(result.current.get(path.nested.ssn).touched).toBe(true);
        result.current.reset();
        expect(result.current.get(path.name).touched).toBe(false);
        expect(result.current.get(path.age).touched).toBe(false);
        expect(result.current.get(path.nested.ssn).touched).toBe(false);
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

        result.current.set(path.name, "Jane Doe");
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
        result.current.set(path.name, "Jane Doe");
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

        result.current.set(path.nested.ssn, "999-99-9999");
        result.current.set(path.nested.ein, "98-7654321");

        expect(result.current.get(path.nested.ssn).value).toBe("999-99-9999");
        expect(result.current.get(path.nested.ein).value).toBe("98-7654321");

        result.current.reset();

        expect(result.current.get(path.nested.ssn).value).toBe("123-45-6789");
        expect(result.current.get(path.nested.ein).value).toBe("");
      });

      it("should handle resetting array values correctly", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );

        result.current.set(path.array, [
          { key: "key1", name: "Modified Name" },
          { key: "key2", name: "New Item" },
        ]);

        expect(result.current.get(path.array).value).toHaveLength(2);
        expect(result.current.get(path.array[0].name).value).toBe("Modified Name");

        result.current.reset();

        expect(result.current.get(path.array).value).toHaveLength(1);
        expect(result.current.get(path.array[0].name).value).toBe("John Doe");
        expect(result.current.get(path.array[0].key).value).toBe("key1");
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

        result.current.set(path.name, "Jane Doe");
        result.current.set(path.age, 25);

        result.current.setCurrentStateAsInitialValues();

        expect(result.current.get(path.name).value).toBe("Jane Doe");
        expect(result.current.get(path.age).value).toBe(25);

        // But touched states should be cleared
        expect(result.current.get(path.name).touched).toBe(false);
        expect(result.current.get(path.age).touched).toBe(false);

        result.current.set(path.name, "Another Name");
        result.current.reset();
        expect(result.current.get(path.name).value).toBe("Jane Doe"); // New initial value
        expect(result.current.get(path.age).value).toBe(25); // New initial value
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

        result.current.set(path.name, "Jane Doe");
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

        result.current.set(path.nested.ssn, "999-99-9999");
        result.current.set(path.array[0].name, "New Array Name");

        result.current.setCurrentStateAsInitialValues();

        expect(result.current.get(path.nested.ssn).value).toBe("999-99-9999");
        expect(result.current.get(path.nested.ssn).touched).toBe(false);
        expect(result.current.get(path.array[0].name).value).toBe("New Array Name");
        expect(result.current.get(path.array[0].name).touched).toBe(false);

        // Changing back to original values should now mark as touched
        result.current.set(path.nested.ssn, "123-45-6789");
        expect(result.current.get(path.nested.ssn).touched).toBe(true);
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
      result.current.set(path.name, "Jane Doe");
      const field = result.current.get(path.name);
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
        result.current.set(path.name, "Jane Doe", { markTouched: false });
        const field = result.current.get(path.name);
        expect(field.touched).toBe(false);
      });

      it("should explicitly mark field as touched when markTouched is true", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        result.current.set(path.name, "Jane Doe", { markTouched: true });
        const field = result.current.get(path.name);
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
        result.current.set(path.name, "Jane Doe", { markTouched: false });
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
        result.current.set(path.name, "Jane Doe", { markTouched: true });
        expect(onHasTouched).toHaveBeenCalledWith(true);
      });

      it("should work with nested fields and markTouched option", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        result.current.set(path.nested.ssn, "999-99-9999", { markTouched: false });
        expect(result.current.get(path.nested.ssn).touched).toBe(false);

        result.current.set(path.nested.ein, "12-3456789", { markTouched: true });
        expect(result.current.get(path.nested.ein).touched).toBe(true);
      });

      it("should work with array fields and markTouched option", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        result.current.set(path.array[0].name, "Modified Name", { markTouched: false });
        expect(result.current.get(path.array[0].name).touched).toBe(false);

        result.current.set(path.array[0].key, "modifiedKey", { markTouched: true });
        expect(result.current.get(path.array[0].key).touched).toBe(true);
      });

      it("should still clear touched when value equals initial regardless of markTouched", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        result.current.set(path.name, "Jane Doe", { markTouched: true });
        expect(result.current.get(path.name).touched).toBe(true);
        result.current.set(path.name, "John Doe", { markTouched: true });
        expect(result.current.get(path.name).touched).toBe(false);
      });

      it("should affect validation when markTouched is false", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        result.current.set(path.age, 3, { markTouched: false });
        const isValid = result.current.validate();
        expect(isValid).toBe(false);
        expect(result.current.get(path.age).status.variant).toBe("error");
      });

      it("should trigger validation when markTouched is true and value is invalid", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        result.current.set(path.age, 3, { markTouched: true });
        const isValid = result.current.validate();
        expect(isValid).toBe(false);
        expect(result.current.get(path.age).status.variant).toBe("error");
      });

      it("should handle multiple fields with different markTouched settings", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        result.current.set(path.name, "Jane Doe", { markTouched: false });
        result.current.set(path.age, 30, { markTouched: true });
        result.current.set(path.nested.ssn, "111-11-1111", { markTouched: false });

        expect(result.current.get(path.name).touched).toBe(false);
        expect(result.current.get(path.age).touched).toBe(true);
        expect(result.current.get(path.nested.ssn).touched).toBe(false);
      });

      it("should preserve markTouched behavior through form operations", () => {
        const { result } = renderHook(() =>
          Form.use({
            values: deep.copy(initialFormValues),
            schema: basicFormSchema,
          }),
        );
        result.current.set(path.name, "Jane Doe", { markTouched: false });
        expect(result.current.get(path.name).touched).toBe(false);
        result.current.set(path.age, 30);
        expect(result.current.get(path.age).touched).toBe(true);
        expect(result.current.get(path.name).touched).toBe(false);
      });
    });

    it("should not mark a field as touched when setting it to its initial value", () => {
      const { result } = renderHook(() =>
        Form.use({
          values: deep.copy(initialFormValues),
          schema: basicFormSchema,
        }),
      );
      result.current.set(path.name, "John Doe");
      const field = result.current.get(path.name);
      expect(field.touched).toBe(false);
    });

    it("should mark a field as untouched when resetting to initial value", () => {
      const { result } = renderHook(() =>
        Form.use({
          values: deep.copy(initialFormValues),
          schema: basicFormSchema,
        }),
      );
      result.current.set(path.name, "Jane Doe");
      expect(result.current.get(path.name).touched).toBe(true);
      result.current.set(path.name, "John Doe");
      expect(result.current.get(path.name).touched).toBe(false);
    });

    it("should clear all touched states when resetting the form", () => {
      const { result } = renderHook(() =>
        Form.use({
          values: deep.copy(initialFormValues),
          schema: basicFormSchema,
        }),
      );
      result.current.set(path.name, "Jane Doe");
      result.current.set(path.age, 25);
      expect(result.current.get(path.name).touched).toBe(true);
      expect(result.current.get(path.age).touched).toBe(true);
      result.current.reset(deep.copy(initialFormValues));
      expect(result.current.get(path.name).touched).toBe(false);
      expect(result.current.get(path.age).touched).toBe(false);
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
      result.current.set(path.name, "Jane Doe");
      expect(onHasTouched).toHaveBeenLastCalledWith(true);
      result.current.set(path.age, 25);
      expect(onHasTouched).toHaveBeenCalledTimes(1);
      result.current.set(path.name, "John Doe");
      result.current.set(path.age, 42);
      expect(onHasTouched).toHaveBeenLastCalledWith(false);
    });

    it("should snapshot touched state when requested", () => {
      const { result } = renderHook(() =>
        Form.use({
          values: deep.copy(initialFormValues),
          schema: basicFormSchema,
        }),
      );
      result.current.set(path.name, "Jane Doe");
      result.current.set(path.age, 25);
      expect(result.current.get(path.name).touched).toBe(true);
      expect(result.current.get(path.age).touched).toBe(true);
      result.current.setCurrentStateAsInitialValues();
      expect(result.current.get(path.name).touched).toBe(false);
      expect(result.current.get(path.age).touched).toBe(false);
      result.current.set(path.name, "John Doe");
      expect(result.current.get(path.name).touched).toBe(true);
    });

    it("should properly track touched state through reset and setCurrentStateAsInitialValues", () => {
      const { result } = renderHook(() =>
        Form.use({
          values: deep.copy(initialFormValues),
          schema: basicFormSchema,
        }),
      );
      result.current.set(path.name, "Jane Doe");
      expect(result.current.get(path.name).touched).toBe(true);
      result.current.reset();
      expect(result.current.get(path.name).touched).toBe(false);
      result.current.set(path.name, "Jane Doe");
      expect(result.current.get(path.name).touched).toBe(true);
      result.current.setCurrentStateAsInitialValues();
      expect(result.current.get(path.name).touched).toBe(false);
      result.current.set(path.name, "John Doe");
      expect(result.current.get(path.name).touched).toBe(true);
      result.current.set(path.name, "Jane Doe");
      expect(result.current.get(path.name).touched).toBe(false);
    });
  });
});

describe("typed paths", () => {
  it("should read and edit through the context", () => {
    const { result } = renderHook(() => Form.useContext<typeof basicFormSchema>(), {
      wrapper,
    });
    act(() => result.current.set(path.name, "Jane"));
    expect(result.current.get(path.name).value).toEqual("Jane");
    expect(result.current.has(path.nested.ein)).toBe(true);
    act(() => result.current.insert(path.array, { key: "key2", name: "Jane" }));
    act(() => result.current.move(path.array, "key1", "key2"));
    expect(result.current.value().array.map(({ key }) => key)).toEqual([
      "key2",
      "key1",
    ]);
    act(() => result.current.remove(path.array.key2));
    expect(result.current.has(path.array.key2)).toBe(false);
  });

  it("should validate a path and set its status", () => {
    const { result } = renderHook(() => Form.useContext<typeof basicFormSchema>(), {
      wrapper,
    });
    act(() => result.current.set(path.name, ""));
    let valid = true;
    act(() => {
      valid = result.current.validate(path.name);
    });
    expect(valid).toBe(false);
    act(() => {
      valid = result.current.validate(path.age);
    });
    expect(valid).toBe(true);
    act(() =>
      result.current.setStatus(path.age, { variant: "warning", message: "Odd" }),
    );
    expect(result.current.get(path.age).status.message).toEqual("Odd");
  });

  it("should render a field with a label from the last part of the path", () => {
    render(<Form.TextField path={path.nested.ssn} />, { wrapper });
    expect(screen.getByText("Ssn")).toBeTruthy();
    expect(screen.getByDisplayValue("123-45-6789")).toBeTruthy();
  });

  it("should take an optional value in a built field", () => {
    render(<Form.TextField path={path.optionalField} />, { wrapper });
    expect(screen.getByText("Optional field")).toBeTruthy();
  });

  it("should validate only the path it gets when validating async", async () => {
    const { result } = renderHook(() => Form.useContext<typeof basicFormSchema>(), {
      wrapper,
    });
    act(() => result.current.set(path.name, ""));
    let valid = true;
    await act(async () => {
      valid = await result.current.validateAsync(path.name);
    });
    expect(valid).toBe(false);
    await act(async () => {
      valid = await result.current.validateAsync(path.age);
    });
    expect(valid).toBe(true);
  });

  it("should flag a value that does not fit its path", () => {
    const { result } = renderHook(() => Form.useContext<typeof basicFormSchema>(), {
      wrapper,
    });
    // @ts-expect-error age is a number.
    act(() => result.current.set(path.age, "42"));
    expect(result.current.get(path.age).status.variant).toEqual("error");
  });

  // Nothing here runs. `check-types` fails if any line compiles.
  it("should refuse a plain string path at every entry point", () => {
    const _useRefused = (ctx: Form.ContextValue) => {
      // @ts-expect-error a path is not a string.
      Form.useField("name");
      // @ts-expect-error a path is not a string.
      Form.useFieldValid("name");
      // @ts-expect-error a path is not a string.
      Form.useStatuses("name");
      // @ts-expect-error a path is not a string.
      Form.useCheck("name", z.string());
      // @ts-expect-error a path is not a string.
      Form.fieldListUtils(ctx, "array");
      // @ts-expect-error a path is not a string.
      Form.setChanged(ctx, "nested", {}, {});
      // @ts-expect-error a path is not a string.
      Form.useFieldValue("name");
      // @ts-expect-error a path is not a string.
      Form.useFieldState("name");
      // @ts-expect-error a path is not a string.
      Form.useFieldList("array");
      // @ts-expect-error a path is not a string.
      Form.useFieldListUtils("array");
      // @ts-expect-error a path is not a string.
      ctx.get("name");
      // @ts-expect-error a path is not a string.
      ctx.set("name", "a");
      // @ts-expect-error a path is not a string.
      ctx.remove("array.a");
      // @ts-expect-error a path is not a string.
      ctx.insert("array", { key: "a" });
      // @ts-expect-error a path is not a string.
      ctx.move("array", "a");
      // @ts-expect-error a path is not a string.
      ctx.has("name");
      // @ts-expect-error a path is not a string.
      ctx.validate("name");
      // @ts-expect-error a path is not a string.
      void ctx.validateAsync("name");
      // @ts-expect-error a path is not a string.
      ctx.setStatus("name", { variant: "error", message: "" });
      // @ts-expect-error a path is not a string.
      ctx.addCheck("name", z.string());
    };
    const _Refused = (): ReactElement => (
      <>
        {/* @ts-expect-error a path is not a string. */}
        <Form.TextField path="name" />
        {/* @ts-expect-error a path is not a string. */}
        <Form.NumericField path="age" />
        {/* @ts-expect-error a path is not a string. */}
        <Form.CheckboxField path="enabled" />
        <Form.Field<string>
          // @ts-expect-error a path is not a string.
          path="name"
        >
          {(p) => <Input.Text {...p} />}
        </Form.Field>
      </>
    );
  });
});
