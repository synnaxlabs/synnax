// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/media";

import { type Service } from "../service";

export const ONTOLOGY_SERVICE: Service = {
  type: "builtin",
  icon: <Icon.Cluster />,
  hasChildren: true,
  allowRename: () => false,
  onRename: undefined,
  canDrop: () => false,
  onMosaicDrop: () => {},
  TreeContextMenu: () => <></>,
  onSelect: () => {},
  haulItems: () => [],
};
