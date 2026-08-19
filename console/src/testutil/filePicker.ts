// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { File as NodeFile } from "node:buffer";

import { vi } from "vitest";

/**
 * fakePickedFile builds a File for driving the browser picker boundary. It is built
 * from node:buffer so undici's fetch accepts it as a request body — jsdom's File is
 * another realm's Blob and would be stringified.
 */
export const fakePickedFile = (
  name: string,
  contents: string | Uint8Array<ArrayBuffer> = "content",
  webkitRelativePath?: string,
): File => {
  const file = new NodeFile([contents], name) as unknown as File;
  if (webkitRelativePath != null)
    Object.defineProperty(file, "webkitRelativePath", { value: webkitRelativePath });
  return file;
};

/**
 * FilePickerInterceptor drives the browser file-picker boundary from a test: the
 * production <input type="file"> element is real, only its click into the native
 * dialog is captured.
 */
export interface FilePickerInterceptor {
  /** lastInput returns the most recently opened picker input. */
  lastInput: () => HTMLInputElement;
  /** selectFiles resolves the pending pick with the given files. */
  selectFiles: (files: File[]) => void;
  /** cancel dismisses the pending pick. */
  cancel: () => void;
}

/**
 * interceptFilePicker captures <input type="file"> clicks so tests can drive the
 * real browser pickFiles/pickDirectory code paths without a native dialog. The
 * returned interceptor selects files or cancels on the captured input. Installs a
 * spy on HTMLElement.prototype.click; callers restore it via vi.restoreAllMocks()
 * in afterEach.
 */
export const interceptFilePicker = (): FilePickerInterceptor => {
  const inputs: HTMLInputElement[] = [];
  vi.spyOn(HTMLElement.prototype, "click").mockImplementation(function (
    this: HTMLElement,
  ) {
    if (this instanceof HTMLInputElement) inputs.push(this);
  });
  const lastInput = (): HTMLInputElement => inputs[inputs.length - 1];
  return {
    lastInput,
    selectFiles: (files) => {
      const input = lastInput();
      Object.defineProperty(input, "files", { configurable: true, value: files });
      input.dispatchEvent(new Event("change"));
    },
    cancel: () => lastInput().dispatchEvent(new Event("cancel")),
  };
};
