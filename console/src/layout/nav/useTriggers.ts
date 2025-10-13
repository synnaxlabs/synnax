// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Triggers } from "@synnaxlabs/pluto";
import { useMemo } from "react";
import { useDispatch } from "react-redux";

import { type Layout } from "@/layout";
import { setNavDrawerVisible, toggleNavHover } from "@/layout/slice";

const createModeConfig = (items: Layout.NavDrawerItem[]): Triggers.ModeConfig<string> =>
  Object.fromEntries(
    items
      .filter((item) => item.trigger?.length > 0)
      .flatMap((item) => [
        [item.key, [item.trigger]],
        [`${item.key}-double`, [[item.trigger[0], item.trigger[0]]]],
      ]),
  ) as Triggers.ModeConfig<string>;

export interface UseTriggersProps {
  items: Layout.NavDrawerItem[];
}

export const useTriggers = ({ items }: UseTriggersProps) => {
  const dispatch = useDispatch();
  const modeConfig = useMemo(() => createModeConfig(items), [items]);
  const flattenedConfig = Triggers.flattenConfig(modeConfig);
  const windowKey = useSelectWindowKey();
  Triggers.use({
    triggers: flattenedConfig,
    loose: false,
    callback: (e) => {
      if (
        e.stage !== "start" ||
        windowKey == null ||
        (e.prevTriggers.length > 0 && e.prevTriggers[0].length > 1)
      )
        return;
      const mode = Triggers.determineMode(modeConfig, e.triggers, { loose: false });
      if (mode.length === 0) return;
      if (mode.includes("double")) {
        const key = mode.split("-")[0];
        dispatch(setNavDrawerVisible({ windowKey, key, value: true }));
      } else dispatch(toggleNavHover({ windowKey, key: mode }));
    },
  });
};
