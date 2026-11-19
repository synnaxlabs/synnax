// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { ReactFlowProvider } from "@xyflow/react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { aether } from "@/aether/aether";
import { CSS } from "@/css";
import { Node } from "@/schematic/node";
import { createSynnaxWrapper } from "@/testutil/Synnax";
import { value } from "@/vis/value/aether";

const deletedState = z.object({ deleted: z.boolean().default(false) }).loose();

// Stands in for the value worker component, reporting its channel as deleted.
class DeletedValue extends aether.Leaf<typeof deletedState> {
  static readonly TYPE = value.Value.TYPE;
  schema = deletedState;

  afterUpdate(): void {
    if (!this.state.deleted) this.setState((s) => ({ ...s, deleted: true }));
  }
}

class LiveValue extends aether.Leaf<typeof deletedState> {
  static readonly TYPE = value.Value.TYPE;
  schema = deletedState;
}

const renderValue = (registry: aether.ComponentRegistry) => {
  const Wrapper = createSynnaxWrapper({ client: null, additionalRegistry: registry });
  const Value = Node.resolveSpec("value").Node;
  return render(
    <Wrapper>
      <ReactFlowProvider>
        <Value
          nodeKey="v1"
          selected={false}
          onConfigChange={() => {}}
          config={Node.createConfig({ variant: "value", channel: 7 })}
        />
      </ReactFlowProvider>
    </Wrapper>,
  );
};

describe("Value.Symbol deleted channel", () => {
  it("should say Deleted channel inside its box, with no badge below", async () => {
    const { findByText, container } = renderValue({
      [DeletedValue.TYPE]: DeletedValue,
    });
    const text = await findByText("Deleted channel");
    expect(text.closest(`.${CSS.BE("value", "content")}`)).not.toBeNull();
    expect(container.querySelector(".lyra-badge")).toBeNull();
  });

  it("should say nothing while its channel exists", () => {
    const { queryByText } = renderValue({ [LiveValue.TYPE]: LiveValue });
    expect(queryByText("Deleted channel")).toBeNull();
  });
});
