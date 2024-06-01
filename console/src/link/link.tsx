// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax } from "@synnaxlabs/client";
import { Synnax as PSynnax } from "@synnaxlabs/pluto";
import { useEffect } from "react";
import { onOpenUrl } from "@tauri-apps/plugin-deep-link";
import { Dispatch, UnknownAction } from "@reduxjs/toolkit";
import { useDispatch } from "react-redux";

import { Cluster } from "@/cluster";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { Range } from "@/range";
import { Schematic } from "@/schematic";
import { Workspace } from "@/workspace";

export interface HandlerProps {
  // url is a string of two parts, the first part is the resource identifier and
  // the second part is the resource key.
  resource: string;
  resourceKey: string;
  client: Synnax;
  dispatch: Dispatch<UnknownAction>;
  placer: Layout.Placer;
  clusters: Cluster.Cluster[];
}

export type Handler = (props: HandlerProps) => boolean;

export interface UseDeepLinkProps {
  handlers: Handler[];
}

export const useDeepLink = ({ handlers }: UseDeepLinkProps): void => {
  const client = PSynnax.use();
  const dispatch = useDispatch();
  const placer = Layout.usePlacer();
  const clusters = Cluster.useSelectMany();

  useEffect(() => {
    onOpenUrl((urls) => {
      if (client == null) {
        console.error("Error: Cannot open URL, client is null");
        return;
      }
      const scheme = "synnax://";
      if (urls.length === 0 || !urls[0].startsWith(scheme)) {
        console.error("Error: Cannot open URL, URLs must start with synnax://");
        return;
      }
      const urlParts = urls[0].slice(scheme.length).split("/");
      if (
        !Cluster.linkHandler({
          resource: urlParts[0],
          resourceKey: urlParts[1],
          client,
          dispatch,
          placer,
          clusters,
        })
      ) {
        return;
      }
      handlers.find((h) =>
        h({
          resource: urlParts[2],
          resourceKey: urlParts[3],
          client,
          dispatch,
          placer,
          clusters,
        }),
      );
    });
  }, [client]);
};

export const HANDLERS: Handler[] = [
  Cluster.linkHandler,
  Schematic.linkHandler,
  Range.linkHandler,
  Workspace.linkHandler,
  LinePlot.linkHandler,
];
