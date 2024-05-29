// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Layout } from "@/layout";
import { Synnax } from "@synnaxlabs/pluto";
import { useEffect } from "react";
import { useDispatch } from "react-redux";

export const useDeepLink = () => {
  const url0 = "synnax://workspace:92cf58a7-adba-44b0-97f7-fbbc1cf9432e";
  const client = Synnax.use();
  const urls = [url0];
  const d = useDispatch();

  useEffect(() => {
    console.log("Using a deep link");
    const prefix = "synnax://workspace:";
    if (urls.length === 0 || !urls[0].startsWith(prefix)) return;
    const workspaceKey = urls[0].slice(prefix.length);
    if (workspaceKey == null) return;
    console.log("Using workspace with key: ", workspaceKey, client);

    const promise = client?.workspaces.retrieve(workspaceKey);

    if (promise == undefined) {
      console.log("Promise is undefined");
      return;
    }
    
    console.log("ID DEFIN")
    promise
      .then((ws) => {
        console.log("Do you see me?");
        if (ws == null) return;
        console.log("Workspace is not null");
        console.log("Workspace Name: ", ws.name);
        d(Layout.setWorkspace({
          slice: ws.layout as unknown as Layout.SliceState,
        }));
        console.log("Set workspace");
      })
      .catch((error) => {
        console.log("Ruhruh");
        console.error("Error: ", error);
      });
  }, [client]);
};
