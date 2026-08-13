// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/pluto";
import { errors, filename } from "@synnaxlabs/x";
import { save } from "@tauri-apps/plugin-dialog";
import { writeFile } from "@tauri-apps/plugin-fs";
import { useCallback } from "react";

import { type FileFilter } from "@/platform/runtime/files";
import { Session } from "@/session";

interface WindowWithShowSaveFilePicker extends Window {
  showSaveFilePicker: (options: {
    suggestedName?: string;
  }) => Promise<FileSystemFileHandle>;
}

export interface DownloadParams {
  /** The stream to download. */
  stream: ReadableStream<Uint8Array>;
  /**
   * The resource name. It titles the save dialog and status messages; its sanitized
   * form names the file.
   */
  name: string;
  /** The extension of the file to download, without the leading dot. */
  extension: string;
  /** File-type filters for the Tauri save dialog. */
  filters?: FileFilter[];
  onDownloadStart?: () => void;
}

/**
 * Returns a callback that downloads a stream to the file system, using the most
 * performant method the runtime offers and reporting progress through statuses.
 */
export const useDownload = (): ((params: DownloadParams) => Promise<void>) => {
  const addStatus = Status.useAdder();
  return useCallback(
    async ({
      stream,
      name,
      extension,
      filters,
      onDownloadStart,
    }: DownloadParams): Promise<void> => {
      const nameWithExtension = filename.sanitize(name, `.${extension}`);
      const addStartStatus = (location: string) => {
        onDownloadStart?.();
        addStatus({
          variant: "info",
          message: `Downloading ${name} to ${location}`,
        });
      };
      const addFinishStatus = (location: string) =>
        addStatus({
          variant: "success",
          message: `Downloaded ${name} to ${location}`,
        });
      // Case 1: we can use the browser's file stream download, which is the most
      // performant and preferred method.
      if (
        "showSaveFilePicker" in window &&
        typeof (window as WindowWithShowSaveFilePicker).showSaveFilePicker ===
          "function"
      )
        try {
          const fileHandle = await (
            window as WindowWithShowSaveFilePicker
          ).showSaveFilePicker({ suggestedName: nameWithExtension });
          const writable = await fileHandle.createWritable();
          addStartStatus(fileHandle.name);
          await stream.pipeTo(writable);
          addFinishStatus(fileHandle.name);
          return;
        } catch (error) {
          if (error instanceof DOMException && error.name === "AbortError") {
            await stream.cancel();
            return;
          }
          throw errors.fromUnknown(error);
        }
      // Case 2: we use Tauri's stream writer, where at least we don't have to load
      // everything into memory.
      if (Session.Runtime.ENGINE === "tauri") {
        const savePath = await save({
          title: `Download ${name}`,
          defaultPath: nameWithExtension,
          filters,
        });
        if (savePath == null) {
          await stream.cancel();
          return;
        }
        addStartStatus(savePath);
        await writeFile(savePath, stream);
        addFinishStatus(savePath);
        return;
      }
      // Case 3: we load everything into memory and download it
      addStartStatus("Downloads");
      const blob = await new Response(stream).blob();
      const link = document.createElement("a");
      link.href = URL.createObjectURL(blob);
      link.download = nameWithExtension;
      link.click();
      URL.revokeObjectURL(link.href);
      link.remove();
      addFinishStatus("Downloads");
    },
    [addStatus],
  );
};
