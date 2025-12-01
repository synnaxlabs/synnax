// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { Ontology } from "@/ontology";
import { VIEW_REGISTRY } from "@/status/external";

const handleSelect: Ontology.HandleSelect = ({
  selection,
  placeLayout,
  client,
  handleError,
}) => {
  if (selection.length === 0) return;
  handleError(async () => {
    const view = await client.views.retrieve({ key: selection[0].id.key });
    const { layout } = VIEW_REGISTRY[view.type];
    if (layout == null) return;
    placeLayout({ ...layout, name: view.name, args: { initialRequest: view.query } });
  }, `Failed to select ${selection[0].name}`);
};

const editedViewRegistryIcons: Record<string, Icon.FC | undefined> = Object.fromEntries(
  Object.entries(VIEW_REGISTRY).map(([type, { icon }]) => [
    type,
    Icon.createComposite(icon, { topRight: Icon.View }),
  ]),
);

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "view",
  onSelect: handleSelect,
  icon: (resource) => {
    const type = resource.data?.type;
    if (typeof type !== "string") return <Icon.View />;
    const I = editedViewRegistryIcons[type];
    if (I == null) return <Icon.View />;
    return <I />;
  },
  hasChildren: false,
};
