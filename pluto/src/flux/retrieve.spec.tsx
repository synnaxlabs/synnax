// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type cache, type label } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { color } from "@synnaxlabs/x";
import { act, render, renderHook, waitFor } from "@testing-library/react";
import { type ReactElement, useCallback, useState } from "react";
import { describe, expect, it, vi } from "vitest";

import { Errors } from "@/errors";
import { Flux } from "@/flux";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();
const Wrapper = createSynnaxWrapper({ client });

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

      it("should allow null client when allowDisconnected is true", async () => {
        const { useRetrieve } = Flux.createRetrieve<{}, number, true>({
          name: "Resource",
          retrieve: async ({ client }) => {
            if (client == null) return 42;
            return 0;
          },
          allowDisconnected: true,
        });

        const { result } = renderHook(() => useRetrieve({ params: {} }), {
          wrapper: createSynnaxWrapper({ client: null }),
        });
        await waitFor(() => {
          expect(result.current.variant).toEqual("success");
          expect(result.current.data).toEqual(42);
        });
      });
    });

    describe("subscriptions", () => {
      it("should update the result when the subscription pushes a change", async () => {
        const ch = await client.labels.create({
          name: "Test Label",
          color: color.construct("#000000"),
        });
        let handler: cache.ChangeHandler<label.Label> | null = null;
        const { useRetrieve } = Flux.createRetrieve<{ key: label.Key }, label.Label>({
          name: "Resource",
          retrieve: async ({ client, query: { key } }) =>
            await client.labels.retrieve({ key }),
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
          handler?.({ variant: "changed", data: { ...ch, name: "Test Label 2" } });
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
        let handler: cache.ChangeHandler<label.Label> | null = null;
        const { useRetrieve } = Flux.createRetrieve<{ key: label.Key }, label.Label>({
          name: "Resource",
          retrieve: async ({ client, query: { key } }) =>
            await client.labels.retrieve({ key }),
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
          handler?.({ variant: "deleted", corpse: ch });
        });
        await waitFor(() => {
          expect(result.current.variant).toEqual("error");
          expect(result.current.status.description).toEqual("Resource was deleted");
        });
      });
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

  it("refetches when the subscription reports an invalidated answer", async () => {
    let capturedHandler: cache.ChangeHandler<number> | null = null;
    let cached: cache.Cached<number> | undefined;
    const retrieve = vi.fn(async () => {
      const value = retrieve.mock.calls.length;
      cached = { variant: "changed", data: value };
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
});
