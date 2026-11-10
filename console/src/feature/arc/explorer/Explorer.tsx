// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { arc } from "@synnaxlabs/client";
import { Arc } from "@synnaxlabs/pluto";

import { RootContextMenu } from "@/feature/arc/tree";
import { Empty } from "@/platform/empty";
import { type Panel } from "@/platform/panel";
import { Tree } from "@/platform/tree";

export const Explorer: Panel.Content = () => {
  const rootKey = Arc.Directory.useRootKey({});
  return (
    <Tree.Tree
      root={arc.directory.resourceID(rootKey)}
      emptyContent={<Empty.Action message="No Arcs" />}
      rootContextMenu={RootContextMenu}
    />
  );
};
