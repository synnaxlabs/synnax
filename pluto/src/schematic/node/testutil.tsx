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
import type { telem } from "@/telem/aether";
import { telemTest } from "@/telem/aether/testutil";
import { control } from "@/telem/control/aether";

/** Gives the forms below it a create prompt that never opens. */
export const CreatePromptStub = ({ children }: PropsWithChildren): ReactElement => (
  <Enumeration.CreatePromptProvider value={vi.fn()}>
    {children}
  </Enumeration.CreatePromptProvider>
);

/**
 * @returns a telem factory that stands in for the schematic controller. It answers each
 * command sink with a test sink, kept in sinks by its channel.
 */
export const createCommandSinks = () => {
  const sinks = new Map<number, telemTest.TestSink<number>>();
  const factory: telem.Factory = {
    type: "commands",
    create: (spec) => {
      if (spec.type !== control.SetChannelValue.TYPE) return null;
      const sink = telemTest.sink<number>();
      sinks.set(spec.props.channel, sink);
      return sink;
    },
  };
  return { sinks, factory };
};
