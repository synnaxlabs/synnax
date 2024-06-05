// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Layout } from "@/layout";
import { Link } from "@/link";
import { setActive } from "@/workspace/slice";

export const linkHandler: Link.Handler = async ({
  resource,
  resourceKey,
  client,
  dispatch,
}): Promise<boolean> => {
  if (resource !== "workspace") {
    console.log("resource is not workspace");
    return false;
  }
  console.log("workspace/link.tsx: Resource is a workspace");
  console.log("Client port", client.props.port);
  try {
    const workspace = await client.workspaces.retrieve(resourceKey);
    if (workspace == null) {
      console.log("workspace is null");
      return false;
    }
    dispatch(
      Layout.setWorkspace({
        slice: workspace.layout as unknown as Layout.SliceState,
      }),
    );
    console.log("Workspace layout set");
    dispatch(setActive(workspace.key));
    console.log("Active workspace set");
    return true;
  } catch (error) {
    console.error("Error: ", error);
    return false;
  }
};
