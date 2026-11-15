// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd, query } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { ICD } from "@synnaxlabs/pluto";

import { Panel } from "@/platform/panel";

export * from "@/feature/icd/commands";
export * from "@/feature/icd/link";
export * from "@/feature/icd/search";
export * from "@/feature/icd/Toolbar";
export * from "@/feature/icd/useCreate";
export * from "@/feature/icd/useImport";

const TAB: Panel.Tab = {
  Content: Panel.lazyComponent(() => import("@/feature/icd/editor/Editor"), "Editor"),
  Icon: Icon.Library,
  // Only these hooks, so the bundler can drop the rest of the namespace.
  Name: Panel.createEditableTabName(
    {
      useEnsure: ICD.useEnsure,
      useName: ICD.useName,
      useRename: ICD.useRename,
    },
    <Icon.Library />,
  ),
  restore: async ({ client, resource }) => {
    await client.icds.create(
      query.requireCorpse(client.icds.watch(resource.key).getCached()),
    );
  },
  // Only these hooks, so the bundler can drop the rest of the namespace.
  Guard: Panel.createGuard({ useTombstone: ICD.useTombstone }),
};

export const TABS: Panel.Tabs = { [icd.TYPE_RESOURCE_ID.type]: TAB };
