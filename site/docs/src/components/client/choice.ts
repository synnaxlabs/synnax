// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Tabs } from "@/components/tabs";

export const CLIENTS = ["console", "python", "typescript", "arc"] as const;

export type Client = (typeof CLIENTS)[number];

/** Marks an inline value that shows its TypeScript form when TypeScript is chosen. */
export const VAR_ATTRIBUTE = "data-client-var";

const STORAGE_KEY = "docs.client";

const isClient = (value: string | null): value is Client =>
  CLIENTS.some((c) => c === value);

const getStored = (): Client | null => {
  try {
    const value = localStorage.getItem(STORAGE_KEY);
    return isClient(value) ? value : null;
  } catch {
    return null;
  }
};

const store = (client: string): void => {
  try {
    localStorage.setItem(STORAGE_KEY, client);
  } catch {
    // Blocked storage only costs the reader one tab choice on the next page.
  }
};

/**
 * Remembers the chosen client for later pages and shows inline values in its naming
 * style.
 */
export const QUERY: Tabs.Query = {
  initial: getStored,
  onChange: (client) => {
    store(client);
    for (const el of document.querySelectorAll<HTMLElement>(`[${VAR_ATTRIBUTE}]`))
      el.textContent = (client === "typescript" ? el.dataset.ts : el.dataset.py) ?? "";
  },
};
