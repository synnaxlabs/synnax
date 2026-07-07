import { Icon, LinePlot, Panel } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { type Panel as PlatformPanel } from "@/platform/panel";

export const TabName: PlatformPanel.TabName = ({ onRename: _, ...props }) => {
  const { key } = Panel.useSelectTabResource();
  LinePlot.useEnsureRetrieved({ key });
  const name = LinePlot.useSelectName({ key });
  const { update } = LinePlot.useRename();
  const handleRename = useCallback(
    (_: string, name: string) => update({ key, name }),
    [update, key],
  );
  return (
    <Panel.DefaultTabName
      {...props}
      icon={<Icon.LinePlot />}
      name={name}
      onRename={handleRename}
    />
  );
};
