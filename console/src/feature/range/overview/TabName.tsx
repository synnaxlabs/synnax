import { Icon, Panel as PPanel, Ranger } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { type Panel } from "@/platform/panel";

export const TabName: Panel.TabName = ({ onRename: _drop, ...props }) => {
  const { key } = PPanel.useSelectTabResource();
  Ranger.useEnsureRetrieved({ key });
  const name = Ranger.useSelectName({ key });
  const { update } = Ranger.useRename();
  const handleRename = useCallback(
    (_: string, next: string) => update({ key, name: next }),
    [update, key],
  );
  return (
    <PPanel.DefaultTabName
      {...props}
      icon={<Icon.Range />}
      name={name}
      onRename={handleRename}
    />
  );
};
