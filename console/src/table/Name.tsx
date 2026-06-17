// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon, Panel, Table } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { type Tabs } from "@/panel/tabs";
import { Tab } from "@/table/tab";

export const Name: Tabs.Name = (props) => {
  const { key } = Tab.useArgs();
  const name = Table.useRetrieveName({ key });
  const { update } = Table.useRename();
  const handleRename = useCallback((name: string) => update({ key, name }), [key]);
  return (
    <Panel.DefaultTabName
      icon={<Icon.Table />}
      name={name}
      onRename={handleRename}
      {...props}
    />
  );
};
