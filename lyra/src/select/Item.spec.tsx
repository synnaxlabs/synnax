// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { act, fireEvent, render, type RenderResult } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { type ReactElement, useState } from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { Dialog } from "@/dialog";
import { List } from "@/list";
import { Select } from "@/select";
import { mockGeometry } from "@/testutil/dom";
import { createRenderCounter } from "@/testutil/renders";
import { Triggers } from "@/triggers";

interface RenderOptions {
  value?: string;
  visible?: boolean;
  onChange?: (key: string | null) => void;
  search?: boolean;
  data?: string[];
  /** Adds a "time" item below the list, with a hint that its label leaves out. */
  pinned?: boolean;
}

const renderSelect = ({
  value,
  visible = false,
  onChange = vi.fn(),
  search = false,
  data = ["alpha", "bravo"],
  pinned = false,
}: RenderOptions = {}): RenderResult =>
  render(
    <Triggers.Provider>
      <Dialog.Frame visible={visible}>
        <Select.Frame<string, undefined>
          data={data}
          value={value}
          onChange={onChange}
          allowNone
        >
          <Select.SingleTrigger placeholder="Pick" />
          <Select.Dialog>
            {search && <Select.Search />}
            <Select.List>
              <Select.Item itemKey="none">
                <span>None</span>
              </Select.Item>
              <Select.Items<string>>
                {({ key, ...p }): ReactElement => (
                  <Select.Item key={key} {...p}>
                    {p.itemKey}
                  </Select.Item>
                )}
              </Select.Items>
              <Select.Item itemKey="custom">Custom</Select.Item>
            </Select.List>
            {pinned && (
              <Select.Item itemKey="time" label="Time">
                <span>Time</span>
                <span>Uses each channel's time.</span>
              </Select.Item>
            )}
          </Select.Dialog>
        </Select.Frame>
      </Dialog.Frame>
    </Triggers.Provider>,
  );

const hovered = (c: RenderResult): string | undefined =>
  c.baseElement.querySelector(".lyra-list__item.lyra--hovered")?.id;

const keyDown = (code: string): void => {
  act(() => {
    fireEvent.keyDown(window, { code });
  });
  act(() => {
    fireEvent.keyUp(window, { code });
  });
};

describe("Select.Item", () => {
  describe("closed dialog", () => {
    it("should show the selected fixed item's children in the trigger", () => {
      const c = renderSelect({ value: "custom" });
      const trigger = c.getByRole("button");
      expect(trigger.textContent).toBe("Custom");
      expect(trigger.querySelector(".lyra-select__label")).not.toBeNull();
    });

    it("should not render unselected fixed items", () => {
      const c = renderSelect({ value: "custom" });
      expect(c.queryByText("None")).toBeNull();
    });

    it("should show the placeholder when nothing is selected", () => {
      const c = renderSelect();
      expect(c.getByRole("button").textContent).toBe("Pick");
    });
  });

  describe("open dialog", () => {
    it("should render fixed items around the data block in page order", () => {
      const c = renderSelect({ visible: true });
      const ids = Array.from(
        c.baseElement.querySelectorAll('[role="option"]'),
        (el) => el.id,
      );
      expect(ids).toEqual(["none", "alpha", "bravo", "custom"]);
    });

    it("should show the selected label in the row and the trigger at once", () => {
      const c = renderSelect({ value: "custom", visible: true });
      expect(c.getAllByText("Custom")).toHaveLength(2);
    });

    it("should walk fixed items and data keys in page order and wrap", () => {
      const c = renderSelect({ visible: true });
      const walk: (string | undefined)[] = [];
      for (let i = 0; i < 5; i++) {
        keyDown("ArrowDown");
        walk.push(hovered(c));
      }
      expect(walk).toEqual(["none", "alpha", "bravo", "custom", "none"]);
    });

    it("should wrap upward from the first fixed item to the last", () => {
      const c = renderSelect({ visible: true });
      keyDown("ArrowDown");
      keyDown("ArrowUp");
      expect(hovered(c)).toBe("custom");
    });

    it("should select the hovered fixed item on Enter", () => {
      const onChange = vi.fn();
      renderSelect({ visible: true, onChange });
      keyDown("ArrowDown");
      keyDown("Enter");
      expect(onChange).toHaveBeenCalledWith("none", { clicked: "none" });
    });

    it("should select a fixed item on click", () => {
      const onChange = vi.fn();
      const c = renderSelect({ visible: true, onChange });
      fireEvent.click(c.getByText("Custom"));
      expect(onChange).toHaveBeenCalledWith("custom", { clicked: "custom" });
    });

    it("should move the hover onto the fixed item under the pointer", () => {
      const c = renderSelect({ visible: true });
      const custom = c.getByText("Custom");
      fireEvent.pointerMove(custom, { pointerType: "mouse", clientX: 1 });
      fireEvent.pointerMove(custom, { pointerType: "mouse", clientX: 2 });
      expect(hovered(c)).toBe("custom");
    });
  });

  describe("search", () => {
    const type = (c: RenderResult, term: string): void => {
      fireEvent.change(c.getByPlaceholderText("Search..."), {
        target: { value: term },
      });
    };

    it("should hide fixed items whose text does not match", () => {
      const c = renderSelect({ visible: true, search: true });
      type(c, "cus");
      expect(c.baseElement.querySelector("#none")?.hasAttribute("hidden")).toBe(true);
      expect(c.baseElement.querySelector("#custom")?.hasAttribute("hidden")).toBe(
        false,
      );
    });

    it("should match without regard to case", () => {
      const c = renderSelect({ visible: true, search: true });
      type(c, "NON");
      expect(c.baseElement.querySelector("#none")?.hasAttribute("hidden")).toBe(false);
    });

    it("should skip hidden fixed items in the walk", () => {
      const c = renderSelect({ visible: true, search: true });
      type(c, "cus");
      keyDown("ArrowDown");
      expect(hovered(c)).toBe("alpha");
      keyDown("ArrowUp");
      expect(hovered(c)).toBe("custom");
    });

    it("should never hide a fixed item outside the list", () => {
      const c = renderSelect({ visible: true, search: true, pinned: true });
      type(c, "zzz");
      expect(c.baseElement.querySelector("#custom")?.hasAttribute("hidden")).toBe(true);
      expect(c.baseElement.querySelector("#time")?.hasAttribute("hidden")).toBe(false);
    });

    it("should leave a fixed item outside the list out of its rows", () => {
      const c = renderSelect({ visible: true, search: true, pinned: true, data: [] });
      type(c, "channel");
      const scroll = c.baseElement.querySelector(".lyra-list__scroll");
      expect(scroll?.classList.contains("lyra-list__scroll--empty")).toBe(true);
    });
  });

  describe("pinned", () => {
    it("should walk a fixed item outside the list after it", () => {
      const c = renderSelect({ visible: true, pinned: true });
      keyDown("ArrowDown");
      keyDown("ArrowUp");
      expect(hovered(c)).toBe("time");
    });

    it("should select a fixed item outside the list on click", () => {
      const onChange = vi.fn();
      const c = renderSelect({ visible: true, pinned: true, onChange });
      fireEvent.click(c.getByText("Uses each channel's time."));
      expect(onChange).toHaveBeenCalledWith("time", { clicked: "time" });
    });
  });

  describe("label", () => {
    it("should show the label in a closed trigger instead of the children", () => {
      const c = renderSelect({ value: "time", pinned: true });
      expect(c.getByRole("button").textContent).toBe("Time");
    });

    it("should show the label in the trigger and the children in the row", () => {
      const c = renderSelect({ value: "time", visible: true, pinned: true });
      const trigger = c.baseElement.querySelector(".lyra-dialog__trigger");
      expect(trigger?.textContent).toBe("Time");
      expect(c.baseElement.querySelector("#time")?.textContent).toBe(
        "TimeUses each channel's time.",
      );
    });
  });
});

describe("Select.MultipleTrigger", () => {
  it("should show a selected fixed item's children in its tag", () => {
    const c = render(
      <Dialog.Frame>
        <Select.Frame<string, undefined>
          multiple
          data={["alpha"]}
          value={["all"]}
          onChange={vi.fn()}
        >
          <Select.MultipleTrigger placeholder="Pick" />
          <Select.Dialog>
            <Select.List>
              <Select.Item itemKey="all">All</Select.Item>
            </Select.List>
          </Select.Dialog>
        </Select.Frame>
      </Dialog.Frame>,
    );
    const tag = c.baseElement.querySelector(".lyra-tag");
    expect(tag?.textContent).toContain("All");
    expect(tag?.className).not.toContain("error");
  });
});

describe("Select render isolation", () => {
  const KEYS = Array.from({ length: 100 }, (_, i) => `key-${i}`);

  it("should re-render only the changed rows when an inline onChange selects", () => {
    const { Counted, counts, reset } = createRenderCounter();
    const item = ({ key, ...p }: Select.ItemProps<string>): ReactElement => (
      <Counted key={key} id={p.itemKey}>
        <Select.Item {...p}>{p.itemKey}</Select.Item>
      </Counted>
    );
    const Harness = (): ReactElement => {
      const [value, setValue] = useState<string | null>(null);
      return (
        <Triggers.Provider>
          <Dialog.Frame visible>
            <Select.Frame<string, undefined>
              data={KEYS}
              value={value ?? undefined}
              onChange={(next: string | null) => setValue(next)}
              allowNone
            >
              <Select.List>
                <Select.Items<string>>{item}</Select.Items>
              </Select.List>
            </Select.Frame>
          </Dialog.Frame>
        </Triggers.Provider>
      );
    };
    const c = render(<Harness />);
    fireEvent.click(c.getByText("key-3"));
    reset();
    fireEvent.click(c.getByText("key-7"));
    expect([...counts.keys()].sort()).toEqual(["key-3", "key-7"]);
  });

  it("should re-render only the two items an arrow key moves the hover between", () => {
    const { Counted, counts, reset } = createRenderCounter();
    render(
      <Triggers.Provider>
        <Dialog.Frame visible>
          <Select.Frame<string, undefined>
            data={KEYS}
            onChange={vi.fn()}
            allowNone
            initialHover={3}
          >
            <Select.List>
              <Select.Items<string>>
                {({ key, ...p }) => (
                  <Counted key={key} id={p.itemKey}>
                    <Select.Item {...p}>{p.itemKey}</Select.Item>
                  </Counted>
                )}
              </Select.Items>
            </Select.List>
          </Select.Frame>
        </Dialog.Frame>
      </Triggers.Provider>,
    );
    reset();
    act(() => {
      fireEvent.keyDown(window, { code: "ArrowDown" });
    });
    act(() => {
      fireEvent.keyUp(window, { code: "ArrowDown" });
    });
    expect([...counts.entries()].sort()).toEqual([
      ["key-3", 1],
      ["key-4", 1],
    ]);
  });

  it("should render only the fixed items whose visibility changes on a search", () => {
    const { Counted, counts, reset } = createRenderCounter();
    const c = render(
      <Triggers.Provider>
        <Dialog.Frame visible>
          <Select.Frame<string, undefined> onChange={vi.fn()} allowNone>
            <Select.Dialog>
              <Select.Search />
              <Select.List>
                {KEYS.map((key) => (
                  <Counted key={key} id={key}>
                    <Select.Item itemKey={key}>{key}</Select.Item>
                  </Counted>
                ))}
              </Select.List>
            </Select.Dialog>
          </Select.Frame>
        </Dialog.Frame>
      </Triggers.Provider>,
    );
    const input = c.getByPlaceholderText("Search...");
    fireEvent.change(input, { target: { value: "key-1" } });
    reset();
    // key-1 and key-10 to key-19 hide. key-2 and key-20 to key-29 show.
    fireEvent.change(input, { target: { value: "key-2" } });
    const changed = ["key-1", ...Array.from({ length: 10 }, (_, i) => `key-1${i}`)];
    changed.push("key-2", ...Array.from({ length: 10 }, (_, i) => `key-2${i}`));
    expect([...counts.keys()].sort()).toEqual(changed.sort());
    counts.forEach((count) => expect(count).toBe(1));
  });

  describe("checkable", () => {
    interface CheckableProps {
      onActivate?: () => void;
      onChange?: (keys: string[]) => void;
      disabled?: boolean;
    }

    const Checkable = ({
      onActivate,
      onChange,
      disabled,
    }: CheckableProps): ReactElement => {
      const [value, setValue] = useState<string[]>([]);
      const handleChange = (next: string[]): void => {
        setValue(next);
        onChange?.(next);
      };
      return (
        <Triggers.Provider>
          <Select.Frame<string, undefined>
            multiple
            data={["alpha", "bravo"]}
            value={value}
            onChange={handleChange}
          >
            <List.Scroll>
              <List.Items<string>>
                {({ key, ...p }): ReactElement => (
                  <Select.Item
                    key={key}
                    {...p}
                    checkable
                    onActivate={onActivate}
                    disabled={disabled}
                  >
                    {p.itemKey}
                  </Select.Item>
                )}
              </List.Items>
            </List.Scroll>
          </Select.Frame>
        </Triggers.Provider>
      );
    };

    const item = (c: RenderResult, key: string): HTMLElement =>
      c.baseElement.querySelector<HTMLElement>(`#${key}`)!;

    const checkbox = (c: RenderResult, key: string): HTMLInputElement =>
      item(c, key).querySelector<HTMLInputElement>('input[type="checkbox"]')!;

    it("should call onActivate on a plain click without selecting", () => {
      const onActivate = vi.fn();
      const c = render(<Checkable onActivate={onActivate} />);
      fireEvent.click(item(c, "alpha"));
      expect(onActivate).toHaveBeenCalledTimes(1);
      expect(checkbox(c, "alpha").checked).toBe(false);
    });

    it("should select on a click with Command held without activating", () => {
      const onActivate = vi.fn();
      const c = render(<Checkable onActivate={onActivate} />);
      fireEvent.click(item(c, "alpha"), { metaKey: true });
      expect(onActivate).not.toHaveBeenCalled();
      expect(checkbox(c, "alpha").checked).toBe(true);
    });

    it("should add to the selection through each checkbox", () => {
      const onChange = vi.fn();
      const c = render(<Checkable onChange={onChange} />);
      fireEvent.click(checkbox(c, "alpha"));
      fireEvent.click(checkbox(c, "bravo"));
      expect(onChange).toHaveBeenLastCalledWith(["alpha", "bravo"]);
    });

    it("should empty the selection when the last checkbox clears", () => {
      const onChange = vi.fn();
      const c = render(<Checkable onChange={onChange} />);
      fireEvent.click(checkbox(c, "alpha"));
      fireEvent.click(checkbox(c, "alpha"));
      expect(onChange).toHaveBeenLastCalledWith([]);
      expect(checkbox(c, "alpha").checked).toBe(false);
    });

    it("should not activate when the checkbox is clicked", () => {
      const onActivate = vi.fn();
      const c = render(<Checkable onActivate={onActivate} />);
      fireEvent.click(checkbox(c, "alpha"));
      expect(onActivate).not.toHaveBeenCalled();
    });

    it("should not select a disabled item through its checkbox", async () => {
      const onChange = vi.fn();
      const c = render(<Checkable onChange={onChange} disabled />);
      await userEvent.click(checkbox(c, "alpha"));
      expect(onChange).not.toHaveBeenCalled();
      expect(checkbox(c, "alpha").checked).toBe(false);
    });
  });
});

describe("Select.CreateItem", () => {
  beforeAll(() => mockGeometry(100, 100));

  interface Fruit {
    key: string;
    name: string;
  }

  const FRUITS: Fruit[] = [{ key: "apple", name: "Apple Pie" }];

  interface CreatableProps {
    modal?: boolean;
    onCreate: (name: string) => void;
    onChange?: (key: string | undefined) => void;
    onVisibleChange?: Select.SingleProps<string, Fruit>["onVisibleChange"];
  }

  const Creatable = ({
    modal,
    onCreate,
    onChange = vi.fn(),
    onVisibleChange,
  }: CreatableProps): ReactElement => {
    const { data, getItem, search } = List.useStaticData<string, Fruit>({
      data: FRUITS,
    });
    return (
      <Select.Single<string, Fruit>
        visible
        onVisibleChange={onVisibleChange}
        data={data}
        getItem={getItem}
        value={undefined}
        onChange={onChange}
        resourceName="fruit"
        onSearch={search}
        fixedItems={
          <Select.Item<string> itemKey="pear" textValue="Pear">
            Pear
          </Select.Item>
        }
        footer={
          <Select.CreateItem<string, Fruit>
            resourceName="fruit"
            modal={modal}
            onCreate={onCreate}
          />
        }
      >
        {({ key, ...p }): ReactElement => (
          <Select.Item key={key} {...p}>
            {getItem?.(p.itemKey)?.name}
          </Select.Item>
        )}
      </Select.Single>
    );
  };

  const renderCreatable = (props: CreatableProps): RenderResult =>
    render(<Creatable {...props} />, { wrapper: Triggers.Provider });

  const type = (c: RenderResult, term: string): void => {
    fireEvent.change(c.getByPlaceholderText("Search fruits..."), {
      target: { value: term },
    });
  };

  it("should read New and the noun with an empty search", () => {
    const c = renderCreatable({ onCreate: vi.fn() });
    expect(c.getByRole("option", { name: "New fruit" })).toBeTruthy();
  });

  it("should name the trimmed search text while the user types", () => {
    const c = renderCreatable({ onCreate: vi.fn() });
    type(c, "  Kiwi ");
    expect(c.getByRole("option", { name: 'Create fruit "Kiwi"' })).toBeTruthy();
  });

  it("should hide while an option has the search text as its name, in any case", () => {
    const c = renderCreatable({ onCreate: vi.fn() });
    type(c, "apple pie");
    expect(c.getByRole("option", { name: "Apple Pie" })).toBeTruthy();
    expect(c.queryByText(/^Create fruit/)).toBeNull();
  });

  it("should hide while a fixed option has the search text as its name", () => {
    const c = renderCreatable({ onCreate: vi.fn() });
    type(c, "PEAR");
    expect(c.getByRole("option", { name: "Pear" })).toBeTruthy();
    expect(c.queryByText(/^Create fruit/)).toBeNull();
  });

  it("should pass the trimmed search text to onCreate and leave the selection alone", () => {
    const onCreate = vi.fn();
    const onChange = vi.fn();
    const c = renderCreatable({ onCreate, onChange });
    type(c, "Kiwi ");
    fireEvent.click(c.getByRole("option", { name: 'Create fruit "Kiwi"' }));
    expect(onCreate).toHaveBeenCalledWith("Kiwi");
    expect(onChange).not.toHaveBeenCalled();
  });

  it("should hide a picked name until the search changes", () => {
    const onCreate = vi.fn();
    const c = renderCreatable({ onCreate });
    type(c, "Kiwi");
    fireEvent.click(c.getByRole("option", { name: 'Create fruit "Kiwi"' }));
    expect(c.queryByText(/^Create fruit/)).toBeNull();
    type(c, "Kiw");
    expect(c.getByRole("option", { name: 'Create fruit "Kiw"' })).toBeTruthy();
    expect(onCreate).toHaveBeenCalledExactlyOnceWith("Kiwi");
  });

  it("should close the dialog and pass an empty name to onCreate from New", () => {
    const onCreate = vi.fn();
    const onVisibleChange = vi.fn();
    const c = renderCreatable({ onCreate, onVisibleChange });
    fireEvent.click(c.getByRole("option", { name: "New fruit" }));
    expect(onCreate).toHaveBeenCalledWith("");
    expect(onVisibleChange).toHaveBeenCalledWith(false);
  });

  it("should keep the dialog open when it creates from the search text", () => {
    const onVisibleChange = vi.fn();
    const c = renderCreatable({ onCreate: vi.fn(), onVisibleChange });
    type(c, "Kiwi");
    fireEvent.click(c.getByRole("option", { name: 'Create fruit "Kiwi"' }));
    expect(onVisibleChange).not.toHaveBeenCalled();
  });

  it("should close the dialog and keep the option on a modal create from the search", () => {
    const onCreate = vi.fn();
    const onVisibleChange = vi.fn();
    const c = renderCreatable({ modal: true, onCreate, onVisibleChange });
    type(c, "Kiwi");
    fireEvent.click(c.getByRole("option", { name: 'Create fruit "Kiwi"' }));
    expect(onCreate).toHaveBeenCalledExactlyOnceWith("Kiwi");
    expect(onVisibleChange).toHaveBeenCalledWith(false);
    expect(c.getByRole("option", { name: 'Create fruit "Kiwi"' })).toBeTruthy();
  });

  it("should create on Enter once the arrow keys reach it", () => {
    const onCreate = vi.fn();
    const c = renderCreatable({ onCreate });
    type(c, "Kiwi");
    act(() => {
      fireEvent.keyDown(window, { code: "ArrowDown" });
      fireEvent.keyDown(window, { code: "Enter" });
    });
    expect(onCreate).toHaveBeenCalledWith("Kiwi");
  });
});
