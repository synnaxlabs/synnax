// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type enumeration } from "@synnaxlabs/client";
import { createTestClient, RoleClients } from "@synnaxlabs/client/testutil";
import { Haul } from "@synnaxlabs/lyra/haul";
import { theme } from "@synnaxlabs/lyra/theme";
import { color, id } from "@synnaxlabs/x";
import { act, fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement, useState } from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { Enumeration } from "@/enumeration";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();

const ALARM = theme.themeZ.parse(theme.SYNNAX_LIGHT).colors.error.z;

const createEntry = (
  name: string,
  value: number,
  extra: Partial<enumeration.Entry> = {},
): enumeration.Entry => ({ key: id.create(), name, value, abnormal: false, ...extra });

const input = (label: string): HTMLInputElement => screen.getByLabelText(label);

const commit = (el: HTMLInputElement, value: string): void => {
  fireEvent.focus(el);
  fireEvent.change(el, { target: { value } });
  fireEvent.blur(el);
};

const marked = (el: HTMLInputElement): boolean =>
  el.getAttribute("aria-invalid") === "true";

describe("Enumeration.Table", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  interface HarnessProps {
    initial: enumeration.Entry[];
    onSet?: (entry: enumeration.Entry) => void;
    abnormalDisabled?: boolean;
    colorDisabled?: boolean;
    preview?: boolean;
  }

  // Holds the entries and applies each set, as a form would.
  const Harness = ({ initial, onSet, ...rest }: HarnessProps): ReactElement => {
    const [value, setValue] = useState(initial);
    return (
      <Enumeration.Table
        value={value}
        onSet={(entry) => {
          onSet?.(entry);
          setValue((prev) => prev.map((e) => (e.key === entry.key ? entry : e)));
        }}
        onAdd={(added) => setValue((prev) => [...prev, ...added])}
        onRemove={(i) => setValue((prev) => prev.filter((_, j) => j !== i))}
        onMove={vi.fn()}
        {...rest}
      />
    );
  };

  const renderTable = (props: HarnessProps) =>
    render(<Harness {...props} />, { wrapper });

  const paste = (el: HTMLInputElement, text: string): void => {
    fireEvent.focus(el);
    fireEvent.paste(el, { clipboardData: { getData: () => text } });
  };

  describe("values", () => {
    it("should hold a value that another entry has, marked and unsent", () => {
      const onSet = vi.fn();
      renderTable({ initial: [createEntry("Off", 0), createEntry("On", 1)], onSet });
      commit(input("Value 2"), "0");
      expect(onSet).not.toHaveBeenCalled();
      expect(input("Value 2").value).toBe("0");
      expect(marked(input("Value 2"))).toBe(true);
    });

    it("should send a held value once the user frees it", () => {
      const onSet = vi.fn();
      renderTable({ initial: [createEntry("Off", 0), createEntry("On", 1)], onSet });
      commit(input("Value 2"), "0");
      commit(input("Value 1"), "5");
      expect(onSet.mock.calls.map(([e]) => [e.name, e.value])).toEqual([
        ["Off", 5],
        ["On", 0],
      ]);
      expect(marked(input("Value 2"))).toBe(false);
    });

    it("should send another field of an entry whose value it holds", () => {
      const onSet = vi.fn();
      renderTable({ initial: [createEntry("Off", 0), createEntry("On", 1)], onSet });
      commit(input("Value 2"), "0");
      commit(input("Name 2"), "Running");
      expect(onSet).toHaveBeenCalledWith(
        expect.objectContaining({ name: "Running", value: 1 }),
      );
      expect(marked(input("Value 2"))).toBe(true);
    });

    it("should send a held value once the user removes the entry that has it", () => {
      const onSet = vi.fn();
      renderTable({ initial: [createEntry("Off", 0), createEntry("On", 1)], onSet });
      commit(input("Value 2"), "0");
      fireEvent.click(screen.getByRole("button", { name: "Remove entry 1" }));
      expect(onSet).toHaveBeenCalledWith(
        expect.objectContaining({ name: "On", value: 0 }),
      );
      expect(marked(input("Value 1"))).toBe(false);
    });

    it("should show the stored value once another user frees a held value", () => {
      const off = createEntry("Off", 0);
      const on = createEntry("On", 1);
      const props = {
        onSet: vi.fn(),
        onAdd: vi.fn(),
        onRemove: vi.fn(),
        onMove: vi.fn(),
      };
      const { rerender } = render(<Enumeration.Table value={[off, on]} {...props} />, {
        wrapper,
      });
      commit(input("Value 2"), "0");
      rerender(<Enumeration.Table value={[{ ...off, value: 5 }, on]} {...props} />);
      expect(input("Value 2").value).toBe("1");
      expect(marked(input("Value 2"))).toBe(false);
    });

    it("should hold a pasted value that another entry has", () => {
      const onSet = vi.fn();
      renderTable({ initial: [createEntry("Off", 0), createEntry("On", 1)], onSet });
      paste(input("Name 2"), "On\t1\nIdle\t0");
      expect(input("Value 3").value).toBe("0");
      expect(marked(input("Value 3"))).toBe(true);
      commit(input("Value 1"), "5");
      expect(onSet).toHaveBeenCalledWith(
        expect.objectContaining({ name: "Idle", value: 0 }),
      );
    });

    it("should mark both entries that store the same value", () => {
      renderTable({ initial: [createEntry("Off", 0), createEntry("On", 0)] });
      expect(marked(input("Value 1"))).toBe(true);
      expect(marked(input("Value 2"))).toBe(true);
    });

    it("should not mark the entry whose value another entry holds", () => {
      renderTable({ initial: [createEntry("Off", 0), createEntry("On", 1)] });
      commit(input("Value 2"), "0");
      expect(marked(input("Value 1"))).toBe(false);
    });

    it("should not add an entry with the value that a held entry stores", () => {
      renderTable({ initial: [createEntry("Off", 0), createEntry("On", 1)] });
      commit(input("Value 2"), "0");
      fireEvent.click(screen.getByRole("button", { name: "Add entry" }));
      expect(input("Value 3").value).toBe("2");
    });

    it("should add an entry with the next whole value", () => {
      renderTable({ initial: [createEntry("Off", 0), createEntry("On", 2.5)] });
      fireEvent.click(screen.getByRole("button", { name: "Add entry" }));
      expect(input("Value 3").value).toBe("3");
    });
  });

  describe("colors", () => {
    it("should set a fixed color from a typed hex code", () => {
      const onSet = vi.fn();
      renderTable({ initial: [createEntry("Off", 0)], onSet });
      commit(input("Color 1"), "#F85149");
      expect(onSet).toHaveBeenCalledWith(
        expect.objectContaining({
          color: { type: "fixed", value: color.construct("#f85149") },
        }),
      );
    });

    it("should set a named color picked from the swatch", async () => {
      const palette = await client.groups.create({
        scope: "color",
        name: `palette-${id.create()}`,
      });
      const named = await client.colors.create({
        name: `color-${id.create()}`,
        light: "#00FF00",
        group: palette.key,
      });
      const onSet = vi.fn();
      renderTable({ initial: [createEntry("Off", 0)], onSet });
      fireEvent.click(screen.getByRole("button", { name: "Pick color 1" }));
      fireEvent.click(await screen.findByRole("button", { name: named.name }));
      expect(onSet).toHaveBeenCalledWith(
        expect.objectContaining({ color: { type: "named", key: named.key } }),
      );
    });

    it("should set a fixed color with alpha from an eight digit hex code", () => {
      const onSet = vi.fn();
      renderTable({ initial: [createEntry("Off", 0)], onSet });
      commit(input("Color 1"), "F8514980");
      expect(onSet).toHaveBeenCalledWith(
        expect.objectContaining({
          color: { type: "fixed", value: color.construct("#f8514980") },
        }),
      );
    });

    it("should show a named color by its name", async () => {
      const palette = await client.groups.create({
        scope: "color",
        name: `palette-${id.create()}`,
      });
      const named = await client.colors.create({
        name: `color-${id.create()}`,
        light: "#00FF00",
        group: palette.key,
      });
      renderTable({
        initial: [createEntry("Off", 0, { color: { type: "named", key: named.key } })],
      });
      await waitFor(() => expect(input("Color 1").value).toBe(named.name));
    });

    it("should clear a color to Auto when its text is erased", () => {
      const onSet = vi.fn();
      const fixed = { type: "fixed" as const, value: color.construct("#2e7d32") };
      renderTable({ initial: [createEntry("Off", 0, { color: fixed })], onSet });
      expect(input("Color 1").value).toBe("2E7D32");
      commit(input("Color 1"), "");
      expect(onSet).toHaveBeenCalledWith(expect.not.objectContaining({ color: fixed }));
    });

    it("should show an entry's warnings on its color cell, not as a mark", () => {
      renderTable({ initial: [createEntry("", 0)] });
      expect(screen.getByLabelText(/It has no name/)).toBeTruthy();
      expect(marked(input("Color 1"))).toBe(false);
    });

    it("should warn about entries that have the same name", () => {
      renderTable({ initial: [createEntry("Open", 0), createEntry("Open", 1)] });
      expect(screen.getAllByLabelText(/Another entry has its name/)).toHaveLength(2);
    });

    it("should warn about a color that is hard to see on the canvas", () => {
      const fixed = { type: "fixed" as const, value: color.construct("#ffffff") };
      renderTable({ initial: [createEntry("Off", 0, { color: fixed })] });
      expect(screen.getByLabelText(/hard to see in the light theme/)).toBeTruthy();
    });

    it("should name the entry whose color is almost the same", () => {
      const fixed = (hex: string) => ({
        type: "fixed" as const,
        value: color.construct(hex),
      });
      renderTable({
        initial: [
          createEntry("Open", 0, { color: fixed("#2e7d32") }),
          createEntry("Running", 1, { color: fixed("#2e7e32") }),
        ],
      });
      expect(screen.getByLabelText(/almost the color of Running/)).toBeTruthy();
    });

    it("should not check the color of an entry with no color", () => {
      renderTable({ initial: [createEntry("Off", 0)] });
      expect(screen.queryByRole("img")).toBeNull();
    });

    it("should not warn about an alarm color on an abnormal entry", () => {
      const fixed = { type: "fixed" as const, value: ALARM };
      renderTable({
        initial: [createEntry("Fault", 0, { color: fixed, abnormal: true })],
      });
      expect(screen.queryByLabelText(/alarm color/)).toBeNull();
    });

    it("should picture a dragged entry with its name and color", () => {
      const fixed = { type: "fixed" as const, value: color.construct("#00ff00") };
      render(
        <Haul.Provider>
          <Harness initial={[createEntry("Running", 1, { color: fixed })]} />
        </Haul.Provider>,
        { wrapper },
      );
      fireEvent.dragStart(screen.getByRole("button", { name: "Move entry 1" }));
      const tile = document.querySelector<HTMLElement>(".lyra-haul-preview__tile");
      expect(tile?.style.background).toBe("rgb(0, 255, 0)");
      expect(screen.getByText("Value 1")).toBeTruthy();
    });

    it("should warn about an alarm color on an entry that is not abnormal", () => {
      const fixed = { type: "fixed" as const, value: ALARM };
      renderTable({ initial: [createEntry("Locked out", 0, { color: fixed })] });
      expect(screen.getByLabelText(/alarm color/)).toBeTruthy();
    });

    it("should not warn about an alarm color when no entry can be abnormal", () => {
      const fixed = { type: "fixed" as const, value: ALARM };
      renderTable({
        initial: [createEntry("Locked out", 0, { color: fixed })],
        abnormalDisabled: true,
      });
      expect(screen.queryByLabelText(/alarm color/)).toBeNull();
    });

    it("should show empty color cells and no warnings when color is disabled", () => {
      renderTable({ initial: [createEntry("", 0)], colorDisabled: true });
      expect(input("Color 1").disabled).toBe(true);
      expect(input("Color 1").value).toBe("");
      expect(screen.queryByLabelText(/It has no name/)).toBeNull();
    });
  });

  describe("abnormal", () => {
    it("should edit abnormal with a switch", () => {
      const onSet = vi.fn();
      renderTable({ initial: [createEntry("Fault", 0)], onSet });
      const cell = input("Abnormal 1");
      expect(cell.classList).toContain("lyra-input__switch-input");
      fireEvent.click(cell);
      expect(onSet).toHaveBeenCalledWith(expect.objectContaining({ abnormal: true }));
    });

    it("should show empty abnormal cells when no entry can be abnormal", () => {
      renderTable({ initial: [createEntry("Open", 0)], abnormalDisabled: true });
      expect(input("Abnormal 1").disabled).toBe(true);
      expect(input("Abnormal 1").value).toBe("");
    });

    it("should show a read-only switch in preview", () => {
      renderTable({
        initial: [createEntry("Fault", 0, { abnormal: true })],
        preview: true,
      });
      expect(input("Abnormal 1").checked).toBe(true);
      expect(input("Abnormal 1").disabled).toBe(true);
    });
  });
});

describe("Enumeration.Entries", () => {
  let wrapper: FC<PropsWithChildren>;
  beforeAll(async () => {
    wrapper = await createAsyncSynnaxWrapper({ client });
  });

  const createEnumeration = async (
    entries: enumeration.Entry[],
  ): Promise<enumeration.Enumeration> =>
    await client.enumerations.create({ name: id.create(), entries });

  const stored = async (key: enumeration.Key): Promise<enumeration.Entry[]> =>
    (await client.enumerations.retrieve(key)).entries;

  it("should show read-only to a user who cannot update the enumeration", async () => {
    const e = await createEnumeration([createEntry("Fault", 0, { abnormal: true })]);
    const viewer = await new RoleClients(client).get("Viewer");
    const viewerWrapper = await createAsyncSynnaxWrapper({ client: viewer });
    // A render as a user who is not an admin suspends at first.
    await act(async () => {
      render(<Enumeration.Entries enumeration={e.key} />, { wrapper: viewerWrapper });
    });
    const cell = await screen.findByLabelText<HTMLInputElement>("Abnormal 1");
    await act(async () => {});
    expect(cell.checked).toBe(true);
    expect(cell.disabled).toBe(true);
  });

  it("should let a user who can update the enumeration edit it", async () => {
    const e = await createEnumeration([createEntry("Fault", 0, { abnormal: true })]);
    render(<Enumeration.Entries enumeration={e.key} />, { wrapper });
    const cell = await screen.findByLabelText<HTMLInputElement>("Abnormal 1");
    await waitFor(() => expect(cell.disabled).toBe(false));
  });

  it("should save an edit to the Core", async () => {
    const e = await createEnumeration([createEntry("Off", 0), createEntry("On", 1)]);
    render(<Enumeration.Entries enumeration={e.key} />, { wrapper });
    commit(await screen.findByLabelText<HTMLInputElement>("Name 2"), "Running");
    await waitFor(async () =>
      expect((await stored(e.key)).map((en) => en.name)).toEqual(["Off", "Running"]),
    );
  });

  it("should save an added entry after the last one", async () => {
    const e = await createEnumeration([createEntry("Off", 0)]);
    render(<Enumeration.Entries enumeration={e.key} />, { wrapper });
    fireEvent.click(await screen.findByRole("button", { name: "Add entry" }));
    await waitFor(async () =>
      expect((await stored(e.key)).map((en) => en.value)).toEqual([0, 1]),
    );
  });

  it("should save a move with the arrow keys", async () => {
    const e = await createEnumeration([createEntry("Off", 0), createEntry("On", 1)]);
    render(<Enumeration.Entries enumeration={e.key} />, { wrapper });
    fireEvent.keyDown(await screen.findByRole("button", { name: "Move entry 2" }), {
      key: "ArrowUp",
    });
    await waitFor(async () =>
      expect((await stored(e.key)).map((en) => en.name)).toEqual(["On", "Off"]),
    );
  });

  it("should save a removed entry", async () => {
    const e = await createEnumeration([createEntry("Off", 0), createEntry("On", 1)]);
    render(<Enumeration.Entries enumeration={e.key} />, { wrapper });
    fireEvent.click(await screen.findByRole("button", { name: "Remove entry 1" }));
    await waitFor(async () =>
      expect((await stored(e.key)).map((en) => en.name)).toEqual(["On"]),
    );
  });
});
