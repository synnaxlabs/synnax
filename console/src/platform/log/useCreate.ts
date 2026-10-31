// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log, type panel, type project } from "@synnaxlabs/client";
import { Log } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { Panel } from "@/platform/panel";
import { Project } from "@/platform/project";
import { Session } from "@/session";

export interface UseCreateProps {
  project?: project.Key;
  tabKey?: panel.TabKey;
}

export const useCreate = ({ project, tabKey }: UseCreateProps = {}): ((
  params?: Partial<log.New>,
) => void) => {
  const getActiveProject = Session.Project.useGetSelected();
  const maybeChangeProject = Project.useMaybeChange();
  const openTab = Panel.useOpenTab();
  const dispatch = useDispatch();
  const { update } = Log.useCreate({
    afterOptimistic: ({ data: { key } }) => {
      maybeChangeProject(project ?? getActiveProject());
      dispatch(Session.Log.create({ key }));
      openTab({ variant: "resource", resource: log.ontologyID(key), key: tabKey });
    },
  });
  return (params = {}) =>
    update({ name: "Log", ...params, project: project ?? getActiveProject() });
};
