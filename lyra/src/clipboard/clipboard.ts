// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/* The deprecated copy command, and the only copy an insecure (http) context allows.
   Feature-detected, so its removal surfaces as a copy failure, not a crash. */
/* eslint-disable @typescript-eslint/no-deprecated */
const copyByCommand = (text: string): boolean => {
  if (typeof document.execCommand !== "function") return false;
  const el = document.createElement("textarea");
  el.value = text;
  el.style.position = "fixed";
  el.style.opacity = "0";
  document.body.append(el);
  el.select();
  try {
    return document.execCommand("copy");
  } finally {
    el.remove();
  }
};
/* eslint-enable @typescript-eslint/no-deprecated */

/**
 * Writes text to the clipboard. Falls back to the deprecated copy command where the
 * clipboard API is absent or denied, as on an insecure (http) origin.
 * @throws {Error} if neither path could write the text, caused by the API's denial.
 */
export const writeText = async (text: string): Promise<void> => {
  let denial: unknown;
  if (navigator.clipboard != null)
    try {
      await navigator.clipboard.writeText(text);
      return;
    } catch (err) {
      denial = err;
    }
  if (!copyByCommand(text))
    throw new Error("Failed to copy to the clipboard", { cause: denial });
};
