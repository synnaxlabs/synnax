// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { act, fireEvent, render } from "@testing-library/react";
import { type ReactElement } from "react";
import { beforeAll, describe, expect, it, vi } from "vitest";

import { Dialog } from "@/dialog";
import { List } from "@/list";
import { Select } from "@/select";
import { createRenderCounter } from "@/testutil";
import { mockGeometry } from "@/testutil/dom";
import { Triggers } from "@/triggers";

const DATA = Array.from({ length: 300 }, (_, i) => `${i}`);

const row = ({ key, ...rest }: List.ItemProps<string>): ReactElement => (
  <Select.Item key={key} {...rest}>
    {rest.itemKey}
  </Select.Item>
);

interface PinnedProps {
  pinned?: string[];
  initialHover?: number;
}

const Pinned = ({ pinned, initialHover }: PinnedProps): ReactElement => (
  <Triggers.Provider>
    <Dialog.Frame visible>
      <Select.Frame
        data={DATA}
        onChange={vi.fn()}
        initialHover={initialHover}
        pinned={pinned}
        itemHeight={33}
        virtual
      >
        <List.Scroll>
          <Select.Items<string>>{row}</Select.Items>
        </List.Scroll>
      </Select.Frame>
    </Dialog.Frame>
  </Triggers.Provider>
);

describe("Select.Frame pinned", () => {
  beforeAll(() => mockGeometry(100, 100));

  it("should keep a pinned option mounted outside the window", () => {
    const c = render(<Pinned />);
    expect(c.queryByText("150", { exact: true })).toBeNull();
    c.rerender(<Pinned pinned={["150"]} />);
    expect(c.queryByText("150", { exact: true })).not.toBeNull();
  });

  it("should keep a pinned option and the hovered option mounted together", () => {
    const c = render(<Pinned pinned={["150"]} initialHover={DATA.length - 1} />);
    expect(c.queryByText("150", { exact: true })).not.toBeNull();
    expect(c.queryByText("299", { exact: true })).not.toBeNull();
    expect(c.queryByText("151", { exact: true })).toBeNull();
  });
});

const { Counted, counts, reset } = createRenderCounter();

const OPTIONS = ["x"];

const nested = ({ key, ...rest }: List.ItemProps<string>): ReactElement => (
  <div key={key}>
    <Counted id={rest.itemKey}>
      <Select.Frame multiple data={OPTIONS} value={OPTIONS} onChange={vi.fn()}>
        <span>{rest.itemKey}</span>
      </Select.Frame>
    </Counted>
  </div>
);

const Outer = ({ data }: { data: string[] }): ReactElement => (
  <List.Frame data={data}>
    <List.Scroll>
      <List.Items<string>>{nested}</List.Items>
    </List.Scroll>
  </List.Frame>
);

describe("Select.Frame nested in a list", () => {
  it("should not render when the keys of the outer list change", () => {
    const c = render(<Outer data={["a", "b"]} />);
    reset();
    c.rerender(<Outer data={["a", "b", "c"]} />);
    expect(Object.fromEntries(counts)).toEqual({});
  });
});

interface OrderedProps extends Pick<
  List.FrameProps<string, record.Keyed<string>>,
  "getItem" | "subscribe"
> {
  selectedFirst?: boolean;
  visible?: boolean;
  value?: string[];
}

const Ordered = ({
  selectedFirst,
  visible = true,
  value = ["b", "none"],
  getItem,
  subscribe,
}: OrderedProps): ReactElement => (
  <Triggers.Provider>
    <Dialog.Frame visible={visible} onVisibleChange={vi.fn()}>
      <Select.Frame
        multiple
        data={["a", "b"]}
        value={value}
        onChange={vi.fn()}
        getItem={getItem}
        subscribe={subscribe}
        selectedFirst={selectedFirst}
      >
        <Select.List>
          <Select.Item itemKey="none">None</Select.Item>
          <Select.Items<string>>{row}</Select.Items>
        </Select.List>
      </Select.Frame>
    </Dialog.Frame>
  </Triggers.Provider>
);

interface OrderedSingleProps extends Pick<
  List.FrameProps<string, record.Keyed<string>>,
  "getItem" | "subscribe"
> {
  value: string;
  search?: boolean;
}

const OrderedSingle = ({
  value,
  search = false,
  getItem,
  subscribe,
}: OrderedSingleProps): ReactElement => (
  <Triggers.Provider>
    <Dialog.Frame visible onVisibleChange={vi.fn()}>
      <Select.Frame
        data={["a", "b"]}
        value={value}
        onChange={vi.fn()}
        getItem={getItem}
        subscribe={subscribe}
        selectedFirst
      >
        {search && <Select.Search />}
        <Select.List>
          <Select.Item itemKey="none">None</Select.Item>
          <Select.Items<string>>{row}</Select.Items>
        </Select.List>
      </Select.Frame>
    </Dialog.Frame>
  </Triggers.Provider>
);

/** A fetched list whose entries load when the spec loads them. */
const createLoader = (...loaded: string[]) => {
  const ready = new Set(loaded);
  const listeners = new Map<string, () => void>();
  const subscribed: string[] = [];
  const getOne = (key: string): record.Keyed<string> | undefined =>
    ready.has(key) ? { key } : undefined;
  const getItem = ((keys: string | string[]) =>
    Array.isArray(keys)
      ? keys.flatMap((key) => getOne(key) ?? [])
      : getOne(keys)) as List.GetItem<string, record.Keyed<string>>;
  const subscribe = (callback: () => void, key: string): (() => void) => {
    subscribed.push(key);
    listeners.set(key, callback);
    return () => listeners.delete(key);
  };
  const load = (key: string): void => {
    ready.add(key);
    act(() => listeners.get(key)?.());
  };
  return { getItem, subscribe, load, listeners, subscribed };
};

describe("Select.Frame selectedFirst", () => {
  const options = (c: ReturnType<typeof render>): string[] =>
    c.getAllByRole("option").map((option) => option.textContent ?? "");

  it("should keep the order of its data when unset", () => {
    expect(options(render(<Ordered />))).toEqual(["None", "a", "b"]);
  });

  it("should read each entry again when the dialog opens again", () => {
    const readable = new Set<string>();
    const getOne = (key: string): record.Keyed<string> | undefined =>
      readable.has(key) ? { key } : undefined;
    const getItem = ((keys: string | string[]) =>
      Array.isArray(keys)
        ? keys.flatMap((key) => getOne(key) ?? [])
        : getOne(keys)) as List.GetItem<string, record.Keyed<string>>;
    const c = render(<Ordered selectedFirst getItem={getItem} />);
    expect(options(c)).toEqual(["None", "a", "b"]);
    readable.add("b");
    c.rerender(<Ordered selectedFirst getItem={getItem} visible={false} />);
    c.rerender(<Ordered selectedFirst getItem={getItem} />);
    expect(options(c)).toEqual(["None", "b", "a"]);
  });

  it("should list the selected entries of its data first", () => {
    expect(options(render(<Ordered selectedFirst />))).toEqual(["None", "b", "a"]);
  });

  it("should list the selected entry of a single selection first", () => {
    expect(options(render(<OrderedSingle value="b" />))).toEqual(["None", "b", "a"]);
  });

  it("should list a selected entry first once it loads", () => {
    const { getItem, subscribe, load, listeners } = createLoader("a");
    const c = render(
      <OrderedSingle value="b" getItem={getItem} subscribe={subscribe} />,
    );
    expect(options(c)).toEqual(["None", "a", "b"]);
    expect([...listeners.keys()]).toEqual(["b"]);
    load("b");
    expect(options(c)).toEqual(["None", "b", "a"]);
    expect(listeners.size).toBe(0);
  });

  it("should subscribe once to each entry that loads after the dialog opens", () => {
    const { getItem, subscribe, load, listeners, subscribed } = createLoader();
    const c = render(
      <Ordered
        selectedFirst
        value={["b", "a"]}
        getItem={getItem}
        subscribe={subscribe}
      />,
    );
    load("b");
    load("a");
    expect(options(c)).toEqual(["None", "b", "a"]);
    expect(subscribed).toEqual(["b", "a"]);
    expect(listeners.size).toBe(0);
  });

  it.each([
    ["scrolls", () => window.dispatchEvent(new WheelEvent("wheel"))],
    [
      "moves the cursor",
      () => window.dispatchEvent(new KeyboardEvent("keydown", { key: "ArrowDown" })),
    ],
    [
      "searches",
      (c: ReturnType<typeof render>) => {
        const search = c.getByRole("textbox");
        fireEvent.change(search, { target: { value: "a" } });
        fireEvent.change(search, { target: { value: "" } });
      },
    ],
  ])("should not list an entry first that loads after the user %s", (_, input) => {
    const { getItem, subscribe, load, listeners } = createLoader("a");
    const c = render(
      <OrderedSingle value="b" search getItem={getItem} subscribe={subscribe} />,
    );
    input(c);
    expect(listeners.size).toBe(0);
    load("b");
    expect(options(c)).toEqual(["None", "a", "b"]);
  });

  it("should keep a selected fixed item of a single selection in its place", () => {
    expect(options(render(<OrderedSingle value="none" />))).toEqual(["None", "a", "b"]);
  });
});
