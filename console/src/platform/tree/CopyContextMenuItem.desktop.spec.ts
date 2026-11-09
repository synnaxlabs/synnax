// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { resource } from "@synnaxlabs/client";
import { stubClipboardWrite } from "@synnaxlabs/lyra/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Tree } from "@/platform/tree";
import { createResource, renderCopyContextMenuItem } from "@/platform/tree/testutil";
import { createCore, createCoreState } from "@/session/core/testutil";

const CORE = createCore("Alpha", { clusterKey: "cluster-9" });

describe("CopyContextMenuItem", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("should copy names without links, since a link opens nothing", async () => {
    const write = stubClipboardWrite();
    await renderCopyContextMenuItem({
      resources: [createResource(resource.idZ.parse("channel:1"), "pressure")],
      items: { channel: Tree.createItem({ type: "channel", linkable: true }) },
      preloadedState: createCoreState([CORE], CORE.key),
    });
    fireEvent.click(screen.getByText("Copy"));
    await waitFor(() => expect(write).toHaveBeenCalledTimes(1));
    const [item] = write.mock.calls[0][0];
    expect(await item.data["text/html"].text()).toBe("pressure");
  });
});
