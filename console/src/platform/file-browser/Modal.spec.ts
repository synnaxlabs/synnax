// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type certificate, TimeSpan, TimeStamp } from "@synnaxlabs/client";
import { createTestClient } from "@synnaxlabs/client/testutil";
import { cleanup, fireEvent, screen, waitFor } from "@testing-library/react";
import { afterEach, describe, expect, it } from "vitest";

import { FileBrowser } from "@/platform/file-browser";
import {
  createFileRack,
  createListing,
  serveListings,
} from "@/platform/file-browser/testutil";
import { findButton, renderModalOpener } from "@/platform/modals/testutil";

const client = createTestClient();

afterEach(() => cleanup());

const DETAILS: certificate.Details = {
  subject: "CN=Plant client",
  issuer: "CN=Plant CA",
  notBefore: TimeStamp.now().sub(TimeSpan.days(1)),
  notAfter: TimeStamp.now().add(TimeSpan.days(365)),
  sha1: "AB",
  sha256: "CD",
};

const TOP = createListing({
  secretsDir: "/etc/synnax/secrets",
  entries: [
    { name: "/", path: "/", kind: "directory" },
    { name: "secrets", path: "/etc/synnax/secrets", kind: "directory" },
  ],
});

const ROOT = createListing({
  path: "/",
  entries: [{ name: "etc", path: "/etc", kind: "directory", size: 0 }],
});

const ETC = createListing({
  path: "/etc",
  parent: "/",
  entries: [
    { name: "ssl", path: "/etc/ssl", kind: "directory", size: 0 },
    {
      name: "client.crt",
      path: "/etc/client.crt",
      kind: "file",
      size: 1200,
      certificate: DETAILS,
    },
    { name: "hosts", path: "/etc/hosts", kind: "file", size: 40 },
  ],
});

const openBrowser = async (listings: ReturnType<typeof createListing>[]) => {
  const files = await createFileRack(client);
  await serveListings(client, files, listings);
  const handle = await renderModalOpener(
    FileBrowser.useModal,
    [{ rack: files.rack.key }],
    { client },
  );
  return { ...handle, rack: files.rack };
};

const openFolder = async (name: string): Promise<void> => {
  fireEvent.click(await screen.findByText(name));
};

describe("FileBrowser.useModal", () => {
  it("should resolve with a certificate file picked in a folder", async () => {
    const { result, rack } = await openBrowser([TOP, ROOT, ETC]);
    await screen.findByText(rack.name);
    await openFolder("/");
    await openFolder("etc");
    fireEvent.click(await screen.findByText("client.crt"));
    expect(screen.getByText(/^CN=Plant client · Expires /)).toBeDefined();
    fireEvent.click(findButton("Select"));
    await expect(result()).resolves.toEqual(
      expect.objectContaining({
        path: "/etc/client.crt",
        certificate: expect.objectContaining({ subject: DETAILS.subject }),
      }),
    );
  });

  it("should resolve with a file on a double click", async () => {
    const { result } = await openBrowser([TOP, ROOT, ETC]);
    await openFolder("/");
    await openFolder("etc");
    fireEvent.doubleClick(await screen.findByText("hosts"));
    await expect(result()).resolves.toEqual(
      expect.objectContaining({ path: "/etc/hosts" }),
    );
  });

  it("should open a folder on a double click and not resolve with it", async () => {
    const { result } = await openBrowser([TOP, ROOT, ETC]);
    await openFolder("/");
    const etc = await screen.findByText("etc");
    fireEvent.click(etc);
    fireEvent.doubleClick(etc);
    await screen.findByText("hosts");
    fireEvent.click(findButton("Cancel"));
    await expect(result()).resolves.toBeNull();
  });

  it("should go up to the parent folder", async () => {
    await openBrowser([TOP, ROOT, ETC]);
    await openFolder("/");
    await openFolder("etc");
    await screen.findByText("hosts");
    fireEvent.click(screen.getByRole("button", { name: "Up" }));
    await screen.findByText("etc");
    expect(screen.queryByText("hosts")).toBeNull();
  });

  it("should keep the listing and show the error of a folder that fails", async () => {
    await openBrowser([TOP, ROOT, ETC]);
    await openFolder("/");
    await openFolder("etc");
    await openFolder("ssl");
    await screen.findByText("/etc/ssl does not exist.");
    expect(screen.getByText("Failed to list files")).toBeDefined();
    expect(screen.getByText("hosts")).toBeDefined();
  });

  it("should say when the rack stopped before it listed every entry", async () => {
    await openBrowser([{ ...TOP, truncated: true }]);
    await screen.findByText(`Shows the first ${TOP.entries.length} entries.`);
  });

  it("should resolve with null on cancel", async () => {
    const { result } = await openBrowser([TOP]);
    await screen.findByText("secrets");
    fireEvent.click(findButton("Cancel"));
    await expect(result()).resolves.toBeNull();
    await waitFor(() => expect(screen.queryByText("secrets")).toBeNull());
  });
});
