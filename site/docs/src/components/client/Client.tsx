// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { type Edition } from "@/components/edition/edition";

export type Client = "console" | "python" | "typescript" | "cpp";

export interface Info {
  key: Client;
  name: string;
  icon: Icon.ReactElement;
  edition?: Edition;
}

export const CLIENTS: Info[] = [
  { key: "console", name: "Console", icon: <Icon.Visualize /> },
  { key: "python", name: "Python", icon: <Icon.Python />, edition: "enterprise" },
  {
    key: "typescript",
    name: "TypeScript",
    icon: <Icon.TypeScript />,
    edition: "enterprise",
  },
  { key: "cpp", name: "C++", icon: <Icon.CPlusPlus />, edition: "enterprise" },
];

const find = (key: string | null): Client | null =>
  CLIENTS.find((c) => c.key === key)?.key ?? null;

export const getFromURL = (): Client | null =>
  find(new URL(window.location.href).searchParams.get("client"));

const STORAGE_KEY = "docs.client";

// Storage is a convenience: a private window or blocked site data leaves it unusable,
// and the Console tab is then the right default.
export const getStored = (): Client | null => {
  try {
    return find(localStorage.getItem(STORAGE_KEY));
  } catch {
    return null;
  }
};

export const store = (client: Client): void => {
  try {
    localStorage.setItem(STORAGE_KEY, client);
  } catch {
    // Blocked storage only costs the reader one tab choice on the next page.
  }
};

/**
 * Selects client in every synced block on the page and remembers it for later pages.
 * replace keeps the change out of the browser history.
 */
export const setInURL = (client: Client, replace = false) => {
  store(client);
  const url = new URL(window.location.href);
  url.searchParams.set("client", client);
  if (replace) window.history.replaceState({}, "", url.toString());
  else window.history.pushState({}, "", url.toString());
  window.dispatchEvent(new CustomEvent("urlchange"));
};
