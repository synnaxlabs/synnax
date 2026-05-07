// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client";
import { act, render } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it, vi } from "vitest";

import { Flux } from "@/flux";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();
const Wrapper = createSynnaxWrapper({ client });

describe("Flux.Suspense.createRetrieve", () => {
  it("suspends until the retrieve resolves, then returns the value", async () => {
    let resolveRetrieve: (value: number) => void = () => {};
    const useRetrieve = Flux.Suspense.createRetrieve<{ key: string }, number>({
      name: "Number",
      retrieve: () =>
        new Promise<number>((resolve) => {
          resolveRetrieve = resolve;
        }),
    });

    const Display = (): ReactElement => {
      const value = useRetrieve({ key: "first-test" });
      return <div data-testid="value">{value}</div>;
    };

    let utils!: ReturnType<typeof render>;
    await act(async () => {
      utils = render(
        <Wrapper>
          <Flux.Suspense.Boundary loading={<div>loading-1</div>}>
            <Display />
          </Flux.Suspense.Boundary>
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
    const useRetrieve = Flux.Suspense.createRetrieve<{ key: string }, number>({
      name: "Number",
      retrieve,
    });

    const Display = (): ReactElement => {
      const value = useRetrieve({ key: "dedupe-test" });
      return <div>{value}</div>;
    };

    await act(async () => {
      render(
        <Wrapper>
          <Flux.Suspense.Boundary loading={<div>loading-2</div>}>
            <Display />
            <Display />
            <Display />
          </Flux.Suspense.Boundary>
        </Wrapper>,
      );
    });

    expect(retrieve).toHaveBeenCalledTimes(1);

    await act(async () => {
      resolveRetrieve(7);
    });
  });

  it("routes a thrown error to the error fallback", async () => {
    const useRetrieve = Flux.Suspense.createRetrieve<{ key: string }, number>({
      name: "Number",
      retrieve: async () => {
        throw new Error("boom");
      },
    });

    const Display = (): ReactElement => {
      const value = useRetrieve({ key: "error-test" });
      return <div>{value}</div>;
    };

    let utils!: ReturnType<typeof render>;
    await act(async () => {
      utils = render(
        <Wrapper>
          <Flux.Suspense.Boundary
            loading={<div>loading-3</div>}
            error={(status) => <div data-testid="error">{status.message}</div>}
          >
            <Display />
          </Flux.Suspense.Boundary>
        </Wrapper>,
      );
    });

    // Drain microtasks so the rejected promise propagates to the error boundary.
    await act(async () => {
      await Promise.resolve();
    });

    expect(utils.queryByTestId("error")?.textContent).toBe("Failed to retrieve Number");
  });
});
