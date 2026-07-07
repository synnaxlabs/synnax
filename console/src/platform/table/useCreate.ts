// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type project, table, UnexpectedError } from "@synnaxlabs/client";
import { Table } from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { Panel } from "@/platform/panel";
import { useMaybeChange } from "@/platform/project/useMaybeChange";
import { Session } from "@/session";

export interface UseCreateProps {
  project?: project.Key;
}

export const useCreate = ({ project }: UseCreateProps = {}): ((
  params?: Partial<table.New>,
) => void) => {
  const activeProject = Session.Project.useSelectOptionalSelected();
  const target = project ?? activeProject;
  const maybeChangeProject = useMaybeChange();
  const openTab = Panel.useOpenTab();
  const dispatch = useDispatch();
  const { update } = Table.useCreate({
    afterOptimistic: ({ data: { key } }) => {
      if (target != null) maybeChangeProject(target);
      dispatch(Session.Table.create({ key, editable: true }));
      openTab({ variant: "resource", resource: table.ontologyID(key) });
    },
  });
  return useCallback(
    (params = {}) => {
      if (target == null)
        throw new UnexpectedError("cannot create a resource without a project");
      update({ name: "Table", ...params, project: target });
    },
    [update, target],
  );
};
