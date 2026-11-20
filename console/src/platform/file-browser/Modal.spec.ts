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

const FILES = createListing({
  path: "/",
  entries: [
    {
      name: "plant.crt",
      path: "/plant.crt",
      kind: "file",
      size: 999_999,
      certificate: { ...DETAILS, subject: "O=Synnax Labs\\, Inc.,C=US" },
    },
  ],
});

const HOME = createListing({
  path: "/home/op",
  parent: "/home",
  entries: [{ name: "notes.txt", path: "/home/op/notes.txt", kind: "file" }],
});

const HOME_TOP = createListing({
  secretsDir: SECRETS_DIR,
  entries: [...TOP.entries, { name: "op", path: "/home/op", kind: "directory" }],
});

const dir = (path: string, name: string) =>
  ({ name, path, kind: "directory" }) as const;

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
    it("should show the last listing of a folder at once on reopen", async () => {
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

    it("should list a reopened folder again and keep the selection", async () => {
      const listings = [TOP];
      const { result, reopen, served } = await openBrowser(listings);
      await findEntry("secrets");
      fireEvent.click(findButton("Cancel"));
      await expect(result()).resolves.toBeNull();
      const extra: FileBrowser.Entry = {
        name: "extra",
        path: "/extra",
        kind: "directory",
      };
      listings[0] = { ...TOP, entries: [...TOP.entries, extra] };
      served.hold();
      reopen();
      fireEvent.click(await findEntry("secrets"));
      await findEntry("secrets", true);
      await served.release();
      await findEntry("extra");
      expect(await findEntry("secrets", true)).toBeDefined();
    });

    it("should find the file browser and places once the rack gains it", async () => {
      const r = await createRunningRack(client, "file");
      const { result, reopen } = await renderModalOpener(
        FileBrowser.useModal,
        [{ rack: r.key, path: "/etc/client.crt" }],
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
      await serveListings(client, { rack: r, browser }, LISTINGS);
      reopen();
      expect(await findEntry("hosts")).toBeDefined();
      expect(await screen.findByText("Secrets folder")).toBeDefined();
    });

    it("should list one folder at a time in the background, then the last", async () => {
      const folders = ["a", "b", "c"].map((name) => dir(`/${name}`, name));
      const { served } = await openBrowser([
        TOP,
        createListing({ path: "/", entries: folders }),
        ...folders.map(({ path }) => createListing({ path, parent: "/" })),
      ]);
      await openFolder("/");
      await findEntry("c");
      served.hold();
      for (const name of ["a", "b", "c"]) fireEvent.mouseEnter(await findEntry(name));
      await waitFor(() => expect(served.paths).toContain("/a"));
      await served.release();
      await waitFor(() => expect(served.paths).toContain("/c"));
      expect(served.paths).not.toContain("/b");
    });

    it("should not list a folder that failed again in the background", async () => {
      const etc = createListing({
        ...ETC,
        entries: [...ETC.entries, dir("/etc/pki", "pki")],
      });
      const { served } = await openBrowser([
        TOP,
        ROOT,
        etc,
        createListing({ path: "/etc/pki", parent: "/etc" }),
      ]);
      await openEtc();
      await openFolder("ssl");
      fireEvent.click(await screen.findByRole("button", { name: "Go back" }));
      served.hold();
      fireEvent.mouseEnter(await findEntry("ssl"));
      fireEvent.mouseEnter(await findEntry("pki"));
      await waitFor(() => expect(served.paths).toContain("/etc/pki"));
      await served.release();
      expect(served.paths.filter((p) => p === "/etc/ssl")).toHaveLength(1);
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
      await openBrowser([HOME_TOP, HOME]);
      expect(await findEntry("notes.txt")).toBeDefined();
    });

    it("should not show the top level while the home folder loads", async () => {
      const files = await createFileRack(client);
      const served = await serveListings(client, files, [HOME_TOP, HOME]);
      served.hold();
      await renderModalOpener(FileBrowser.useModal, [{ rack: files.rack.key }], {
        client,
      });
      await waitFor(() => expect(served.paths).toContain(""));
      const released = served.release();
      served.hold();
      await released;
      await waitFor(() => expect(served.paths).toContain("/home/op"));
      expect(screen.getByText("Listing files")).toBeDefined();
      expect(screen.queryByRole("option")).toBeNull();
      await served.release();
      expect(await findEntry("notes.txt")).toBeDefined();
    });

    it("should open the home folder for a file that is not absolute", async () => {
      await openBrowser([HOME_TOP, HOME], "certs/client.crt");
      expect(await findEntry("notes.txt")).toBeDefined();
    });

    it("should select a Windows file given with forward slashes", async () => {
      await openBrowser(
        [
          createListing({ entries: [dir("C:\\", "C:\\")] }),
          createListing({
            path: "C:\\Users\\op",
            entries: [{ name: "a.crt", path: "C:\\Users\\op\\a.crt", kind: "file" }],
          }),
        ],
        "C:/Users/op/./a.crt",
      );
      expect(await findEntry("a.crt", true)).toBeDefined();
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

    it("should keep the selection when the open folder opens again", async () => {
      await openBrowser(LISTINGS);
      await openEtc();
      fireEvent.click(await findEntry("hosts"));
      await findEntry("hosts", true);
      const path = screen.getByLabelText("Path");
      fireEvent.click(within(path).getByRole("button", { name: "etc" }));
      expect(
        screen.getByRole("option", { name: /^hosts/, selected: true }),
      ).toBeDefined();
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

    it("should treat a network share as the root of a UNC path", async () => {
      const share = "\\\\srv\\share";
      await openBrowser(
        [
          createListing({ path: share, entries: [dir(`${share}\\certs`, "certs")] }),
          createListing({
            path: `${share}\\certs`,
            entries: [dir(`${share}\\certs\\sub`, "sub")],
          }),
          createListing({
            path: `${share}\\certs\\sub`,
            entries: [
              { name: "a.crt", path: `${share}\\certs\\sub\\a.crt`, kind: "file" },
            ],
          }),
        ],
        `${share}\\certs\\sub\\a.crt`,
      );
      await findEntry("a.crt", true);
      const path = screen.getByLabelText("Path");
      fireEvent.click(within(path).getByRole("button", { name: "certs" }));
      await findEntry("sub");
      const up = screen.getByRole("button", { name: "Up" });
      fireEvent.click(up);
      await findEntry("certs", true);
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

    it.each([
      ["Alt", { altKey: true }, "ArrowLeft", "ArrowRight"],
      ["Cmd", { metaKey: true }, "[", "]"],
    ])("should go up, back, and forward on %s keys", async (_, mod, back, forward) => {
      await openBrowser(LISTINGS);
      await openEtc();
      fireEvent.keyDown(findFiles(), { key: "ArrowUp", ...mod });
      await findEntry("etc", true);
      fireEvent.keyDown(findFiles(), { key: back, ...mod });
      await findEntry("hosts");
      fireEvent.keyDown(findFiles(), { key: forward, ...mod });
      expect(await findEntry("etc")).toBeDefined();
      expect(screen.queryByRole("option", { name: /^hosts/ })).toBeNull();
    });
  });

  describe("path field", () => {
    const openField = (): HTMLElement => {
      fireEvent.click(screen.getByRole("group", { name: "Path" }));
      return screen.getByRole("textbox", { name: "Path" });
    };

    const submit = (field: HTMLElement, text: string): void => {
      fireEvent.change(field, { target: { value: text } });
      fireEvent.keyDown(field, { key: "Enter" });
    };

    it("should open a typed folder", async () => {
      await openBrowser(LISTINGS);
      await findEntry("secrets");
      submit(openField(), "/etc");
      expect(await findEntry("hosts")).toBeDefined();
    });

    it("should open the folder of a typed file with the file selected", async () => {
      await openBrowser(LISTINGS);
      await findEntry("secrets");
      submit(openField(), "/etc/client.crt");
      expect(await findEntry("client.crt", true)).toBeDefined();
    });

    it("should read a leading ~ as the home folder", async () => {
      const top = createListing({
        secretsDir: SECRETS_DIR,
        entries: [...TOP.entries, { name: "op", path: "/home/op", kind: "directory" }],
      });
      const home = createListing({ path: "/home/op", parent: "/home" });
      const certs = createListing({
        path: "/home/op/certs",
        parent: "/home/op",
        entries: [{ name: "a.crt", path: "/home/op/certs/a.crt", kind: "file" }],
      });
      await openBrowser([top, home, certs]);
      await screen.findByText("This folder is empty");
      submit(openField(), "~/certs");
      expect(await findEntry("a.crt")).toBeDefined();
    });

    it("should suggest folders and complete one on Tab", async () => {
      await openBrowser(LISTINGS);
      await findEntry("secrets");
      const field = openField();
      fireEvent.change(field, { target: { value: "/e" } });
      await screen.findByRole("option", { name: "etc/" });
      fireEvent.keyDown(field, { key: "Tab" });
      expect(field).toHaveProperty("value", "/etc/");
    });

    it("should suggest a hidden folder only after a typed dot", async () => {
      const root = createListing({
        path: "/",
        entries: [
          ...ROOT.entries,
          { name: ".cache", path: "/.cache", kind: "directory" },
        ],
      });
      await openBrowser([TOP, root, ETC]);
      await findEntry("secrets");
      const field = openField();
      fireEvent.change(field, { target: { value: "/" } });
      await screen.findByRole("option", { name: "etc/" });
      expect(screen.queryByRole("option", { name: ".cache/" })).toBeNull();
      fireEvent.change(field, { target: { value: "/." } });
      expect(await screen.findByRole("option", { name: ".cache/" })).toBeDefined();
    });

    it("should close the field on Escape and keep the dialog open", async () => {
      const { result } = await openBrowser(LISTINGS);
      await findEntry("secrets");
      fireEvent.keyDown(openField(), { key: "Escape" });
      expect(screen.queryByRole("textbox", { name: "Path" })).toBeNull();
      fireEvent.click(findButton("Cancel"));
      await expect(result()).resolves.toBeNull();
    });

    it("should open the field with a / typed in the list", async () => {
      await openBrowser(LISTINGS);
      await openEtc();
      fireEvent.keyDown(findFiles(), { key: "/" });
      expect(screen.getByRole("textbox", { name: "Path" })).toHaveProperty(
        "value",
        "/",
      );
    });

    it("should open the field with a / typed in an empty folder", async () => {
      await openBrowser([TOP, createListing({ path: "/" })]);
      await openFolder("/");
      fireEvent.keyDown(await screen.findByText("This folder is empty"), { key: "/" });
      expect(screen.getByRole("textbox", { name: "Path" })).toHaveProperty(
        "value",
        "/",
      );
    });

    it("should open the field with the open folder on Ctrl+L", async () => {
      await openBrowser(LISTINGS);
      await openEtc();
      fireEvent.keyDown(findFiles(), { key: "l", ctrlKey: true });
      expect(screen.getByRole("textbox", { name: "Path" })).toHaveProperty(
        "value",
        "/etc/",
      );
    });

    it("should highlight the last suggestion on ArrowUp and open it on Enter", async () => {
      await openBrowser(LISTINGS);
      await findEntry("secrets");
      const field = openField();
      fireEvent.change(field, { target: { value: "/etc/" } });
      await screen.findByRole("option", { name: "hosts" });
      fireEvent.keyDown(field, { key: "ArrowUp" });
      expect(
        screen.getByRole("option", { name: "hosts", selected: true }),
      ).toBeDefined();
      fireEvent.keyDown(field, { key: "Enter" });
      expect(await findEntry("hosts", true)).toBeDefined();
    });

    it("should complete the prefix that all matches share on Tab", async () => {
      const certs = createListing({
        path: "/certs",
        entries: ["client-a.crt", "client-b.crt"].map((name) => ({
          name,
          path: `/certs/${name}`,
          kind: "file" as const,
        })),
      });
      await openBrowser([...LISTINGS, certs]);
      await findEntry("secrets");
      const field = openField();
      fireEvent.change(field, { target: { value: "/certs/c" } });
      await screen.findByRole("option", { name: "client-a.crt" });
      fireEvent.keyDown(field, { key: "Tab" });
      expect(field).toHaveProperty("value", "/certs/client-");
    });

    it("should open the folder of a file suggestion picked with the mouse", async () => {
      await openBrowser(LISTINGS);
      await findEntry("secrets");
      fireEvent.change(openField(), { target: { value: "/etc/cl" } });
      fireEvent.mouseDown(await screen.findByRole("option", { name: "client.crt" }));
      expect(await findEntry("client.crt", true)).toBeDefined();
    });

    it("should read a typed Windows path in the Driver's form", async () => {
      await openBrowser([
        createListing({ entries: [dir("C:\\", "C:\\")] }),
        createListing({
          path: "C:\\certs",
          entries: [{ name: "a.crt", path: "C:\\certs\\a.crt", kind: "file" }],
        }),
      ]);
      await findEntry("C:");
      submit(openField(), "C:/certs/../certs//a.crt");
      expect(await findEntry("a.crt", true)).toBeDefined();
    });

    it("should not open a typed path that resolves after a later navigation", async () => {
      const { served } = await openBrowser([...LISTINGS, SECRETS]);
      await findEntry("secrets");
      served.hold();
      submit(openField(), "/etc/client.crt");
      expect(await screen.findByRole("progressbar")).toBeDefined();
      fireEvent.click(screen.getByRole("button", { name: "Secrets folder" }));
      await served.release();
      expect(await findEntry("ca.crt")).toBeDefined();
      await waitFor(() => expect(screen.queryByRole("progressbar")).toBeNull());
      expect(screen.queryByRole("option", { name: /^client\.crt/ })).toBeNull();
    });

    it("should open the field with the open folder on Ctrl+Shift+G", async () => {
      await openBrowser(LISTINGS);
      await openEtc();
      fireEvent.keyDown(findFiles(), { key: "G", ctrlKey: true, shiftKey: true });
      expect(screen.getByRole("textbox", { name: "Path" })).toHaveProperty(
        "value",
        "/etc/",
      );
    });

    it("should open a typed Windows file in another case", async () => {
      const file = (name: string): FileBrowser.Entry => ({
        name,
        path: `C:\\certs\\${name}`,
        kind: "file",
      });
      await openBrowser(
        [createListing({ path: "C:\\certs", entries: [file("a.crt"), file("b.crt")] })],
        "C:\\certs\\b.crt",
      );
      await findEntry("b.crt", true);
      fireEvent.keyDown(findFiles(), { key: "l", ctrlKey: true });
      const field = screen.getByRole("textbox", { name: "Path" });
      fireEvent.change(field, { target: { value: "C:\\certs\\A.CRT" } });
      fireEvent.keyDown(field, { key: "Enter" });
      expect(await findEntry("a.crt", true)).toBeDefined();
    });

    it("should open a typed file that came after its folder listed", async () => {
      const listings = [...LISTINGS];
      await openBrowser(listings);
      await openEtc();
      const added: FileBrowser.Entry = {
        name: "new.crt",
        path: "/etc/new.crt",
        kind: "file",
      };
      listings[2] = { ...ETC, entries: [...ETC.entries, added] };
      fireEvent.keyDown(findFiles(), { key: "l", ctrlKey: true });
      const field = screen.getByRole("textbox", { name: "Path" });
      fireEvent.change(field, { target: { value: "/etc/new.crt" } });
      fireEvent.keyDown(field, { key: "Enter" });
      expect(await findEntry("new.crt", true)).toBeDefined();
    });

    it.each([
      ["", {}],
      [" with AltGr", { ctrlKey: true, altKey: true, modifierAltGraph: true }],
    ])("should open the field with a ~ typed in the list%s", async (_, mods) => {
      await openBrowser(LISTINGS);
      await openEtc();
      fireEvent.keyDown(findFiles(), { key: "~", ...mods });
      expect(screen.getByRole("textbox", { name: "Path" })).toHaveProperty(
        "value",
        "~",
      );
    });

    it("should select the entry that starts with letters typed in a row", async () => {
      await openBrowser([
        TOP,
        createListing({ path: "/", entries: [dir("/sa", "sa"), dir("/sb", "sb")] }),
      ]);
      await openFolder("/");
      await findEntry("sb");
      fireEvent.keyDown(findFiles(), { key: "s" });
      await findEntry("sa", true);
      fireEvent.keyDown(findFiles(), { key: "b" });
      expect(await findEntry("sb", true)).toBeDefined();
    });

    it("should select the first entry that starts with a typed letter", async () => {
      await openBrowser(LISTINGS);
      await openEtc();
      fireEvent.keyDown(findFiles(), { key: "h" });
      expect(await findEntry("hosts", true)).toBeDefined();
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

    it("should pick other files only under All files", async () => {
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

    it("should pick any file in the secrets folder under Certificates", async () => {
      const { result } = await openBrowser([...LISTINGS, SECRETS]);
      fireEvent.click(await screen.findByText("Secrets folder"));
      fireEvent.click(await findEntry("ca.crt"));
      await findEntry("ca.crt", true);
      expect(findButton("Select").getAttribute("aria-disabled")).not.toBe("true");
      fireEvent.click(findButton("Select"));
      await expect(result()).resolves.toEqual(
        expect.objectContaining({ path: `${SECRETS_DIR}/ca.crt` }),
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

    it("should pick the selected file on the Select shortcut", async () => {
      const { result } = await openBrowser(LISTINGS);
      await openEtc();
      fireEvent.click(await findEntry("client.crt"));
      within(findButton("Select")).getByText("Ctrl");
      pressSave();
      await expect(result()).resolves.toEqual(
        expect.objectContaining({ path: "/etc/client.crt" }),
      );
    });

    it("should not pick a file of the last folder while the next one loads", async () => {
      const { served } = await openBrowser(LISTINGS);
      await openEtc();
      served.hold();
      await openFolder("ssl");
      fireEvent.click(await findEntry("client.crt"));
      await findEntry("client.crt", true);
      expect(findButton("Select").getAttribute("aria-disabled")).toBe("true");
      await served.release();
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

    it.each(["Size", "Modified", "Expires"])(
      "should sort by %s, with ties in the order of their names",
      async (heading) => {
        const start = TimeStamp.now();
        const file = (name: string, value: number): FileBrowser.Entry => ({
          name,
          path: `/${name}`,
          kind: "file",
          size: value,
          modified: start.add(TimeSpan.days(value)),
          certificate: {
            ...DETAILS,
            notAfter: start.add(TimeSpan.days(value)),
          },
        });
        await openBrowser([
          TOP,
          createListing({
            path: "/",
            entries: [file("a", 3), file("c", 1), file("b", 1)],
          }),
        ]);
        await openFolder("/");
        await findEntry("c");
        fireEvent.click(
          screen.getByRole("button", { name: new RegExp(`^${heading}`) }),
        );
        await waitFor(() =>
          expect(entryNames()).toEqual([
            expect.stringMatching(/^b/),
            expect.stringMatching(/^c/),
            expect.stringMatching(/^a/),
          ]),
        );
      },
    );

    it("should say that it lists files before the first listing comes", async () => {
      const files = await createFileRack(client);
      await renderModalOpener(FileBrowser.useModal, [{ rack: files.rack.key }], {
        client,
      });
      expect(await screen.findByText("Listing files")).toBeDefined();
    });

    it("should keep the last folder in view while the next one loads", async () => {
      const { served } = await openBrowser(LISTINGS);
      await openFolder("/");
      const etc = await findEntry("etc");
      served.stop();
      fireEvent.doubleClick(etc);
      const path = screen.getByLabelText("Path");
      await within(path).findByRole("button", { name: "etc" });
      expect(await findEntry("etc")).toBeDefined();
    });

    it.each([
      ["Expired", TimeSpan.days(-30), TimeSpan.days(-1)],
      ["Not yet valid", TimeSpan.days(1), TimeSpan.days(365)],
    ])("should mark a certificate as %s", async (label, notBefore, notAfter) => {
      const now = TimeStamp.now();
      const certificate = {
        ...DETAILS,
        notBefore: now.add(notBefore),
        notAfter: now.add(notAfter),
      };
      await openBrowser([
        TOP,
        createListing({
          path: "/",
          entries: [{ name: "a.crt", path: "/a.crt", kind: "file", certificate }],
        }),
      ]);
      await openFolder("/");
      expect(await within(await findEntry("a.crt")).findByText(label)).toBeDefined();
    });

    it("should name a certificate by a subject with an escaped comma", async () => {
      await openBrowser([TOP, FILES]);
      await openFolder("/");
      expect(await screen.findByText("Synnax Labs, Inc.")).toBeDefined();
    });

    it("should round a size up to the next unit", async () => {
      await openBrowser([TOP, FILES]);
      await openFolder("/");
      expect(await screen.findByText("1.0 MB")).toBeDefined();
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

    it("should list a folder that failed again on Try again", async () => {
      const listings = [TOP, ROOT];
      await openBrowser(listings, "/etc/client.crt");
      await screen.findByText("/etc does not exist.");
      expect(screen.queryByRole("button", { name: "Go back" })).toBeNull();
      listings.push(ETC);
      fireEvent.click(screen.getByRole("button", { name: "Try again" }));
      expect(await findEntry("client.crt", true)).toBeDefined();
    });

    it("should list the places that failed again on Try again", async () => {
      const listings = [ROOT];
      await openBrowser(listings, "/etc/client.crt");
      await screen.findByText("/etc does not exist.");
      listings.push(TOP, ETC);
      fireEvent.click(screen.getByRole("button", { name: "Try again" }));
      expect(await screen.findByText("Secrets folder")).toBeDefined();
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
