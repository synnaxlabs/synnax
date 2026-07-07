// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc, type ontology, type Synnax } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/pluto";
import { strings } from "@synnaxlabs/x";

import { Ontology } from "@/platform/ontology";
import { type Panel } from "@/platform/panel";

const load = async (client: Synnax, { key }: ontology.ID, openTab: Panel.OpenTab) => {
  const a = await client.arcs.retrieve({ key });
  openTab({ variant: "resource", resource: arc.ontologyID(a.key) });
};

const handleSelect: Ontology.HandleSelect = ({
  client,
  selection,
  openTab,
  handleError,
}) => {
  load(client, selection[0].id, openTab).catch((e: unknown) => {
    const names = strings.naturalLanguageJoin(
      selection.map(({ name }) => name),
      "Arc",
    );
    handleError(e, `Failed to load ${names}`);
  });
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "arc",
  icon: <Icon.Arc />,
  canDrop: () => true,
  onSelect: handleSelect,
};
