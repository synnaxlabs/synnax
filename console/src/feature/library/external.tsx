// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { library, query } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Library } from "@synnaxlabs/pluto";

import { Panel } from "@/platform/panel";

export * from "@/feature/library/commands";
export * from "@/feature/library/link";
export * from "@/feature/library/search";
export * from "@/feature/library/Toolbar";
export * from "@/feature/library/tree";
export * from "@/feature/library/useCreate";
export * from "@/feature/library/useImport";

const TAB: Panel.Tab = {
  Content: Panel.lazyComponent(
    () => import("@/feature/library/editor/Editor"),
    "Editor",
  ),
  Icon: Icon.Library,
  // Only these hooks, so the bundler can drop the rest of the namespace.
  Name: Panel.createEditableTabName(
    {
      useEnsure: Library.useEnsure,
      useName: Library.useName,
      useRename: Library.useRename,
    },
    <Icon.Library />,
  ),
  restore: async ({ client, resource }) => {
    await client.libraries.create(
      query.requireCorpse(client.libraries.getCached({ key: resource.key })),
    );
  },
  // Only these hooks, so the bundler can drop the rest of the namespace.
  useTombstone: Panel.createTombstoneReader({ useTombstone: Library.useTombstone }),
};

export const TABS: Panel.Tabs = {
  [library.TYPE_ONTOLOGY_ID.type]: TAB,
};
