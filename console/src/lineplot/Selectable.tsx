// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { Access, type Flux, Icon, LinePlot, Panel } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Tab } from "@/lineplot/tab";
import { Project } from "@/project";
import { Range } from "@/range";
import { Selector } from "@/selector";

const NAME = "Line Plot";

export const Selectable: Selector.Selectable = () => {
  const project = Project.useSelectActiveKey();
  const activeRange = Range.useSelectActiveKey() ?? Range.RECENT_KEY;
  const setTabContent = Panel.useSetCurrentTabContent();
  const { update: create } = LinePlot.useCreate({
    afterOptimistic: useCallback(
      ({ data: { key } }: Flux.AfterOptimisticParams<lineplot.LinePlot>) =>
        setTabContent({ type: Tab.TYPE, args: { key } }),
      [setTabContent],
    ),
  });
  const handleClick = () =>
    create({ project, name: NAME, ranges: { x1: [activeRange] } });
  return (
    <Selector.Item
      key={Tab.TYPE}
      title={NAME}
      icon={<Icon.LinePlot />}
      onClick={handleClick}
    />
  );
};
Selectable.type = Tab.TYPE;
Selectable.useVisible = () => Access.useCreateGranted(lineplot.TYPE_ONTOLOGY_ID);
