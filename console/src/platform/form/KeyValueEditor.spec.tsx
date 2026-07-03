// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form as PForm } from "@synnaxlabs/pluto";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";
import { type z } from "zod";

import { KeyValueEditor } from "@/platform/form";
import { getBySelector, renderWithConsole } from "@/testutil";

const PATH = "headers";

type Row = Record<string, unknown>;
type OnChange = (args: { values: unknown }) => void;

interface HarnessProps {
  onChange?: OnChange;
  initialHeaders?: unknown;
  keyField?: string;
  valueType?: "string" | "number";
  valueFirst?: boolean;
}

const Harness = ({
  onChange = vi.fn(),
  initialHeaders,
  keyField = "name",
  valueType,
  valueFirst,
}: HarnessProps): ReactElement => {
  const methods = PForm.use<z.ZodType>({
    values: { [PATH]: initialHeaders },
    onChange,
  });
  return (
    <PForm.Form {...methods}>
      <KeyValueEditor
        path={PATH}
        label="Headers"
        keyField={keyField}
        valueType={valueType}
        valueFirst={valueFirst}
      />
    </PForm.Form>
  );
};
Harness.displayName = "KeyValueEditorHarness";

const lastHeaders = (onChange: OnChange): Row[] | undefined =>
  (
    (onChange as ReturnType<typeof vi.fn>).mock.calls.at(-1)?.[0] as {
      values: { [PATH]?: Row[] };
    }
  ).values[PATH];

describe("KeyValueEditor", () => {
  it("should append an empty row when the add button is clicked", async () => {
    const onChange = vi.fn();
    await renderWithConsole(<Harness onChange={onChange} />);
    expect(screen.queryByPlaceholderText("Key")).toBeNull();
    fireEvent.click(screen.getByRole("button"));
    await waitFor(() => expect(screen.getByPlaceholderText("Key")).toBeTruthy());
    expect(lastHeaders(onChange)).toEqual([{ name: "", value: "" }]);
  });

  it("should use the provided key field name for appended rows", async () => {
    const onChange = vi.fn();
    await renderWithConsole(<Harness onChange={onChange} keyField="parameter" />);
    fireEvent.click(screen.getByRole("button"));
    await waitFor(() =>
      expect(lastHeaders(onChange)).toEqual([{ parameter: "", value: "" }]),
    );
  });

  it("should update a row's key when the key input changes", async () => {
    const onChange = vi.fn();
    await renderWithConsole(
      <Harness onChange={onChange} initialHeaders={[{ name: "", value: "" }]} />,
    );
    const keyInput = await screen.findByPlaceholderText("Key");
    fireEvent.change(keyInput, { target: { value: "Authorization" } });
    await waitFor(() =>
      expect(lastHeaders(onChange)).toEqual([{ name: "Authorization", value: "" }]),
    );
  });

  it("should update a row's value when the value input changes", async () => {
    const onChange = vi.fn();
    await renderWithConsole(
      <Harness onChange={onChange} initialHeaders={[{ name: "X", value: "" }]} />,
    );
    const valueInput = await screen.findByPlaceholderText("Value");
    fireEvent.change(valueInput, { target: { value: "Bearer" } });
    await waitFor(() =>
      expect(lastHeaders(onChange)).toEqual([{ name: "X", value: "Bearer" }]),
    );
  });

  it("should remove only the targeted row when several exist", async () => {
    const onChange = vi.fn();
    const { container } = await renderWithConsole(
      <Harness
        onChange={onChange}
        initialHeaders={[
          { name: "A", value: "1" },
          { name: "B", value: "2" },
          { name: "C", value: "3" },
        ]}
      />,
    );
    await screen.findByDisplayValue("B");
    const rows = container.querySelectorAll(".console-kv-row");
    expect(rows).toHaveLength(3);
    fireEvent.click(getBySelector(rows[1], "button"));
    await waitFor(() =>
      expect(lastHeaders(onChange)).toEqual([
        { name: "A", value: "1" },
        { name: "C", value: "3" },
      ]),
    );
    expect(screen.queryByDisplayValue("B")).toBeNull();
  });

  it("should only edit the targeted row when two rows share the same key", async () => {
    const onChange = vi.fn();
    const { container } = await renderWithConsole(
      <Harness
        onChange={onChange}
        initialHeaders={[
          { name: "dup", value: "first" },
          { name: "dup", value: "second" },
        ]}
      />,
    );
    await screen.findByDisplayValue("second");
    const rows = container.querySelectorAll(".console-kv-row");
    const secondValueInput = rows[1].querySelector(
      "input[placeholder='Value']",
    ) as HTMLInputElement;
    fireEvent.change(secondValueInput, { target: { value: "changed" } });
    await waitFor(() =>
      expect(lastHeaders(onChange)).toEqual([
        { name: "dup", value: "first" },
        { name: "dup", value: "changed" },
      ]),
    );
  });

  it("should remove the last row and clear the field value", async () => {
    const onChange = vi.fn();
    const { container } = await renderWithConsole(
      <Harness onChange={onChange} initialHeaders={[{ name: "X", value: "Y" }]} />,
    );
    await screen.findByPlaceholderText("Key");
    const buttons = container.querySelectorAll("button");
    fireEvent.click(buttons[buttons.length - 1]);
    await waitFor(() => expect(screen.queryByPlaceholderText("Key")).toBeNull());
    expect(lastHeaders(onChange)).toEqual([]);
  });

  it("should render a numeric input and default to 0 for number value types", async () => {
    const onChange = vi.fn();
    await renderWithConsole(<Harness onChange={onChange} valueType="number" />);
    fireEvent.click(screen.getByRole("button"));
    await waitFor(() =>
      expect(lastHeaders(onChange)).toEqual([{ name: "", value: 0 }]),
    );
    expect(screen.getByDisplayValue("0")).toBeTruthy();
  });

  it("should render the value input before the key input when valueFirst is set", async () => {
    const { container } = await renderWithConsole(
      <Harness initialHeaders={[{ name: "", value: "" }]} valueFirst />,
    );
    await screen.findByPlaceholderText("Key");
    const inputs = Array.from(container.querySelectorAll<HTMLInputElement>("input"));
    expect(inputs[0].placeholder).toBe("Value");
    expect(inputs[1].placeholder).toBe("Key");
  });

  it("should reset a non-array (legacy) value to an empty array on mount", async () => {
    const onChange = vi.fn();
    await renderWithConsole(
      <Harness onChange={onChange} initialHeaders={{ legacy: "object" }} />,
    );
    await waitFor(() => expect(lastHeaders(onChange)).toEqual([]));
    expect(screen.queryByPlaceholderText("Key")).toBeNull();
  });
});
