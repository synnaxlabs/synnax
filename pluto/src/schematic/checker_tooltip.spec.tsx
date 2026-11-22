// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { Tooltip as Base } from "@synnaxlabs/lyra/tooltip";
import { id, TimeSpan } from "@synnaxlabs/x";
import { render, waitFor } from "@testing-library/react";
import { type PropsWithChildren, type ReactElement } from "react";
import { describe, expect, it } from "vitest";

import { Node } from "@/schematic/node";
import { Tooltip } from "@/schematic/Tooltip";
import { telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/testutil";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { latestSample } from "@/vis/latestSample/aether";

const ANCHOR_RECT = {
  left: 100,
  top: 100,
  right: 200,
  bottom: 140,
  width: 100,
  height: 40,
  x: 100,
  y: 100,
  toJSON: () => ({}),
} as DOMRect;

const createAnchor = (): HTMLElement => {
  const el = document.createElement("div");
  el.getBoundingClientRect = () => ANCHOR_RECT;
  document.body.appendChild(el);
  return el;
};

const STREAM_VALUE_TYPE = telem.streamChannelValue({ channel: 0 }).type;

class StampFactory implements telem.Factory {
  type = "remote";
  create(spec: telem.Spec): telem.Telem | null {
    if (spec.type !== STREAM_VALUE_TYPE) return null;
    return telemTest.source<number>(1);
  }
}

const TOOLTIP = ".pluto-schematic-tooltip";

describe("checker 8f-2: tooltip kind icon of a calculated channel", () => {
  it("should never show a calculated channel with the virtual icon", async () => {
    const author = createTestClient();
    const calc = await author.calculations.create({
      name: id.create(),
      expression: "return 2",
    });
    // A viewer whose client never saw the calculation, and whose symbol has already
    // read the channel.
    const viewer = createTestClient();
    const [output] = await viewer.channels.retrieve([calc.output]);
    expect(output.virtual).toBe(true);
    const Providers = await createAsyncSynnaxWrapper({
      client: viewer,
      telemFactories: [new StampFactory()],
      additionalRegistry: latestSample.REGISTRY,
    });
    const Wrapper = ({ children }: PropsWithChildren): ReactElement => (
      <Providers>
        <Base.Config delay={TimeSpan.milliseconds(50)}>{children}</Base.Config>
      </Providers>
    );
    const seen = new Set<string>();
    const record = () => {
      const tip = document.querySelector(TOOLTIP);
      if (tip == null) return;
      if (tip.querySelector(".lyra-icon--virtual") != null) seen.add("virtual");
      if (tip.querySelector(".lyra-icon--calculation") != null) seen.add("calculation");
    };
    const observer = new MutationObserver(record);
    observer.observe(document.body, { childList: true, subtree: true });
    try {
      render(
        <Tooltip
          anchor={createAnchor()}
          config={Node.createConfig({ variant: "value", channel: calc.output })}
        />,
        { wrapper: Wrapper },
      );
      await waitFor(() => expect(seen.has("calculation")).toBe(true), {
        timeout: 5000,
      });
    } finally {
      observer.disconnect();
    }
    expect([...seen]).toEqual(["calculation"]);
  });
});
