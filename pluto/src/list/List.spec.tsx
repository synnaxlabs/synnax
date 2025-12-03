// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { observe, type record } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { act, useState } from "react";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { Button } from "@/button";
import { renderProp } from "@/component/renderProp";
import { List } from "@/list";
import { mockBoundingClientRect } from "@/testutil/dom";

describe("List", () => {
  interface Context {
    virtual: boolean;
    name: string;
  }
  const CONTEXTS: Context[] = [
    { name: "non-virtual", virtual: false },
    { name: "virtual", virtual: true },
  ];
  CONTEXTS.forEach((context) => {
    beforeAll(() => {
      Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
    });
    describe(context.name, () => {
      describe("basic item rendering", () => {
        it("should render a list of items", () => {
          const result = render(
            <List.Frame data={["1", "2", "3"]} virtual={context.virtual}>
              <List.Items>
                {({ key, ...rest }: List.ItemProps<string>) => (
                  <List.Item key={key} {...rest}>
                    {key}
                  </List.Item>
                )}
              </List.Items>
            </List.Frame>,
          );
          expect(result.getByText("1")).toBeTruthy();
          expect(result.getByText("2")).toBeTruthy();
          expect(result.getByText("3")).toBeTruthy();
        });

        it("should allow the caller to provide a custom item getter", () => {
          const ITEMS: record.KeyedNamed<string>[] = [
            { key: "1", name: "one" },
            { key: "2", name: "two" },
            { key: "3", name: "three" },
          ];
          const getItem = List.createGetItem(
            (key) => ITEMS.find((item) => item.key === key),
            (keys) =>
              keys
                .map((key) => ITEMS.find((item) => item.key === key))
                .filter((item) => item != null),
          );
          const result = render(
            <List.Frame<string, record.KeyedNamed<string>>
              data={["1", "2", "3"]}
              getItem={getItem}
              virtual={context.virtual}
            >
              <List.Items<string>>
                {({ itemKey }) => <div key={itemKey}>{getItem(itemKey)?.name}</div>}
              </List.Items>
            </List.Frame>,
          );
          expect(result.getByText("one")).toBeTruthy();
          expect(result.getByText("two")).toBeTruthy();
          expect(result.getByText("three")).toBeTruthy();
        });

        it("should allow the caller to pass a subscription function for whenever the item content changes", () => {
          const data: record.KeyedNamed<string>[] = [
            { key: "1", name: "one" },
            { key: "2", name: "two" },
            { key: "3", name: "three" },
          ];
          const getItem = ((
            key: string | string[],
          ): record.KeyedNamed<string> | record.KeyedNamed<string>[] | undefined => {
            if (Array.isArray(key)) return key.map((k) => ({ key: k, name: k }));
            if (key === "1") return data[0];
            if (key === "2") return data[1];
            if (key === "3") return data[2];
            return undefined;
          }) as List.GetItem<string, record.KeyedNamed<string>>;
          const obs = new observe.Observer<void>();
          const itemProp = renderProp(({ itemKey }: List.ItemProps<string>) => {
            const item = List.useItem<string, record.KeyedNamed<string>>(itemKey);
            return <div key={itemKey}>{item?.name}</div>;
          });
          const result = render(
            <List.Frame<string, record.KeyedNamed<string>>
              data={["1", "2", "3"]}
              getItem={getItem}
              subscribe={(callback) => obs.onChange(callback)}
              virtual={context.virtual}
            >
              <List.Items<string>>{itemProp}</List.Items>
            </List.Frame>,
          );
          expect(result.getByText("one")).toBeTruthy();
          expect(result.getByText("two")).toBeTruthy();
          expect(result.getByText("three")).toBeTruthy();
          data[0] = { key: "1", name: "one-updated" };
          act(() => {
            obs.notify();
          });
          expect(result.getByText("one-updated")).toBeTruthy();
        });
      });
      describe("on fetch more", () => {
        it("should not call on fetchMore when no list items are", () => {
          const fetchMore = vi.fn();
          render(
            <List.Frame data={[]} virtual={context.virtual} onFetchMore={fetchMore} />,
          );
          expect(fetchMore).not.toHaveBeenCalled();
        });
        it("should call onFetchMore the first time list items are mounted", () => {
          const fetchMore = vi.fn();
          render(
            <List.Frame
              data={["1", "2", "3"]}
              virtual={context.virtual}
              onFetchMore={fetchMore}
            >
              <List.Items>{({ key }) => <div key={key}>{key}</div>}</List.Items>
            </List.Frame>,
          );
          expect(fetchMore).toHaveBeenCalled();
        });

        it("should not call onFetchMore if the list items are re-mounted", () => {
          const fetchMore = vi.fn();
          const Component = () => {
            const [listItemsVisible, setListItemsVisible] = useState(true);
            return (
              <List.Frame
                data={["1", "2", "3"]}
                virtual={context.virtual}
                onFetchMore={fetchMore}
              >
                <Button.Button onClick={() => setListItemsVisible(!listItemsVisible)}>
                  Toggle
                </Button.Button>
                {listItemsVisible && (
                  <List.Items>{({ key }) => <div key={key}>{key}</div>}</List.Items>
                )}
              </List.Frame>
            );
          };
          const result = render(<Component />);
          const expectedCalls = context.virtual ? 2 : 1;
          expect(fetchMore).toHaveBeenCalledTimes(expectedCalls);
          result.getByText("Toggle").click();
          expect(fetchMore).toHaveBeenCalledTimes(expectedCalls);
          result.getByText("Toggle").click();
          expect(fetchMore).toHaveBeenCalledTimes(expectedCalls);
        });
      });
    });
  });

  describe("scroll-based pagination (non-virtual)", () => {
    let mockObserverCallback: IntersectionObserverCallback;
    let mockObserverInstance: {
      observe: ReturnType<typeof vi.fn>;
      disconnect: ReturnType<typeof vi.fn>;
      unobserve: ReturnType<typeof vi.fn>;
    };
    const MockIntersectionObserver = vi.fn((callback: IntersectionObserverCallback) => {
      mockObserverCallback = callback;
      mockObserverInstance = {
        observe: vi.fn(),
        disconnect: vi.fn(),
        unobserve: vi.fn(),
      };
      return mockObserverInstance;
    });

    beforeAll(() => {
      Element.prototype.getBoundingClientRect = mockBoundingClientRect(0, 0, 100, 100);
    });

    beforeEach(() => {
      vi.stubGlobal("IntersectionObserver", MockIntersectionObserver);
    });

    afterEach(() => {
      vi.unstubAllGlobals();
      vi.clearAllMocks();
    });

    it("should create an IntersectionObserver for the sentinel element", () => {
      const fetchMore = vi.fn();
      render(
        <List.Frame data={["1", "2", "3"]} virtual={false} onFetchMore={fetchMore}>
          <List.Items>{({ key }) => <div key={key}>{key}</div>}</List.Items>
        </List.Frame>,
      );
      expect(MockIntersectionObserver).toHaveBeenCalled();
      expect(mockObserverInstance.observe).toHaveBeenCalled();
    });

    it("should call onFetchMore when sentinel intersects", () => {
      const fetchMore = vi.fn();
      render(
        <List.Frame data={["1", "2", "3"]} virtual={false} onFetchMore={fetchMore}>
          <List.Items>{({ key }) => <div key={key}>{key}</div>}</List.Items>
        </List.Frame>,
      );
      expect(fetchMore).toHaveBeenCalledTimes(1);

      act(() => {
        mockObserverCallback(
          [{ isIntersecting: true } as IntersectionObserverEntry],
          {} as IntersectionObserver,
        );
      });

      expect(fetchMore).toHaveBeenCalledTimes(2);
    });

    it("should not call onFetchMore multiple times while fetching", () => {
      const fetchMore = vi.fn();
      render(
        <List.Frame data={["1", "2", "3"]} virtual={false} onFetchMore={fetchMore}>
          <List.Items>{({ key }) => <div key={key}>{key}</div>}</List.Items>
        </List.Frame>,
      );
      expect(fetchMore).toHaveBeenCalledTimes(1);

      act(() => {
        mockObserverCallback(
          [{ isIntersecting: true } as IntersectionObserverEntry],
          {} as IntersectionObserver,
        );
      });
      expect(fetchMore).toHaveBeenCalledTimes(2);

      act(() => {
        mockObserverCallback(
          [{ isIntersecting: true } as IntersectionObserverEntry],
          {} as IntersectionObserver,
        );
      });
      expect(fetchMore).toHaveBeenCalledTimes(2);
    });

    it("should allow another fetch after data length changes", () => {
      const fetchMore = vi.fn();
      const Component = ({ data }: { data: string[] }) => (
        <List.Frame data={data} virtual={false} onFetchMore={fetchMore}>
          <List.Items>{({ key }) => <div key={key}>{key}</div>}</List.Items>
        </List.Frame>
      );

      const { rerender } = render(<Component data={["1", "2", "3"]} />);
      expect(fetchMore).toHaveBeenCalledTimes(1);

      act(() => {
        mockObserverCallback(
          [{ isIntersecting: true } as IntersectionObserverEntry],
          {} as IntersectionObserver,
        );
      });
      expect(fetchMore).toHaveBeenCalledTimes(2);

      act(() => {
        mockObserverCallback(
          [{ isIntersecting: true } as IntersectionObserverEntry],
          {} as IntersectionObserver,
        );
      });
      expect(fetchMore).toHaveBeenCalledTimes(2);

      rerender(<Component data={["1", "2", "3", "4", "5"]} />);

      act(() => {
        mockObserverCallback(
          [{ isIntersecting: true } as IntersectionObserverEntry],
          {} as IntersectionObserver,
        );
      });
      expect(fetchMore).toHaveBeenCalledTimes(3);
    });

    it("should not call onFetchMore when sentinel is not intersecting", () => {
      const fetchMore = vi.fn();
      render(
        <List.Frame data={["1", "2", "3"]} virtual={false} onFetchMore={fetchMore}>
          <List.Items>{({ key }) => <div key={key}>{key}</div>}</List.Items>
        </List.Frame>,
      );
      expect(fetchMore).toHaveBeenCalledTimes(1);

      act(() => {
        mockObserverCallback(
          [{ isIntersecting: false } as IntersectionObserverEntry],
          {} as IntersectionObserver,
        );
      });
      expect(fetchMore).toHaveBeenCalledTimes(1);
    });

    it("should disconnect observer on unmount", () => {
      const fetchMore = vi.fn();
      const { unmount } = render(
        <List.Frame data={["1", "2", "3"]} virtual={false} onFetchMore={fetchMore}>
          <List.Items>{({ key }) => <div key={key}>{key}</div>}</List.Items>
        </List.Frame>,
      );

      unmount();
      expect(mockObserverInstance.disconnect).toHaveBeenCalled();
    });
  });
});
