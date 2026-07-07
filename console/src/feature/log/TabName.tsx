import { Icon, Log, Panel } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { type Panel as PlatformPanel } from "@/platform/panel";

export const TabName: PlatformPanel.TabName = ({ onRename: _, ...props }) => {
  const { key } = Panel.useSelectTabResource();
  Log.useEnsureRetrieved({ key });
  const name = Log.useSelectName({ key });
  const { update } = Log.useRename();
  const handleRename = useCallback(
    (_: string, name: string) => update({ key, name }),
    [update, key],
  );
  return (
    <Panel.DefaultTabName
      {...props}
      icon={<Icon.Log />}
      name={name}
      onRename={handleRename}
    />
  );
};
