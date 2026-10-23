import { type ontology, panel } from "@synnaxlabs/client";
import { type Icon } from "@synnaxlabs/lyra/icon";

import { renderIcon } from "@/feature/panel/icon";
import { Tree } from "@/platform/tree";

const appearanceZ = panel.panelZ.pick({ icon: true, color: true });

// The panel's ontology resource carries the panel itself, so the row needs no retrieve.
const icon = (resource: ontology.Resource): Icon.ReactElement =>
  renderIcon(appearanceZ.parse(resource.data ?? {}));

const TreeItem = Tree.createItem({ type: "panel", icon });

export const TREE_ITEMS: Tree.Items = { panel: TreeItem };
