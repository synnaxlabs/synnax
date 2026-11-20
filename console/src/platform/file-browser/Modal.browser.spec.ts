// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { screen, waitFor, within } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { page } from "vitest/browser";

import { FileBrowser } from "@/platform/file-browser";
import {
  createFileRack,
  createListing,
  serveListings,
} from "@/platform/file-browser/testutil";
import { renderModalOpener } from "@/platform/modals/testutil";

const client = createTestClient();

describe("FileBrowser.Modal", () => {
  it("should keep the headings over their columns on a sideways scroll", async () => {
    // The columns need 100rem, more than the dialog has at this size.
    await page.viewport(1600, 900);
    const files = await createFileRack(client);
    await serveListings(client, files, [
      createListing({ entries: [{ name: "/", path: "/", kind: "directory" }] }),
      createListing({
        path: "/",
        entries: [{ name: "hosts", path: "/hosts", kind: "file", size: 40 }],
      }),
    ]);
    await renderModalOpener(
      FileBrowser.useModal,
      [{ rack: files.rack.key, path: "/hosts" }],
      { client },
    );
    const size = within(
      await screen.findByRole("option", { name: /^hosts/ }),
    ).getByText("40 B");
    const heading = screen.getByRole("button", { name: /^Size/ });
    const gap = (): number =>
      heading.getBoundingClientRect().right - size.getBoundingClientRect().right;
    const before = gap();
    const list = screen.getByRole("listbox", { name: "Files" });
    list.scrollLeft = list.scrollWidth;
    expect(list.scrollLeft).toBeGreaterThan(0);
    await waitFor(() => expect(gap()).toBeCloseTo(before, 0));
  });
});
