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
import { beforeAll, describe, expect, it, type Mock, vi } from "vitest";

import { Enumeration } from "@/enumeration";
import { createAsyncSynnaxWrapper, createSynnaxWrapper } from "@/testutil/Synnax";

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

type OnSet = (entries: enumeration.Entry[]) => void;

/** @returns the name and value of each sent entry, grouped by the call that sent it. */
const sentValues = (onSet: Mock<OnSet>): [string, number][][] =>
  onSet.mock.calls.map(([entries]) =>
    entries.map((e): [string, number] => [e.name, e.value]),
  );

const marked = (el: HTMLInputElement): boolean =>
  el.getAttribute("aria-invalid") === "true";

describe("Enumeration.Table", () => {
  const wrapper = createSynnaxWrapper({ client: null });

  interface HarnessProps {
    initial: enumeration.Entry[];
    onSet?: (entries: enumeration.Entry[]) => void;
    onAdd?: (entries: enumeration.Entry[]) => void;
    preview?: boolean;
  }

  // Holds the entries and applies each set, as a form would.
  const Harness = ({ initial, onSet, onAdd, ...rest }: HarnessProps): ReactElement => {
    const [value, setValue] = useState(initial);
    return (
      <Enumeration.Table
        value={value}
        onSet={(changed) => {
          onSet?.(changed);
          setValue((prev) =>
            prev.map((e) => changed.find((c) => c.key === e.key) ?? e),
          );
        }}
        onAdd={(added) => {
          onAdd?.(added);
          setValue((prev) => [...prev, ...added]);
        }}
        onRemove={(i) => setValue((prev) => prev.filter((_, j) => j !== i))}
        onMove={vi.fn()}
        {...rest}
      />
    );
  };

  const renderTable = (props: HarnessProps, as: FC<PropsWithChildren> = wrapper) =>
    render(<Harness {...props} />, { wrapper: as });

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
      const onSet = vi.fn<OnSet>();
      renderTable({ initial: [createEntry("Off", 0), createEntry("On", 1)], onSet });
      commit(input("Value 2"), "0");
      commit(input("Value 1"), "5");
      expect(sentValues(onSet)).toEqual([
        [
          ["Off", 5],
          ["On", 0],
        ],
      ]);
      expect(marked(input("Value 2"))).toBe(false);
    });

    it("should send another field of an entry whose value it holds", () => {
      const onSet = vi.fn();
      renderTable({ initial: [createEntry("Off", 0), createEntry("On", 1)], onSet });
      commit(input("Value 2"), "0");
      commit(input("Name 2"), "Running");
      expect(onSet).toHaveBeenCalledWith([
        expect.objectContaining({ name: "Running", value: 1 }),
      ]);
      expect(marked(input("Value 2"))).toBe(true);
    });

    it("should send a held value once the user removes the entry that has it", () => {
      const onSet = vi.fn();
      renderTable({ initial: [createEntry("Off", 0), createEntry("On", 1)], onSet });
      commit(input("Value 2"), "0");
      fireEvent.click(screen.getByRole("button", { name: "Remove entry 1" }));
      expect(onSet).toHaveBeenCalledWith([
        expect.objectContaining({ name: "On", value: 0 }),
      ]);
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
      const onSet = vi.fn<OnSet>();
      renderTable({ initial: [createEntry("Off", 0), createEntry("On", 1)], onSet });
      paste(input("Name 2"), "On\t1\nIdle\t0");
      expect(input("Value 3").value).toBe("0");
      expect(marked(input("Value 3"))).toBe(true);
      commit(input("Value 1"), "5");
      expect(sentValues(onSet).at(-1)).toEqual([
        ["Off", 5],
        ["Idle", 0],
      ]);
    });

    it("should hold a value that a held entry still stores", () => {
      const onSet = vi.fn();
      renderTable({
        initial: [createEntry("Off", 0), createEntry("On", 1), createEntry("Idle", 2)],
        onSet,
      });
      commit(input("Value 3"), "0");
      commit(input("Value 2"), "2");
      expect(onSet).not.toHaveBeenCalled();
      expect(input("Value 2").value).toBe("2");
      expect(marked(input("Value 2"))).toBe(true);
    });

    it("should save a swap of two values", () => {
      const onSet = vi.fn<OnSet>();
      renderTable({ initial: [createEntry("Off", 0), createEntry("On", 1)], onSet });
      commit(input("Value 2"), "0");
      commit(input("Value 1"), "1");
      expect(sentValues(onSet)).toEqual([
        [
          ["Off", 1],
          ["On", 0],
        ],
      ]);
      expect(marked(input("Value 2"))).toBe(false);
    });

    it("should save a pasted entry with a free value that a held entry stores", () => {
      const onAdd = vi.fn();
      renderTable({ initial: [createEntry("Off", 0), createEntry("On", 1)], onAdd });
      commit(input("Value 2"), "0");
      paste(input("Name 2"), "On\t0\nIdle\t1");
      expect(onAdd).toHaveBeenCalledWith([
        expect.objectContaining({ name: "Idle", value: 2 }),
      ]);
      expect(input("Value 3").value).toBe("1");
      expect(marked(input("Value 3"))).toBe(true);
    });

    it("should keep a hold on a remove while a held entry stores its value", () => {
      const onSet = vi.fn();
      renderTable({
        initial: [
          createEntry("Off", 0),
          createEntry("On", 1),
          createEntry("Idle", 2),
          createEntry("Fault", 3),
        ],
        onSet,
      });
      commit(input("Value 3"), "0");
      commit(input("Value 2"), "2");
      fireEvent.click(screen.getByRole("button", { name: "Remove entry 4" }));
      expect(onSet).not.toHaveBeenCalled();
      expect(marked(input("Value 2"))).toBe(true);
    });

    it("should release each hold that a remove frees", () => {
      const onSet = vi.fn<OnSet>();
      renderTable({
        initial: [createEntry("Off", 0), createEntry("On", 1), createEntry("Idle", 2)],
        onSet,
      });
      commit(input("Value 3"), "0");
      commit(input("Value 2"), "2");
      fireEvent.click(screen.getByRole("button", { name: "Remove entry 1" }));
      expect(sentValues(onSet)).toEqual([
        [
          ["On", 2],
          ["Idle", 0],
        ],
      ]);
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
      expect(onSet).toHaveBeenCalledWith([
        expect.objectContaining({
          color: { type: "fixed", value: color.construct("#f85149") },
        }),
      ]);
    });

    it("should set a fixed color with alpha from an eight digit hex code", () => {
      const onSet = vi.fn();
      renderTable({ initial: [createEntry("Off", 0)], onSet });
      commit(input("Color 1"), "F8514980");
      expect(onSet).toHaveBeenCalledWith([
        expect.objectContaining({
          color: { type: "fixed", value: color.construct("#f8514980") },
        }),
      ]);
    });

    it("should clear a color to Auto when its text is erased", () => {
      const onSet = vi.fn();
      const fixed = { type: "fixed" as const, value: color.construct("#2e7d32") };
      renderTable({ initial: [createEntry("Off", 0, { color: fixed })], onSet });
      expect(input("Color 1").value).toBe("2E7D32");
      commit(input("Color 1"), "");
      expect(onSet).toHaveBeenCalledWith([
        expect.not.objectContaining({ color: fixed }),
      ]);
    });

    it("should set a fixed color from a pasted hex code", () => {
      const onSet = vi.fn();
      renderTable({ initial: [createEntry("Off", 0)], onSet });
      paste(input("Name 1"), "Open\t0\t#2E7D32\tfalse");
      expect(onSet).toHaveBeenCalledWith([
        expect.objectContaining({
          name: "Open",
          color: { type: "fixed", value: color.construct("#2e7d32") },
        }),
      ]);
      expect(marked(input("Color 1"))).toBe(false);
    });

    it("should clear a color to Auto from a pasted empty cell", () => {
      const onSet = vi.fn();
      const fixed = { type: "fixed" as const, value: color.construct("#2e7d32") };
      renderTable({ initial: [createEntry("Off", 0, { color: fixed })], onSet });
      paste(input("Name 1"), "Open\t0\t\tfalse");
      expect(onSet).toHaveBeenCalledOnce();
      const [[sent]] = onSet.mock.calls[0];
      expect(sent.name).toBe("Open");
      expect(sent).not.toHaveProperty("color");
    });

    it("should mark a pasted color that is not a hex code", () => {
      renderTable({ initial: [createEntry("Off", 0)] });
      paste(input("Name 1"), "Open\t0\tgreen\tfalse");
      expect(marked(input("Color 1"))).toBe(true);
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

    it("should name the entry whose color looks the same with color blindness", () => {
      const fixed = (hex: string) => ({
        type: "fixed" as const,
        value: color.construct(hex),
      });
      renderTable({
        initial: [
          createEntry("Open", 0, { color: fixed("#388e3c") }),
          createEntry("Running", 1, { color: fixed("#00838f") }),
        ],
      });
      expect(
        screen.getByLabelText(/With color blindness, its color looks like Running/),
      ).toBeTruthy();
      expect(screen.queryByLabelText(/almost the color of/)).toBeNull();
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
  });

  describe("abnormal", () => {
    it("should edit abnormal with a switch", () => {
      const onSet = vi.fn();
      renderTable({ initial: [createEntry("Fault", 0)], onSet });
      const cell = input("Abnormal 1");
      expect(cell.classList).toContain("lyra-input__switch-input");
      fireEvent.click(cell);
      expect(onSet).toHaveBeenCalledWith([expect.objectContaining({ abnormal: true })]);
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

  // A live wrapper turns off React's act environment for the rest of the file, so these
  // run last.
  describe("named colors", () => {
    let live: FC<PropsWithChildren>;
    beforeAll(async () => {
      live = await createAsyncSynnaxWrapper({ client });
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
      renderTable({ initial: [createEntry("Off", 0)], onSet }, live);
      fireEvent.click(screen.getByRole("button", { name: "Pick color 1" }));
      fireEvent.click(await screen.findByRole("button", { name: named.name }));
      expect(onSet).toHaveBeenCalledWith([
        expect.objectContaining({ color: { type: "named", key: named.key } }),
      ]);
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
      renderTable(
        {
          initial: [
            createEntry("Off", 0, { color: { type: "named", key: named.key } }),
          ],
        },
        live,
      );
      await waitFor(() => expect(input("Color 1").value).toBe(named.name));
    });

    it("should warn about a named color hard to see in the dark theme", async () => {
      const palette = await client.groups.create({
        scope: "color",
        name: `palette-${id.create()}`,
      });
      const named = await client.colors.create({
        name: `color-${id.create()}`,
        light: "#2e7d32",
        dark: "#101010",
        group: palette.key,
      });
      renderTable(
        {
          initial: [
            createEntry("Off", 0, { color: { type: "named", key: named.key } }),
          ],
        },
        live,
      );
      expect(
        await screen.findByLabelText(/hard to see in the dark theme/),
      ).toBeTruthy();
      expect(screen.queryByLabelText(/hard to see in the light theme/)).toBeNull();
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

  // The editor's client shows its pending actions, so the Core reads through another.
  const core = createTestClient();
  const stored = async (key: enumeration.Key): Promise<enumeration.Entry[]> =>
    (await core.enumerations.retrieve(key)).entries;

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

  it("should save a move down with the arrow keys", async () => {
    const e = await createEnumeration([
      createEntry("Off", 0),
      createEntry("On", 1),
      createEntry("Idle", 2),
    ]);
    render(<Enumeration.Entries enumeration={e.key} />, { wrapper });
    fireEvent.keyDown(await screen.findByRole("button", { name: "Move entry 1" }), {
      key: "ArrowDown",
    });
    await waitFor(async () =>
      expect((await stored(e.key)).map((en) => en.name)).toEqual(["On", "Off", "Idle"]),
    );
  });

  it("should save the rows of a paste below the last entry in order", async () => {
    const e = await createEnumeration([createEntry("Off", 0)]);
    render(<Enumeration.Entries enumeration={e.key} />, { wrapper });
    const name = await screen.findByLabelText<HTMLInputElement>("Name 1");
    fireEvent.focus(name);
    fireEvent.paste(name, {
      clipboardData: { getData: () => "Off\t0\nIdle\t1\nFault\t2" },
    });
    await waitFor(async () =>
      expect((await stored(e.key)).map((en) => [en.name, en.value])).toEqual([
        ["Off", 0],
        ["Idle", 1],
        ["Fault", 2],
      ]),
    );
  });

  it("should save a swap of two values", async () => {
    const e = await createEnumeration([createEntry("Off", 0), createEntry("On", 1)]);
    render(<Enumeration.Entries enumeration={e.key} />, { wrapper });
    commit(await screen.findByLabelText<HTMLInputElement>("Value 2"), "0");
    commit(screen.getByLabelText<HTMLInputElement>("Value 1"), "1");
    await waitFor(async () =>
      expect((await stored(e.key)).map((en) => en.value)).toEqual([1, 0]),
    );
  });

  it("should save each hold that a remove frees", async () => {
    const e = await createEnumeration([
      createEntry("Off", 0),
      createEntry("On", 1),
      createEntry("Idle", 2),
    ]);
    render(<Enumeration.Entries enumeration={e.key} />, { wrapper });
    commit(await screen.findByLabelText<HTMLInputElement>("Value 3"), "0");
    commit(screen.getByLabelText<HTMLInputElement>("Value 2"), "2");
    fireEvent.click(screen.getByRole("button", { name: "Remove entry 1" }));
    await waitFor(async () =>
      expect((await stored(e.key)).map((en) => [en.name, en.value])).toEqual([
        ["On", 2],
        ["Idle", 0],
      ]),
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
