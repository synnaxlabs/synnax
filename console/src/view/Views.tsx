// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type view } from "@synnaxlabs/client";
import { Button, Icon, Select, Text, View as PView } from "@synnaxlabs/pluto";
import { type ReactElement, useCallback, useState } from "react";

import { useContext } from "@/view/context";
import { type Query, type UseQueryReturn } from "@/view/useQuery";

export interface ViewsProps<Q extends Query>
  extends Pick<UseQueryReturn<Q>, "onQueryChange" | "resetQuery"> {}

export const Views = <Q extends Query>({
  onQueryChange,
  resetQuery,
}: ViewsProps<Q>): ReactElement | null => {
  const { editable, resourceType } = useContext("View.Views");
  const query = PView.useList({ initialQuery: { types: [resourceType] } });
  const [selected, setSelected] = useState<view.Key>();
  const handleSelectView = useCallback(
    (view: view.Key) => {
      const v = query.getItem(view);
      if (v == null) {
        setSelected(undefined);
        resetQuery();
        return;
      }
      setSelected(view);
      onQueryChange(v.query as Q);
    },
    [onQueryChange, resetQuery],
  );
  if (!editable || query.variant !== "success" || query.data.length === 0) return null;
  return (
    <Select.Buttons
      keys={query.data}
      x
      value={selected}
      multiple={false}
      pack={false}
      gap="medium"
      style={buttonsStyle}
      onChange={handleSelectView}
      allowNone
    >
      {query.data.map((key) => (
        <ViewItem key={key} itemKey={key} />
      ))}
    </Select.Buttons>
  );
};

const buttonsStyle = { padding: "1rem 1.5rem", overflow: "scroll" } as const;

interface ViewItemProps {
  itemKey: view.Key;
}

const ViewItem = ({ itemKey }: ViewItemProps): ReactElement | null => {
  const query = PView.useRetrieve({ key: itemKey });
  const { update: del } = PView.useDelete();
  if (query.variant !== "success") return null;
  return (
    <Select.Button itemKey={itemKey} justify="between" style={viewItemStyle}>
      <Text.Text>{query.data.name}</Text.Text>
      <Button.Button onClick={() => del(itemKey)} size="small">
        <Icon.Delete />
      </Button.Button>
    </Select.Button>
  );
};

const viewItemStyle = { padding: "0.5rem 1rem" } as const;
