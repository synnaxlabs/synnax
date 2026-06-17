import { Icon, Panel, Schematic } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { type Tabs } from "@/panel/tabs";
import { Tab } from "@/schematic/tab";

export const Name: Tabs.Name = (props) => {
  const { key } = Tab.useArgs();
  const name = Schematic.useRetrieveName({ key });
  const { update } = Schematic.useRename();
  const handleRename = useCallback((name: string) => update({ key, name }), [name]);
  return (
    <Panel.DefaultTabName
      icon={<Icon.Schematic />}
      name={name}
      onRename={handleRename}
      {...props}
    />
  );
};
