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
import { uniqueName } from "@synnaxlabs/x/testutil";
import { cleanup, fireEvent, screen, waitFor, within } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

// The shortcut reads "Ctrl" on Windows. On macOS it shows ⌘ and listens for Command.
await vi.hoisted(async () => {
  const { pinOS } = await import("@/testutil/pinOS");
  pinOS("Windows");
});

import { createRunningRack } from "@/platform/device/testutil";
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
  const served = await serveListings(client, files, listings);
  const handle = await renderModalOpener(
    FileBrowser.useModal,
    [{ rack: files.rack.key }],
    { client },
  );
  return { ...handle, rack: files.rack, served };
};

const openFolder = async (name: string): Promise<void> => {
  fireEvent.doubleClick(await screen.findByText(name));
};

const pressEnter = (): void => {
  const files = screen.getByRole("listbox", { name: "Files" });
  fireEvent.keyDown(files, { key: "Enter", code: "Enter" });
  fireEvent.keyUp(files, { key: "Enter", code: "Enter" });
};

const pressSave = (): void => {
  fireEvent.keyDown(window, { code: "ControlLeft" });
  fireEvent.keyDown(window, { code: "Enter" });
  fireEvent.keyUp(window, { code: "Enter" });
  fireEvent.keyUp(window, { code: "ControlLeft" });
};

describe("FileBrowser.useModal", () => {
  it("should show the last listing of a folder at once when it opens again", async () => {
    const { result, reopen, rack, served } = await openBrowser([TOP]);
    await screen.findByText("secrets");
    fireEvent.click(findButton("Cancel"));
    await expect(result()).resolves.toBeNull();
    // Nothing answers now, so the entries can only come from the last listing.
    served.stop();
    reopen();
    await screen.findByText(rack.name);
    expect(await screen.findByText("secrets")).toBeDefined();
    expect(served.paths).toEqual([""]);
  });

  it("should find the file browser again once the rack gains it", async () => {
    const r = await createRunningRack(client, "file");
    const { result, reopen } = await renderModalOpener(
      FileBrowser.useModal,
      [{ rack: r.key }],
      { client },
    );
    await screen.findByText("Failed to list files");
    fireEvent.click(findButton("Cancel"));
    await expect(result()).resolves.toBeNull();
    const browser = await r.createTask({
      name: uniqueName("file_browser"),
      type: FileBrowser.TASK_TYPE,
      config: {},
    });
    await serveListings(client, { rack: r, browser }, [TOP]);
    reopen();
    expect(await screen.findByText("secrets")).toBeDefined();
  });

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

  it("should pick a file on a double click", async () => {
    const { result } = await openBrowser([TOP, ROOT, ETC]);
    await openFolder("/");
    await openFolder("etc");
    fireEvent.doubleClick(await screen.findByText("hosts"));
    await expect(result()).resolves.toEqual(
      expect.objectContaining({ path: "/etc/hosts" }),
    );
  });

  it("should select a folder on a single click and not open it", async () => {
    await openBrowser([TOP, ROOT, ETC]);
    await openFolder("/");
    fireEvent.click(await screen.findByText("etc"));
    await screen.findByRole("option", { name: "etc", selected: true });
    expect(findButton("Select").getAttribute("aria-disabled")).toBe("true");
  });

  it("should open a folder on a double click", async () => {
    const { result } = await openBrowser([TOP, ROOT, ETC]);
    await openFolder("/");
    fireEvent.doubleClick(await screen.findByText("etc"));
    await screen.findByText("hosts");
    fireEvent.click(findButton("Cancel"));
    await expect(result()).resolves.toBeNull();
  });

  it("should open a selected folder and pick a selected file on Enter", async () => {
    const { result } = await openBrowser([TOP, ROOT, ETC]);
    await openFolder("/");
    fireEvent.click(await screen.findByText("etc"));
    await screen.findByRole("option", { name: "etc", selected: true });
    pressEnter();
    fireEvent.click(await screen.findByText("hosts"));
    await screen.findByRole("option", { name: "hosts", selected: true });
    pressEnter();
    await expect(result()).resolves.toEqual(
      expect.objectContaining({ path: "/etc/hosts" }),
    );
  });

  it("should pick the selected file on the shortcut its Select button shows", async () => {
    const { result } = await openBrowser([TOP, ROOT, ETC]);
    await openFolder("/");
    await openFolder("etc");
    fireEvent.click(await screen.findByText("client.crt"));
    within(findButton("Select")).getByText("Ctrl");
    pressSave();
    await expect(result()).resolves.toEqual(
      expect.objectContaining({ path: "/etc/client.crt" }),
    );
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
