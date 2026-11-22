// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, DataType } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { assertDefined, uniqueName } from "@synnaxlabs/x/testutil";
import { act, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Channel } from "@/feature/channel";
import { createResource, renderTreeContextMenu } from "@/platform/tree/testutil";

const client = createTestClient();

const createCalculated = async () =>
  await client.channels.create({
    name: uniqueName("calc"),
    dataType: DataType.FLOAT32,
    virtual: true,
    expression: "return 1",
  });

describe("channel tree check", () => {
  it("should withhold Edit calculation from a selection of two calculated channels", async () => {
    const first = await createCalculated();
    const second = await createCalculated();
    const reader = createTestClient();
    const { ContextMenu } = Channel.TREE_ITEMS.channel;
    assertDefined(ContextMenu);
    await renderTreeContextMenu(ContextMenu, {
      client: reader,
      resources: [
        createResource(channel.resourceID(first.key), first.name),
        createResource(channel.resourceID(second.key), second.name),
      ],
    });
    expect(await screen.findByText("Group selection")).toBeTruthy();
    await waitFor(() =>
      expect(reader.calculations.outputs.get(first.key)).toMatchObject({
        hidden: false,
      }),
    );
    await act(async () => {});
    expect(screen.queryByText("Edit calculation")).toBeNull();
  });
});
