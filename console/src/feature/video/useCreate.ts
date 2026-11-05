// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel, type project, video } from "@synnaxlabs/client";
import { Video } from "@synnaxlabs/pluto";

import { Panel } from "@/platform/panel";
import { useMaybeChange } from "@/platform/project/useMaybeChange";
import { Session } from "@/session";

export interface UseCreateProps {
  project?: project.Key;
  tabKey?: panel.TabKey;
}

/** @returns a function that creates a video view and opens it in a tab. */
export const useCreate = ({ project, tabKey }: UseCreateProps = {}): ((
  params?: Partial<video.New>,
) => void) => {
  const getActiveProject = Session.Project.useGetSelected();
  const maybeChangeProject = useMaybeChange();
  const openTab = Panel.useOpenTab();
  const { update } = Video.useCreate({
    afterOptimistic: ({ data: { key } }) => {
      maybeChangeProject(project ?? getActiveProject());
      openTab({ variant: "resource", resource: video.ontologyID(key), key: tabKey });
    },
  });
  return (params = {}) =>
    update({ name: "Video", ...params, project: project ?? getActiveProject() });
};
