import { Ontology } from "@/ontology";
import { Icon } from "@synnaxlabs/media";
import { opc } from "@/hardware/opc";

const handleSelect: Ontology.HandleSelect = ({ selection, placeLayout }) => {
  placeLayout({
    ...opc.readTaskLayout,
    key: selection[0].id.key,
  });
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: "task",
  hasChildren: true,
  icon: <Icon.Task />,
  canDrop: () => true,
  onSelect: handleSelect,
  TreeContextMenu: undefined,
  haulItems: () => [],
  allowRename: () => true,
  onRename: undefined,
};
