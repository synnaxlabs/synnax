// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { resource } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { stubClipboardWrite, stubClipboardWriteText } from "@synnaxlabs/lyra/testutil";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { Tree } from "@/platform/tree";
import { createResource, renderCopyContextMenuItem } from "@/platform/tree/testutil";
import { createCore, createCoreState } from "@/session/core/testutil";

const client = createTestClient();

const CORE = createCore("Alpha", { clusterKey: "cluster-9" });

const ITEMS: Tree.Items = {
  channel: Tree.createItem({ type: "channel", linkable: true }),
  rack: Tree.createItem({ type: "rack" }),
};

const render = async (resources: Tree.Resource[], selected?: resource.ID[]) =>
  await renderCopyContextMenuItem({
    client,
    resources,
    selected,
    items: ITEMS,
    preloadedState: createCoreState([CORE], CORE.key),
  });

describe("CopyContextMenuItem", () => {
  const ch = createResource(resource.idZ.parse("channel:1"), "R&D <pressure>");
  const rack = createResource(resource.idZ.parse("rack:65537"), "my_rack");

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("should copy the names of the selected resources, one per line", async () => {
    const writeText = stubClipboardWriteText();
    const statuses = await render([ch, rack]);
    fireEvent.click(screen.getByText("Copy"));
    await waitFor(() =>
      expect(writeText).toHaveBeenCalledWith("R&D <pressure>\nmy_rack"),
    );
    await waitFor(() =>
      expect(statuses()).toContainEqual(
        expect.objectContaining({
          variant: "success",
          name: "",
          message: "Copied 2 names to clipboard",
        }),
      ),
    );
  });

  it("should copy links for the types a link opens as HTML", async () => {
    const write = stubClipboardWrite();
    await render([ch, rack]);
    fireEvent.click(screen.getByText("Copy"));
    await waitFor(() => expect(write).toHaveBeenCalledTimes(1));
    const [item] = write.mock.calls[0][0];
    expect(await item.data["text/plain"].text()).toBe("R&D <pressure>\nmy_rack");
    expect(await item.data["text/html"].text()).toBe(
      '<a href="synnax://cluster/cluster-9/channel/1">R&amp;D &lt;pressure&gt;</a>' +
        "<br>my_rack",
    );
  });

  it("should name the resource in the status when one is copied", async () => {
    stubClipboardWriteText();
    const statuses = await render([rack]);
    fireEvent.click(screen.getByText("Copy"));
    await waitFor(() =>
      expect(statuses()).toContainEqual(
        expect.objectContaining({
          variant: "success",
          name: "my_rack",
          message: "Copied name to clipboard",
        }),
      ),
    );
  });

  it("should report a copy the clipboard refuses", async () => {
    stubClipboardWriteText(async () => {
      throw new Error("Write permission denied.");
    });
    const statuses = await render([rack]);
    fireEvent.click(screen.getByText("Copy"));
    await waitFor(() =>
      expect(statuses()).toContainEqual(
        expect.objectContaining({
          variant: "error",
          name: "my_rack",
          message: "Failed to copy name to clipboard",
          description: "Write permission denied.",
        }),
      ),
    );
  });

  it("should render nothing when the tree no longer holds the resources", async () => {
    await render([], [ch.id]);
    expect(screen.queryByText("Copy")).toBeNull();
  });
});
