import { arc } from "@synnaxlabs/client";
import { Access, Icon } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { useCreate } from "@/arc/editor/useCreate";
import { TYPE } from "@/arc/slice";
import { Selector } from "@/selector";

export const Selectable: Selector.Selectable = ({ layoutKey }) => {
  const hasCreatePermission = Access.useCreateGranted(arc.TYPE_ONTOLOGY_ID);
  const createArc = useCreate();
  const handleClick = useCallback(
    () => createArc({ key: layoutKey }),
    [createArc, layoutKey],
  );
  if (!hasCreatePermission) return null;
  return (
    <Selector.Item title="Arc Automation" icon={<Icon.Arc />} onClick={handleClick} />
  );
};
Selectable.type = TYPE;
Selectable.useVisible = () => Access.useCreateGranted(arc.TYPE_ONTOLOGY_ID);
