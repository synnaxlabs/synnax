// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

export type ArcContext = "Flow" | "Function";

export interface Info {
  key: ArcContext;
  name: string;
  icon: Icon.ReactElement;
}

export const TABS: Info[] = [
  { key: "Flow", name: "Flow", icon: <Icon.ArcFlow /> },
  { key: "Function", name: "Func", icon: <Icon.ArcFunc /> },
];

export const getFromURL = (): ArcContext | null => {
  const url = new URL(window.location.href);
  const context = url.searchParams.get("context");
  return TABS.find((c) => c.key === context)?.key ?? null;
};

export const setInURL = (context: ArcContext) => {
  const url = new URL(window.location.href);
  url.searchParams.set("context", context);
  window.history.pushState({}, "", url.toString());
  window.dispatchEvent(new CustomEvent("urlchange"));
};
