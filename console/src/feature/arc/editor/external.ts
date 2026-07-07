// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";

import { Editor } from "@/feature/arc/editor/Editor";
import { TabName } from "@/feature/arc/editor/TabName";
import { type Panel } from "@/platform/panel";

export * from "@/feature/arc/editor/Editor";
export * from "@/feature/arc/editor/toolbar/Toolbar";

const TAB: Panel.Tab = {
  Content: Editor,
  Name: TabName,
};

export const TABS: Panel.Tabs = {
  [arc.TYPE_ONTOLOGY_ID.type]: TAB,
};
