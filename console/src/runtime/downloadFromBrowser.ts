// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export const downloadFromBrowser = (data: Blob, fileName: string) => {
  const link = document.createElement("a");
  link.href = URL.createObjectURL(data);
  link.download = fileName;
  link.click();
  URL.revokeObjectURL(link.href);
  link.remove();
};

/**
 * Downloads a stream to the browser.
 * @param stream - The stream to download.
 * @param fileName - The name of the file to download.
 * @returns The name of the file that was downloaded.
 */
export const downloadStreamFromBrowser = async (
  stream: ReadableStream<Uint8Array>,
  fileName: string,
): Promise<string> => {
  // Hacky thing to check if the browser supports showSaveFilePicker, in which case we
  // will use that stream to the file system.
  // https://developer.mozilla.org/en-US/docs/Web/API/Window/showSaveFilePicker#browser_compatibility
  if (
    "showSaveFilePicker" in window &&
    typeof (window as Window & { showSaveFilePicker?: unknown }).showSaveFilePicker ===
      "function"
  ) {
    const fileHandle: FileSystemFileHandle = await (
      window as unknown as Window & { showSaveFilePicker?: any }
    ).showSaveFilePicker({ suggestedName: fileName });
    const writable = await fileHandle.createWritable();
    try {
      await stream.pipeTo(writable);
    } catch (error) {
      await writable.abort();
      throw error;
    }
    return fileHandle.name;
  }
  const blob = await new Response(stream).blob();
  downloadFromBrowser(blob, fileName);
  return "Downloads";
};
