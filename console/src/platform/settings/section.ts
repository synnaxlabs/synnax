// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Icon } from "@synnaxlabs/lyra/icon";
import { type FC } from "react";

/** A page of Settings. The section list shows it by its icon and name. */
export interface Section {
  key: string;
  name: string;
  icon: Icon.ReactElement;
  Content: FC;
  /** Reports whether the user can use the section. Settings hides it when false. */
  useVisible?: () => boolean;
}

/** A named run of sections in the section list, such as Preferences. */
export interface Group {
  key: string;
  name: string;
  sections: Section[];
}

/**
 * Returns the groups with only the sections the user can use, and drops a group that
 * has none. Calls every section's visibility hook, so groups has to be a module-level
 * constant to keep the hook order stable.
 */
export const useVisibleGroups = (groups: Group[]): Group[] => {
  "use no memo"; // Calls a hook per section, so the group list sets the hook order.
  const sections = groups.flatMap((group) => group.sections);
  const visibilities = sections.map((section) => section.useVisible?.() ?? true);
  const visible = new Set(sections.filter((_, i) => visibilities[i]));
  return groups
    .map((group) => ({
      ...group,
      sections: group.sections.filter((s) => visible.has(s)),
    }))
    .filter((group) => group.sections.length > 0);
};
