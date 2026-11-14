// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { query, ranger } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Ranger } from "@synnaxlabs/pluto";

import { FLAGS } from "@/flags";
import { Panel } from "@/platform/panel";

const TAB: Panel.Tab = {
  Content: FLAGS.ranges
    ? Panel.lazyComponent(
        () => import("@/feature/range/next/overview/Overview"),
        "Overview",
      )
    : Panel.lazyComponent(
        () => import("@/feature/range/overview/Overview"),
        "Overview",
      ),
  Icon: Icon.Range,
  // Only these hooks, so the bundler can drop the rest of the namespace.
  Name: Panel.createEditableTabName(
    {
      useEnsure: Ranger.useEnsure,
      useName: Ranger.useName,
      useRename: Ranger.useRename,
    },
    <Icon.Range />,
  ),
  restore: async ({ client, resource }) => {
    const corpse = query.requireCorpse(client.ranges.watch(resource.key).getCached());
    await client.ranges.create(corpse.payload);
  },
  // Only these hooks, so the bundler can drop the rest of the namespace.
  Guard: Panel.createGuard({ useTombstone: Ranger.useTombstone }),
};

export const TABS: Panel.Tabs = {
  [ranger.TYPE_RESOURCE_ID.type]: TAB,
};
