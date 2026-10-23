// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export const preventDefault = (e: { preventDefault: () => void }): void =>
  e.preventDefault();

export const stopPropagation = (e: { stopPropagation: () => void }): void =>
  e.stopPropagation();

/** The keys whose keydown or keyup the browser turns into a click. */
export const ACTIVATION_KEYS = [" ", "Enter"];

/**
 * Cancels the synthetic click for Space and Enter. Attach to both keydown and keyup:
 * Enter clicks on keydown, Space on keyup.
 */
export const blockActivation = (e: {
  key: string;
  preventDefault: () => void;
}): void => {
  if (ACTIVATION_KEYS.includes(e.key)) e.preventDefault();
};

/** True when the target accepts text entry, so the keystroke belongs to it. */
export const isInputOrContentEditable = (e: {
  target: EventTarget | null;
}): boolean => {
  if (e.target instanceof HTMLInputElement || e.target instanceof HTMLTextAreaElement)
    return true;
  if (!(e.target instanceof HTMLElement)) return false;
  return (
    e.target.getAttribute("contenteditable") === "true" || e.target.role === "textbox"
  );
};
