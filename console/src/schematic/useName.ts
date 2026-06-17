import { Schematic } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { type Panel } from "@/panel";
import { Tab } from "@/schematic/tab";

export const useName = (): Panel.UseNameReturn => {
  const { key } = Tab.useArgs();
  const name = Schematic.useRetrieveName({ key });
  const { update } = Schematic.useRename();
  const rename = useCallback((name: string) => update({ key, name }), [name]);
  return { name, rename };
};
