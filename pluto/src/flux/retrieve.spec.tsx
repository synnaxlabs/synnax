// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  DisconnectedError,
  type label,
  NotFoundError,
  query,
} from "@synnaxlabs/client";
import {
  createSeverableProxy,
  createTestClient,
  TEST_CLIENT_PARAMS,
} from "@synnaxlabs/client/testutil";
import { Unreachable } from "@synnaxlabs/freighter";
import { color, id, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, render, renderHook, waitFor } from "@testing-library/react";
import {
  type FC,
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useMemo,
  useState,
} from "react";
import { describe, expect, it, vi } from "vitest";

import { aetherTest } from "@/aether/test";
import { Errors } from "@/errors";
import { Flux } from "@/flux";
import { status } from "@/status/aether";
import { Status } from "@/status/base";
import { Synnax } from "@/synnax";
import { synnax } from "@/synnax/aether";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();
const Wrapper = createSynnaxWrapper({ client });

/** Mounts the real provider against a live cluster reached through `port`. */
const createLiveWrapper = (port: number): FC<PropsWithChildren> => {
  const AetherProvider = aetherTest.createProvider({
    ...synnax.REGISTRY,
    ...status.REGISTRY,
  });
  const connParams = {
    ...TEST_CLIENT_PARAMS,
    port,
    // Unbounded retries: a capped breaker stops the streamer from ever reopening.
    retry: {
      baseInterval: TimeSpan.milliseconds(10),
      maxInterval: TimeSpan.milliseconds(50),
      scale: 1.5,
    },
  };
  const Live = ({ children }: PropsWithChildren): ReactElement => (
    <AetherProvider>
      <Status.Aggregator>
        <Synnax.Provider connParams={connParams}>{children}</Synnax.Provider>
      </Status.Aggregator>
    </AetherProvider>
  );
  Live.displayName = "LiveWrapper";
  return Live;
};

describe("retrieve", () => {
  describe("useDirect", () => {
    describe("basic retrieval", () => {
      it("should return a loading result as its initial state", () => {
        const { useRetrieve } = Flux.createRetrieve<{}, number>({
          name: "Resource",
          retrieve: async () => 0,
        });

        const { result } = renderHook(() => useRetrieve({ params: {} }), {
          wrapper: Wrapper,
        });
        expect(result.current.variant).toEqual("loading");
        expect(result.current.data).toEqual(undefined);
        expect(result.current.status.message).toEqual("Retrieving Resource");
      });

      it("should return a success result when the data is fetched", async () => {
        const { useRetrieve } = Flux.createRetrieve<{}, number>({
          name: "Resource",
          retrieve: async () => 12,
        });

        const { result } = renderHook(() => useRetrieve({ params: {} }), {
          wrapper: Wrapper,
        });
        await waitFor(() => {
          expect(result.current.variant).toEqual("success");
          expect(result.current.data).toEqual(12);
          expect(result.current.status.message).toEqual(
            "Successfully retrieved Resource",
          );
        });
      });

      it("should return an error result when the retrieve function throws an error", async () => {
        const { useRetrieve } = Flux.createRetrieve<{}, number>({
          name: "Resource",
          retrieve: async () => {
            throw new Error("test");
          },
        });

        const { result } = renderHook(() => useRetrieve({ params: {} }), {
          wrapper: Wrapper,
        });
        await waitFor(() => {
          expect(result.current.variant).toEqual("error");
          expect(result.current.data).toEqual(undefined);
          expect(result.current.status.message).toEqual("Failed to retrieve Resource");
          expect(result.current.status.description).toEqual("test");
        });
      });

      it("should return an error result when no client is connected", async () => {
        const { useRetrieve } = Flux.createRetrieve<{}, number>({
          name: "Resource",
          retrieve: async () => 0,
        });

        const { result } = renderHook(() => useRetrieve({ params: {} }), {
          wrapper: createSynnaxWrapper({ client: null }),
        });
        await waitFor(() => {
          expect(result.current.variant).toEqual("disabled");
          expect(result.current.data).toEqual(undefined);
          expect(result.current.status.message).toEqual("Failed to retrieve Resource");
          expect(result.current.status.description).toEqual(
            "Cannot retrieve Resource because no Core is connected.",
          );
        });
      });
    });

    describe("failure notifications", () => {
      const renderFailing = (error: Error) => {
        const { useRetrieve } = Flux.createRetrieve<{}, number>({
          name: "Resource",
          retrieve: async () => {
            throw error;
          },
        });
        return renderHook(
          () => ({
            retrieve: useRetrieve({ params: {} }),
            notifications: Status.useNotifications(),
          }),
          { wrapper: Wrapper },
        );
      };

      it("should add a status when the retrieve fails for an ordinary reason", async () => {
        const { result } = renderFailing(new Error("test"));
        await waitFor(() => {
          expect(result.current.retrieve.variant).toEqual("error");
          expect(result.current.notifications.statuses).toHaveLength(1);
          expect(result.current.notifications.statuses[0].message).toEqual(
            "Failed to retrieve Resource",
          );
        });
      });

      it("should not add a status when the Core is unreachable", async () => {
        const { result } = renderFailing(new Unreachable());
        await waitFor(() => expect(result.current.retrieve.variant).toEqual("error"));
        expect(result.current.notifications.statuses).toHaveLength(0);
      });

      it("should not add a status when the retrieve short circuits as disconnected", async () => {
        const { result } = renderFailing(new DisconnectedError());
        await waitFor(() => expect(result.current.retrieve.variant).toEqual("error"));
        expect(result.current.notifications.statuses).toHaveLength(0);
      });
    });

    describe("subscriptions", () => {
      it("should update the result when the subscription pushes a change", async () => {
        const ch = await client.labels.create({
          name: "Test Label",
          color: color.construct("#000000"),
        });
        let handler: query.ChangeHandler<label.Label> | null = null;
        const { useRetrieve } = Flux.createRetrieve<{ key: label.Key }, label.Label>({
          name: "Resource",
          retrieve: async ({ client, query: { key } }) =>
            await client.labels.retrieve(key),
          subscribe: (_, h) => {
            handler = h;
            return () => {};
          },
        });

        const { result } = renderHook(() => useRetrieve({ key: ch.key }), {
          wrapper: Wrapper,
        });
        await waitFor(() => {
          expect(result.current.variant).toEqual("success");
          expect(result.current.data).toEqual(ch);
          expect(handler).not.toBeNull();
        });
        act(() => {
          handler?.({ ...ch, name: "Test Label 2" });
        });
        await waitFor(
          () => {
            expect(result.current.data?.name).toEqual("Test Label 2");
            expect(
              result.current.variant,
              `${result.current.status.message}:${result.current.status.description}`,
            ).toEqual("success");
          },
          { timeout: 1000 },
        );
      });

      it("should move to an error result when the subscription reports a deletion", async () => {
        const ch = await client.labels.create({
          name: "Corpse Label",
          color: color.construct("#000000"),
        });
        let handler: query.ChangeHandler<label.Label> | null = null;
        const { useRetrieve } = Flux.createRetrieve<{ key: label.Key }, label.Label>({
          name: "Resource",
          retrieve: async ({ client, query: { key } }) =>
            await client.labels.retrieve(key),
          subscribe: (_, h) => {
            handler = h;
            return () => {};
          },
        });

        const { result } = renderHook(() => useRetrieve({ key: ch.key }), {
          wrapper: Wrapper,
        });
        await waitFor(() => expect(result.current.variant).toEqual("success"));
        act(() => {
          handler?.(new query.Deleted(ch, TimeStamp.now()));
        });
        await waitFor(() => {
          expect(result.current.variant).toEqual("error");
          expect(result.current.status.description).toEqual("Resource was deleted");
        });
      });
    });
  });

  describe("useObservable", () => {
    it("should drop the subscription of a query superseded mid-fetch", async () => {
      const handlers = new Map<string, query.ChangeHandler<string>>();
      let resolveSlow: ((value: string) => void) | null = null;
      const { useRetrieveObservable } = Flux.createRetrieve<{ key: string }, string>({
        name: "Resource",
        retrieve: async ({ query: { key } }) =>
          key === "slow"
            ? await new Promise<string>((resolve) => (resolveSlow = resolve))
            : key,
        subscribe: ({ query: { key } }, h) => {
          handlers.set(key, h);
          return () => handlers.delete(key);
        },
      });
      const onChange = vi.fn();
      const { result } = renderHook(() => useRetrieveObservable({ onChange }), {
        wrapper: Wrapper,
      });
      let slow: Promise<void> | null = null;
      await act(async () => {
        slow = result.current.retrieveAsync({ key: "slow" });
        await result.current.retrieveAsync({ key: "fast" });
      });
      await act(async () => {
        resolveSlow?.("slow");
        await slow;
      });
      expect(handlers.has("slow")).toBe(false);
      act(() => handlers.get("fast")?.("updated"));
      const [lastResult, lastQuery] = onChange.mock.calls.at(-1) as [
        Flux.Result<string>,
        { key: string },
      ];
      expect(lastResult.data).toEqual("updated");
      expect(lastQuery.key).toEqual("fast");
    });
  });

  describe("useEffect", () => {
    it("should call the onChange handler when the data is fetched", async () => {
      const onChangeMock = vi.fn();
      const { result } = renderHook(
        () => {
          const [result, setResult] = useState<Flux.Result<number>>(
            Flux.loadingResult<number>("retrieving Resource", undefined),
          );
          const handleChange = useCallback(
            (value: Flux.Result<number>) => {
              setResult(value);
              onChangeMock(value);
            },
            [onChangeMock],
          );
          const { useRetrieveEffect } = Flux.createRetrieve<{ key: string }, number>({
            name: "Resource",
            retrieve: async () => 12,
          });
          useRetrieveEffect({
            query: { key: "test" },
            onChange: handleChange,
          });
          return result;
        },
        { wrapper: Wrapper },
      );
      await waitFor(() => {
        expect(onChangeMock).toHaveBeenCalledTimes(2);
        expect(result.current.data).toEqual(12);
      });
    });
  });
});

describe("useRetrieveSuspended", () => {
  it("suspends until the retrieve resolves, then returns the value", async () => {
    let resolveRetrieve: (value: number) => void = () => {};
    const { useRetrieveSuspended } = Flux.createRetrieve<{ key: string }, number>({
      name: "Number",
      retrieve: () =>
        new Promise<number>((resolve) => {
          resolveRetrieve = resolve;
        }),
    });

    const Display = (): ReactElement => {
      const value = useRetrieveSuspended({ key: "first-test" });
      return <div data-testid="value">{value}</div>;
    };

    let utils!: ReturnType<typeof render>;
    await act(async () => {
      utils = render(
        <Wrapper>
          <Errors.SuspenseBoundary loading={<div>loading-1</div>}>
            <Display />
          </Errors.SuspenseBoundary>
        </Wrapper>,
      );
    });

    expect(utils.queryByText("loading-1")).toBeTruthy();
    expect(utils.queryByTestId("value")).toBeNull();

    await act(async () => {
      resolveRetrieve(42);
    });

    expect(utils.queryByTestId("value")?.textContent).toBe("42");
  });

  it("dedupes concurrent reads of the same query", async () => {
    let resolveRetrieve: (value: number) => void = () => {};
    const retrieve = vi.fn(
      () =>
        new Promise<number>((resolve) => {
          resolveRetrieve = resolve;
        }),
    );
    const { useRetrieveSuspended } = Flux.createRetrieve<{ key: string }, number>({
      name: "Number",
      retrieve,
    });

    const Display = (): ReactElement => {
      const value = useRetrieveSuspended({ key: "dedupe-test" });
      return <div>{value}</div>;
    };

    await act(async () => {
      render(
        <Wrapper>
          <Errors.SuspenseBoundary loading={<div>loading-2</div>}>
            <Display />
            <Display />
            <Display />
          </Errors.SuspenseBoundary>
        </Wrapper>,
      );
    });

    expect(retrieve).toHaveBeenCalledTimes(1);

    await act(async () => {
      resolveRetrieve(7);
    });
  });

  it("routes a thrown error to the error fallback", async () => {
    const { useRetrieveSuspended } = Flux.createRetrieve<{ key: string }, number>({
      name: "Number",
      retrieve: async () => {
        throw new Error("boom");
      },
    });

    const Display = (): ReactElement => {
      const value = useRetrieveSuspended({ key: "error-test" });
      return <div>{value}</div>;
    };

    let utils!: ReturnType<typeof render>;
    await act(async () => {
      utils = render(
        <Wrapper>
          <Errors.SuspenseBoundary
            loading={<div>loading-3</div>}
            FallbackComponent={({ error }) => (
              <div data-testid="error">{error.message}</div>
            )}
          >
            <Display />
          </Errors.SuspenseBoundary>
        </Wrapper>,
      );
    });

    // Drain microtasks so the rejected promise propagates to the error boundary.
    await act(async () => {
      await Promise.resolve();
    });

    expect(utils.queryByTestId("error")?.textContent).toBe("Failed to retrieve Number");
  });

  it("routes a rejection to the error fallback without refetching when the query is domain-cached", async () => {
    const retrieve = vi.fn(async (): Promise<number> => {
      throw new Error("boom");
    });
    const { useRetrieveSuspended } = Flux.createRetrieve<{ key: string }, number>({
      name: "Number",
      retrieve,
      getCached: () => undefined,
    });

    const Display = (): ReactElement => {
      const value = useRetrieveSuspended({ key: "cached-error-test" });
      return <div>{value}</div>;
    };

    let utils!: ReturnType<typeof render>;
    await act(async () => {
      utils = render(
        <Wrapper>
          <Errors.SuspenseBoundary
            loading={<div>loading-cached-error</div>}
            FallbackComponent={({ error }) => (
              <div data-testid="error">{error.message}</div>
            )}
          >
            <Display />
          </Errors.SuspenseBoundary>
        </Wrapper>,
      );
    });

    await act(async () => {
      await Promise.resolve();
    });

    expect(utils.queryByTestId("error")?.textContent).toBe("Failed to retrieve Number");
    expect(retrieve).toHaveBeenCalledTimes(1);
  });

  it("resolves synchronously without suspending when the cache hits", async () => {
    const retrieve = vi.fn(async () => 99);
    const cached: query.Cached<number> = 42;
    const { useRetrieveSuspended } = Flux.createRetrieve<{ key: string }, number>({
      name: "Number",
      retrieve,
      getCached: () => cached,
    });

    const Display = (): ReactElement => {
      const value = useRetrieveSuspended({ key: "cached-hit" });
      return <div data-testid="value">{value}</div>;
    };

    let utils!: ReturnType<typeof render>;
    await act(async () => {
      utils = render(
        <Wrapper>
          <Errors.SuspenseBoundary loading={<div>loading-cached</div>}>
            <Display />
          </Errors.SuspenseBoundary>
        </Wrapper>,
      );
    });

    expect(utils.queryByText("loading-cached")).toBeNull();
    expect(utils.queryByTestId("value")?.textContent).toBe("42");
    expect(retrieve).not.toHaveBeenCalled();
  });

  it("resolves from deriveCached when the query's own cache misses", async () => {
    const retrieve = vi.fn(async () => 99);
    const { useRetrieveSuspended } = Flux.createRetrieve<{ key: string }, number>({
      name: "Number",
      retrieve,
      getCached: () => undefined,
      deriveCached: () => 13,
    });

    const Display = (): ReactElement => {
      const value = useRetrieveSuspended({ key: "derived" });
      return <div data-testid="value">{value}</div>;
    };

    let utils!: ReturnType<typeof render>;
    await act(async () => {
      utils = render(
        <Wrapper>
          <Errors.SuspenseBoundary loading={<div>loading-derived</div>}>
            <Display />
          </Errors.SuspenseBoundary>
        </Wrapper>,
      );
    });

    expect(utils.queryByText("loading-derived")).toBeNull();
    expect(utils.queryByTestId("value")?.textContent).toBe("13");
    expect(retrieve).not.toHaveBeenCalled();
  });

  it("falls through to the async retrieve when the cache misses", async () => {
    let resolveRetrieve: (value: number) => void = () => {};
    let cached: query.Cached<number> | undefined;
    const retrieve = vi.fn(
      () =>
        new Promise<number>((resolve) => {
          resolveRetrieve = (value) => {
            cached = value;
            resolve(value);
          };
        }),
    );
    const { useRetrieveSuspended } = Flux.createRetrieve<{ key: string }, number>({
      name: "Number",
      retrieve,
      subscribe: () => () => {},
      getCached: () => cached,
    });

    const Display = (): ReactElement => {
      const value = useRetrieveSuspended({ key: "cached-miss" });
      return <div data-testid="value">{value}</div>;
    };

    let utils!: ReturnType<typeof render>;
    await act(async () => {
      utils = render(
        <Wrapper>
          <Errors.SuspenseBoundary loading={<div>loading-miss</div>}>
            <Display />
          </Errors.SuspenseBoundary>
        </Wrapper>,
      );
    });

    expect(utils.queryByText("loading-miss")).toBeTruthy();
    expect(retrieve).toHaveBeenCalledTimes(1);

    await act(async () => {
      resolveRetrieve(7);
    });

    expect(utils.queryByTestId("value")?.textContent).toBe("7");
  });

  it("refetches when the subscription reports an invalidated answer", async () => {
    let capturedHandler: query.ChangeHandler<number> | null = null;
    let cached: query.Cached<number> | undefined;
    const retrieve = vi.fn(async () => {
      const value = retrieve.mock.calls.length;
      cached = value;
      return value;
    });
    const { useRetrieveSuspended } = Flux.createRetrieve<{ key: string }, number>({
      name: "Number",
      retrieve,
      subscribe: (_, handler) => {
        capturedHandler = handler;
        return () => {};
      },
      getCached: () => cached,
    });

    const Display = (): ReactElement => {
      const value = useRetrieveSuspended({ key: "invalidate-test" });
      return <div data-testid="value">{value}</div>;
    };

    let utils!: ReturnType<typeof render>;
    await act(async () => {
      utils = render(
        <Wrapper>
          <Errors.SuspenseBoundary loading={<div>loading</div>}>
            <Display />
          </Errors.SuspenseBoundary>
        </Wrapper>,
      );
    });

    await waitFor(() => expect(utils.queryByTestId("value")?.textContent).toBe("1"));

    await act(async () => {
      cached = undefined;
      capturedHandler!(undefined);
    });

    await waitFor(() => expect(utils.queryByTestId("value")?.textContent).toBe("2"));
    expect(retrieve).toHaveBeenCalledTimes(2);
  });

  describe("equal", () => {
    const sameNumbers = (a: number[], b: number[]): boolean =>
      a.length === b.length && a.every((v, i) => v === b[i]);

    interface Harness {
      utils: ReturnType<typeof render>;
      seen: number[][];
      push: (next: number[]) => Promise<void>;
    }

    // getCached builds a fresh array on every call, the way a query projecting a
    // cached record down to a narrower shape does. That is the shape `equal` exists
    // to make usable: without it useSyncExternalStore never settles.
    const renderProjection = async (key: string): Promise<Harness> => {
      let source: number[] = [1, 2];
      let handler: query.ChangeHandler<number[]> | null = null;
      const { useRetrieveSuspended } = Flux.createRetrieve<{ key: string }, number[]>({
        name: "Numbers",
        retrieve: async () => [...source],
        subscribe: (_, h) => {
          handler = h;
          return () => {};
        },
        getCached: () => [...source],
        equal: sameNumbers,
      });

      const seen: number[][] = [];
      const Display = (): ReactElement => {
        const value = useRetrieveSuspended({ key });
        seen.push(value);
        return <div data-testid="value">{value.join(",")}</div>;
      };

      let utils!: ReturnType<typeof render>;
      await act(async () => {
        utils = render(
          <Wrapper>
            <Errors.SuspenseBoundary
              loading={null}
              FallbackComponent={({ error }) => (
                <div data-testid="error">{error.message}</div>
              )}
            >
              <Display />
            </Errors.SuspenseBoundary>
          </Wrapper>,
        );
      });
      return {
        utils,
        seen,
        push: async (next) => {
          await act(async () => {
            source = next;
            handler?.([...next]);
          });
        },
      };
    };

    it("makes a query that rebuilds its answer on every read renderable", async () => {
      const { utils, seen } = await renderProjection("equal-stable");
      // Without the comparator this shape spins useSyncExternalStore until React
      // gives up with a max-update-depth error, and the boundary eats the tree.
      expect(utils.queryByTestId("error")).toBeNull();
      expect(utils.queryByTestId("value")?.textContent).toEqual("1,2");
      expect(seen.length).toEqual(1);
    });

    it("hands back the previous answer when the next one compares equal", async () => {
      const { utils, seen, push } = await renderProjection("equal-hold");
      const before = seen[seen.length - 1];
      const renders = seen.length;

      await push([1, 2]);

      expect(utils.queryByTestId("error")).toBeNull();
      expect(seen.length).toEqual(renders);
      expect(seen[seen.length - 1]).toBe(before);
    });

    it("hands back the next answer when it compares unequal", async () => {
      const { utils, seen, push } = await renderProjection("equal-change");
      const before = seen[seen.length - 1];

      await push([1, 2, 3]);

      expect(seen[seen.length - 1]).not.toBe(before);
      expect(utils.queryByTestId("value")?.textContent).toEqual("1,2,3");
    });

    it("still throws a deleted error for a tombstoned answer", async () => {
      const tombstone = new query.Deleted<number[]>([1, 2], TimeStamp.now());
      const { useRetrieveSuspended } = Flux.createRetrieve<{ key: string }, number[]>({
        name: "Numbers",
        retrieve: async () => [1, 2],
        subscribe: () => () => {},
        getCached: () => tombstone,
        equal: sameNumbers,
      });

      const Display = (): ReactElement => {
        const value = useRetrieveSuspended({ key: "equal-deleted" });
        return <div data-testid="value">{value.join(",")}</div>;
      };

      let utils!: ReturnType<typeof render>;
      await act(async () => {
        utils = render(
          <Wrapper>
            <Errors.SuspenseBoundary
              loading={null}
              FallbackComponent={({ error }) => (
                <div data-testid="error">{error.message}</div>
              )}
            >
              <Display />
            </Errors.SuspenseBoundary>
          </Wrapper>,
        );
      });

      expect(utils.queryByTestId("error")?.textContent).toEqual("Numbers was deleted");
    });

    it("still refetches when the subscription invalidates the answer", async () => {
      let cached: query.Cached<number[]> | undefined;
      let handler: query.ChangeHandler<number[]> | null = null;
      const retrieve = vi.fn(async () => {
        cached = [retrieve.mock.calls.length];
        return cached;
      });
      const { useRetrieveSuspended } = Flux.createRetrieve<{ key: string }, number[]>({
        name: "Numbers",
        retrieve,
        subscribe: (_, h) => {
          handler = h;
          return () => {};
        },
        getCached: () => cached,
        equal: sameNumbers,
      });

      const Display = (): ReactElement => {
        const value = useRetrieveSuspended({ key: "equal-invalidate" });
        return <div data-testid="value">{value.join(",")}</div>;
      };

      let utils!: ReturnType<typeof render>;
      await act(async () => {
        utils = render(
          <Wrapper>
            <Errors.SuspenseBoundary loading={null}>
              <Display />
            </Errors.SuspenseBoundary>
          </Wrapper>,
        );
      });
      await waitFor(() =>
        expect(utils.queryByTestId("value")?.textContent).toEqual("1"),
      );

      await act(async () => {
        cached = undefined;
        handler?.(undefined);
      });

      await waitFor(() =>
        expect(utils.queryByTestId("value")?.textContent).toEqual("2"),
      );
      expect(retrieve).toHaveBeenCalledTimes(2);
    });

    it("passes the cached answer straight through when no comparator is given", async () => {
      const cached = [4, 5];
      const { useRetrieveSuspended } = Flux.createRetrieve<{ key: string }, number[]>({
        name: "Numbers",
        retrieve: async () => [],
        getCached: () => cached,
      });

      const seen: number[][] = [];
      const Display = (): ReactElement => {
        seen.push(useRetrieveSuspended({ key: "no-equal" }));
        return <div data-testid="value">{seen[seen.length - 1].join(",")}</div>;
      };

      let utils!: ReturnType<typeof render>;
      await act(async () => {
        utils = render(
          <Wrapper>
            <Errors.SuspenseBoundary loading={null}>
              <Display />
            </Errors.SuspenseBoundary>
          </Wrapper>,
        );
      });

      expect(utils.queryByTestId("value")?.textContent).toEqual("4,5");
      expect(seen[seen.length - 1]).toBe(cached);
    });
  });

  describe("not-found wait", () => {
    interface Harness {
      utils: ReturnType<typeof render>;
      retrieve: ReturnType<typeof vi.fn>;
      push: (result: query.Cached<number> | undefined) => void;
    }

    // Renders a suspending read whose fetch rejects with NotFoundError while
    // the domain cache is empty, so the read enters the pending wait.
    const renderNotFound = async (key: string): Promise<Harness> => {
      let cached: query.Cached<number> | undefined;
      let handler: query.ChangeHandler<number> | null = null;
      const retrieve = vi.fn(async (): Promise<number> => {
        throw new NotFoundError("no such number");
      });
      const { useRetrieveSuspended } = Flux.createRetrieve<{ key: string }, number>({
        name: "Number",
        retrieve,
        subscribe: (_, h) => {
          handler = h;
          return () => {};
        },
        getCached: () => cached,
      });

      const Display = (): ReactElement => {
        const value = useRetrieveSuspended({ key });
        return <div data-testid="value">{value}</div>;
      };

      let utils!: ReturnType<typeof render>;
      await act(async () => {
        utils = render(
          <Wrapper>
            <Errors.SuspenseBoundary
              loading={<div>waiting</div>}
              FallbackComponent={({ error, resetErrorBoundary }) => (
                <>
                  <div data-testid="error">{error.message}</div>
                  <button data-testid="reset" onClick={resetErrorBoundary}>
                    reset
                  </button>
                </>
              )}
            >
              <Display />
            </Errors.SuspenseBoundary>
          </Wrapper>,
        );
      });
      return {
        utils,
        retrieve,
        push: (result) => {
          cached = result;
          handler?.(result);
        },
      };
    };

    it("stays suspended on not-found and resolves when the document appears", async () => {
      const { utils, push } = await renderNotFound("nf-wait");
      expect(utils.queryByText("waiting")).toBeTruthy();
      expect(utils.queryByTestId("error")).toBeNull();

      await act(async () => {
        push(21);
      });

      await waitFor(() => expect(utils.queryByTestId("value")?.textContent).toBe("21"));
    });

    it("rejects with the not-found error after the wait expires, without refetching", async () => {
      vi.useFakeTimers();
      try {
        const { utils, retrieve } = await renderNotFound("nf-timeout");
        expect(utils.queryByText("waiting")).toBeTruthy();

        await act(async () => {
          vi.advanceTimersByTime(TimeSpan.seconds(6).milliseconds);
          await Promise.resolve();
        });

        expect(utils.queryByTestId("error")?.textContent).toBe(
          "Failed to retrieve Number",
        );
        expect(retrieve).toHaveBeenCalledTimes(1);
      } finally {
        vi.useRealTimers();
      }
    });

    it("refetches after the settled not-found is invalidated", async () => {
      let cached: query.Cached<number> | undefined;
      let found = false;
      const retrieve = vi.fn(async (): Promise<number> => {
        if (!found) throw new NotFoundError("no such number");
        cached = 42;
        return 42;
      });
      const { useRetrieveSuspended, useInvalidate } = Flux.createRetrieve<
        { key: string },
        number
      >({
        name: "Number",
        retrieve,
        subscribe: () => () => {},
        getCached: () => cached,
      });
      const Display = (): ReactElement => (
        <div data-testid="value">{useRetrieveSuspended({ key: "invalidate" })}</div>
      );
      const Retry = ({ onRetry }: { onRetry: () => void }): ReactElement => {
        const invalidate = useInvalidate();
        return (
          <button
            data-testid="retry"
            onClick={() => {
              invalidate({ key: "invalidate" });
              onRetry();
            }}
          >
            retry
          </button>
        );
      };
      let utils!: ReturnType<typeof render>;
      vi.useFakeTimers();
      try {
        await act(async () => {
          utils = render(
            <Wrapper>
              <Errors.SuspenseBoundary
                loading={<div>waiting</div>}
                FallbackComponent={({ resetErrorBoundary }) => (
                  <Retry onRetry={resetErrorBoundary} />
                )}
              >
                <Display />
              </Errors.SuspenseBoundary>
            </Wrapper>,
          );
        });
        await act(async () => {
          vi.advanceTimersByTime(TimeSpan.seconds(6).milliseconds);
          await Promise.resolve();
        });
      } finally {
        vi.useRealTimers();
      }
      expect(utils.queryByTestId("retry")).toBeTruthy();
      expect(retrieve).toHaveBeenCalledTimes(1);
      found = true;
      await act(async () => {
        fireEvent.click(utils.getByTestId("retry"));
      });
      await waitFor(() => expect(utils.queryByTestId("value")?.textContent).toBe("42"));
      expect(retrieve).toHaveBeenCalledTimes(2);
    });

    it("stays on the settled not-found when the boundary resets without invalidating", async () => {
      vi.useFakeTimers();
      let utils!: ReturnType<typeof render>;
      let retrieve!: ReturnType<typeof vi.fn>;
      try {
        const harness = await renderNotFound("nf-reset-only");
        utils = harness.utils;
        retrieve = harness.retrieve;
        await act(async () => {
          vi.advanceTimersByTime(TimeSpan.seconds(6).milliseconds);
          await Promise.resolve();
        });
      } finally {
        vi.useRealTimers();
      }
      await act(async () => {
        fireEvent.click(utils.getByTestId("reset"));
      });
      expect(utils.queryByTestId("reset")).toBeTruthy();
      expect(retrieve).toHaveBeenCalledTimes(1);
    });

    it("rejects with a deleted error when a tombstone arrives during the wait", async () => {
      const { utils, push } = await renderNotFound("nf-tombstone");
      expect(utils.queryByText("waiting")).toBeTruthy();

      await act(async () => {
        push(new query.Deleted(3, TimeStamp.now()));
      });

      await waitFor(() =>
        expect(utils.queryByTestId("error")?.textContent).toBe("Number was deleted"),
      );
    });

    it("tears down the subscription when it delivers a tombstone synchronously", async () => {
      const tombstone = new query.Deleted<number>(3, TimeStamp.now());
      let cached: query.Cached<number> | undefined;
      const disconnect = vi.fn();
      const retrieve = vi.fn(async (): Promise<number> => {
        throw new NotFoundError("no such number");
      });
      const { useRetrieveSuspended } = Flux.createRetrieve<{ key: string }, number>({
        name: "Number",
        retrieve,
        // A row tombstoned between the failed fetch and the subscription
        // mounting answers during onChange itself, so the handler fires
        // before the destructor is returned.
        subscribe: (_, h) => {
          cached = tombstone;
          h(tombstone);
          return disconnect;
        },
        getCached: () => cached,
      });

      const Display = (): ReactElement => {
        const value = useRetrieveSuspended({ key: "nf-sync-tombstone" });
        return <div data-testid="value">{value}</div>;
      };

      let utils!: ReturnType<typeof render>;
      await act(async () => {
        utils = render(
          <Wrapper>
            <Errors.SuspenseBoundary
              loading={<div>waiting</div>}
              FallbackComponent={({ error }) => (
                <div data-testid="error">{error.message}</div>
              )}
            >
              <Display />
            </Errors.SuspenseBoundary>
          </Wrapper>,
        );
      });

      await waitFor(() =>
        expect(utils.queryByTestId("error")?.textContent).toBe("Number was deleted"),
      );
      expect(disconnect).toHaveBeenCalled();
    });

    it("settles a not-found immediately when the query has no subscription", async () => {
      const retrieve = vi.fn(async (): Promise<number> => {
        throw new NotFoundError("no such number");
      });
      const { useRetrieveSuspended } = Flux.createRetrieve<{ key: string }, number>({
        name: "Number",
        retrieve,
        getCached: () => undefined,
      });

      const Display = (): ReactElement => {
        const value = useRetrieveSuspended({ key: "nf-unsubscribed" });
        return <div>{value}</div>;
      };

      let utils!: ReturnType<typeof render>;
      await act(async () => {
        utils = render(
          <Wrapper>
            <Errors.SuspenseBoundary
              loading={<div>waiting</div>}
              FallbackComponent={({ error }) => (
                <div data-testid="error">{error.message}</div>
              )}
            >
              <Display />
            </Errors.SuspenseBoundary>
          </Wrapper>,
        );
      });

      await act(async () => {
        await Promise.resolve();
      });

      expect(utils.queryByTestId("error")?.textContent).toBe(
        "Failed to retrieve Number",
      );
      expect(retrieve).toHaveBeenCalledTimes(1);
    });
  });

  it("refetches a read that failed during an outage once the connection returns", async () => {
    const proxy = await createSeverableProxy();
    try {
      const [first, second] = await client.labels.create([
        { name: `first-${id.create()}`, color: "#000000" },
        { name: `second-${id.create()}`, color: "#000000" },
      ]);
      const { useRetrieveSuspended } = Flux.createRetrieve<{ key: string }, string>({
        name: "Label",
        retrieve: async ({ client, query }) =>
          (await client.labels.retrieve(query.key)).name,
      });
      const Display = ({ labelKey }: { labelKey: string }): ReactElement => (
        <div>{useRetrieveSuspended({ key: labelKey })}</div>
      );
      const Live = createLiveWrapper(proxy.port);
      const tree = (labelKey: string): ReactElement => (
        <Live>
          <Errors.SuspenseBoundary
            loading={<div>loading</div>}
            FallbackComponent={() => <div data-testid="error">failed</div>}
          >
            <Display labelKey={labelKey} />
          </Errors.SuspenseBoundary>
        </Live>
      );

      // The first read opens the change stream, which is what advances the epoch
      // once the connection returns.
      const utils = render(tree(first.key));
      await waitFor(() => expect(utils.getByText(first.name)).toBeTruthy());

      // The second label was never read, so it cannot be served from the cache.
      await proxy.sever();
      utils.rerender(tree(second.key));
      await waitFor(() => expect(utils.getByTestId("error")).toBeTruthy());

      await proxy.restore();
      await waitFor(() => expect(utils.getByText(second.name)).toBeTruthy(), {
        timeout: 20000,
      });
    } finally {
      await proxy.close();
    }
  }, 30000);
});

describe("useEnsureRetrieved", () => {
  it("does not suspend when the cache hits", async () => {
    const retrieve = vi.fn(async () => 5);
    const { useEnsureRetrieved } = Flux.createRetrieve<{ key: string }, number>({
      name: "Number",
      retrieve,
      getCached: () => 5,
    });

    const Display = (): ReactElement => {
      useEnsureRetrieved({ key: "ensure-cached" });
      return <div data-testid="ready">ready</div>;
    };

    let utils!: ReturnType<typeof render>;
    await act(async () => {
      utils = render(
        <Wrapper>
          <Errors.SuspenseBoundary loading={<div>loading-ensure</div>}>
            <Display />
          </Errors.SuspenseBoundary>
        </Wrapper>,
      );
    });

    expect(utils.queryByText("loading-ensure")).toBeNull();
    expect(utils.queryByTestId("ready")).toBeTruthy();
    expect(retrieve).not.toHaveBeenCalled();
  });

  it("routes a rejection to the error fallback without refetching when the query is domain-cached", async () => {
    const retrieve = vi.fn(async (): Promise<number> => {
      throw new Error("boom");
    });
    const { useEnsureRetrieved } = Flux.createRetrieve<{ key: string }, number>({
      name: "Number",
      retrieve,
      getCached: () => undefined,
    });

    const Display = (): ReactElement => {
      useEnsureRetrieved({ key: "ensure-error" });
      return <div data-testid="ready">ready</div>;
    };

    let utils!: ReturnType<typeof render>;
    await act(async () => {
      utils = render(
        <Wrapper>
          <Errors.SuspenseBoundary
            loading={<div>loading-ensure-error</div>}
            FallbackComponent={({ error }) => (
              <div data-testid="error">{error.message}</div>
            )}
          >
            <Display />
          </Errors.SuspenseBoundary>
        </Wrapper>,
      );
    });

    await act(async () => {
      await Promise.resolve();
    });

    expect(utils.queryByTestId("error")?.textContent).toBe("Failed to retrieve Number");
    expect(retrieve).toHaveBeenCalledTimes(1);
  });
});

describe("useRetrieveSuspended connection changes", () => {
  it("surfaces a disconnect that lands after the read resolved", async () => {
    const { useRetrieveSuspended } = Flux.createRetrieve<{ key: string }, number>({
      name: "Number",
      retrieve: async () => 42,
    });

    const Display = (): ReactElement => {
      const value = useRetrieveSuspended({ key: "disconnect-test" });
      const label = useMemo(() => `value-${value}`, [value]);
      return <div data-testid="value">{label}</div>;
    };

    const Harness = ({ connected }: { connected: boolean }): ReactElement => (
      <Wrapper>
        <Synnax.TestProvider client={connected ? client : null}>
          <Errors.SuspenseBoundary
            loading={<div>loading</div>}
            FallbackComponent={({ error }) => (
              <div data-testid="error">{error.message}</div>
            )}
          >
            <Display />
          </Errors.SuspenseBoundary>
        </Synnax.TestProvider>
      </Wrapper>
    );

    let utils!: ReturnType<typeof render>;
    await act(async () => {
      utils = render(<Harness connected />);
    });
    await waitFor(() =>
      expect(utils.queryByTestId("value")?.textContent).toBe("value-42"),
    );

    await act(async () => {
      utils.rerender(<Harness connected={false} />);
    });
    expect(utils.queryByTestId("error")?.textContent).toContain("no Core connected");
  });
});
