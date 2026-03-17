// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/pluto";

import { create as createEditor } from "@/agent/editor/Editor";
import { Ontology } from "@/ontology";

const handleSelect: Ontology.HandleSelect = ({ selection, placeLayout }) => {
  const { id, name } = selection[0];
  placeLayout(createEditor({ key: id.key, name }));
};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  ...Ontology.NOOP_SERVICE,
  type: "agent",
  icon: <Icon.Auto />,
  hasChildren: false,
  canDrop: () => false,
  haulItems: () => [],
  onSelect: handleSelect,
};
