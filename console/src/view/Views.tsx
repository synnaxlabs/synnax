// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type view } from "@synnaxlabs/client";
import { Button, Flex, Icon, List, View as PView } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback } from "react";

import { type Request, useContext } from "@/view/context";

export const Views = (): ReactElement | null => {
  const { editable, resourceType, onRequestChange } = useContext("Views");
  const listProps = PView.useList({ initialQuery: { types: [resourceType] } });

  const handleSelectView = useCallback(
    (v: view.View) => {
      onRequestChange(v.query as Request);
    },
    [onRequestChange],
  );
  console.log("VIEWS", listProps.data);

  if (!editable || listProps.data.length === 0) return null;

  return (
    <List.Frame<view.Key, view.View> {...listProps}>
      <List.Items<view.Key, view.View>
        displayItems={Infinity}
        x
        gap="medium"
        bordered
        align="center"
        style={itemsStyle}
      >
        {({ key, ...rest }) => (
          <ViewItem key={key} {...rest} onSelectView={handleSelectView} />
        )}
      </List.Items>
    </List.Frame>
  );
};

const itemsStyle = { padding: "1rem 1.5rem" } as const;

interface ViewItemProps extends List.ItemProps<string> {
  onSelectView: (view: view.View) => void;
}

const ViewItem = ({ itemKey, onSelectView }: ViewItemProps): ReactElement | null => {
  const query = PView.useRetrieve({ key: itemKey });
  const { update: del } = PView.useDelete();
  if (query.variant !== "success") return null;
  const view = query.data;
  return (
    <Flex.Box x pack>
      <Button.Button onClick={() => onSelectView(view)}>{view.name}</Button.Button>
      <Button.Button onClick={() => del(itemKey)}>
        <Icon.Delete />
      </Button.Button>
    </Flex.Box>
  );
};
