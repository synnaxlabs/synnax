// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { vi } from "vitest";

/**
 * fakePickedFile builds a File for driving the browser picker boundary. Under Node it
 * is built from node:buffer so undici's fetch accepts it as a request body — jsdom's
 * File is another realm's Blob and would be stringified. A browser has no `process`
 * and uses its own File.
 */
export const fakePickedFile = (
  name: string,
  contents: string | Uint8Array<ArrayBuffer> = "content",
  webkitRelativePath?: string,
): File => {
  const file =
    typeof process === "undefined"
      ? new File([contents], name)
      : (new (process.getBuiltinModule("node:buffer").File)(
          [contents],
          name,
        ) as unknown as File);
  if (webkitRelativePath != null)
    Object.defineProperty(file, "webkitRelativePath", { value: webkitRelativePath });
  return file;
};

/**
 * FilePickerInterceptor drives the browser file-picker boundary from a test: the
 * production <input type="file"> element is real, only its click into the native dialog
 * is captured.
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
 * interceptFilePicker captures <input type="file"> clicks so tests can drive the real
 * browser pickFiles/pickDirectory code paths without a native dialog. The returned
 * interceptor selects files or cancels on the captured input. Installs a spy on
 * HTMLInputElement.prototype.click; callers restore it via vi.restoreAllMocks() in
 * afterEach.
 */
export const interceptFilePicker = (): FilePickerInterceptor => {
  // A file input's click would open the browser's own picker, which a headless browser
  // cancels at once. Other inputs click as usual.
  const spy = vi
    .spyOn(HTMLInputElement.prototype, "click")
    .mockImplementation(function (this: HTMLInputElement) {
      if (this.type !== "file") HTMLElement.prototype.click.call(this);
    });
  const lastInput = (): HTMLInputElement => {
    const input = spy.mock.contexts.findLast(
      (ctx): ctx is HTMLInputElement =>
        ctx instanceof HTMLInputElement && ctx.type === "file",
    );
    if (input == null) throw new Error("no file picker was opened");
    return input;
  };
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
