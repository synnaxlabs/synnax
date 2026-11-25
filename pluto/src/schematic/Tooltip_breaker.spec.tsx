// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DataType } from "@synnaxlabs/client";
import { createTestClient, spyOnSend } from "@synnaxlabs/client/testutil";
import { Tooltip as Base } from "@synnaxlabs/lyra/tooltip";
import { id, sleep, TimeSpan } from "@synnaxlabs/x";
import { render } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement } from "react";
import { beforeAll, describe, expect, it } from "vitest";

import { Node } from "@/schematic/node";
import { Tooltip } from "@/schematic/Tooltip";
import { telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/testutil";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { latestSample } from "@/vis/latestSample/aether";

const client = createTestClient();

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

const getTooltip = (): HTMLElement | null =>
  document.querySelector<HTMLElement>(".pluto-schematic-tooltip");

const outputLookups = (send: ReturnType<typeof spyOnSend>): number =>
  send.mock.calls.filter(([target]) => target === "/calculation/retrieve").length;

describe("Schematic.Tooltip breaker", () => {
  let Providers: FC<PropsWithChildren>;
  beforeAll(async () => {
    Providers = await createAsyncSynnaxWrapper({
      client,
      telemFactories: [new StampFactory()],
      additionalRegistry: latestSample.REGISTRY,
    });
  });
  const wrapperWith =
    (delay: TimeSpan) =>
    ({ children }: PropsWithChildren): ReactElement => (
      <Providers>
        <Base.Config delay={delay}>{children}</Base.Config>
      </Providers>
    );

  it("should fire no output lookup while a mouse sweeps over a symbol", async () => {
    const ch = await client.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    // The symbol under the cursor has already read its channel.
    await client.channels.retrieve(ch.key);
    const send = spyOnSend(client);
    const { unmount } = render(
      <Tooltip
        anchor={createAnchor()}
        config={Node.createConfig({ variant: "value", channel: ch.key })}
      />,
      { wrapper: wrapperWith(TimeSpan.seconds(10)) },
    );
    await sleep.sleep(TimeSpan.milliseconds(300));
    unmount();
    await sleep.sleep(TimeSpan.milliseconds(100));
    expect(getTooltip()).toBeNull();
    expect(outputLookups(send)).toBe(0);
  });
});
