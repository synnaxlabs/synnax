import { Icon, Panel, Schematic } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { type Panel as PlatformPanel } from "@/platform/panel";

export const TabName: PlatformPanel.TabName = ({ onRename: _, ...props }) => {
  const { key } = Panel.useSelectTabResource();
  Schematic.useEnsureRetrieved({ key });
  const name = Schematic.useSelectName({ key });
  const { update } = Schematic.useRename();
  const handleRename = useCallback(
    (_: string, name: string) => update({ key, name }),
    [update, key],
  );
  return (
    <Panel.DefaultTabName
      {...props}
      icon={<Icon.Schematic />}
      name={name}
      onRename={handleRename}
    />
  );
};
