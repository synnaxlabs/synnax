// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  renameTab,
  resetTabSelection,
  Tabs as CoreTabs,
  TabsContent,
  TabsContext,
  useStaticTabs,
  useTabsContext,
} from "./Tabs";
import { TabsSelector } from "./TabsSelector";

export type { Tab, TabsProps } from "./Tabs";

type CoreTabsType = typeof CoreTabs;

interface TabsType extends CoreTabsType {
  useStatic: typeof useStaticTabs;
  resetSelection: typeof resetTabSelection;
  rename: typeof renameTab;
  Provider: typeof TabsContext.Provider;
  Content: typeof TabsContent;
  Selector: typeof TabsSelector;
  useContext: typeof useTabsContext;
}

export const Tabs = CoreTabs as TabsType;

Tabs.useStatic = useStaticTabs;
Tabs.resetSelection = resetTabSelection;
Tabs.rename = renameTab;
Tabs.Provider = TabsContext.Provider;
Tabs.Content = TabsContent;
Tabs.Selector = TabsSelector;
Tabs.useContext = useTabsContext;
