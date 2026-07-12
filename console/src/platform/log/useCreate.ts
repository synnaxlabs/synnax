// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { log, type project, UnexpectedError } from "@synnaxlabs/client";
import { Log } from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { Panel } from "@/platform/panel";
import { useMaybeChange } from "@/platform/project/useMaybeChange";
import { Session } from "@/session";

export interface UseCreateProps {
  project?: project.Key;
}

export const useCreate = ({ project }: UseCreateProps = {}): ((
  params?: Partial<log.New>,
) => void) => {
  const activeProject = Session.Project.useSelectOptionalSelected();
  const target = project ?? activeProject;
  const maybeChangeProject = useMaybeChange();
  const openTab = Panel.useOpenTab();
  const dispatch = useDispatch();
  const { update } = Log.useCreate({
    afterOptimistic: ({ data: { key } }) => {
      if (target != null) maybeChangeProject(target);
      dispatch(Session.Log.create({ key }));
      openTab({ variant: "resource", resource: log.ontologyID(key) });
    },
  });
  return useCallback(
    (params = {}) => {
      if (target == null)
        throw new UnexpectedError("cannot create a resource without a project");
      update({ name: "Log", ...params, project: target });
    },
    [update, target],
  );
};
