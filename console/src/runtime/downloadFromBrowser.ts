// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Status } from "@synnaxlabs/pluto";
import { save } from "@tauri-apps/plugin-dialog";
import { writeFile } from "@tauri-apps/plugin-fs";

import { ENGINE } from "@/runtime/runtime";

export const downloadFromBrowser = (data: Blob, fileName: string) => {
  const link = document.createElement("a");
  link.href = URL.createObjectURL(data);
  link.download = fileName;
  link.click();
  URL.revokeObjectURL(link.href);
  link.remove();
};

interface WindowWithShowSaveFilePicker extends Window {
  showSaveFilePicker: (options: {
    suggestedName?: string;
  }) => Promise<FileSystemFileHandle>;
}

export interface DownloadStreamParams {
  stream: ReadableStream<Uint8Array>;
  name: string;
  extension: string;
  addStatus: Status.Adder;
  onDownloadStart?: () => void;
}

/**
 * Downloads a stream to the file system. The function will try to use the most
 * performant method to download the stream.
 * @param stream - The stream to download.
 * @param name - The name of the file to download.
 * @param extension - The extension of the file to download.
 * @param addStatus - The function to add a status message.
 */
export const downloadStream = async ({
  stream,
  name,
  extension,
  onDownloadStart,
  addStatus,
}: DownloadStreamParams): Promise<void> => {
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
  // Case 1: we can use the browser's file stream download, which is the most performant
  // and preferred method.
  if (
    "showSaveFilePicker" in window &&
    typeof (window as WindowWithShowSaveFilePicker).showSaveFilePicker === "function"
  )
    try {
      const fileHandle = await (
        window as WindowWithShowSaveFilePicker
      ).showSaveFilePicker({ suggestedName: name });
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
      throw error;
    }
  // Case 2: we use Tauri's stream writer, where are least we don't have to load
  // everything into memory.
  if (ENGINE === "tauri") {
    const savePath = await save({
      title: `Download ${name}`,
      defaultPath: `${name}.${extension}`,
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
  downloadFromBrowser(blob, name);
  addFinishStatus("Downloads");
};
