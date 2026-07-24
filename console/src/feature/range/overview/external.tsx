// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ranger } from "@synnaxlabs/client";
import { Icon, Ranger } from "@synnaxlabs/pluto";

import { Overview } from "@/feature/range/overview/Overview";
import { Panel } from "@/platform/panel";

export { Overview };

const TAB: Panel.Tab = {
  Content: Overview,
  Name: Panel.createEditableTabName(Ranger, <Icon.Range />),
};

export const TABS: Panel.Tabs = {
  [ranger.TYPE_ONTOLOGY_ID.type]: TAB,
};
