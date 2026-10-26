// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { errors } from "@synnaxlabs/x";

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
 * @throws {Error} the API's denial if neither path could write the text, or a new
 * error if the API is absent and the copy command fails.
 */
export const writeText = async (text: string): Promise<void> => {
  let denial: Error | undefined;
  if (navigator.clipboard != null)
    try {
      await navigator.clipboard.writeText(text);
      return;
    } catch (err) {
      denial = errors.fromUnknown(err);
    }
  if (!copyByCommand(text))
    throw denial ?? new Error("The browser did not allow the copy.");
};
