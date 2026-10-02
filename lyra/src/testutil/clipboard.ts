// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Mock, vi } from "vitest";

/**
 * Installs a spy in place of navigator.clipboard.writeText (an unmockable browser seam
 * with no injection point) and returns it. By default the spy resolves; pass an
 * implementation to simulate a rejection. Call from a beforeEach so each test gets a
 * fresh spy.
 */
export const stubClipboardWriteText = (
  impl: (text: string) => Promise<void> = async () => {},
): Mock => {
  const writeText = vi.fn(impl);
  Object.defineProperty(navigator, "clipboard", {
    value: { writeText },
    configurable: true,
  });
  return writeText;
};

/** Removes navigator.clipboard, as an insecure context or an old engine does. */
export const stubClipboardUnavailable = (): void => {
  Object.defineProperty(navigator, "clipboard", {
    value: undefined,
    configurable: true,
  });
};

/** Removes document.execCommand, as an engine that dropped it would. */
export const stubCopyCommandUnavailable = (): void => {
  Object.defineProperty(document, "execCommand", {
    value: undefined,
    configurable: true,
  });
};

/** Replaces document.execCommand with one that throws, as a locked-down engine does. */
export const stubCopyCommandThrowing = (): void => {
  Object.defineProperty(document, "execCommand", {
    value: (): boolean => {
      throw new Error("blocked");
    },
    configurable: true,
  });
};

/**
 * Replaces document.execCommand, which jsdom does not implement. The returned spy
 * receives the text the browser would have copied, taken from the caller's selection.
 * Pass false for a command the browser refuses.
 */
export const stubCopyCommand = (succeeds: boolean = true): Mock => {
  const copied = vi.fn();
  Object.defineProperty(document, "execCommand", {
    value: (command: string): boolean => {
      if (command !== "copy") return false;
      // The last one, since the caller appends its scratch element to the body.
      const el = Array.from(
        document.querySelectorAll<HTMLTextAreaElement>("textarea"),
      ).at(-1);
      if (el == null) return false;
      copied(el.value.slice(el.selectionStart ?? 0, el.selectionEnd ?? 0));
      return succeeds;
    },
    configurable: true,
  });
  return copied;
};
