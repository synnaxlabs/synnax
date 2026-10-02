// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { observe, type record } from "@synnaxlabs/x";
import { act, fireEvent, render } from "@testing-library/react";
import { type ReactElement, useState } from "react";
import { afterEach, beforeAll, beforeEach, describe, expect, it, vi } from "vitest";

import { Button } from "@/button";
import { renderProp } from "@/component/renderProp";
import { List } from "@/list";
import { mockGeometry } from "@/testutil/dom";
import { createRenderCounter } from "@/testutil/renders";

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
    beforeAll(() => mockGeometry(100, 100));
    describe(context.name, () => {
      describe("basic item rendering", () => {
        it("should render a list of items", () => {
          const result = render(
            <List.Frame data={["1", "2", "3"]} virtual={context.virtual}>
              <List.Scroll>
                <List.Items>
                  {({ key, ...rest }: List.ItemProps<string>) => (
                    <List.Item key={key} {...rest}>
                      {key}
                    </List.Item>
                  )}
                </List.Items>
              </List.Scroll>
            </List.Frame>,
          );
          expect(result.getByText("1")).toBeTruthy();
          expect(result.getByText("2")).toBeTruthy();
          expect(result.getByText("3")).toBeTruthy();
        });

        it("should hand onSelect the click event alongside the key", () => {
          const onSelect = vi.fn();
          const result = render(
            <List.Frame data={["1"]} virtual={context.virtual}>
              <List.Scroll>
                <List.Items>
                  {({ key, ...rest }: List.ItemProps<string>) => (
                    <List.Item key={key} onSelect={onSelect} {...rest}>
                      {key}
                    </List.Item>
                  )}
                </List.Items>
              </List.Scroll>
            </List.Frame>,
          );
          fireEvent.click(result.getByText("1"), { shiftKey: true });
          expect(onSelect).toHaveBeenCalledWith(
            "1",
            expect.objectContaining({ shiftKey: true }),
          );
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
              <List.Scroll>
                <List.Items<string>>
                  {({ itemKey }) => <div key={itemKey}>{getItem(itemKey)?.name}</div>}
                </List.Items>
              </List.Scroll>
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
              <List.Scroll>
                <List.Items<string>>{itemProp}</List.Items>
              </List.Scroll>
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
              <List.Scroll>
                <List.Items>{({ key }) => <div key={key}>{key}</div>}</List.Items>
              </List.Scroll>
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
                  <List.Scroll>
                    <List.Items>{({ key }) => <div key={key}>{key}</div>}</List.Items>
                  </List.Scroll>
                )}
              </List.Frame>
            );
          };
          const result = render(<Component />);
          expect(fetchMore).toHaveBeenCalledTimes(1);
          fireEvent.click(result.getByText("Toggle"));
          expect(fetchMore).toHaveBeenCalledTimes(1);
          fireEvent.click(result.getByText("Toggle"));
          expect(fetchMore).toHaveBeenCalledTimes(1);
        });
      });
    });
  });

  describe("windowing", () => {
    const ITEM_HEIGHT = 27;
    const DATA = Array.from({ length: 500 }, (_, i) => `${i}`);

    beforeAll(() => mockGeometry(100, 100));

    const renderWindowed = (overscan?: number) =>
      render(
        <List.Frame data={DATA} virtual itemHeight={ITEM_HEIGHT} overscan={overscan}>
          <List.Scroll>
            <List.Items>
              {({ key, ...rest }: List.ItemProps<string>) => (
                <List.Item key={key} {...rest}>
                  {key}
                </List.Item>
              )}
            </List.Items>
          </List.Scroll>
        </List.Frame>,
      );

    const rows = (result: ReturnType<typeof render>): HTMLElement[] =>
      Array.from(result.container.querySelectorAll(".pluto-list__item"));

    it("should render only the rows near the window", () => {
      const rendered = rows(renderWindowed());
      expect(rendered.length).toBeGreaterThan(0);
      expect(rendered.length).toBeLessThan(DATA.length);
      expect(rendered[0].textContent).toBe("0");
    });

    it("should offset each row by its index", () => {
      const rendered = rows(renderWindowed());
      rendered.forEach((row, index) =>
        expect(row.style.top).toBe(`${index * ITEM_HEIGHT}px`),
      );
    });

    it("should render more rows when overscan grows", () => {
      const tight = rows(renderWindowed(0)).length;
      const loose = rows(renderWindowed(40)).length;
      expect(loose).toBeGreaterThan(tight);
    });

    it("should not re-render the rows that stay in view when the list scrolls", () => {
      const { Counted, counts } = createRenderCounter();
      const result = render(
        <List.Frame data={DATA} virtual itemHeight={ITEM_HEIGHT} overscan={0}>
          <List.Scroll>
            <List.Items>
              {({ key, ...rest }: List.ItemProps<string>) => (
                <Counted key={key} id={rest.itemKey}>
                  <List.Item {...rest}>{key}</List.Item>
                </Counted>
              )}
            </List.Items>
          </List.Scroll>
        </List.Frame>,
      );
      const scroller =
        result.container.querySelector<HTMLElement>(".pluto-list__scroll");
      if (scroller == null) throw new Error("scroll container not found");
      const before = rows(result).map((row) => row.id);
      act(() => {
        scroller.scrollTop = ITEM_HEIGHT;
        fireEvent.scroll(scroller);
      });
      const after = rows(result).map((row) => row.id);
      expect(after).not.toEqual(before);
      expect([...counts.keys()]).toEqual([]);
    });

    it("should show a changed record in an item whose element is reused", () => {
      interface Entry {
        key: string;
        name: string;
      }
      const Name = (props: List.ItemProps<string>): ReactElement => {
        const entry = List.useItem<string, Entry>(props.itemKey);
        return <List.Item {...props}>{entry?.name}</List.Item>;
      };
      const item = ({ key, ...rest }: List.ItemProps<string>): ReactElement => (
        <Name key={key} {...rest} />
      );
      const Harness = (): ReactElement => {
        const [entries, setEntries] = useState<Entry[]>([{ key: "a", name: "before" }]);
        const props = List.useStaticData<string, Entry>({ data: entries });
        return (
          <>
            <button onClick={() => setEntries([{ key: "a", name: "after" }])}>
              rename
            </button>
            <List.Frame {...props} virtual itemHeight={ITEM_HEIGHT}>
              <List.Scroll>
                <List.Items>{item}</List.Items>
              </List.Scroll>
            </List.Frame>
          </>
        );
      };
      const result = render(<Harness />);
      expect(result.getByText("before")).toBeTruthy();
      fireEvent.click(result.getByText("rename"));
      expect(result.getByText("after")).toBeTruthy();
    });

    describe("pinned", () => {
      interface PinnedProps {
        pinned?: string[];
        onFetchMore?: () => void;
      }

      const Pinned = ({ pinned, onFetchMore }: PinnedProps): ReactElement => (
        <List.Frame
          data={DATA}
          virtual
          itemHeight={ITEM_HEIGHT}
          pinned={pinned}
          onFetchMore={onFetchMore}
        >
          <List.Scroll>
            <List.Items>
              {({ key, ...rest }: List.ItemProps<string>) => (
                <List.Item key={key} {...rest}>
                  {key}
                </List.Item>
              )}
            </List.Items>
          </List.Scroll>
        </List.Frame>
      );

      const texts = (result: ReturnType<typeof render>): (string | null)[] =>
        rows(result).map((row) => row.textContent);

      it("should keep pinned rows mounted outside the window", () => {
        const result = render(<Pinned pinned={["250", "499"]} />);
        expect(texts(result)).toContain("250");
        expect(texts(result)).toContain("499");
        expect(texts(result)).not.toContain("251");
      });

      it("should place a pinned row at the offset of its index", () => {
        const result = render(<Pinned pinned={["250"]} />);
        const row = rows(result).find((r) => r.textContent === "250");
        expect(row?.style.top).toBe(`${250 * ITEM_HEIGHT}px`);
      });

      it("should mount a row when it becomes pinned and release it after", () => {
        const result = render(<Pinned />);
        expect(texts(result)).not.toContain("250");
        result.rerender(<Pinned pinned={["250"]} />);
        expect(texts(result)).toContain("250");
        result.rerender(<Pinned />);
        expect(texts(result)).not.toContain("250");
      });

      it("should ignore a pinned key that is not in the data", () => {
        const unpinned = texts(render(<Pinned />));
        expect(texts(render(<Pinned pinned={["missing"]} />))).toEqual(unpinned);
      });

      it("should not fetch more when only the pinned last row is mounted", () => {
        const onFetchMore = vi.fn();
        const result = render(<Pinned pinned={["499"]} onFetchMore={onFetchMore} />);
        const scroller =
          result.container.querySelector<HTMLElement>(".pluto-list__scroll");
        if (scroller == null) throw new Error("scroll container not found");
        const calls = onFetchMore.mock.calls.length;
        act(() => {
          scroller.scrollTop = ITEM_HEIGHT;
          fireEvent.scroll(scroller);
        });
        expect(texts(result)).toContain("499");
        expect(onFetchMore).toHaveBeenCalledTimes(calls);
      });
    });

    it("should keep the container tall enough to scroll the whole data set", () => {
      const result = renderWindowed();
      const virtualizer = result.container.querySelector<HTMLElement>(
        ".pluto-list__virtualizer",
      );
      expect(virtualizer?.style.minHeight).toBe(`${DATA.length * ITEM_HEIGHT}px`);
    });
  });

  describe("scroll container", () => {
    const ITEM_HEIGHT = 27;
    const DATA = Array.from({ length: 500 }, (_, i) => `${i}`);
    const item = ({ key, ...rest }: List.ItemProps<string>) => (
      <List.Item key={key} {...rest}>
        {key}
      </List.Item>
    );

    beforeAll(() => mockGeometry(100, 100));

    it("should render the items inside the enclosing Scroll", () => {
      const result = render(
        <List.Frame data={["1", "2"]}>
          <List.Scroll>
            <div>fixed</div>
            <List.Items>{item}</List.Items>
          </List.Scroll>
        </List.Frame>,
      );
      const scrolls = result.container.querySelectorAll(".pluto-list__scroll");
      expect(scrolls).toHaveLength(1);
      expect(scrolls[0].textContent).toBe("fixed12");
    });

    it("should throw when no Scroll encloses the items", () => {
      vi.spyOn(console, "error").mockImplementation(() => {});
      expect(() =>
        render(
          <List.Frame data={["1"]}>
            <List.Items>{item}</List.Items>
          </List.Frame>,
        ),
      ).toThrow("List.Items");
      vi.restoreAllMocks();
    });

    it("should offset virtual rows from the items, not from content above them", () => {
      const margin = 50;
      const offsetTop = vi
        .spyOn(HTMLElement.prototype, "offsetTop", "get")
        .mockImplementation(function (this: HTMLElement) {
          return this.classList.contains("pluto-list__virtualizer") ? margin : 0;
        });
      try {
        const result = render(
          <List.Frame data={DATA} virtual itemHeight={ITEM_HEIGHT} overscan={0}>
            <List.Scroll>
              <div>fixed</div>
              <List.Items>{item}</List.Items>
            </List.Scroll>
          </List.Frame>,
        );
        const rows = Array.from(
          result.container.querySelectorAll<HTMLElement>(".pluto-list__item"),
        );
        // A 100px window with 50px of content above the items fits two 27px rows.
        expect(rows).toHaveLength(2);
        rows.forEach((row, index) =>
          expect(row.style.top).toBe(`${index * ITEM_HEIGHT}px`),
        );
        const virtualizer = result.container.querySelector<HTMLElement>(
          ".pluto-list__virtualizer",
        );
        expect(virtualizer?.style.minHeight).toBe(`${DATA.length * ITEM_HEIGHT}px`);
      } finally {
        offsetTop.mockRestore();
      }
    });

    it("should remeasure the offset when content above the items mounts", async () => {
      const offsetTop = vi
        .spyOn(HTMLElement.prototype, "offsetTop", "get")
        .mockImplementation(function (this: HTMLElement) {
          if (!this.classList.contains("pluto-list__virtualizer")) return 0;
          return this.previousElementSibling == null ? 0 : 50;
        });
      let show: (visible: boolean) => void = () => {};
      const Above = (): ReactElement | null => {
        const [visible, setVisible] = useState(false);
        show = setVisible;
        return visible ? <div>fixed</div> : null;
      };
      try {
        const result = render(
          <List.Frame data={DATA} virtual itemHeight={ITEM_HEIGHT} overscan={0}>
            <List.Scroll>
              <Above />
              <List.Items>{item}</List.Items>
            </List.Scroll>
          </List.Frame>,
        );
        const rows = () => result.container.querySelectorAll(".pluto-list__item");
        // A 100px window fits four 27px rows, and two once 50px sits above them.
        expect(rows()).toHaveLength(4);
        await act(async () => show(true));
        expect(rows()).toHaveLength(2);
      } finally {
        offsetTop.mockRestore();
      }
    });

    it("should throw when the enclosing Scroll belongs to another frame", () => {
      vi.spyOn(console, "error").mockImplementation(() => {});
      expect(() =>
        render(
          <List.Frame data={["1"]}>
            <List.Scroll>
              <List.Frame data={["2"]}>
                <List.Items>{item}</List.Items>
              </List.Frame>
            </List.Scroll>
          </List.Frame>,
        ),
      ).toThrow("List.Items must be inside the List.Scroll of its own List.Frame");
      vi.restoreAllMocks();
    });

    it("should scroll to an item, not to content above the items", () => {
      const scrolled: string[] = [];
      Element.prototype.scrollIntoView = vi.fn(function (this: Element) {
        scrolled.push(this.textContent);
      });
      let scrollToIndex: ((index: number) => void) | undefined;
      const Capture = () => {
        ({ scrollToIndex } = List.useScroller());
        return null;
      };
      render(
        <List.Frame data={["1", "2", "3"]}>
          <Capture />
          <List.Scroll>
            <div>fixed</div>
            <List.Items>{item}</List.Items>
          </List.Scroll>
        </List.Frame>,
      );
      act(() => scrollToIndex?.(2));
      expect(scrolled).toEqual(["2"]);
    });
  });

  describe("height animation", () => {
    const hasAnimateClass = (animateHeight?: boolean) =>
      render(
        <List.Frame data={["1"]}>
          <List.Scroll animateHeight={animateHeight}>
            <List.Items>
              {({ key, ...rest }: List.ItemProps<string>) => (
                <List.Item key={key} {...rest}>
                  {key}
                </List.Item>
              )}
            </List.Items>
          </List.Scroll>
        </List.Frame>,
      )
        .container.querySelector(".pluto-list__scroll")
        ?.classList.contains("pluto-list__scroll--animate-height");

    it("should not animate height unless the caller opts in", () => {
      expect(hasAnimateClass()).toBe(false);
    });

    it("should animate height when the caller opts in", () => {
      expect(hasAnimateClass(true)).toBe(true);
    });
  });

  describe("scroll-based pagination (non-virtual)", () => {
    let mockObserverCallback: IntersectionObserverCallback;
    const mockObserve = vi.fn();
    const mockDisconnect = vi.fn();

    class MockIntersectionObserver {
      observe = mockObserve;
      disconnect = mockDisconnect;

      constructor(callback: IntersectionObserverCallback) {
        mockObserverCallback = callback;
      }
    }

    beforeAll(() => mockGeometry(100, 100));

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
          <List.Scroll>
            <List.Items>{({ key }) => <div key={key}>{key}</div>}</List.Items>
          </List.Scroll>
        </List.Frame>,
      );
      expect(mockObserve).toHaveBeenCalled();
    });

    it("should call onFetchMore when sentinel intersects", () => {
      const fetchMore = vi.fn();
      render(
        <List.Frame data={["1", "2", "3"]} virtual={false} onFetchMore={fetchMore}>
          <List.Scroll>
            <List.Items>{({ key }) => <div key={key}>{key}</div>}</List.Items>
          </List.Scroll>
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
          <List.Scroll>
            <List.Items>{({ key }) => <div key={key}>{key}</div>}</List.Items>
          </List.Scroll>
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
          <List.Scroll>
            <List.Items>{({ key }) => <div key={key}>{key}</div>}</List.Items>
          </List.Scroll>
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
          <List.Scroll>
            <List.Items>{({ key }) => <div key={key}>{key}</div>}</List.Items>
          </List.Scroll>
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
          <List.Scroll>
            <List.Items>{({ key }) => <div key={key}>{key}</div>}</List.Items>
          </List.Scroll>
        </List.Frame>,
      );

      unmount();
      expect(mockDisconnect).toHaveBeenCalled();
    });
  });
});
