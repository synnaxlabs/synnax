// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep } from "@synnaxlabs/x";
import { fireEvent, render } from "@testing-library/react";
import { type ReactElement, type ReactNode } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Form } from "@/form";

const channelZ = z.object({ name: z.string(), range: z.number() });
const schema = z.object({ a: channelZ, b: channelZ, c: channelZ.optional() });
type Values = z.infer<typeof schema>;
const path = deep.path<Values>();

interface RenderedForm {
  values: () => Values;
  getByDisplayValue: (value: string) => HTMLElement;
  queryByText: (text: string) => HTMLElement | null;
}

const renderForm = (values: Values, children: ReactNode): RenderedForm => {
  let ctx: Form.ContextValue<typeof schema> | null = null;
  const Container = (): ReactElement => {
    const methods = Form.use<typeof schema>({ values, schema });
    ctx = methods;
    return <Form.Form<typeof schema> {...methods}>{children}</Form.Form>;
  };
  const c = render(<Container />);
  return {
    values: () => {
      if (ctx == null) throw new Error("the form did not render");
      return ctx.value();
    },
    getByDisplayValue: c.getByDisplayValue,
    queryByText: c.queryByText,
  };
};

const A = { name: "a", range: 10 };
const B = { name: "b", range: 10 };

describe("Form.Fanout", () => {
  it("should show the value that every subtree shares", () => {
    const c = renderForm(
      { a: A, b: B },
      <Form.Fanout paths={[path.a, path.b]}>
        <Form.NumericField path={path.a.range} />
      </Form.Fanout>,
    );
    expect(c.getByDisplayValue("10")).toBeTruthy();
    expect(c.queryByText("Mixed")).toBeNull();
  });

  it("should show Mixed where the subtrees differ", () => {
    const c = renderForm(
      { a: A, b: { ...B, range: 5 } },
      <Form.Fanout paths={[path.a, path.b]}>
        <Form.NumericField path={path.a.range} />
      </Form.Fanout>,
    );
    expect(c.queryByText("Mixed")).toBeTruthy();
  });

  it("should show Mixed where a subtree lacks the field", () => {
    const c = renderForm(
      { a: A, b: B },
      <Form.Fanout paths={[path.a, path.c]}>
        <Form.NumericField path={path.a.range} />
      </Form.Fanout>,
    );
    expect(c.queryByText("Mixed")).toBeTruthy();
  });

  it("should write an edit to every subtree", () => {
    const c = renderForm(
      { a: A, b: { ...B, range: 5 } },
      <Form.Fanout paths={[path.a, path.b]}>
        <Form.NumericField path={path.a.range} />
      </Form.Fanout>,
    );
    const input = c.getByDisplayValue("10");
    fireEvent.change(input, { target: { value: "20" } });
    fireEvent.blur(input);
    expect(c.values()).toMatchObject({ a: { range: 20 }, b: { range: 20 } });
    expect(c.queryByText("Mixed")).toBeNull();
  });

  it("should write a field outside the first subtree to that field alone", () => {
    const c = renderForm(
      { a: A, b: B },
      <Form.Fanout paths={[path.a, path.b]}>
        <Form.Field path={path.b.name} />
      </Form.Fanout>,
    );
    fireEvent.change(c.getByDisplayValue("b"), { target: { value: "bee" } });
    expect(c.values()).toMatchObject({ a: { name: "a" }, b: { name: "bee" } });
  });
});
