// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { createGetItem, type FrameProps } from "@/list/Frame";
import { useTransformedData } from "@/list/useTransformedData";

interface Source {
  key: number;
  name: string;
}

interface Transformed {
  key: string;
  label: string;
}

const PREFIX = "item:";

const createSource = (
  items: Source[],
  subscribe?: FrameProps<number, Source>["subscribe"],
): Pick<FrameProps<number, Source>, "data" | "getItem" | "subscribe"> => {
  const getOne = (key: number) => items.find((item) => item.key === key);
  return {
    data: items.map(({ key }) => key),
    getItem: createGetItem<number, Source>(getOne, (keys) =>
      keys.map(getOne).filter((item) => item != null),
    ),
    subscribe,
  };
};

const render = (source: ReturnType<typeof createSource>) => {
  const transform = vi.fn(({ key, name }: Source): Transformed => ({
    key: `${PREFIX}${key}`,
    label: name,
  }));
  const hook = renderHook(
    ({ source }) =>
      useTransformedData({
        source,
        toKey: (key): string => `${PREFIX}${key}`,
        fromKey: (key) =>
          key.startsWith(PREFIX) ? Number(key.slice(PREFIX.length)) : null,
        transform,
      }),
    { initialProps: { source } },
  );
  return { ...hook, transform };
};

describe("useTransformedData", () => {
  it("should transform the keys of the source data", () => {
    const { result } = render(
      createSource([
        { key: 1, name: "a" },
        { key: 2, name: "b" },
      ]),
    );
    expect(result.current.data).toEqual(["item:1", "item:2"]);
  });

  it("should read one item or many by transformed key", () => {
    const { result } = render(
      createSource([
        { key: 1, name: "a" },
        { key: 2, name: "b" },
      ]),
    );
    expect(result.current.getItem("item:1")).toEqual({ key: "item:1", label: "a" });
    expect(result.current.getItem(["item:2", "item:3", "other"])).toEqual([
      { key: "item:2", label: "b" },
    ]);
  });

  it("should return nothing for a key with no source key", () => {
    const { result } = render(createSource([{ key: 1, name: "a" }]));
    expect(result.current.getItem("other")).toBeUndefined();
  });

  it("should keep one transformed item for each source item", () => {
    const source = createSource([{ key: 1, name: "a" }]);
    const { result, rerender, transform } = render(source);
    const first = result.current.getItem("item:1");
    rerender({ source });
    expect(result.current.getItem("item:1")).toBe(first);
    expect(transform).toHaveBeenCalledTimes(1);
  });

  it("should transform an item again when the source replaces it", () => {
    const { result, rerender } = render(createSource([{ key: 1, name: "a" }]));
    const first = result.current.getItem("item:1");
    rerender({ source: createSource([{ key: 1, name: "b" }]) });
    const second = result.current.getItem("item:1");
    expect(second).not.toBe(first);
    expect(second).toEqual({ key: "item:1", label: "b" });
  });

  it("should subscribe to the source under the source key", () => {
    const unsubscribe = vi.fn();
    const subscribe = vi.fn(() => unsubscribe);
    const { result } = render(createSource([{ key: 1, name: "a" }], subscribe));
    const callback = vi.fn();
    result.current.subscribe(callback, "item:1")();
    expect(subscribe).toHaveBeenCalledWith(callback, 1);
    expect(unsubscribe).toHaveBeenCalledOnce();
  });

  it("should not subscribe to the source for a key with no source key", () => {
    const subscribe = vi.fn(() => () => {});
    const { result } = render(createSource([{ key: 1, name: "a" }], subscribe));
    result.current.subscribe(vi.fn(), "other")();
    expect(subscribe).not.toHaveBeenCalled();
  });
});
