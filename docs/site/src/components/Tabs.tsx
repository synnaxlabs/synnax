// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import { Tabs as Core } from "@synnaxlabs/pluto/tabs";

export type TabsProps = {
  tabs: Core.Tab[];
} & Record<string, ReactElement | any>;

export const Tabs = ({ tabs, ...props }: TabsProps): ReactElement => {
  tabs = tabs.map((tab) => ({ ...tab, icon: props[`${tab.tabKey}-icon`] }));
  const staticProps = Core.useStatic({ tabs });

  return (
    <Core.Tabs {...staticProps}>{(tab) => <div>{props[tab.tabKey]}</div>}</Core.Tabs>
  );
};
