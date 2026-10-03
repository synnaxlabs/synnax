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

import { Panel } from "@/platform/panel";

const TAB: Panel.Tab = {
  Content: Panel.lazyComponent(
    () => import("@/feature/range/overview/Overview"),
    "Overview",
  ),
  Icon: Icon.Range,
  // Passing the whole namespace would keep all of its code in the startup bundle.
  Name: Panel.createEditableTabName(
    {
      useEnsure: Ranger.useEnsure,
      useName: Ranger.useName,
      useRename: Ranger.useRename,
    },
    <Icon.Range />,
  ),
  restore: async ({ client, resource }) => {
    const corpse = query.requireCorpse(client.ranges.getCached(resource.key));
    await client.ranges.create(corpse.payload);
  },
  // Passing the whole namespace would keep all of its code in the startup bundle.
  useTombstone: Panel.createTombstoneReader({ useTombstone: Ranger.useTombstone }),
};

export const TABS: Panel.Tabs = {
  [ranger.TYPE_ONTOLOGY_ID.type]: TAB,
};
