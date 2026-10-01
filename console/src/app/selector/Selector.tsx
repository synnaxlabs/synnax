// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";

import { SELECTABLES } from "@/app/selector/selectables";
import { type Panel } from "@/platform/panel";
import { Selector as Base } from "@/platform/selector";

const Selector = Base.create({
  selectables: SELECTABLES,
  icon: <Icon.Component />,
  tabTitle: "Create component",
  text: "Create component",
});

export const TABS: Panel.Tabs = { [Base.TAB_TYPE]: Selector };
