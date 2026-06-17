// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log } from "@synnaxlabs/client";
import { Access, type Flux, Icon, Log, Panel } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Tab } from "@/log/tab";
import { Project } from "@/project";
import { Selector } from "@/selector";

const NAME = "Log";

export const Selectable: Selector.Selectable = () => {
  const project = Project.useSelectActiveKey();
  const setTabContent = Panel.useSetCurrentTabContent();
  const { update: create } = Log.useCreate({
    afterOptimistic: useCallback(
      ({ data: { key } }: Flux.AfterOptimisticParams<log.Log>) =>
        setTabContent({ type: Tab.TYPE, args: { key } }),
      [setTabContent],
    ),
  });
  const handleClick = () => create({ project, name: NAME });
  return (
    <Selector.Item
      key={Tab.TYPE}
      title={NAME}
      icon={<Icon.Log />}
      onClick={handleClick}
    />
  );
};
Selectable.type = Tab.TYPE;
Selectable.useVisible = () => Access.useCreateGranted(log.TYPE_ONTOLOGY_ID);
