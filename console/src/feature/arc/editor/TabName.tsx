import { Arc, Icon, Panel as PPanel } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { type Panel } from "@/platform/panel";

export const TabName: Panel.TabName = ({ onRename: _onRename, ...props }) => {
  const { key } = PPanel.useSelectTabResource();
  Arc.useEnsureRetrieved({ key });
  const name = Arc.useSelectName({ key });
  const { update } = Arc.useRename();
  const handleRename = useCallback(
    (_: string, next: string) => update({ key, name: next }),
    [update, key],
  );
  return (
    <PPanel.DefaultTabName
      {...props}
      icon={<Icon.Arc />}
      name={name}
      onRename={handleRename}
    />
  );
};
