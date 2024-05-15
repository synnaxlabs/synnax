// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Ontology } from "@/ontology";
import { Icon } from "@synnaxlabs/media";

export const handleSelect: Ontology.HandleSelect = ({ selection, placeLayout }) => {};

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: "device",
  hasChildren: false,
  icon: <Icon.Task />,
  canDrop: () => false,
  onSelect: handleSelect,
  TreeContextMenu: undefined,
  haulItems: () => [],
  allowRename: () => false,
  onRename: undefined,
};
