// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Nav as PNav, OS } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Nav as PlatformNav } from "@/platform/nav";
import { Version } from "@/platform/version";
import { Window } from "@/platform/window";
import { Session } from "@/session";

/**
 * Window chrome for pre-workspace surfaces. Decides its own visibility: web
 * builds have no window chrome, and child windows never show it.
 */
export const Nav = (): ReactElement | null => {
  const os = OS.use();
  if (Session.Runtime.ENGINE === "web" || !Session.Runtime.isMainWindow()) return null;
  return (
    <PlatformNav.Bar location="top" size="7rem" bordered data-tauri-drag-region>
      <PNav.Bar.Start data-tauri-drag-region>
        <Window.Controls visibleIfOS="macOS" forceOS={os} />
      </PNav.Bar.Start>
      <PNav.Bar.End data-tauri-drag-region justify="end">
        <Version.Badge />
        <Window.Controls visibleIfOS="Windows" forceOS={os} />
      </PNav.Bar.End>
    </PlatformNav.Bar>
  );
};
