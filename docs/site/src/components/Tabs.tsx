// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Tab, Tabs as PTabs } from "@synnaxlabs/pluto";

export type TabsProps = {
  tabs: Tab[];
} & Record<string, JSX.Element | any>;

export const Tabs = ({ tabs, ...props }: TabsProps): JSX.Element => {
  tabs = tabs.map((tab) => ({ ...tab, icon: props[`${tab.tabKey}-icon`] }));
  const staticProps = PTabs.useStatic({ tabs });

  return <PTabs {...staticProps}>{(tab) => <div>{props[tab.tabKey]}</div>}</PTabs>;
};
