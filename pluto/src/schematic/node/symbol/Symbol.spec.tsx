// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.


import { schematic } from "@synnaxlabs/client";
import { render } from "@testing-library/react";
import { ReactFlowProvider } from "@xyflow/react";
import { describe, expect, it, vi } from "vitest";

import { symbolSpec } from "@/schematic/node/symbol/spec";
import { createSynnaxWrapper } from "@/testutil";
import { symbolNode } from "@/vis/symbolNode/aether";

const Wrapper = createSynnaxWrapper({
  client: null,
  additionalRegistry: symbolNode.REGISTRY,
});

const renderNode = (config: Partial<schematic.SymbolNodeConfig>) =>
  render(
    <Wrapper>
      <ReactFlowProvider>
        <symbolSpec.Node
          nodeKey="s1"
          selected={false}
          onConfigChange={vi.fn()}
          config={schematic.symbolNodeConfigZ.parse({ variant: "symbol", ...config })}
        />
      </ReactFlowProvider>
    </Wrapper>,
  );

describe("Symbol node", () => {
  it("should fill a built-in symbol while a local state reads active", async () => {
    const { container } = renderNode({
      spec: { type: "builtin", name: "valve" },
      color: { type: "fixed", value: "#abcdef" },
      states: [{ key: "active", source: { type: "local", value: 1 } }],
    });
    await vi.waitFor(() =>
      expect(container.querySelector("[fill='#a0c2e3']")).not.toBeNull(),
    );
  });
});
