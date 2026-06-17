// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { table } from "@synnaxlabs/client";
import { Access, type Flux, Icon, Panel, Table } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Project } from "@/project";
import { Selector } from "@/selector";
import { Tab } from "@/table/tab";

const NAME = "Table";

export const Selectable: Selector.Selectable = () => {
  const project = Project.useSelectActiveKey();
  const setTabContent = Panel.useSetCurrentTabContent();
  const { update: create } = Table.useCreate({
    afterOptimistic: useCallback(
      ({ data: { key } }: Flux.AfterOptimisticParams<table.Table>) =>
        setTabContent({ type: Tab.TYPE, args: { key } }),
      [setTabContent],
    ),
  });
  const handleClick = () => create({ project, name: NAME });
  return (
    <Selector.Item
      key={Tab.TYPE}
      title={NAME}
      icon={<Icon.Table />}
      onClick={handleClick}
    />
  );
};
Selectable.type = Tab.TYPE;
Selectable.useVisible = () => Access.useCreateGranted(table.TYPE_ONTOLOGY_ID);
