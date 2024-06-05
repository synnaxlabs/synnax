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

export const linkHandler: Link.Handler = ({
  resource,
  resourceKey,
  client,
  dispatch,
}) => {
  if (resource != "workspace") return false;
  client.workspaces
    .retrieve(resourceKey)
    .then((workspace) => {
      if (workspace == null) return false;
      dispatch(
        Layout.setWorkspace({
          slice: workspace.layout as unknown as Layout.SliceState,
        }),
      );
      dispatch(setActive(workspace.key));
      return true;
    })
    .catch((error) => {
      console.error("Error: ", error);
      return false;
    });
  return false;
};
