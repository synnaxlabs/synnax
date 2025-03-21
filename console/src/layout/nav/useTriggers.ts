import { useSelectWindowKey } from "@synnaxlabs/drift/react";
import { Triggers } from "@synnaxlabs/pluto";
import { useMemo } from "react";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";

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
      if (e.stage !== "start" || windowKey == null) return;
      const mode = Triggers.determineMode(modeConfig, e.triggers, { loose: false });
      if (mode.length === 0) return;
      if (mode.includes("double")) {
        const key = mode.split("-")[0];
        dispatch(Layout.setNavDrawerVisible({ windowKey, key, value: true }));
      } else dispatch(Layout.toggleNavHover({ windowKey, key: mode }));
    },
  });
};
