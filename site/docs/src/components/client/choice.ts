// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Query } from "@/components/tabs/sync";

export const CLIENTS = ["console", "python", "typescript", "cpp"] as const;

export type Client = (typeof CLIENTS)[number];

/** Marks an inline value that shows its TypeScript form when TypeScript is chosen. */
export const VAR_ATTRIBUTE = "data-client-var";

/** Shows inline values in the naming style of the chosen client. */
export const QUERY: Query = {
  onChange: (client) => {
    for (const el of document.querySelectorAll<HTMLElement>(`[${VAR_ATTRIBUTE}]`))
      el.textContent = (client === "typescript" ? el.dataset.ts : el.dataset.py) ?? "";
  },
};
