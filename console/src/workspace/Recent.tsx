// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement, useState } from "react";

import { type workspace } from "@synnaxlabs/client";
import { List as Core, Synnax, Text, useAsyncEffect } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";

import { add, setActive } from "./slice";

export const Recent = (): ReactElement | null => {
  const client = Synnax.use();
  const d = useDispatch();
  if (client == null) return null;
  const key = client.auth?.user?.key;
  if (key == null) return null;
  const [data, setData] = useState<workspace.Workspace[]>([]);

  useAsyncEffect(async () => {
    const workspaces = await client.workspaces.retrieveByAuthor(key);
    setData(workspaces);
  }, [client]);

  const handleClick = (key: string) => {
    void (async () => {
      const ws = await client.workspaces.retrieve(key);
      d(add({ workspaces: [ws] }));
      d(Layout.setWorkspace({ slice: ws.layout as unknown as Layout.SliceState }));
    })();
  };

  return (
    <Core.List<workspace.Key, workspace.Workspace> data={data}>
      <Core.Core style={{ height: 200 }}>
        {({ onSelect, entry: { key, name } }) => {
          return (
            <Text.Link level="h4" onClick={() => handleClick(key)}>
              {name}
            </Text.Link>
          );
        }}
      </Core.Core>
    </Core.List>
  );
};
