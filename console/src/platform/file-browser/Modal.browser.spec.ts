// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createTestClient } from "@synnaxlabs/client/testutil";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
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

const TOP = createListing({ entries: [{ name: "/", path: "/", kind: "directory" }] });

const ROOT = createListing({
  path: "/",
  entries: [
    { name: "empty", path: "/empty", kind: "directory" },
    { name: "hosts", path: "/hosts", kind: "file", size: 40 },
  ],
});

const EMPTY = createListing({ path: "/empty", parent: "/" });

const openWide = async (): Promise<void> => {
  // The columns need 100rem, more than the dialog has at this size.
  await page.viewport(1600, 900);
  const files = await createFileRack(client);
  await serveListings(client, files, [TOP, ROOT, EMPTY]);
  await renderModalOpener(
    FileBrowser.useModal,
    [{ rack: files.rack.key, path: "/hosts" }],
    { client },
  );
  await screen.findByRole("option", { name: /^hosts/ });
};

const sizeGap = (): number => {
  const heading = screen.getByRole("button", { name: /^Size/ });
  const cell = within(screen.getByRole("option", { name: /^hosts/ })).getByText("40 B");
  return heading.getBoundingClientRect().right - cell.getBoundingClientRect().right;
};

const scrollRight = (): void => {
  const list = screen.getByRole("listbox", { name: "Files" });
  list.scrollLeft = list.scrollWidth;
  expect(list.scrollLeft).toBeGreaterThan(0);
};

describe("FileBrowser.Modal", () => {
  it("should keep the headings over their columns on a sideways scroll", async () => {
    await openWide();
    const before = sizeGap();
    scrollRight();
    await waitFor(() => expect(sizeGap()).toBeCloseTo(before, 0));
  });

  it("should line up the headings again after an empty folder", async () => {
    await openWide();
    const before = sizeGap();
    scrollRight();
    fireEvent.doubleClick(screen.getByRole("option", { name: /^empty/ }));
    await screen.findByText("This folder is empty");
    fireEvent.click(screen.getByRole("button", { name: "Back" }));
    await screen.findByRole("option", { name: /^hosts/ });
    await waitFor(() => expect(sizeGap()).toBeCloseTo(before, 0));
  });
});
