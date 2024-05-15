import { Ontology } from "@/ontology";
import { Icon } from "@synnaxlabs/media";
import { OPC } from "@/hardware/opc";

const handleSelect: Ontology.HandleSelect = ({ selection, placeLayout }) => {
  placeLayout({
    ...OPC.readTaskLayout,
    key: selection[0].id.key,
  });
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: "task",
  hasChildren: false,
  icon: <Icon.Task />,
  canDrop: () => false,
  onSelect: handleSelect,
  TreeContextMenu: undefined,
  haulItems: () => [],
  allowRename: () => false,
  onRename: undefined,
};
