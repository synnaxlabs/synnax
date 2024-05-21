import { Ontology } from "@/ontology";
import { Icon } from "@synnaxlabs/media";
import { OPC } from "@/hardware/opc";
import { NI } from "@/hardware/ni";
import { Layout } from "@/layout";

const ZERO_LAYOUT_STATES: Record<string, Layout.State> = {
  [OPC.Task.configureReadLayout.type]: OPC.Task.configureReadLayout,
  [NI.Task.configureAnalogReadLayout.type]: NI.Task.configureAnalogReadLayout,
  [NI.Task.configureDigitalWriteLayout.type]: NI.Task.configureDigitalWriteLayout,
  [NI.Task.configureDigitalReadLayout.type]: NI.Task.configureDigitalReadLayout,
};

const handleSelect: Ontology.HandleSelect = ({
  selection,
  placeLayout,
  client,
  addStatus,
}) => {
  console.log("selection", selection);
  if (selection.length === 0) return;
  const task = selection[0].id;
  void (async () => {
    try {
      const t = await client.hardware.tasks.retrieve(task.key);
      const baseLayout = ZERO_LAYOUT_STATES[t.type];
      console.log(t, baseLayout);
      return placeLayout({ ...baseLayout, key: selection[0].id.key });
    } catch (e) {
      addStatus({ variant: "error", message: (e as Error).message });
    }
  })();
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
