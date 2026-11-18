// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type enumeration } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { theme } from "@synnaxlabs/lyra/theme";
import { color, id } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
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
    onSet?: (index: number, entry: enumeration.Entry) => void;
    preview?: boolean;
  }

  // Holds the entries and applies each set, as a form would.
  const Harness = ({ initial, onSet, ...rest }: HarnessProps): ReactElement => {
    const [value, setValue] = useState(initial);
    return (
      <Enumeration.Table
        value={value}
        onSet={(i, entry) => {
          onSet?.(i, entry);
          setValue((prev) => prev.map((e, j) => (j === i ? entry : e)));
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
      expect(onSet.mock.calls.map(([i, e]) => [i, e.value])).toEqual([
        [0, 5],
        [1, 0],
      ]);
      expect(marked(input("Value 2"))).toBe(false);
    });

    it("should send another field of an entry whose value it holds", () => {
      const onSet = vi.fn();
      renderTable({ initial: [createEntry("Off", 0), createEntry("On", 1)], onSet });
      commit(input("Value 2"), "0");
      commit(input("Name 2"), "Running");
      expect(onSet).toHaveBeenCalledWith(
        1,
        expect.objectContaining({ name: "Running", value: 1 }),
      );
      expect(marked(input("Value 2"))).toBe(true);
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
        0,
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
        0,
        expect.objectContaining({ color: { type: "named", key: named.key } }),
      );
    });

    it("should clear a color to Auto when its text is erased", () => {
      const onSet = vi.fn();
      const fixed = { type: "fixed" as const, value: color.construct("#2e7d32") };
      renderTable({ initial: [createEntry("Off", 0, { color: fixed })], onSet });
      expect(input("Color 1").value).toBe("2E7D32");
      commit(input("Color 1"), "");
      expect(onSet).toHaveBeenCalledWith(
        0,
        expect.not.objectContaining({ color: fixed }),
      );
    });

    it("should show an entry's warnings on its color cell, not as a mark", () => {
      renderTable({ initial: [createEntry("", 0)] });
      expect(screen.getByLabelText(/It has no name/)).toBeTruthy();
      expect(marked(input("Color 1"))).toBe(false);
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
      expect(onSet).toHaveBeenCalledWith(
        0,
        expect.objectContaining({ abnormal: true }),
      );
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
