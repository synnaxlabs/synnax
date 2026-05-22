// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { open } from "@tauri-apps/plugin-shell";
import { useEffect } from "react";

import { ENGINE } from "@/runtime/runtime";

const handle = (e: MouseEvent) => {
  if (!(e.target instanceof Element)) return;
  const a = e.target.closest("a");
  if (a == null || a.target !== "_blank" || a.href === "") return;
  e.preventDefault();
  e.stopPropagation();
  open(a.href).catch((err) => console.error(`failed to open ${a.href}`, err));
};

/**
 * Routes `<a target="_blank">` clicks to the system browser when running in Tauri.
 * Tauri's WebView silently drops `target="_blank"` clicks by default; this hook
 * intercepts those and hands the URL to the shell plugin's `open`. In the browser build
 * this is a no-op — native `target="_blank"` already opens a new tab.
 */
export const useExternalLinkHandler = (): void => {
  useEffect(() => {
    if (ENGINE !== "tauri") return;
    document.addEventListener("click", handle, true);
    document.addEventListener("auxclick", handle, true);
    return () => {
      document.removeEventListener("click", handle, true);
      document.removeEventListener("auxclick", handle, true);
    };
  }, []);
};
