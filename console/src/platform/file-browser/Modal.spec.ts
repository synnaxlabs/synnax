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
  subject: "O=Plant,CN=Plant client",
  issuer: "CN=Plant CA",
  notBefore: TimeStamp.now().sub(TimeSpan.days(1)),
  notAfter: TimeStamp.now().add(TimeSpan.days(365)),
  sha1: "AB",
  sha256: "CD",
};

const SECRETS_DIR = "/etc/synnax/secrets";

const TOP = createListing({
  secretsDir: SECRETS_DIR,
  entries: [
    { name: "/", path: "/", kind: "directory" },
    { name: "secrets", path: SECRETS_DIR, kind: "directory" },
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
    { name: "hosts", path: "/etc/hosts", kind: "file", size: 40 },
    { name: "ssl", path: "/etc/ssl", kind: "directory", size: 0 },
    {
      name: "client.crt",
      path: "/etc/client.crt",
      kind: "file",
      size: 1200,
      certificate: DETAILS,
    },
  ],
});

const SECRETS = createListing({
  path: SECRETS_DIR,
  parent: "/etc/synnax",
  secret: true,
  entries: [{ name: "ca.crt", path: `${SECRETS_DIR}/ca.crt`, kind: "file", size: 9 }],
});

const LISTINGS = [TOP, ROOT, ETC];

const openBrowser = async (
  listings: ReturnType<typeof createListing>[],
  path?: string,
) => {
  const files = await createFileRack(client);
  const served = await serveListings(client, files, listings);
  const handle = await renderModalOpener(
    FileBrowser.useModal,
    [{ rack: files.rack.key, path }],
    { client },
  );
  return { ...handle, rack: files.rack, served };
};

const findEntry = async (name: string, selected?: boolean): Promise<HTMLElement> =>
  await screen.findByRole("option", { name: new RegExp(`^${name}`), selected });

const openFolder = async (name: string): Promise<void> => {
  fireEvent.doubleClick(await findEntry(name));
};

const openEtc = async (): Promise<void> => {
  await openFolder("/");
  await openFolder("etc");
  await findEntry("hosts");
};

const findFiles = (): HTMLElement => screen.getByRole("listbox", { name: "Files" });

const pressEnter = (): void => {
  const files = findFiles();
  fireEvent.keyDown(files, { key: "Enter", code: "Enter" });
  fireEvent.keyUp(files, { key: "Enter", code: "Enter" });
};

const pressSave = (): void => {
  fireEvent.keyDown(window, { code: "ControlLeft" });
  fireEvent.keyDown(window, { code: "Enter" });
  fireEvent.keyUp(window, { code: "Enter" });
  fireEvent.keyUp(window, { code: "ControlLeft" });
};

const entryNames = (): string[] =>
  screen.getAllByRole("option").map((o) => o.textContent);

describe("FileBrowser.useModal", () => {
  describe("session", () => {
    it("should show the last listing of a folder at once when it opens again", async () => {
      const { result, reopen, rack, served } = await openBrowser([TOP]);
      await findEntry("secrets");
      fireEvent.click(findButton("Cancel"));
      await expect(result()).resolves.toBeNull();
      // Nothing answers now, so the entries can only come from the last listing.
      served.stop();
      reopen();
      await screen.findByText(rack.name);
      expect(await findEntry("secrets")).toBeDefined();
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
      expect(await findEntry("secrets")).toBeDefined();
    });

    it("should list a selected folder before it opens", async () => {
      const { served } = await openBrowser(LISTINGS);
      await openFolder("/");
      fireEvent.click(await findEntry("etc"));
      await waitFor(() => expect(served.paths).toContain("/etc"));
    });
  });

  describe("start folder", () => {
    it("should open the home folder first", async () => {
      const home = createListing({
        path: "/home/op",
        parent: "/home",
        entries: [{ name: "notes.txt", path: "/home/op/notes.txt", kind: "file" }],
      });
      const top = createListing({
        secretsDir: SECRETS_DIR,
        entries: [...TOP.entries, { name: "op", path: "/home/op", kind: "directory" }],
      });
      await openBrowser([top, home]);
      expect(await findEntry("notes.txt")).toBeDefined();
    });

    it("should open the folder of a given file with the file selected", async () => {
      const { result } = await openBrowser(LISTINGS, "/etc/client.crt");
      await findEntry("client.crt", true);
      fireEvent.click(findButton("Select"));
      await expect(result()).resolves.toEqual(
        expect.objectContaining({ path: "/etc/client.crt" }),
      );
    });
  });

  describe("navigation", () => {
    it("should open a place of the Driver", async () => {
      await openBrowser([...LISTINGS, SECRETS]);
      await findEntry("secrets");
      fireEvent.click(screen.getByText("Secrets folder"));
      expect(await findEntry("ca.crt")).toBeDefined();
    });

    it("should open a folder on the way to the open one", async () => {
      const ssl = createListing({
        path: "/etc/ssl",
        parent: "/etc",
        entries: [{ name: "ca.pem", path: "/etc/ssl/ca.pem", kind: "file" }],
      });
      await openBrowser([...LISTINGS, ssl]);
      await openEtc();
      await openFolder("ssl");
      await findEntry("ca.pem");
      const path = screen.getByLabelText("Path");
      fireEvent.click(within(path).getByRole("button", { name: "etc" }));
      expect(await findEntry("hosts")).toBeDefined();
    });

    it("should select a folder on a single click and not open it", async () => {
      await openBrowser(LISTINGS);
      await openFolder("/");
      fireEvent.click(await findEntry("etc"));
      await findEntry("etc", true);
      expect(findButton("Select").getAttribute("aria-disabled")).toBe("true");
    });

    it("should go up and select the folder it came from", async () => {
      await openBrowser(LISTINGS);
      await openEtc();
      fireEvent.click(screen.getByRole("button", { name: "Up" }));
      expect(await findEntry("etc", true)).toBeDefined();
    });

    it("should go up on Backspace in the list", async () => {
      await openBrowser(LISTINGS);
      await openEtc();
      fireEvent.keyDown(findFiles(), { key: "Backspace" });
      expect(await findEntry("etc", true)).toBeDefined();
    });

    it("should go up a Windows path to its drive", async () => {
      const dir = (path: string, name: string) =>
        ({ name, path, kind: "directory" }) as const;
      await openBrowser(
        [
          createListing({ entries: [dir("C:\\", "C:\\")] }),
          createListing({ path: "C:\\", entries: [dir("C:\\Users", "Users")] }),
          createListing({ path: "C:\\Users", entries: [dir("C:\\Users\\op", "op")] }),
          createListing({
            path: "C:\\Users\\op",
            entries: [{ name: "a.crt", path: "C:\\Users\\op\\a.crt", kind: "file" }],
          }),
        ],
        "C:\\Users\\op\\a.crt",
      );
      await findEntry("a.crt", true);
      const up = screen.getByRole("button", { name: "Up" });
      fireEvent.click(up);
      await findEntry("op", true);
      fireEvent.click(up);
      await findEntry("Users", true);
      expect(up.getAttribute("aria-disabled")).toBe("true");
    });

    it("should go back and forward", async () => {
      await openBrowser(LISTINGS);
      await openEtc();
      fireEvent.click(screen.getByRole("button", { name: "Back" }));
      await findEntry("etc", true);
      fireEvent.click(screen.getByRole("button", { name: "Forward" }));
      expect(await findEntry("hosts")).toBeDefined();
    });

    it("should go back on Alt+Left", async () => {
      await openBrowser(LISTINGS);
      await openEtc();
      fireEvent.keyDown(findFiles(), { key: "ArrowLeft", altKey: true });
      expect(await findEntry("etc", true)).toBeDefined();
    });
  });

  describe("pick", () => {
    it("should resolve with a certificate file picked in a folder", async () => {
      const { result } = await openBrowser(LISTINGS);
      await openEtc();
      fireEvent.click(await findEntry("client.crt"));
      expect(screen.getByText("Plant client")).toBeDefined();
      fireEvent.click(findButton("Select"));
      await expect(result()).resolves.toEqual(
        expect.objectContaining({
          path: "/etc/client.crt",
          certificate: expect.objectContaining({ subject: DETAILS.subject }),
        }),
      );
    });

    it("should pick a certificate on a double click", async () => {
      const { result } = await openBrowser(LISTINGS);
      await openEtc();
      fireEvent.doubleClick(await findEntry("client.crt"));
      await expect(result()).resolves.toEqual(
        expect.objectContaining({ path: "/etc/client.crt" }),
      );
    });

    it("should pick a file that is not a certificate only under All files", async () => {
      const { result } = await openBrowser(LISTINGS);
      await openEtc();
      const hosts = await findEntry("hosts");
      fireEvent.click(hosts);
      fireEvent.doubleClick(hosts);
      await findEntry("hosts", true);
      expect(findButton("Select").getAttribute("aria-disabled")).toBe("true");
      fireEvent.click(screen.getByText("All files"));
      fireEvent.doubleClick(await findEntry("hosts"));
      await expect(result()).resolves.toEqual(
        expect.objectContaining({ path: "/etc/hosts" }),
      );
    });

    it("should open a selected folder and pick a selected file on Enter", async () => {
      const { result } = await openBrowser(LISTINGS);
      await openFolder("/");
      fireEvent.click(await findEntry("etc"));
      await findEntry("etc", true);
      pressEnter();
      fireEvent.click(await findEntry("client.crt"));
      await findEntry("client.crt", true);
      pressEnter();
      await expect(result()).resolves.toEqual(
        expect.objectContaining({ path: "/etc/client.crt" }),
      );
    });

    it("should pick the selected file on the shortcut its Select button shows", async () => {
      const { result } = await openBrowser(LISTINGS);
      await openEtc();
      fireEvent.click(await findEntry("client.crt"));
      within(findButton("Select")).getByText("Ctrl");
      pressSave();
      await expect(result()).resolves.toEqual(
        expect.objectContaining({ path: "/etc/client.crt" }),
      );
    });

    it("should resolve with null on cancel", async () => {
      const { result } = await openBrowser([TOP]);
      await findEntry("secrets");
      fireEvent.click(findButton("Cancel"));
      await expect(result()).resolves.toBeNull();
      await waitFor(() => expect(screen.queryByText("secrets")).toBeNull());
    });
  });

  describe("list", () => {
    it("should list folders first, then files by name", async () => {
      await openBrowser(LISTINGS);
      await openEtc();
      expect(entryNames()).toEqual([
        expect.stringMatching(/^ssl/),
        expect.stringMatching(/^client\.crt/),
        expect.stringMatching(/^hosts/),
      ]);
    });

    it("should reverse the order on a click of the sorted heading", async () => {
      await openBrowser(LISTINGS);
      await openEtc();
      fireEvent.click(screen.getByRole("button", { name: /^Name/ }));
      await waitFor(() =>
        expect(entryNames()).toEqual([
          expect.stringMatching(/^ssl/),
          expect.stringMatching(/^hosts/),
          expect.stringMatching(/^client\.crt/),
        ]),
      );
    });

    it("should count the entries of a folder", async () => {
      await openBrowser(LISTINGS);
      await openEtc();
      expect(screen.getByText("3 items")).toBeDefined();
    });

    it("should say when the Driver stopped before it listed every entry", async () => {
      await openBrowser([{ ...TOP, truncated: true }]);
      expect(await screen.findByText("Shows the first 2 items.")).toBeDefined();
    });

    it("should say when a folder is empty", async () => {
      await openBrowser([
        ...LISTINGS,
        createListing({ path: "/etc/ssl", parent: "/etc" }),
      ]);
      await openEtc();
      await openFolder("ssl");
      expect(await screen.findByText("This folder is empty")).toBeDefined();
    });

    it("should show the error of a folder that fails and go back from it", async () => {
      await openBrowser(LISTINGS);
      await openEtc();
      await openFolder("ssl");
      await screen.findByText("/etc/ssl does not exist.");
      expect(screen.getByText("Failed to list files")).toBeDefined();
      expect(screen.queryByRole("listbox", { name: "Files" })).toBeNull();
      fireEvent.click(screen.getByRole("button", { name: "Go back" }));
      expect(await findEntry("ssl", true)).toBeDefined();
    });
  });
});
