// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, DataType } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { Tooltip as Base } from "@synnaxlabs/lyra/tooltip";
import { id, TimeSpan } from "@synnaxlabs/x";
import { render, waitFor, within } from "@testing-library/react";
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

describe("Schematic.Tooltip tried", () => {
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

  it("should not show a calculated channel as virtual when it opens warm", async () => {
    const source = await client.channels.create({
      name: id.create(),
      dataType: DataType.FLOAT32,
      virtual: true,
    });
    const name = id.create();
    const calc = await client.calculations.create({
      name,
      expression: `return ${source.name} * 2`,
    });
    const output: channel.Channel = await client.channels.retrieve(calc.output);
    const Wrapper = wrapperWith(TimeSpan.milliseconds(50));
    const { rerender } = render(
      <Tooltip anchor={createAnchor()} config={Node.createConfig({ variant: "valve" })} />,
      { wrapper: Wrapper },
    );
    await waitFor(() => expect(getTooltip()).not.toBeNull());
    rerender(<div />);
    rerender(
      <Tooltip
        anchor={createAnchor()}
        config={Node.createConfig({ variant: "value", channel: output.key })}
      />,
    );
    await waitFor(() => expect(getTooltip()).not.toBeNull());
    const tooltip = getTooltip() as HTMLElement;
    const value = within(tooltip).getByText(name).nextElementSibling as HTMLElement;
    expect(value.querySelector(".lyra-icon--virtual")).toBeNull();
  });
});
