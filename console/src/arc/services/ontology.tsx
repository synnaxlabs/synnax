// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology, type Synnax } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/pluto";
import { strings } from "@synnaxlabs/x";

import { Arc } from "@/arc";
import { translateGraphToConsole } from "@/arc//types/translate";
import { type Layout } from "@/layout";
import { Ontology } from "@/ontology";

const handleSelect: Ontology.HandleSelect = ({
  client,
  selection,
  placeLayout,
  handleError,
}) => {
  load(client, selection[0].id, placeLayout).catch((e) => {
    const names = strings.naturalLanguageJoin(
      selection.map(({ name }) => name),
      "arc",
    );
    handleError(e, `Failed to load arc ${names}`);
  });
};

const load = async (client: Synnax, id: ontology.ID, placeLayout: Layout.Placer) => {
  const arc = await client.arcs.retrieve({ key: id.key });
  const graph = translateGraphToConsole(arc.graph);
  placeLayout(
    Arc.createEditor({
      name: arc.name,
      version: "0.0.0",
      key: arc.key,
      type: "arc",
      remoteCreated: true,
      graph,
    }),
  );
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "arc",
  icon: <Icon.Arc />,
  canDrop: () => true,
  onSelect: handleSelect,
};
