// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { Arc } from "@/arc";
import { Ontology } from "@/ontology";

const handleSelect: Ontology.HandleSelect = ({ selection, placeLayout }) => {
  selection.forEach((s) => {
    placeLayout(Arc.createEditor({ key: s.id.key }));
  });
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "arc",
  icon: <Icon.Arc />,
  canDrop: () => true,
  onSelect: handleSelect,
};
