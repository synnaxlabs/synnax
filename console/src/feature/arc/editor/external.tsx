// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, query } from "@synnaxlabs/client";
import { Arc, Panel as PPanel } from "@synnaxlabs/pluto";

import { Arc as PlatformArc } from "@/platform/arc";
import { Panel } from "@/platform/panel";

// The tab's Name ensures the Arc is cached. A tab whose Arc is loading or deleted
// draws the automation glyph.
const TypeIcon: Panel.TabIcon = (props) => {
  const { key } = PPanel.useTabResource();
  const type = Arc.useResult({ key }).data?.type ?? "automation";
  const Glyph = PlatformArc.TYPE_ICONS[type];
  return <Glyph {...props} />;
};

const TAB: Panel.Tab = {
  Content: Panel.lazyComponent(() => import("@/feature/arc/editor/Editor"), "Editor"),
  Toolbar: Panel.lazyComponent(
    () => import("@/feature/arc/editor/toolbar/Toolbar"),
    "Toolbar",
  ),
  Icon: TypeIcon,
  // Only these hooks, so the bundler can drop the rest of the namespace.
  Name: Panel.createEditableTabName(
    { useEnsure: Arc.useEnsure, useName: Arc.useName, useRename: Arc.useRename },
    <TypeIcon />,
  ),
  restore: async ({ client, resource }) => {
    const corpse = query.requireCorpse(client.arcs.getCached(resource.key));
    await client.arcs.create(corpse);
  },
  // Only these hooks, so the bundler can drop the rest of the namespace.
  Guard: Panel.createGuard({ useTombstone: Arc.useTombstone }),
};

export const TABS: Panel.Tabs = {
  [arc.TYPE_ONTOLOGY_ID.type]: TAB,
};
