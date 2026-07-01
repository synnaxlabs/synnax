// Copyright 2026 Synnax Labs, Inc.
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

import { create } from "@/component/arc/layout";
import { Ontology } from "@/service/ontology";
import { Session } from "@/session";

const handleSelect: Ontology.HandleSelect = ({
  client,
  selection,
  placeLayout,
  handleError,
}) => {
  load(client, selection[0].id, placeLayout).catch((e: unknown) => {
    const names = strings.naturalLanguageJoin(
      selection.map(({ name }) => name),
      "Arc",
    );
    handleError(e, `Failed to load ${names}`);
  });
};

const load = async (
  client: Synnax,
  id: ontology.ID,
  placeLayout: Session.Layout.Placer,
) => {
  const { name, key } = await client.arcs.retrieve({ key: id.key });
  placeLayout(create({ name, key }));
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "arc",
  icon: <Icon.Arc />,
  canDrop: () => true,
  onSelect: handleSelect,
};
