// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { renderToString } from "react-dom/server";
import { describe, expect, it } from "vitest";

import { Diagram } from "@/components/diagram";

const events = (count: number) =>
  Array.from({ length: count }, (_, i) => ({
    label: `Event ${i + 1}`,
    reason: "Why.",
  }));

const results = (writers: Diagram.HandoffWriter[]): string[] => {
  const count = writers[0].authority.length;
  document.body.innerHTML = renderToString(
    <Diagram.Handoff title="Handoff" writers={writers} events={events(count)} />,
  );
  return [...document.querySelectorAll(".handoff-diagram__event-result")].map(
    (result) => result.textContent ?? "",
  );
};

describe("Handoff", () => {
  it("should give control to the writer with the highest authority", () => {
    expect(
      results([
        { name: "A", authority: [100, 100] },
        { name: "B", authority: [null, 200] },
      ]),
    ).toEqual(["A takes control. Why.", "B takes control. Why."]);
  });

  it("should give a tie to the writer that opened first", () => {
    expect(
      results([
        { name: "A", authority: [100, 100, null, 100] },
        { name: "B", authority: [null, 100, 100, 100] },
      ]),
    ).toEqual([
      "A takes control. Why.",
      "A keeps control. Why.",
      "B takes control. Why.",
      "B keeps control. Why.",
    ]);
  });

  it("should say that no writer controls the channel when every writer is closed", () => {
    expect(results([{ name: "A", authority: [null] }])).toEqual([
      "No writer controls the channel.",
    ]);
  });

  it("should throw when a writer has no authority for an event", () => {
    expect(() =>
      renderToString(
        <Diagram.Handoff
          title="Handoff"
          writers={[{ name: "A", authority: [100] }]}
          events={events(2)}
        />,
      ),
    ).toThrow("A needs an authority for each of the 2 events");
  });
});
