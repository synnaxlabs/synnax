// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type workspace } from "@synnaxlabs/client";
import { Status, Synnax, Text, useAsyncEffect } from "@synnaxlabs/pluto";
import { List } from "@synnaxlabs/pluto/list";
import { type ReactElement, useState } from "react";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";
import { add } from "@/workspace/slice";

export const Recent = (): ReactElement | null => {
  const client = Synnax.use();
  const dispatch = useDispatch();
  const [data, setData] = useState<workspace.Workspace[]>([]);
  const key = client?.auth?.user?.key;

  useAsyncEffect(async () => {
    const workspaces =
      key != null ? await client?.workspaces.retrieveByAuthor(key) : [];
    setData(workspaces ?? []);
  }, [client]);

  const handleException = Status.useExceptionHandler();

  const handleClick = (key: string): void => {
    if (client == null) return;
    client.workspaces
      .retrieve(key)
      .then((ws) => {
        dispatch(add(ws));
        dispatch(Layout.setWorkspace({ slice: ws.layout as Layout.SliceState }));
      })
      .catch((e) => handleException(e, "Failed to open workspace"));
  };

  if (client == null || key == null) return null;

  return (
    <List.List<workspace.Key, workspace.Workspace> data={data}>
      <List.Core<workspace.Key, workspace.Workspace>>
        {({ entry: { key, name } }) => (
          <Text.Link key={key} level="h4" onClick={() => handleClick(key)}>
            {name}
          </Text.Link>
        )}
      </List.Core>
    </List.List>
  );
};
