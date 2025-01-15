// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Layout } from "@/layout";
import { type Link } from "@/link";
import { Workspace } from "@/workspace";

export const linkHandler: Link.Handler = async ({ client, dispatch, key }) => {
  const workspace = await client.workspaces.retrieve(key);
  dispatch(Layout.setWorkspace({ slice: workspace.layout as Layout.SliceState }));
  dispatch(Workspace.setActive(workspace.key));
};
