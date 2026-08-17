// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { join, sep } from "@tauri-apps/api/path";
import { open } from "@tauri-apps/plugin-dialog";
import { readDir, readTextFile } from "@tauri-apps/plugin-fs";

import { Session } from "@/session";

export interface FileFilter {
  name: string;
  extensions: string[];
}

export interface PickedFile {
  /** File's basename, e.g., "manifest.json". */
  name: string;
  /**
   * Path relative to the picked root, e.g., "manifest.json" for top-level or
   * "sub/foo.json" for nested. For pickFiles this always equals name.
   */
  path: string;
  read: () => Promise<string>;
}

export interface PickFilesParams {
  title?: string;
  filters?: FileFilter[];
  multiple?: boolean;
}

const browserAccept = (filters?: FileFilter[]): string | undefined => {
  if (filters == null || filters.length === 0) return undefined;
  return filters
    .flatMap(({ extensions }) => extensions.map((ext) => `.${ext}`))
    .join(",");
};

/**
 * Resolves a settle callback after the user appears to have dismissed a file picker but
 * the `cancel` event never fired (older browsers, sandboxed iframes). Listens for the
 * window regaining focus once, then waits for `change` to fire on its own before
 * falling back to null. Idempotent with the change/cancel listeners — first to settle
 * wins.
 */
const settleOnFocusReturn = (settle: () => void): void => {
  const handler = () => setTimeout(settle, 500);
  window.addEventListener("focus", handler, { once: true });
};

const pickFilesTauri = async ({
  title,
  filters,
  multiple,
}: PickFilesParams): Promise<PickedFile[] | null> => {
  const result = await open({
    title,
    filters,
    multiple: multiple ?? false,
    directory: false,
  });
  if (result == null) return null;
  const paths = Array.isArray(result) ? result : [result];
  if (paths.length === 0) return null;
  const separator = sep();
  return paths.map((path) => {
    const name = path.split(separator).pop() ?? path;
    return { name, path: name, read: () => readTextFile(path) };
  });
};

const pickFilesBrowser = ({
  filters,
  multiple,
}: PickFilesParams): Promise<PickedFile[] | null> =>
  new Promise((resolve) => {
    const input = document.createElement("input");
    input.type = "file";
    if (multiple) input.multiple = true;
    const accept = browserAccept(filters);
    if (accept != null) input.accept = accept;
    let settled = false;
    const settle = (value: PickedFile[] | null) => {
      if (settled) return;
      settled = true;
      resolve(value);
    };
    input.addEventListener("change", () => {
      const files = input.files;
      if (files == null || files.length === 0) return settle(null);
      settle(
        Array.from(files).map((file) => ({
          name: file.name,
          path: file.name,
          read: () => file.text(),
        })),
      );
    });
    input.addEventListener("cancel", () => settle(null));
    settleOnFocusReturn(() => settle(null));
    input.click();
  });

/**
 * Opens a native file picker and returns the selected files as a uniform { name, read }
 * shape. On Tauri this uses the OS dialog and reads via the filesystem plugin; in the
 * browser it uses an <input type="file"> and reads via File.text(). Returns null if the
 * user cancels or selects nothing.
 */
export const pickFiles = (params: PickFilesParams): Promise<PickedFile[] | null> =>
  Session.Runtime.ENGINE === "tauri"
    ? pickFilesTauri(params)
    : pickFilesBrowser(params);

export interface PickedDirectory {
  /** The picked directory's basename. */
  name: string;
  /** Files contained in the directory, with paths relative to it. */
  files: PickedFile[];
}

export interface PickDirectoryParams {
  title?: string;
}

const pickDirectoryTauri = async ({
  title,
}: PickDirectoryParams): Promise<PickedDirectory | null> => {
  const result = await open({ title, directory: true, multiple: false });
  if (result == null || Array.isArray(result)) return null;
  const dirPath = result;
  const separator = sep();
  const name = dirPath.split(separator).pop() ?? dirPath;
  const files: PickedFile[] = [];
  const walk = async (absolute: string, relative: string): Promise<void> => {
    for (const entry of await readDir(absolute)) {
      const fullPath = await join(absolute, entry.name);
      const relPath = relative === "" ? entry.name : `${relative}/${entry.name}`;
      if (entry.isDirectory) await walk(fullPath, relPath);
      else if (entry.isFile)
        files.push({
          name: entry.name,
          path: relPath,
          read: () => readTextFile(fullPath),
        });
    }
  };
  await walk(dirPath, "");
  return { name, files };
};

const pickDirectoryBrowser = (): Promise<PickedDirectory | null> =>
  new Promise((resolve) => {
    const input = document.createElement("input");
    input.type = "file";
    input.webkitdirectory = true;
    let settled = false;
    const settle = (value: PickedDirectory | null) => {
      if (settled) return;
      settled = true;
      resolve(value);
    };
    input.addEventListener("change", () => {
      const files = input.files;
      if (files == null || files.length === 0) return settle(null);
      const arr = Array.from(files);
      const rootName = arr[0].webkitRelativePath.split("/")[0];
      settle({
        name: rootName,
        files: arr.map((file) => {
          const rel = file.webkitRelativePath.startsWith(`${rootName}/`)
            ? file.webkitRelativePath.slice(rootName.length + 1)
            : file.webkitRelativePath;
          return { name: file.name, path: rel, read: () => file.text() };
        }),
      });
    });
    input.addEventListener("cancel", () => settle(null));
    settleOnFocusReturn(() => settle(null));
    input.click();
  });

/**
 * Opens a native directory picker and returns the directory's name plus every file
 * under it, recursively. On Tauri this walks the directory via the filesystem plugin;
 * in the browser it uses `<input type="file" webkitdirectory>`. Each file's `path` is
 * relative to the picked root in forward-slash form. Returns null if the user cancels.
 */
export const pickDirectory = (
  params: PickDirectoryParams = {},
): Promise<PickedDirectory | null> =>
  Session.Runtime.ENGINE === "tauri"
    ? pickDirectoryTauri(params)
    : pickDirectoryBrowser();
