// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";

import { type Ontology } from "@/ontology";

export const ONTOLOGY_SERVICE: Ontology.Service = {
  type: ontology.NODE_TYPE,
  icon: <Icon.Node />,
  hasChildren: true,
  canDrop: () => false,
  onSelect: () => {},
  haulItems: () => [],
  allowRename: () => false,
};
