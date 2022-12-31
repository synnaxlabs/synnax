// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Tabs as CoreTabs, resetTabSelection, useStaticTabs, renameTab } from "./Tabs";

export type { Tab, TabsProps } from "./Tabs";

type CoreTabsType = typeof CoreTabs;

interface TabsType extends CoreTabsType {
  useStatic: typeof useStaticTabs;
  resetSelection: typeof resetTabSelection;
  rename: typeof renameTab;
}

export const Tabs = CoreTabs as TabsType;

Tabs.useStatic = useStaticTabs;
Tabs.resetSelection = resetTabSelection;
Tabs.rename = renameTab;
