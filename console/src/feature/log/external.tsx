// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log, query } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Log as Base } from "@synnaxlabs/pluto";

import { Selectable } from "@/feature/log/Selectable";
import { Panel } from "@/platform/panel";
import { type Selector } from "@/platform/selector";

export * from "@/feature/log/commands";
export * from "@/feature/log/link";
export * from "@/feature/log/search";
export * from "@/feature/log/Selectable";
export * from "@/feature/log/tree";
export * from "@/platform/log/external";

const TAB_TYPE = log.TYPE_ONTOLOGY_ID.type;

export const SELECTABLES: Selector.Selectable[] = [Selectable];

const TAB: Panel.Tab = {
  Content: Panel.lazyComponent(() => import("@/feature/log/Log"), "Log"),
  Toolbar: Panel.lazyComponent(() => import("@/feature/log/toolbar"), "Toolbar"),
  Icon: Icon.Log,
  Name: Panel.createEditableTabName(
    { useEnsure: Base.useEnsure, useName: Base.useName, useRename: Base.useRename },
    <Icon.Log />,
  ),
  restore: async ({ client, project, resource }) => {
    const corpse = query.requireCorpse(client.logs.getCached(resource.key));
    await client.logs.create(project, corpse);
  },
  useTombstone: Panel.createTombstoneReader({ useTombstone: Base.useTombstone }),
};

export const TABS: Panel.Tabs = {
  [TAB_TYPE]: TAB,
};
