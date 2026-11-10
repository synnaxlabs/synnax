// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Access } from "@synnaxlabs/pluto";

import { INTEGRATIONS } from "@/feature/task/types";
import { Panel } from "@/platform/panel";
import { Selector as Base } from "@/platform/selector";

const GROUPS: Base.Group[] = INTEGRATIONS.filter(
  ({ selectables }) => selectables.length > 0,
).map(({ prefix, name, icon, selectables }) => ({
  key: prefix,
  title: name,
  icon,
  selectables,
}));

const useCreateGranted = (): boolean => Access.useCreateGranted(task.TYPE_RESOURCE_ID);

/** The task types, one group per integration, for a user who may create tasks. */
export const SECTION: Base.Section = {
  key: "tasks",
  groups: GROUPS,
  useVisible: useCreateGranted,
};

export const Selector = Base.create({
  sections: [SECTION],
  icon: <Icon.Task />,
  tabTitle: "Create task",
  text: "Create task",
  searchPlaceholder: "Search tasks",
});

export const SELECTOR_TAB_TYPE = "taskSelector";

export const useOpenSelector = (): (() => void) => {
  const openTab = Panel.useOpenTab();
  return () => openTab({ variant: "view", type: SELECTOR_TAB_TYPE });
};
