// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, type panel, type project } from "@synnaxlabs/client";
import { LinePlot } from "@synnaxlabs/pluto";

import { Analytics } from "@/platform/analytics";
import { Panel } from "@/platform/panel";
import { Session } from "@/session";

export interface UseCreateProps {
  project?: project.Key;
  tabKey?: panel.TabKey;
}

export const useCreate = ({ project, tabKey }: UseCreateProps = {}): ((
  params?: Partial<lineplot.New>,
) => void) => {
  const getActiveProject = Session.Project.useGetSelected();
  const getSelectedRef = Session.Range.useGetSelectedRef();
  const openTab = Panel.useOpenTab();
  const { capture } = Analytics.use();
  const { update } = LinePlot.useCreate({
    afterOptimistic: ({ data: { key } }) =>
      openTab({ variant: "resource", resource: lineplot.ontologyID(key), key: tabKey }),
    afterSuccess: () => capture("plot_created", {}),
  });
  return (params = {}) =>
    update({
      name: "Line plot",
      ranges: [getSelectedRef()],
      ...params,
      project: project ?? getActiveProject(),
    });
};
