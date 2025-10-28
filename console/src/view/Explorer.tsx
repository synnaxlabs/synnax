// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type view } from "@synnaxlabs/client";
import {
  Button,
  Flex,
  Icon,
  List,
  type state,
  Status,
  Synnax,
  Text,
  View as PView,
} from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { useState } from "react";

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
  return (
    <View
      {...props}
      request={request}
      onRequestChange={handleRequestChange}
      initialEditable={false}
      onCreateView={handleSaveView}
      views={<Views resourceType={props.resourceType} />}
      hasSaveView
    />
  );
};

interface ViewsProps {
  resourceType: string;
}

const Views = ({ resourceType }: ViewsProps) => {
  const listProps = PView.useList({ initialQuery: { types: [resourceType] } });
  return (
    <Flex.Box x bordered align="center">
      <Text.Text level="p" style={{ padding: "2rem" }}>
        <Icon.View />
        Views
      </Text.Text>
      <List.Frame<string, view.View> {...listProps}>
        <List.Items<string, view.View> displayItems={Infinity} x>
          {({ key, ...rest }) => <Item key={key} {...rest} />}
        </List.Items>
      </List.Frame>
    </Flex.Box>
  );
};

const Item = (props: List.ItemProps<string>) => {
  const { getItem } = List.useUtilContext<string, view.View>();
  const view = getItem?.(props.itemKey);
  return (
    <Button.Button level="small" bordered={false}>
      {view?.name}
    </Button.Button>
  );
};
