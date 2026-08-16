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
import { useCallback } from "react";

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
  const getSelectedRange = Session.Range.useGetSelectedKey();
  const openTab = Panel.useOpenTab();
  const { update } = LinePlot.useCreate({
    afterOptimistic: ({ data: { key } }) =>
      openTab({ variant: "resource", resource: lineplot.ontologyID(key), key: tabKey }),
  });
  return useCallback(
    (params = {}) =>
      update({
        name: "Line Plot",
        ranges: { x1: [getSelectedRange() ?? Session.Range.RECENT_KEY] },
        ...params,
        project: project ?? getActiveProject(),
      }),
    [update, project, getActiveProject, getSelectedRange],
  );
};
