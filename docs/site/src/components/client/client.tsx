// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

export type Client = "console" | "python" | "typescript" | "cpp";

export interface Info {
  key: Client;
  name: string;
  icon: Icon.ReactElement;
}

export const CLIENTS: Info[] = [
  { key: "console", name: "Console", icon: <Icon.Visualize /> },
  { key: "python", name: "Python", icon: <Icon.Python /> },
  { key: "typescript", name: "TypeScript", icon: <Icon.TypeScript /> },
  { key: "cpp", name: "C++", icon: <Icon.CPlusPlus /> },
];

export const getFromURL = (): Client | null => {
  const url = new URL(window.location.href);
  const client = url.searchParams.get("client");
  return CLIENTS.find((c) => c.key === client)?.key ?? null;
};

export const setInURL = (client: Client) => {
  const url = new URL(window.location.href);
  url.searchParams.set("client", client);
  window.history.pushState({}, "", url.toString());
};
