// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot, type project } from "@synnaxlabs/client";
import { LinePlot } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Panel } from "@/platform/panel";
import { Session } from "@/session";

export interface CreateParams extends Partial<lineplot.New> {
  project?: project.Key;
}

export interface UseCreateProps {
  project?: project.Key;
}

export const useCreate = ({ project }: UseCreateProps = {}): ((
  params?: CreateParams,
) => void) => {
  const activeProject = Session.Project.useSelectOptionalSelected();
  const target = project ?? activeProject;
  const openTab = Panel.useOpenTab();
  const { update } = LinePlot.useCreate({
    afterOptimistic: ({ data: { key } }) =>
      openTab({ variant: "resource", resource: lineplot.ontologyID(key) }),
  });
  return useCallback(
    (params: CreateParams = {}) =>
      update({ name: "New Line Plot", project: target, ...params }),
    [update, target],
  );
};
