// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";

import { Sift } from "@/feature/sift";
import { createSiftDevice } from "@/feature/sift/testutil";
import { createDeviceResource, renderMenuItem } from "@/platform/device/testutil";
import {
  createBaseProps,
  createSelection,
  createState,
} from "@/platform/tree/testutil";
import { createTestStore } from "@/testutil";

const client = createTestClient();

describe("Sift.Device.ContextMenuItems", () => {
  it("should open the connection of the selected device for editing", async () => {
    const dev = await createSiftDevice(client);
    const resource = createDeviceResource(dev);
    const onConfigure = vi.fn();
    await renderMenuItem(
      <Sift.Device.ContextMenuItems
        {...createBaseProps({ client, store: await createTestStore() })}
        selection={createSelection({ ids: [resource.id] })}
        state={createState([resource])}
        onConfigure={onConfigure}
      />,
      { client },
    );
    fireEvent.click(await screen.findByText("Edit connection"));
    expect(onConfigure).toHaveBeenCalledWith(dev.key);
  });
});
