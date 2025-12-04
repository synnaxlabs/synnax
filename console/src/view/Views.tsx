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

import { useContext } from "@/view/context";
import { type Query, type UseQueryReturn } from "@/view/useQuery";

export interface ViewsProps<Q extends Query>
  extends Pick<UseQueryReturn<Q>, "onQueryChange"> {}

export const Views = <Q extends Query>({
  onQueryChange,
}: ViewsProps<Q>): ReactElement | null => {
  const { editable, resourceType } = useContext("View.Views");
  const query = PView.useRetrieveMultiple({ types: [resourceType] });

  const handleSelectView = useCallback(
    (v: view.View) => {
      onQueryChange(v.query as Q);
    },
    [onQueryChange],
  );
  if (!editable) return null;
  if (query.variant !== "success" || query.data.length === 0) return null;

  return (
    <List.Frame<view.Key, view.View> data={query.data.map((v) => v.key)}>
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

interface ViewItemProps extends List.ItemProps<view.Key> {
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
