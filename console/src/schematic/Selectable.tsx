import { schematic } from "@synnaxlabs/client";
import { Access, Icon } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { create, LAYOUT_TYPE } from "@/schematic/layout";
import { Selector } from "@/selector";

export const Selectable: Selector.Selectable = ({ layoutKey, onPlace }) => {
  const hasCreatePermission = Access.useCreateGranted(schematic.TYPE_ONTOLOGY_ID);
  const handleClick = useCallback(() => {
    onPlace(create({ key: layoutKey }));
  }, [onPlace, layoutKey]);
  if (!hasCreatePermission) return null;
  return (
    <Selector.Item
      key={LAYOUT_TYPE}
      title="Schematic"
      icon={<Icon.Schematic />}
      onClick={handleClick}
    />
  );
};
Selectable.type = LAYOUT_TYPE;
Selectable.useVisible = () => Access.useCreateGranted(schematic.TYPE_ONTOLOGY_ID);
