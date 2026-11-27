// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { enumeration } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Haul } from "@synnaxlabs/lyra/haul";
import { renderThemed } from "@synnaxlabs/lyra/styletest";
import { color, id } from "@synnaxlabs/x";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement, useState } from "react";
import { beforeAll, describe, expect, it, type Mock, vi } from "vitest";

import { Enumeration } from "@/enumeration";
import { createAsyncSynnaxWrapper, createSynnaxWrapper } from "@/testutil";

const client = createTestClient();

const createEntry = (
  name: string,
  value: number,
  extra: Partial<enumeration.Entry> = {},
): enumeration.Entry => ({ key: id.create(), name, value, abnormal: false, ...extra });

type OnChange = (edit: Enumeration.Edit) => void;

/** @returns the entries that each edit changes, for each edit that changes any. */
const changes = (onChange: Mock<OnChange>): enumeration.Entry[][] =>
  onChange.mock.calls
    .map(([edit]) => edit.changed)
    .filter((changed) => changed.length > 0);

/** @returns whether a pointer at the element's center lands on it. */
const takesPointer = (el: Element): boolean => {
  const { left, top, width, height } = el.getBoundingClientRect();
  return el.contains(document.elementFromPoint(left + width / 2, top + height / 2));
};

const fontOf = (el: Element): string => getComputedStyle(el).fontFamily;

describe("Enumeration.Table", () => {
  it("should show a fixed color in the code font and a named one in the text font", async () => {
    const Wrapper = createSynnaxWrapper({ client: null });
    const fixed = { type: "fixed" as const, value: color.construct("#d67e00") };
    await renderThemed(
      <Wrapper>
        <Enumeration.Table
          value={[
            { key: "off", name: "Off", value: 0, abnormal: false, color: fixed },
            {
              key: "on",
              name: "On",
              value: 1,
              abnormal: false,
              color: { type: "named", key: "unknown" },
            },
          ]}
          onChange={() => {}}
          onMove={() => {}}
        />
      </Wrapper>,
    );
    const text = fontOf(screen.getByLabelText("Name 1"));
    const digits = screen.getByLabelText("Color 1");
    expect(fontOf(digits)).not.toBe(text);
    expect(fontOf(screen.getByText("#"))).toBe(fontOf(digits));
    expect(getComputedStyle(digits).textTransform).toBe("uppercase");
    expect(fontOf(screen.getByLabelText("Color 2"))).toBe(text);
  });

  it("should take a pointer on the swatch and the warning of a color cell", async () => {
    const Wrapper = createSynnaxWrapper({ client: null });
    await renderThemed(
      <Wrapper>
        <Enumeration.Table
          value={[{ key: "off", name: "", value: 0, abnormal: false }]}
          onChange={() => {}}
          onMove={() => {}}
        />
      </Wrapper>,
    );
    expect(takesPointer(screen.getByRole("button", { name: "Pick color 1" }))).toBe(
      true,
    );
    expect(takesPointer(screen.getByRole("img", { name: /It has no name/ }))).toBe(
      true,
    );
  });

  describe("swatch and drag", () => {
    const wrapper = createSynnaxWrapper({ client: null });

    interface HarnessProps {
      initial: enumeration.Entry[];
      onChange?: OnChange;
      preview?: boolean;
    }

    // Holds the entries and applies each edit, as a form would.
    const Harness = ({ initial, onChange, ...rest }: HarnessProps): ReactElement => {
      const [value, setValue] = useState(initial);
      return (
        <Enumeration.Table
          value={value}
          onChange={(edit) => {
            onChange?.(edit);
            const { changed, added, removed } = edit;
            setValue((prev) => [
              ...prev
                .filter((e) => !removed.includes(e.key))
                .map((e) => changed.find((c) => c.key === e.key) ?? e),
              ...added,
            ]);
          }}
          onMove={vi.fn()}
          {...rest}
        />
      );
    };

    const renderTable = (props: HarnessProps, as: FC<PropsWithChildren> = wrapper) =>
      render(<Harness {...props} />, { wrapper: as });

    it("should send a color picked in the swatch once, when the picker closes", () => {
      const onChange = vi.fn<OnChange>();
      renderTable({ initial: [createEntry("Off", 0)], onChange });
      const swatch = screen.getByRole("button", { name: "Pick color 1" });
      fireEvent.click(swatch);
      const hex = screen.getByLabelText("Hex");
      fireEvent.change(hex, { target: { value: "#00ff00" } });
      fireEvent.change(hex, { target: { value: "#0000ff" } });
      expect(changes(onChange)).toEqual([]);
      fireEvent.click(swatch);
      expect(changes(onChange)).toEqual([
        [
          expect.objectContaining({
            color: { type: "fixed", value: color.construct("#0000ff") },
          }),
        ],
      ]);
    });

    it("should picture a dragged entry with its name and color", () => {
      const fixed = { type: "fixed" as const, value: color.construct("#00ff00") };
      render(
        <Haul.Provider>
          <Harness initial={[createEntry("Running", 1, { color: fixed })]} />
        </Haul.Provider>,
        { wrapper },
      );
      fireEvent.dragStart(screen.getByRole("button", { name: "Move entry 1" }), {
        dataTransfer: new DataTransfer(),
      });
      const tile = document.querySelector<HTMLElement>(".lyra-haul-preview__tile");
      expect(tile?.style.background).toBe("rgb(0, 255, 0)");
      expect(screen.getByText("Value 1")).toBeTruthy();
    });

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
        const onChange = vi.fn<OnChange>();
        renderTable({ initial: [createEntry("Off", 0)], onChange }, live);
        fireEvent.click(screen.getByRole("button", { name: "Pick color 1" }));
        fireEvent.click(await screen.findByRole("button", { name: named.name }));
        expect(changes(onChange)).toContainEqual([
          expect.objectContaining({ color: { type: "named", key: named.key } }),
        ]);
      });
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
});
