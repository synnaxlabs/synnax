// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { Component } from "@synnaxlabs/lyra/component";
import { type Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { List } from "@synnaxlabs/lyra/list";
import { Select as Base } from "@synnaxlabs/lyra/select";
import { Telem } from "@synnaxlabs/lyra/telem";
import { type ReactElement } from "react";

import { CSS } from "@/css";
import { type Flux } from "@/flux";
import { Label } from "@/label";
import { Breadcrumb } from "@/ranger/Breadcrumb";
import { type ListQuery, useList, useResult } from "@/ranger/queries";
import { HAUL_TYPE } from "@/ranger/types";

export interface SelectProps
  extends
    Omit<
      Base.SingleProps<ranger.Key, ranger.Payload | undefined>,
      | "data"
      | "getItem"
      | "subscribe"
      | "status"
      | "onFetchMore"
      | "onSearch"
      | "children"
      | "resourceName"
    >,
    Flux.UseListParams<ListQuery, ranger.Key, ranger.Payload> {}

export const Select = ({ filter, query, ...rest }: SelectProps): ReactElement => {
  const { data, retrieve, subscribe, getItem, status } = useList({
    filter,
    query,
  });
  const { fetchMore, search } = List.usePager({ retrieve });
  return (
    <Base.Single<ranger.Key, ranger.Payload | undefined>
      variant="modal"
      haulType={HAUL_TYPE}
      icon={<Icon.Range />}
      itemHeight={45}
      dialogProps={DIALOG_PROPS}
      {...rest}
      resourceName="range"
      data={data}
      subscribe={subscribe}
      getItem={getItem}
      status={status}
      onFetchMore={fetchMore}
      onSearch={search}
    >
      {listItemRenderProp}
    </Base.Single>
  );
};

const DIALOG_PROPS: Dialog.DialogProps = { style: { width: 800 } };

interface ListItemProps extends List.ItemProps<ranger.Key> {
  showParent?: boolean;
  showLabels?: boolean;
}

const ListItem = ({
  className,
  itemKey,
  showParent = true,
  showLabels = true,
  ...rest
}: ListItemProps): ReactElement | null => {
  const item = List.useItem<ranger.Key, ranger.Payload>(itemKey);
  const parentKey = showParent ? item?.parent : undefined;
  const parent = useResult(parentKey == null ? null : { key: parentKey }).data;
  if (item == null) return null;
  const { name, timeRange } = item;
  return (
    <Base.Item
      className={CSS.cls(CSS.BE("range", "list-item"), className)}
      itemKey={itemKey}
      justify="between"
      {...rest}
    >
      <Breadcrumb
        name={name}
        parent={parent}
        showParent={showParent}
        timeRange={timeRange}
      />
      <Flex.Box x>
        {showLabels && <Label.Tags keys={item.labels} />}
        <Telem.Text.TimeRange level="tiny">{timeRange}</Telem.Text.TimeRange>
      </Flex.Box>
    </Base.Item>
  );
};

const listItemRenderProp = Component.renderProp(ListItem);
