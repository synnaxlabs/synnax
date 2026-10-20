// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Arc } from "@synnaxlabs/pluto";

import { type Panel } from "@/platform/panel";
import { Tree } from "@/platform/tree";

export const Explorer: Panel.Content = () => {
  const groupID = Arc.useGroupID({});
  return <Tree.Tree root={groupID} />;
};
