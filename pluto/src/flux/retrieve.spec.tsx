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
  UnexpectedError,
} from "@synnaxlabs/client";
import {
  createSeverableProxy,
  createTestClient,
  TEST_CLIENT_PARAMS,
} from "@synnaxlabs/client/testutil";
import { color, id, TimeSpan, TimeStamp } from "@synnaxlabs/x";
import { act, fireEvent, render, renderHook, waitFor } from "@testing-library/react";
import {
  type FC,
  type PropsWithChildren,
  type ReactElement,
  startTransition,
  useCallback,
  useMemo,
  useState,
} from "react";
import { assert, describe, expect, it, vi } from "vitest";

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

/**
 * Collects the errors React reports globally while `run` executes. React recovers
 * from a corrupt hook order by re-rendering the root synchronously, so the render
 * still produces the right DOM and only the report proves the defect.
 */
const captureUncaught = async (run: () => Promise<void>): Promise<Error[]> => {
  const caught: Error[] = [];
  const onError = (event: ErrorEvent) => {
    caught.push(event.error);
    event.preventDefault();
  };
  window.addEventListener("error", onError);
  try {
    await run();
  } finally {
    window.removeEventListener("error", onError);
  }
  return caught;
};

describe("use", () => {
  it("suspends until the retrieve resolves, then returns the value", async () => {
    let resolveRetrieve: (value: number) => void = () => {};
    const { use } = Flux.createRetrieve<{ key: string }, number>({
      name: "Number",
      retrieve: () =>
        new Promise<number>((resolve) => {
          resolveRetrieve = resolve;
        }),
    });

    const Display = (): ReactElement => {
      const value = use({ key: "first-test" });
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
    const { use } = Flux.createRetrieve<{ key: string }, number>({
      name: "Number",
      retrieve,
    });

    const Display = (): ReactElement => {
      const value = use({ key: "dedupe-test" });
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
    const { use } = Flux.createRetrieve<{ key: string }, number>({
      name: "Number",
      retrieve: async () => {
        throw new Error("boom");
      },
    });

    const Display = (): ReactElement => {
      const value = use({ key: "error-test" });
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
    const { use } = Flux.createRetrieve<{ key: string }, number>({
      name: "Number",
      retrieve,
      getCached: () => undefined,
    });

    const Display = (): ReactElement => {
      const value = use({ key: "cached-error-test" });
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
    const { use } = Flux.createRetrieve<{ key: string }, number>({
      name: "Number",
      retrieve,
      getCached: () => cached,
    });

    const Display = (): ReactElement => {
      const value = use({ key: "cached-hit" });
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
    const { use } = Flux.createRetrieve<{ key: string }, number>({
      name: "Number",
      retrieve,
      getCached: () => undefined,
      deriveCached: () => 13,
    });

    const Display = (): ReactElement => {
      const value = use({ key: "derived" });
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
    const { use } = Flux.createRetrieve<{ key: string }, number>({
      name: "Number",
      retrieve,
      onChange: () => () => {},
      getCached: () => cached,
    });

    const Display = (): ReactElement => {
      const value = use({ key: "cached-miss" });
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
    const { use } = Flux.createRetrieve<{ key: string }, number>({
      name: "Number",
      retrieve,
      onChange: (_, handler) => {
        capturedHandler = handler;
        return () => {};
      },
      getCached: () => cached,
    });

    const Display = (): ReactElement => {
      const value = use({ key: "invalidate-test" });
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

  describe("suspended mount replay", () => {
    // The reader renders a hook after the read, whose slot only exists if the
    // replayed attempt resumes through the same `use` call it suspended on. The
    // mount goes through a transition: a concurrent render is what lets React
    // replay the suspended attempt instead of re-mounting it from scratch.
    const mountThroughTransition = async (
      use: (query: { key: string }) => number,
      key: string,
    ): Promise<Error[]> => {
      const Display = (): ReactElement => {
        const value = use({ key });
        const format = useCallback(() => `n=${value}`, [value]);
        return <div data-testid="value">{format()}</div>;
      };
      const Harness = (): ReactElement => {
        const [mounted, setMounted] = useState(false);
        return (
          <>
            <button
              data-testid="mount"
              onClick={() => startTransition(() => setMounted(true))}
            />
            <Errors.SuspenseBoundary loading={<div>loading</div>}>
              {mounted && <Display />}
            </Errors.SuspenseBoundary>
          </>
        );
      };
      return await captureUncaught(async () => {
        let utils!: ReturnType<typeof render>;
        await act(async () => {
          utils = render(
            <Wrapper>
              <Harness />
            </Wrapper>,
          );
        });
        await act(async () => {
          fireEvent.click(utils.getByTestId("mount"));
        });
        await waitFor(() =>
          expect(utils.queryByTestId("value")?.textContent).toBe("n=7"),
        );
      });
    };

    it("keeps the caller's hook order when the answer lands in the domain cache", async () => {
      let cached: number | undefined;
      const { use } = Flux.createRetrieve<{ key: string }, number>({
        name: "Number",
        retrieve: async () => {
          cached = 7;
          return cached;
        },
        getCached: () => cached,
      });
      expect(await mountThroughTransition(use, "replay-domain-cached")).toEqual([]);
    });

    it("keeps the caller's hook order when the answer settles locally", async () => {
      const { use } = Flux.createRetrieve<{ key: string }, number>({
        name: "Number",
        retrieve: async () => 7,
      });
      expect(await mountThroughTransition(use, "replay-settled")).toEqual([]);
    });
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
      const { use } = Flux.createRetrieve<{ key: string }, number[]>({
        name: "Numbers",
        retrieve: async () => [...source],
        onChange: (_, h) => {
          handler = h;
          return () => {};
        },
        getCached: () => [...source],
        equal: sameNumbers,
      });

      const seen: number[][] = [];
      const Display = (): ReactElement => {
        const value = use({ key });
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
      const { use } = Flux.createRetrieve<{ key: string }, number[]>({
        name: "Numbers",
        retrieve: async () => [1, 2],
        onChange: () => () => {},
        getCached: () => tombstone,
        equal: sameNumbers,
      });

      const Display = (): ReactElement => {
        const value = use({ key: "equal-deleted" });
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
      const { use } = Flux.createRetrieve<{ key: string }, number[]>({
        name: "Numbers",
        retrieve,
        onChange: (_, h) => {
          handler = h;
          return () => {};
        },
        getCached: () => cached,
        equal: sameNumbers,
      });

      const Display = (): ReactElement => {
        const value = use({ key: "equal-invalidate" });
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
      const { use } = Flux.createRetrieve<{ key: string }, number[]>({
        name: "Numbers",
        retrieve: async () => [],
        getCached: () => cached,
      });

      const seen: number[][] = [];
      const Display = (): ReactElement => {
        seen.push(use({ key: "no-equal" }));
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

    it("settles a composed answer when no comparator is given", async () => {
      const source = [4, 5];
      const { use } = Flux.createRetrieve<{ key: string }, number[]>({
        name: "Numbers",
        retrieve: async () => [...source],
        onChange: () => () => {},
        getCached: () => [...source],
      });

      const seen: number[][] = [];
      const Display = (): ReactElement => {
        const value = use({ key: "no-equal-composed" });
        seen.push(value);
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

      expect(utils.queryByTestId("value")?.textContent).toEqual("4,5");
      expect(seen.length).toBeLessThan(10);
      expect(seen[seen.length - 1]).toBe(seen[0]);
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
      const { use } = Flux.createRetrieve<{ key: string }, number>({
        name: "Number",
        retrieve,
        onChange: (_, h) => {
          handler = h;
          return () => {};
        },
        getCached: () => cached,
      });

      const Display = (): ReactElement => {
        const value = use({ key });
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
      const { use, useInvalidate } = Flux.createRetrieve<{ key: string }, number>({
        name: "Number",
        retrieve,
        onChange: () => () => {},
        getCached: () => cached,
      });
      const Display = (): ReactElement => (
        <div data-testid="value">{use({ key: "invalidate" })}</div>
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
      const { use } = Flux.createRetrieve<{ key: string }, number>({
        name: "Number",
        retrieve,
        // A row tombstoned between the failed fetch and the subscription
        // mounting answers during onChange itself, so the handler fires
        // before the destructor is returned.
        onChange: (_, h) => {
          cached = tombstone;
          h(tombstone);
          return disconnect;
        },
        getCached: () => cached,
      });

      const Display = (): ReactElement => {
        const value = use({ key: "nf-sync-tombstone" });
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
      const { use } = Flux.createRetrieve<{ key: string }, number>({
        name: "Number",
        retrieve,
        getCached: () => undefined,
      });

      const Display = (): ReactElement => {
        const value = use({ key: "nf-unsubscribed" });
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
    let mounted: ReturnType<typeof render> | undefined;
    try {
      const [first, second] = await client.labels.create([
        { name: `first-${id.create()}`, color: "#000000" },
        { name: `second-${id.create()}`, color: "#000000" },
      ]);
      const { use } = Flux.createRetrieve<{ key: string }, string>({
        name: "Label",
        retrieve: async ({ client, query }) =>
          (await client.labels.retrieve(query.key)).name,
      });
      const Display = ({ labelKey }: { labelKey: string }): ReactElement => (
        <div>{use({ key: labelKey })}</div>
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
      mounted = utils;
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
      // Close the client before the port dies, or its unbounded reconnect loop
      // churns against a dead proxy and logs into whatever test runs next.
      mounted?.unmount();
      await proxy.close();
    }
  }, 30000);
});

describe("useEnsure", () => {
  it("does not suspend when the cache hits", async () => {
    const retrieve = vi.fn(async () => 5);
    const { useEnsure } = Flux.createRetrieve<{ key: string }, number>({
      name: "Number",
      retrieve,
      getCached: () => 5,
    });

    const Display = (): ReactElement => {
      useEnsure({ key: "ensure-cached" });
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
    const { useEnsure } = Flux.createRetrieve<{ key: string }, number>({
      name: "Number",
      retrieve,
      getCached: () => undefined,
    });

    const Display = (): ReactElement => {
      useEnsure({ key: "ensure-error" });
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

describe("use connection changes", () => {
  it("surfaces a disconnect that lands after the read resolved", async () => {
    const { use } = Flux.createRetrieve<{ key: string }, number>({
      name: "Number",
      retrieve: async () => 42,
    });

    const Display = (): ReactElement => {
      const value = use({ key: "disconnect-test" });
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

describe("useTombstone", () => {
  it("reads a deletion as a value and clears it once the record returns", async () => {
    const lbl = await client.labels.create({
      name: "Tombstone Label",
      color: color.construct("#000000"),
    });
    let cached: query.Cached<label.Label> | undefined = lbl;
    let handler: query.ChangeHandler<label.Label> | null = null;
    const { useTombstone } = Flux.createRetrieve<{ key: label.Key }, label.Label>({
      name: "Resource",
      retrieve: async ({ client, query: { key } }) => await client.labels.retrieve(key),
      onChange: (_, h) => {
        handler = h;
        return () => {};
      },
      getCached: () => cached,
    });

    const { result } = renderHook(() => useTombstone({ key: lbl.key }), {
      wrapper: Wrapper,
    });
    expect(result.current).toBeNull();

    act(() => {
      cached = new query.Deleted(lbl, TimeStamp.now());
      handler?.(cached);
    });
    expect(result.current?.name).toEqual("Tombstone Label");

    // The restore heals the read on its own: no boundary reset, no refetch.
    act(() => {
      cached = lbl;
      handler?.(cached);
    });
    expect(result.current).toBeNull();
  });
});

describe("createSelector", () => {
  interface Data {
    name: string;
    value: number;
  }

  interface Harness {
    retrieve: ReturnType<typeof vi.fn<() => Promise<Data>>>;
    set: (next: query.Cached<Data> | undefined) => void;
    createSelector: Flux.CreateSelector<{ key: string }, Data>;
  }

  const createHarness = (initial?: query.Cached<Data>): Harness => {
    let cached = initial;
    let handler: query.ChangeHandler<Data> | undefined;
    const retrieve = vi.fn(async (): Promise<Data> => ({ name: "fetched", value: 0 }));
    const { createSelector } = Flux.createRetrieve<{ key: string }, Data>({
      name: "Resource",
      retrieve,
      onChange: (_, h) => {
        handler = h;
        return () => {};
      },
      getCached: () => cached,
    });
    return {
      retrieve,
      createSelector,
      set: (next) => {
        cached = next;
        handler?.(next);
      },
    };
  };

  it("returns the selected slice of the cached answer without fetching", () => {
    const harness = createHarness({ name: "cached", value: 1 });
    const useName = harness.createSelector((data) => data.name);
    const { result } = renderHook(() => useName({ key: "a" }), { wrapper: Wrapper });
    expect(result.current).toEqual("cached");
    expect(harness.retrieve).not.toHaveBeenCalled();
  });

  it("throws NotFoundError on a cold miss instead of suspending", async () => {
    const harness = createHarness();
    const useName = harness.createSelector((data) => data.name);
    const Display = (): ReactElement => <div>{useName({ key: "a" })}</div>;
    let utils!: ReturnType<typeof render>;
    await act(async () => {
      utils = render(
        <Wrapper>
          <Errors.SuspenseBoundary
            loading={<div>loading-select</div>}
            FallbackComponent={({ error }) => (
              <div data-testid="error">
                {NotFoundError.matches(error) ? "not-found" : "other"}
              </div>
            )}
          >
            <Display />
          </Errors.SuspenseBoundary>
        </Wrapper>,
      );
    });
    expect(utils.queryByText("loading-select")).toBeNull();
    expect(utils.queryByTestId("error")?.textContent).toEqual("not-found");
    expect(harness.retrieve).not.toHaveBeenCalled();
  });

  it("throws DeletedError when the cached answer is a tombstone", async () => {
    const harness = createHarness(
      new query.Deleted<Data>({ name: "corpse", value: 1 }, TimeStamp.now()),
    );
    const useName = harness.createSelector((data) => data.name);
    const Display = (): ReactElement => <div>{useName({ key: "a" })}</div>;
    let utils!: ReturnType<typeof render>;
    await act(async () => {
      utils = render(
        <Wrapper>
          <Errors.SuspenseBoundary
            loading={null}
            FallbackComponent={({ error }) => (
              <div data-testid="error">
                {Flux.DeletedError.matches(error) ? "deleted" : "other"}
              </div>
            )}
          >
            <Display />
          </Errors.SuspenseBoundary>
        </Wrapper>,
      );
    });
    expect(utils.queryByTestId("error")?.textContent).toEqual("deleted");
  });

  it("throws DisconnectedError when no client is connected", async () => {
    const harness = createHarness({ name: "cached", value: 1 });
    const useName = harness.createSelector((data) => data.name);
    const Display = (): ReactElement => <div>{useName({ key: "a" })}</div>;
    const NullWrapper = createSynnaxWrapper({ client: null });
    let utils!: ReturnType<typeof render>;
    await act(async () => {
      utils = render(
        <NullWrapper>
          <Errors.SuspenseBoundary
            loading={null}
            FallbackComponent={({ error }) => (
              <div data-testid="error">
                {DisconnectedError.matches(error) ? "disconnected" : "other"}
              </div>
            )}
          >
            <Display />
          </Errors.SuspenseBoundary>
        </NullWrapper>,
      );
    });
    expect(utils.queryByTestId("error")?.textContent).toEqual("disconnected");
  });

  it("re-renders when a push changes the selected slice", () => {
    const harness = createHarness({ name: "before", value: 1 });
    const useName = harness.createSelector((data) => data.name);
    const { result } = renderHook(() => useName({ key: "a" }), { wrapper: Wrapper });
    expect(result.current).toEqual("before");
    act(() => harness.set({ name: "after", value: 1 }));
    expect(result.current).toEqual("after");
  });

  it("keeps the previous identity when the selected slice is equal", () => {
    const harness = createHarness({ name: "same", value: 1 });
    const useNames = harness.createSelector(
      (data) => [data.name],
      (a, b) => a.length === b.length && a.every((v, i) => v === b[i]),
    );
    const { result } = renderHook(() => useNames({ key: "a" }), { wrapper: Wrapper });
    const first = result.current;
    act(() => harness.set({ name: "same", value: 2 }));
    expect(result.current).toBe(first);
  });

  it("holds the last live value across an invalidation push", () => {
    const harness = createHarness({ name: "live", value: 1 });
    const useName = harness.createSelector((data) => data.name);
    const { result } = renderHook(() => useName({ key: "a" }), { wrapper: Wrapper });
    expect(result.current).toEqual("live");
    act(() => harness.set(undefined));
    expect(result.current).toEqual("live");
  });

  it("memoizes the selection on the raw answer's reference", () => {
    const raw: Data = { name: "stable", value: 1 };
    const harness = createHarness(raw);
    const select = vi.fn((data: Data) => data.name);
    const useName = harness.createSelector(select);
    const { result } = renderHook(() => useName({ key: "a" }), { wrapper: Wrapper });
    expect(result.current).toEqual("stable");
    const calls = select.mock.calls.length;
    act(() => harness.set(raw));
    expect(select.mock.calls.length).toEqual(calls);
  });

  it("re-selects when the query changes", () => {
    const byKey: Record<string, Data> = {
      a: { name: "alpha", value: 1 },
      b: { name: "beta", value: 2 },
    };
    let handler: query.ChangeHandler<Data> | undefined;
    const { createSelector } = Flux.createRetrieve<{ key: string }, Data>({
      name: "Resource",
      retrieve: async () => byKey.a,
      onChange: (_, h) => {
        handler = h;
        return () => {};
      },
      getCached: ({ query: { key } }) => byKey[key],
    });
    const useName = createSelector((data) => data.name);
    const { result, rerender } = renderHook(({ key }) => useName({ key }), {
      wrapper: Wrapper,
      initialProps: { key: "a" },
    });
    expect(result.current).toEqual("alpha");
    rerender({ key: "b" });
    expect(result.current).toEqual("beta");
    expect(handler).toBeDefined();
  });

  it("passes selector-only query fields through to the projection", () => {
    const harness = createHarness({ name: "alpha,beta", value: 1 });
    const usePart = harness.createSelector<string, { key: string; index: number }>(
      (data, { index }) => data.name.split(",")[index],
    );
    const { result, rerender } = renderHook(
      ({ index }) => usePart({ key: "a", index }),
      {
        wrapper: Wrapper,
        initialProps: { index: 0 },
      },
    );
    expect(result.current).toEqual("alpha");
    rerender({ index: 1 });
    expect(result.current).toEqual("beta");
  });

  it("refuses to mint when the definition has no cache read", () => {
    const { createSelector } = Flux.createRetrieve<{ key: string }, Data>({
      name: "Resource",
      retrieve: async () => ({ name: "fetched", value: 0 }),
    });
    expect(() => createSelector((data) => data.name)).toThrow(UnexpectedError);
  });
});

describe("createResultSelector", () => {
  interface Data {
    name: string;
    value: number;
  }

  interface Harness {
    retrieve: ReturnType<typeof vi.fn<() => Promise<Data>>>;
    set: (next: query.Cached<Data> | undefined) => void;
    createResultSelector: Flux.CreateResultSelector<{ key: string }, Data>;
  }

  const createHarness = (
    initial?: query.Cached<Data>,
    retrieveImpl?: () => Promise<Data>,
  ): Harness => {
    let cached = initial;
    const handlers = new Set<query.ChangeHandler<Data>>();
    const retrieve = vi.fn(
      retrieveImpl ?? (async (): Promise<Data> => ({ name: "fetched", value: 0 })),
    );
    const { createResultSelector } = Flux.createRetrieve<{ key: string }, Data>({
      name: "Resource",
      retrieve,
      onChange: (_, h) => {
        handlers.add(h);
        return () => handlers.delete(h);
      },
      getCached: () => cached,
    });
    return {
      retrieve,
      createResultSelector,
      set: (next) => {
        cached = next;
        handlers.forEach((h) => h(next));
      },
    };
  };

  it("serves the selected slice from the cache without fetching", () => {
    const harness = createHarness({ name: "cached", value: 1 });
    const useName = harness.createResultSelector((data) => data.name);
    const { result } = renderHook(() => useName({ key: "a" }), { wrapper: Wrapper });
    expect(result.current.variant).toEqual("success");
    expect(result.current.data).toEqual("cached");
    expect(harness.retrieve).not.toHaveBeenCalled();
  });

  it("fetches on a cold miss and settles to the selected slice", async () => {
    const harness = createHarness();
    const useName = harness.createResultSelector((data) => data.name);
    const { result } = renderHook(() => useName({ key: "a" }), { wrapper: Wrapper });
    expect(result.current.variant).toEqual("loading");
    await waitFor(() => expect(result.current.variant).toEqual("success"));
    expect(result.current.data).toEqual("fetched");
    expect(harness.retrieve).toHaveBeenCalledTimes(1);
  });

  it("re-renders when a push changes the selected slice", () => {
    const harness = createHarness({ name: "before", value: 1 });
    const useName = harness.createResultSelector((data) => data.name);
    const { result } = renderHook(() => useName({ key: "a" }), { wrapper: Wrapper });
    expect(result.current.data).toEqual("before");
    act(() => harness.set({ name: "after", value: 1 }));
    expect(result.current.data).toEqual("after");
  });

  it("does not re-render when a push changes only unselected fields", () => {
    const harness = createHarness({ name: "same", value: 1 });
    const useName = harness.createResultSelector((data) => data.name);
    const renders = vi.fn();
    const { result } = renderHook(
      () => {
        renders();
        return useName({ key: "a" });
      },
      { wrapper: Wrapper },
    );
    const first = result.current;
    const before = renders.mock.calls.length;
    act(() => harness.set({ name: "same", value: 2 }));
    expect(renders.mock.calls.length).toEqual(before);
    expect(result.current).toBe(first);
  });

  it("compares slices with the provided equality", () => {
    const harness = createHarness({ name: "a,b", value: 1 });
    const useParts = harness.createResultSelector(
      (data) => data.name.split(","),
      (a, b) => a.length === b.length && a.every((v, i) => v === b[i]),
    );
    const { result } = renderHook(() => useParts({ key: "a" }), { wrapper: Wrapper });
    const first = result.current;
    act(() => harness.set({ name: "a,b", value: 2 }));
    expect(result.current).toBe(first);
  });

  it("reports a deleted answer as an error result", () => {
    const harness = createHarness(
      new query.Deleted<Data>({ name: "corpse", value: 1 }, TimeStamp.now()),
    );
    const useName = harness.createResultSelector((data) => data.name);
    const { result } = renderHook(() => useName({ key: "a" }), { wrapper: Wrapper });
    expect(result.current.variant).toEqual("error");
    assert(result.current.variant === "error");
    expect(Flux.DeletedError.matches(result.current.status.details.error)).toBe(true);
  });

  it("reads as disabled with a null query", () => {
    const harness = createHarness({ name: "cached", value: 1 });
    const useName = harness.createResultSelector((data) => data.name);
    const { result } = renderHook(() => useName(null), { wrapper: Wrapper });
    expect(result.current.variant).toEqual("disabled");
    expect(harness.retrieve).not.toHaveBeenCalled();
  });

  it("refuses to mint when the definition has no cache read", () => {
    const { createResultSelector } = Flux.createRetrieve<{ key: string }, Data>({
      name: "Resource",
      retrieve: async () => ({ name: "fetched", value: 0 }),
    });
    expect(() => createResultSelector((data) => data.name)).toThrow(UnexpectedError);
  });
});

describe("useResult", () => {
  interface Data {
    name: string;
    value: number;
  }

  interface Harness {
    retrieve: ReturnType<typeof vi.fn<() => Promise<Data>>>;
    set: (next: query.Cached<Data> | undefined) => void;
    useResult: Flux.UseResult<{ key: string }, Data>;
  }

  const createHarness = (
    initial?: query.Cached<Data>,
    retrieveImpl?: () => Promise<Data>,
  ): Harness => {
    let cached = initial;
    const handlers = new Set<query.ChangeHandler<Data>>();
    const retrieve = vi.fn(
      retrieveImpl ?? (async (): Promise<Data> => ({ name: "fetched", value: 0 })),
    );
    const { useResult } = Flux.createRetrieve<{ key: string }, Data>({
      name: "Resource",
      retrieve,
      onChange: (_, h) => {
        handlers.add(h);
        return () => handlers.delete(h);
      },
      getCached: () => cached,
    });
    return {
      retrieve,
      useResult,
      set: (next) => {
        cached = next;
        handlers.forEach((h) => h(next));
      },
    };
  };

  it("serves the cached answer without fetching", () => {
    const harness = createHarness({ name: "cached", value: 1 });
    const { result } = renderHook(() => harness.useResult({ key: "a" }), {
      wrapper: Wrapper,
    });
    expect(result.current.variant).toEqual("success");
    expect(result.current.data).toEqual({ name: "cached", value: 1 });
    expect(harness.retrieve).not.toHaveBeenCalled();
  });

  it("reports loading on a cold miss and serves the fetch once it lands", async () => {
    let resolveRetrieve: (value: Data) => void = () => {};
    const harness = createHarness(
      undefined,
      () =>
        new Promise<Data>((resolve) => {
          resolveRetrieve = resolve;
        }),
    );
    const { result } = renderHook(() => harness.useResult({ key: "a" }), {
      wrapper: Wrapper,
    });
    expect(result.current.variant).toEqual("loading");
    expect(result.current.data).toBeUndefined();
    expect(harness.retrieve).toHaveBeenCalledTimes(1);
    await act(async () => {
      const data = { name: "fetched", value: 2 };
      resolveRetrieve(data);
      harness.set(data);
    });
    expect(result.current.variant).toEqual("success");
    expect(result.current.data).toEqual({ name: "fetched", value: 2 });
  });

  it("dedupes concurrent cold reads into one fetch", () => {
    const harness = createHarness(undefined, () => new Promise<Data>(() => {}));
    renderHook(
      () => [harness.useResult({ key: "a" }), harness.useResult({ key: "a" })],
      { wrapper: Wrapper },
    );
    expect(harness.retrieve).toHaveBeenCalledTimes(1);
  });

  it("surfaces a failed fetch as an error without refetching or logging", async () => {
    const harness = createHarness(undefined, async () => {
      throw new Error("boom");
    });
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
    try {
      const { result, rerender } = renderHook(() => harness.useResult({ key: "a" }), {
        wrapper: Wrapper,
      });
      await act(async () => {
        await Promise.resolve();
      });
      rerender();
      expect(result.current.variant).toEqual("error");
      expect(result.current.data).toBeUndefined();
      expect(result.current.status.description).toContain("boom");
      expect(harness.retrieve).toHaveBeenCalledTimes(1);
      expect(consoleError).not.toHaveBeenCalled();
    } finally {
      consoleError.mockRestore();
    }
  });

  it("serves a record created after a not-found fetch without logging", async () => {
    const harness = createHarness(undefined, async () => {
      throw new NotFoundError("nope");
    });
    const consoleError = vi.spyOn(console, "error").mockImplementation(() => {});
    try {
      const { result } = renderHook(() => harness.useResult({ key: "a" }), {
        wrapper: Wrapper,
      });
      await act(async () => {
        await Promise.resolve();
      });
      expect(result.current.variant).toEqual("loading");
      await act(async () => {
        harness.set({ name: "created", value: 3 });
      });
      expect(result.current.variant).toEqual("success");
      expect(result.current.data).toEqual({ name: "created", value: 3 });
      expect(consoleError).not.toHaveBeenCalled();
    } finally {
      consoleError.mockRestore();
    }
  });

  it("reports a deleted record as an error without fetching", () => {
    const harness = createHarness(
      new query.Deleted<Data>({ name: "corpse", value: 1 }, TimeStamp.now()),
    );
    const { result } = renderHook(() => harness.useResult({ key: "a" }), {
      wrapper: Wrapper,
    });
    assert(result.current.variant === "error");
    expect(result.current.data).toBeUndefined();
    expect(Flux.DeletedError.matches(result.current.status.details.error)).toBe(true);
    expect(harness.retrieve).not.toHaveBeenCalled();
  });

  it("reports disabled when no client is connected", () => {
    const harness = createHarness({ name: "cached", value: 1 });
    const NullWrapper = createSynnaxWrapper({ client: null });
    const { result } = renderHook(() => harness.useResult({ key: "a" }), {
      wrapper: NullWrapper,
    });
    expect(result.current.variant).toEqual("disabled");
    expect(result.current.data).toBeUndefined();
    expect(harness.retrieve).not.toHaveBeenCalled();
  });

  it("re-renders when the cached answer changes", async () => {
    const harness = createHarness({ name: "one", value: 1 });
    const { result } = renderHook(() => harness.useResult({ key: "a" }), {
      wrapper: Wrapper,
    });
    await act(async () => {
      harness.set({ name: "two", value: 2 });
    });
    expect(result.current.variant).toEqual("success");
    expect(result.current.data).toEqual({ name: "two", value: 2 });
  });

  it("skips the read entirely for a null query", () => {
    const harness = createHarness({ name: "cached", value: 1 });
    const { result } = renderHook(() => harness.useResult(null), { wrapper: Wrapper });
    expect(result.current.variant).toEqual("disabled");
    expect(result.current.data).toBeUndefined();
    expect(harness.retrieve).not.toHaveBeenCalled();
  });

  it("serves a settled answer once for a definition without getCached", async () => {
    const retrieve = vi.fn(async (): Promise<Data> => ({ name: "one-shot", value: 4 }));
    const { useResult } = Flux.createRetrieve<{ key: string }, Data>({
      name: "Resource",
      retrieve,
    });
    const { result } = renderHook(() => useResult({ key: "a" }), { wrapper: Wrapper });
    expect(result.current.variant).toEqual("loading");
    await waitFor(() =>
      expect(result.current.data).toEqual({ name: "one-shot", value: 4 }),
    );
    expect(result.current.variant).toEqual("success");
    expect(retrieve).toHaveBeenCalledTimes(1);
  });

  it("serves a settled answer when the retrieve never reaches the cache", async () => {
    const harness = createHarness(undefined, async () => ({
      name: "off-cache",
      value: 5,
    }));
    const { result } = renderHook(() => harness.useResult({ key: "a" }), {
      wrapper: Wrapper,
    });
    expect(result.current.variant).toEqual("loading");
    await waitFor(() =>
      expect(result.current.data).toEqual({ name: "off-cache", value: 5 }),
    );
    expect(harness.retrieve).toHaveBeenCalledTimes(1);
  });

  describe("identity stability", () => {
    it("returns the identical result across re-renders while the answer holds", () => {
      const harness = createHarness({ name: "cached", value: 1 });
      const { result, rerender } = renderHook(() => harness.useResult({ key: "a" }), {
        wrapper: Wrapper,
      });
      const first = result.current;
      rerender();
      expect(result.current).toBe(first);
    });

    it("returns a new result when the answer changes", async () => {
      const harness = createHarness({ name: "one", value: 1 });
      const { result } = renderHook(() => harness.useResult({ key: "a" }), {
        wrapper: Wrapper,
      });
      const first = result.current;
      await act(async () => {
        harness.set({ name: "two", value: 2 });
      });
      expect(result.current).not.toBe(first);
      expect(result.current.data).toEqual({ name: "two", value: 2 });
    });

    it("returns the identical loading result across re-renders", () => {
      const harness = createHarness(undefined, () => new Promise<Data>(() => {}));
      const { result, rerender } = renderHook(() => harness.useResult({ key: "a" }), {
        wrapper: Wrapper,
      });
      const first = result.current;
      expect(first.variant).toEqual("loading");
      rerender();
      expect(result.current).toBe(first);
    });

    it("returns the identical disabled result across re-renders", () => {
      const harness = createHarness({ name: "cached", value: 1 });
      const { result, rerender } = renderHook(() => harness.useResult(null), {
        wrapper: Wrapper,
      });
      const first = result.current;
      expect(first.variant).toEqual("disabled");
      rerender();
      expect(result.current).toBe(first);
    });
  });
});

describe("normalizeQuery", () => {
  type Query = { key: string; includeStatus?: boolean };

  it("hands every callback the one normalized, identity-stable query", async () => {
    const seen: Query[] = [];
    const { useResult } = Flux.createRetrieve<Query, number>({
      name: "Resource",
      normalizeQuery: (query) => ({ includeStatus: true, ...query }),
      retrieve: async ({ query }) => {
        seen.push(query);
        return 1;
      },
      onChange: ({ query }) => {
        seen.push(query);
        return () => {};
      },
      getCached: ({ query }) => {
        seen.push(query);
        return undefined;
      },
    });
    const { rerender } = renderHook(() => useResult({ key: "a" }), {
      wrapper: Wrapper,
    });
    await act(async () => {
      await Promise.resolve();
    });
    rerender();
    rerender();
    expect(seen.length).toBeGreaterThan(2);
    const identities = new Set(seen);
    expect(identities.size).toBe(1);
    expect(seen[0]).toEqual({ key: "a", includeStatus: true });
  });

  it("keeps the caller's value over the merged default", () => {
    const seen: Query[] = [];
    const { useResult } = Flux.createRetrieve<Query, number>({
      name: "Resource",
      normalizeQuery: (query) => ({ includeStatus: true, ...query }),
      retrieve: async () => 1,
      getCached: ({ query }) => {
        seen.push(query);
        return 1;
      },
    });
    renderHook(() => useResult({ key: "a", includeStatus: false }), {
      wrapper: Wrapper,
    });
    expect(seen[0]).toEqual({ key: "a", includeStatus: false });
  });
});
