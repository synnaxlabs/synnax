// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, schematic } from "@synnaxlabs/client";
import type { SyntheticCore } from "@synnaxlabs/client/testutil";
import { render, type RenderResult } from "@testing-library/react";
import { ReactFlowProvider } from "@xyflow/react";
import type { FC, PropsWithChildren } from "react";
import { type Mock, vi } from "vitest";
import { z } from "zod";

import { aether } from "@/aether/aether";
import type { NodeProps } from "@/schematic/node/spec";
import { symbolSpec } from "@/schematic/node/symbol/spec";
import type { telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/testutil";
import { control } from "@/telem/control/aether";
import { createSynnaxWrapper } from "@/testutil";
import { symbolNode } from "@/vis/symbolNode/aether";

// The light theme of the wrapper shows #abcdef as #a0c2e3.
export const COLOR = { type: "fixed" as const, value: "#abcdef" };
export const FILLED = "[fill='#a0c2e3']";

// Every sample of the synthetic Core reads as 1 through this match.
export const WITHIN = { type: "within" as const, bounds: { lower: -2, upper: 2 } };

const chipState = z.object({ sink: z.unknown() }).loose();
const indicatorState = z.object({}).loose();

class StubIndicator extends aether.Leaf<typeof indicatorState> {
  static readonly TYPE = control.Indicator.TYPE;
  override schema = indicatorState;
}

export interface ControlStubs {
  registry: aether.ComponentRegistry;
  /** chipSinks gets the sink that each control chip acquires with. */
  chipSinks: Mock;
}

/** @returns worker components that stand in for the control chip and indicator. */
export const createControlStubs = (): ControlStubs => {
  const chipSinks = vi.fn();
  class StubChip extends aether.Leaf<typeof chipState> {
    static readonly TYPE = control.Chip.TYPE;
    override schema = chipState;

    override afterUpdate(): void {
      chipSinks(this.state.sink);
    }
  }
  return {
    registry: { [StubChip.TYPE]: StubChip, [StubIndicator.TYPE]: StubIndicator },
    chipSinks,
  };
};

/** CommandFactory records each command write by channel, with no control session. */
export class CommandFactory implements telem.Factory {
  type = "command";
  readonly sinks = new Map<channel.Key, telemTest.TestSink<number>>();

  create(spec: telem.Spec): telem.Telem | null {
    if (spec.type !== control.SetChannelValue.TYPE) return null;
    const sink = telemTest.sink<number>();
    this.sinks.set(control.setChannelValuePropsZ.parse(spec.props).channel, sink);
    return sink;
  }
}

export interface WrapperOptions {
  /** core serves the channels that states read. */
  core?: SyntheticCore;
  commands?: CommandFactory;
  registry?: aether.ComponentRegistry;
}

/** @returns a wrapper that runs symbol nodes with no client. */
export const createWrapper = ({
  core,
  commands,
  registry,
}: WrapperOptions = {}): FC<PropsWithChildren> => {
  const factories = [
    ...(core == null ? [] : [telemTest.createRemoteFactory(core)]),
    ...(commands == null ? [] : [commands]),
  ];
  return createSynnaxWrapper({
    client: null,
    additionalRegistry: { ...symbolNode.REGISTRY, ...registry },
    telemFactories: factories.length === 0 ? undefined : factories,
  });
};

const LocalWrapper = createWrapper();

/** @returns the render of one symbol node with config. */
export const renderNode = (
  config: Omit<z.input<typeof schematic.symbolNodeConfigZ>, "variant">,
  Wrapper: FC<PropsWithChildren> = LocalWrapper,
  onConfigChange: NodeProps["onConfigChange"] = vi.fn(),
): RenderResult =>
  render(
    <Wrapper>
      <ReactFlowProvider>
        <symbolSpec.Node
          nodeKey="s1"
          selected={false}
          onConfigChange={onConfigChange}
          config={schematic.symbolNodeConfigZ.parse({ variant: "symbol", ...config })}
        />
      </ReactFlowProvider>
    </Wrapper>,
  );
