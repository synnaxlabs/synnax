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
import { type ReactElement, useRef } from "react";
import { describe, expect, it } from "vitest";

import { Flux } from "@/flux";
import { createSynnaxWrapper } from "@/testutil/Synnax";

const client = createTestClient();
const Wrapper = createSynnaxWrapper({ client });

interface Doc {
  key: string;
  name: string;
  count: number;
}

describe("Flux.Suspense.createSelector", () => {
  it("suspends until the upstream resolves, then returns the slice", async () => {
    let resolveDoc: (doc: Doc) => void = () => {};
    const useDoc = Flux.Suspense.createRetrieve<{ key: string }, Doc>({
      name: "Doc",
      retrieve: () =>
        new Promise<Doc>((resolve) => {
          resolveDoc = resolve;
        }),
    });

    const useDocName = Flux.Suspense.createSelector({
      upstream: useDoc,
      select: (doc) => doc.name,
    });

    const Display = (): ReactElement => {
      const name = useDocName({ key: "k1" });
      return <div data-testid="name">{name}</div>;
    };

    let utils!: ReturnType<typeof render>;
    await act(async () => {
      utils = render(
        <Wrapper>
          <Flux.Suspense.Boundary loading={<div>loading</div>}>
            <Display />
          </Flux.Suspense.Boundary>
        </Wrapper>,
      );
    });

    expect(utils.queryByText("loading")).toBeTruthy();
    expect(utils.queryByTestId("name")).toBeNull();

    await act(async () => {
      resolveDoc({ key: "k1", name: "Foo", count: 0 });
    });

    expect(utils.queryByTestId("name")?.textContent).toBe("Foo");
  });

  it("skips re-renders when the slice is unchanged under equal", async () => {
    let externalChange:
      | ((updater: (prev: Doc | undefined) => Doc | undefined) => void)
      | null = null;

    const useDoc = Flux.Suspense.createRetrieve<{ key: string }, Doc>({
      name: "Doc",
      retrieve: async () => ({ key: "k1", name: "Foo", count: 0 }),
      mountListeners: ({ onChange }) => {
        externalChange = onChange;
        return () => {
          externalChange = null;
        };
      },
    });

    const useDocName = Flux.Suspense.createSelector({
      upstream: useDoc,
      select: (doc) => doc.name,
      equal: (a, b) => a === b,
    });

    let renderCount = 0;
    const Display = (): ReactElement => {
      renderCount++;
      const name = useDocName({ key: "k1" });
      const initial = useRef(renderCount);
      return (
        <div data-testid="name" data-renders={renderCount}>
          {name}-{initial.current}
        </div>
      );
    };

    let utils!: ReturnType<typeof render>;
    await act(async () => {
      utils = render(
        <Wrapper>
          <Flux.Suspense.Boundary loading={<div>loading</div>}>
            <Display />
          </Flux.Suspense.Boundary>
        </Wrapper>,
      );
    });

    expect(utils.queryByTestId("name")?.textContent).toMatch(/^Foo-/);
    const baselineRenders = Number(utils.queryByTestId("name")?.dataset.renders);
    expect(baselineRenders).toBeGreaterThan(0);

    // Update count without changing name; selector should not re-render the consumer.
    await act(async () => {
      externalChange!((prev) =>
        prev == null ? prev : { ...prev, count: prev.count + 1 },
      );
    });
    expect(Number(utils.queryByTestId("name")?.dataset.renders)).toBe(baselineRenders);

    // Now change name; selector should re-render.
    await act(async () => {
      externalChange!((prev) => (prev == null ? prev : { ...prev, name: "Bar" }));
    });
    expect(utils.queryByTestId("name")?.textContent).toMatch(/^Bar-/);
    expect(Number(utils.queryByTestId("name")?.dataset.renders)).toBeGreaterThan(
      baselineRenders,
    );
  });
});
