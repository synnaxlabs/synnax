// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Project } from "@synnaxlabs/pluto";
import { useEffect } from "react";

import { Session } from "@/session";

/**
 * Puts the name of the selected project in front of the browser tab title. Does nothing
 * on desktop, where the window keeps its own title.
 */
export const useBrowserTabTitle = (): void => {
  const key = Session.Project.useSelectSelected();
  // Browser only, so a user can find the right project among many open tabs. The
  // desktop runs one app at a time, so it has no other title to tell apart.
  const web = Session.Runtime.ENGINE === "web";
  const name = Project.useResult(web ? { key } : null).data?.name;
  useEffect(() => {
    if (name == null) return;
    const base = document.title;
    document.title = `${name} - ${base}`;
    return () => {
      document.title = base;
    };
  }, [name]);
};
