// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createRenderCounter } from "@synnaxlabs/lyra/testutil";
import { deep } from "@synnaxlabs/x";
import { render, screen } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { ICD } from "@/platform/icd";

const root = deep.path<{ sources: Record<string, ICD.ReadSource> }>();

const createPlan = (newIndex: string): ICD.MessagePlan => ({
  fields: [],
  channels: { props: null, newIndex, channels: {}, creates: [], renames: [] },
});

const Reader = ({ source }: { source: string }): ReactElement => {
  const plan = ICD.usePlan(root.sources[source], 0);
  return <span>{`${source}: ${plan?.channels.newIndex ?? "none"}`}</span>;
};

const planned = (plans: Record<string, string>): ICD.PlannedRead[] =>
  Object.entries(plans).map(([source, newIndex]) => ({
    path: root.sources[source],
    read: 0,
    plan: createPlan(newIndex),
  }));

describe("Plans", () => {
  const { Counted, counts, reset } = createRenderCounter();
  const readers = (
    <>
      <Counted id="a">
        <Reader source="a" />
      </Counted>
      <Counted id="b">
        <Reader source="b" />
      </Counted>
    </>
  );

  it("should render again only the reader of a changed plan", async () => {
    const { rerender } = render(
      <ICD.Plans plans={planned({ a: "a1", b: "b1" })}>{readers}</ICD.Plans>,
    );
    await screen.findByText("b: b1");
    reset();
    rerender(<ICD.Plans plans={planned({ a: "a1", b: "b2" })}>{readers}</ICD.Plans>);
    await screen.findByText("b: b2");
    expect(counts.get("a")).toBeUndefined();
    expect(counts.get("b")).toBe(1);
  });

  it("should drop the plan of a read that the plans leave out", async () => {
    const { rerender } = render(
      <ICD.Plans plans={planned({ a: "a1", b: "b1" })}>{readers}</ICD.Plans>,
    );
    await screen.findByText("b: b1");
    rerender(<ICD.Plans plans={planned({ a: "a1" })}>{readers}</ICD.Plans>);
    await screen.findByText("b: none");
    screen.getByText("a: a1");
  });
});
