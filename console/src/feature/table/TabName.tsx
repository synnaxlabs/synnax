import { Icon, Panel, Table } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { type Panel as PlatformPanel } from "@/platform/panel";

export const TabName: PlatformPanel.TabName = ({ onRename: _, ...props }) => {
  const { key } = Panel.useSelectTabResource();
  Table.useEnsureRetrieved({ key });
  const name = Table.useSelectName({ key });
  const { update } = Table.useRename();
  const handleRename = useCallback(
    (_: string, name: string) => update({ key, name }),
    [update, key],
  );
  return (
    <Panel.DefaultTabName
      {...props}
      icon={<Icon.Table />}
      name={name}
      onRename={handleRename}
    />
  );
};
