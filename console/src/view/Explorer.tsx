// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type List,
  type state,
  Status,
  Synnax,
  View as PView,
} from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { useEffect, useState } from "react";

import { View, type ViewProps } from "./View";

export interface ExplorerProps<K extends record.Key, E extends record.Keyed<K>>
  extends Omit<
    ViewProps<K, E>,
    | "initialRequest"
    | "request"
    | "onRequestChange"
    | "onCreateView"
    | "hasSaveView"
    | "initialEditable"
  > {}

interface RequestState extends List.PagerParams {
  [x: string]: unknown;
}

export const Explorer = <K extends record.Key, E extends record.Keyed<K>>(
  props: ExplorerProps<K, E>,
) => {
  const [request, setRequest] = useState<RequestState>({});
  const client = Synnax.use();
  const handleError = Status.useErrorHandler();
  const handleSaveView = () => {
    if (client == null) return;
    handleError(async () => {
      const vw = await client.views.create({
        name: `View for ${props.resourceType}`,
        type: props.resourceType,
        query: request,
      });
      console.log(vw);
    });
  };
  const handleRequestChange = (setter: state.SetArg<List.PagerParams>) => {
    setRequest({ ...setter });
  };
  const views = PView.useList({
    initialQuery: {
      types: [props.resourceType],
    },
  });
  client?.views
    .retrieve({
      types: [props.resourceType],
    })
    .then((vw) => {
      console.log(`views retrieved for ${props.resourceType}`, vw);
    })
    .catch((err) => {
      console.error("error retrieving views", err);
    });
  useEffect(() => {
    console.log("views", views.data);
  }, [views.data.length]);

  return (
    <View
      {...props}
      request={request}
      onRequestChange={handleRequestChange}
      initialEditable={false}
      onCreateView={handleSaveView}
      showViews
      hasSaveView
    />
  );
};
