// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { query, video } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Video as Base } from "@synnaxlabs/pluto";

import { Selectable } from "@/feature/video/Selectable";
import { FLAGS } from "@/flags";
import { Panel } from "@/platform/panel";
import { type Selector } from "@/platform/selector";

export * from "@/feature/video/commands";
export * from "@/feature/video/link";
export * from "@/feature/video/search";
export * from "@/feature/video/Selectable";
export * from "@/feature/video/tree";
export * from "@/feature/video/useCreate";

const TAB_TYPE = video.TYPE_ONTOLOGY_ID.type;

export const SELECTABLES: Selector.Selectable[] = FLAGS.video ? [Selectable] : [];

const TAB: Panel.Tab = {
  Content: Panel.lazyComponent(() => import("@/feature/video/Video"), "Video"),
  Toolbar: Panel.lazyComponent(() => import("@/feature/video/toolbar"), "Toolbar"),
  Icon: Icon.Video,
  // Only these hooks, so the bundler can drop the rest of the namespace.
  Name: Panel.createEditableTabName(
    {
      useEnsure: Base.useEnsure,
      useName: Base.useName,
      useRename: Base.useRename,
    },
    <Icon.Video />,
  ),
  restore: async ({ client, project, resource }) => {
    const corpse = query.requireCorpse(client.videos.getCached(resource.key));
    await client.videos.create(project, corpse);
  },
  // Only these hooks, so the bundler can drop the rest of the namespace.
  Guard: Panel.createGuard({ useTombstone: Base.useTombstone }),
};

export const TABS: Panel.Tabs = FLAGS.video ? { [TAB_TYPE]: TAB } : {};
