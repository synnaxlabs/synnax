import { Ontology } from "@/ontology";
import { Icon } from "@synnaxlabs/media";
import { OPC } from "@/hardware/opc";
import { NI } from "@/hardware/ni";
import { Layout } from "@/layout";

const TASK_LAYOUTS: Record<string, Layout> = {};

const handleSelect: Ontology.HandleSelect = ({ selection, placeLayout, client }) => {
  if (selection.length === 0) return;
  const task = selection[0].id;
  const t = client.hardware.tasks.retrieve(task.key);
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
