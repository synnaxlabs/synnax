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
import { id } from "@synnaxlabs/x";
import { render, waitFor } from "@testing-library/react";
import { ReactFlowProvider } from "@xyflow/react";
import { describe, expect, it } from "vitest";

import { CSS } from "@/css";
import { Node } from "@/schematic/node";
import { createAsyncSynnaxWrapper } from "@/testutil/Synnax";
import { canvasTest } from "@/vis/render/testutil";
import { value } from "@/vis/value/aether";

const client = createTestClient();

const createChannel = async (): Promise<channel.Channel> =>
  await client.channels.create({
    name: `value_${id.create()}`,
    dataType: DataType.FLOAT32,
    virtual: true,
  });

const renderValue = async (channel: channel.Key) => {
  const Providers = await createAsyncSynnaxWrapper({
    client,
    additionalRegistry: value.REGISTRY,
    telemFactories: [],
    renderContext: canvasTest.record(),
  });
  const Value = Node.resolveSpec("value").Node;
  return render(
    <Providers>
      <ReactFlowProvider>
        <Value
          nodeKey={id.create()}
          selected={false}
          onConfigChange={() => {}}
          config={Node.createConfig({ variant: "value", channel })}
        />
      </ReactFlowProvider>
    </Providers>,
  );
};

describe("Value.Symbol deleted channel", () => {
  it("should say Deleted channel inside its box, with no badge below", async () => {
    const ch = await createChannel();
    await client.channels.delete(ch.key);
    const { findByText, container } = await renderValue(ch.key);
    const text = await findByText("Deleted channel");
    expect(text.closest(`.${CSS.BE("value", "content")}`)).not.toBeNull();
    expect(container.querySelector(".lyra-badge")).toBeNull();
  });

  it("should say nothing while its channel exists", async () => {
    const ch = await createChannel();
    const { container, queryByText } = await renderValue(ch.key);
    await waitFor(() =>
      expect(container.querySelector(`.${CSS.BE("value", "content")}`)).not.toBeNull(),
    );
    expect(queryByText("Deleted channel")).toBeNull();
  });
});
