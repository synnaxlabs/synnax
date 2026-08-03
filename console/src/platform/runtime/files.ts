// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { errors } from "@synnaxlabs/x";
import { join, sep } from "@tauri-apps/api/path";
import { open, save } from "@tauri-apps/plugin-dialog";
import {
  exists,
  mkdir,
  readDir,
  readTextFile,
  writeTextFile,
} from "@tauri-apps/plugin-fs";

import { downloadFromBrowser } from "@/platform/runtime/download";
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

const MIME_TYPES: Record<string, string> = {
  json: "application/json",
  svg: "image/svg+xml",
  csv: "text/csv",
  xml: "application/xml",
  txt: "text/plain",
};

const inferMimeType = (filename: string): string => {
  const ext = filename.split(".").pop()?.toLowerCase();
  return (ext != null && MIME_TYPES[ext]) || "application/octet-stream";
};

/**
 * Resolves a settle callback after the user appears to have dismissed a file
 * picker but the `cancel` event never fired (older browsers, sandboxed
 * iframes). Listens for the window regaining focus once, then waits for
 * `change` to fire on its own before falling back to null. Idempotent with
 * the change/cancel listeners — first to settle wins.
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
 * Opens a native file picker and returns the selected files as a uniform
 * { name, read } shape. On Tauri this uses the OS dialog and reads via the
 * filesystem plugin; in the browser it uses an <input type="file"> and reads
 * via File.text(). Returns null if the user cancels or selects nothing.
 */
export const pickFiles = (params: PickFilesParams): Promise<PickedFile[] | null> =>
  Session.Runtime.ENGINE === "tauri"
    ? pickFilesTauri(params)
    : pickFilesBrowser(params);

export interface SaveFileParams {
  title?: string;
  defaultName: string;
  filters?: FileFilter[];
  contents: string;
}

/**
 * Saves a string to disk via a native save dialog on Tauri, or a download to
 * the browser's downloads folder on the web. Returns a human-readable location
 * for status messages (the chosen path on Tauri, the filename on the browser),
 * or null if the user cancels.
 */
export const saveFile = async ({
  title,
  defaultName,
  filters,
  contents,
}: SaveFileParams): Promise<string | null> => {
  if (Session.Runtime.ENGINE === "tauri") {
    const path = await save({ title, defaultPath: defaultName, filters });
    if (path == null) return null;
    await writeTextFile(path, contents);
    return path;
  }
  downloadFromBrowser(
    new Blob([contents], { type: inferMimeType(defaultName) }),
    defaultName,
  );
  return defaultName;
};

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
  const entries = await readDir(dirPath);
  const files: PickedFile[] = [];
  for (const entry of entries) {
    if (!entry.isFile) continue;
    const fullPath = await join(dirPath, entry.name);
    files.push({
      name: entry.name,
      path: entry.name,
      read: () => readTextFile(fullPath),
    });
  }
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
 * Opens a native directory picker and returns the directory's name plus the
 * files inside it. On Tauri this walks the directory's top-level entries via
 * the filesystem plugin; in the browser it uses
 * `<input type="file" webkitdirectory>` which recursively includes every file
 * under the picked folder (each file's `path` is relative to the root).
 * Returns null if the user cancels.
 */
export const pickDirectory = (
  params: PickDirectoryParams = {},
): Promise<PickedDirectory | null> =>
  Session.Runtime.ENGINE === "tauri"
    ? pickDirectoryTauri(params)
    : pickDirectoryBrowser();

export interface WritableDirectory {
  /** A human-readable target path for status messages. */
  displayPath: string;
  /** True if the target directory already had contents before this pick. */
  preExisted: boolean;
  /** Writes a UTF-8 text file at the given top-level filename. */
  writeText: (name: string, contents: string) => Promise<void>;
}

export interface PickWritableDirectoryParams {
  title?: string;
  /**
   * Name of the subdirectory to create under the picked location and use as
   * the write target. The returned handle's writeText writes into this
   * subdirectory.
   */
  subdirectory: string;
}

declare global {
  interface Window {
    showDirectoryPicker?: (options?: {
      mode?: "read" | "readwrite";
    }) => Promise<FileSystemDirectoryHandle>;
  }
}

const pickWritableDirectoryTauri = async ({
  title,
  subdirectory,
}: PickWritableDirectoryParams): Promise<WritableDirectory | null> => {
  const parent = await open({
    title,
    directory: true,
    multiple: false,
    recursive: true,
  });
  if (parent == null || Array.isArray(parent)) return null;
  const dirPath = await join(parent, subdirectory);
  const preExisted = await exists(dirPath);
  let ensured: Promise<void> | null = null;
  const ensureDir = () => (ensured ??= mkdir(dirPath, { recursive: true }));
  return {
    displayPath: dirPath,
    preExisted,
    writeText: async (name, contents) => {
      await ensureDir();
      await writeTextFile(await join(dirPath, name), contents);
    },
  };
};

const pickWritableDirectoryBrowser = async ({
  subdirectory,
}: PickWritableDirectoryParams): Promise<WritableDirectory | null> => {
  if (window.showDirectoryPicker == null)
    throw new Error(
      "This browser does not support writing to a chosen directory. Use Chrome, Edge, or Safari, or run the desktop app.",
    );
  let root: FileSystemDirectoryHandle;
  try {
    root = await window.showDirectoryPicker({ mode: "readwrite" });
  } catch (e) {
    if (e instanceof DOMException && e.name === "AbortError") return null;
    throw errors.fromUnknown(e);
  }
  let subHandle: FileSystemDirectoryHandle | null = null;
  let preExisted: boolean;
  try {
    subHandle = await root.getDirectoryHandle(subdirectory);
    preExisted = true;
  } catch (e) {
    if (!(e instanceof DOMException) || e.name !== "NotFoundError")
      throw errors.fromUnknown(e);
    preExisted = false;
  }
  const ensureSubHandle = async (): Promise<FileSystemDirectoryHandle> =>
    (subHandle ??= await root.getDirectoryHandle(subdirectory, { create: true }));
  return {
    displayPath: `${root.name}/${subdirectory}`,
    preExisted,
    writeText: async (name, contents) => {
      const dir = await ensureSubHandle();
      const fileHandle = await dir.getFileHandle(name, { create: true });
      const writable = await fileHandle.createWritable();
      await writable.write(contents);
      await writable.close();
    },
  };
};

/**
 * Opens a native picker for a directory to write into. On Tauri the user picks
 * a parent directory and {subdirectory} is created inside it. In the browser
 * this uses the File System Access API's showDirectoryPicker (Chrome, Edge,
 * Safari); Firefox does not implement it and throws a clear error. Returns
 * null if the user cancels.
 */
export const pickWritableDirectory = (
  params: PickWritableDirectoryParams,
): Promise<WritableDirectory | null> =>
  Session.Runtime.ENGINE === "tauri"
    ? pickWritableDirectoryTauri(params)
    : pickWritableDirectoryBrowser(params);
