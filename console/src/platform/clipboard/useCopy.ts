// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/pluto";
import { useCallback } from "react";

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

/* navigator.clipboard is permission-gated, and absent altogether in an insecure
   context. Tauri's clipboard plugin is the non-deprecated fallback, but it covers
   the desktop app alone, so it would add an engine branch and an install rather
   than replace this. */
const writeText = async (text: string): Promise<void> => {
  if (navigator.clipboard != null)
    try {
      await navigator.clipboard.writeText(text);
      return;
    } catch {
      // A denied write is expected; the command below is the fallback.
    }
  if (!copyByCommand(text)) throw new Error("clipboard write rejected");
};

export const useCopy = (): ((text: string, name: string) => void) => {
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  return useCallback(
    (text: string, name: string) => {
      handleError(async () => {
        await writeText(text);
        addStatus({ variant: "success", message: `Copied ${name} to clipboard` });
      }, `Failed to copy ${name} to clipboard`);
    },
    [addStatus, handleError],
  );
};
