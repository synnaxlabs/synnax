import { arc } from "@synnaxlabs/client";
import { Access, Icon } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { useCreateModal } from "@/arc/editor/CreateModal";
import { useCreate } from "@/arc/editor/useCreate";
import { TYPE } from "@/arc/slice";
import { Selector } from "@/selector";

export const Selectable: Selector.Selectable = ({ layoutKey, handleError }) => {
  const hasCreatePermission = Access.useCreateGranted(arc.TYPE_ONTOLOGY_ID);
  const createArcModal = useCreateModal();
  const createArc = useCreate();

  const handleClick = useCallback(() => {
    handleError(async () => {
      const result = await createArcModal({});
      if (result != null)
        createArc({ key: layoutKey, name: result.name, mode: result.mode });
    }, "Failed to create Arc program");
  }, [layoutKey, createArcModal, handleError, createArc]);

  if (!hasCreatePermission) return null;

  return (
    <Selector.Item title="Arc Automation" icon={<Icon.Arc />} onClick={handleClick} />
  );
};
Selectable.type = TYPE;
Selectable.useVisible = () => Access.useCreateGranted(arc.TYPE_ONTOLOGY_ID);
