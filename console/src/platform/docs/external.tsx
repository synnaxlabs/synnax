// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { Docs, TAB_TYPE } from "@/platform/docs/Docs";
import { Panel } from "@/platform/panel";

export * from "@/platform/docs/Docs";
export * from "@/platform/docs/OpenButton";
export * from "@/platform/docs/useOpenTab";

const TAB: Panel.Tab = {
  Content: Docs,
  Name: Panel.createStaticTabName({
    name: "Documentation",
    icon: <Icon.QuestionMark />,
  }),
};

export const TABS: Panel.Tabs = {
  [TAB_TYPE]: TAB,
};
