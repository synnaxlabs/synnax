// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { PropsWithChildren, ReactElement } from "react";
import { vi } from "vitest";

import { Enumeration } from "@/enumeration";
import { telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/testutil";
import { control } from "@/telem/control/aether";

/** Gives the forms below it a create prompt that never opens. */
export const CreatePromptStub = ({ children }: PropsWithChildren): ReactElement => (
  <Enumeration.CreatePromptProvider value={vi.fn()}>
    {children}
  </Enumeration.CreatePromptProvider>
);

const CHANNEL_VALUE_TYPE = telem.streamChannelValue({ channel: 0 }).type;

/**
 * @returns a telem factory that stands in for the Core and the schematic controller. It
 * replaces the remote factory. Each channel reads its value in values, or no data, from
 * a test source kept in sources. Each command sink is a test sink kept in sinks. Both
 * are by channel.
 */
export const createChannelTelem = (values: Record<number, number> = {}) => {
  const sources = new Map<number, telemTest.TestSource<number>>();
  const sinks = new Map<number, telemTest.TestSink<number>>();
  const factory: telem.Factory = {
    type: "remote",
    create: (spec) => {
      const channel = Number(spec.props.channel);
      if (spec.type === control.SetChannelValue.TYPE) {
        const sink = telemTest.sink<number>();
        sinks.set(channel, sink);
        return sink;
      }
      if (spec.type !== CHANNEL_VALUE_TYPE) return null;
      const source = telemTest.source(channel in values ? values[channel] : NaN);
      sources.set(channel, source);
      return source;
    },
  };
  return { sources, sinks, factory };
};
