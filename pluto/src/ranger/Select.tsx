// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Component } from "@/component";
import { CSS } from "@/css";
import { type Dialog } from "@/dialog";
import { Flex } from "@/flex";
import { type Flux } from "@/flux";
import { Icon } from "@/icon";
import { List } from "@/list";
import { Breadcrumb } from "@/ranger/Breadcrumb";
import { type ListQuery, useList } from "@/ranger/queries";
import { HAUL_TYPE } from "@/ranger/types";
import { Select as Core } from "@/select";
import { Tag } from "@/tag";
import { Telem } from "@/telem";

export interface SelectProps
  extends
    Omit<
      Core.SingleProps<ranger.Key, ranger.Payload | undefined>,
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

export const Select = ({
  filter,
  initialQuery,
  ...rest
}: SelectProps): ReactElement => {
  const { data, retrieve, subscribe, getItem, status } = useList({
    filter,
    initialQuery,
  });
  const { fetchMore, search } = List.usePager({ retrieve });
  return (
    <Core.Single<ranger.Key, ranger.Payload | undefined>
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
    </Core.Single>
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
  if (item == null) return null;
  const { name, timeRange, parent, labels } = item;
  return (
    <Core.ListItem
      className={CSS(CSS.BE("range", "list-item"), className)}
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
        {showLabels && labels != null && labels.length > 0 && (
          <>
            {labels.map(({ key, name, color }) => (
              <Tag.Tag key={key} color={color} size="small">
                {name}
              </Tag.Tag>
            ))}
          </>
        )}
        <Telem.Text.TimeRange level="small">{timeRange}</Telem.Text.TimeRange>
      </Flex.Box>
    </Core.ListItem>
  );
};

const listItemRenderProp = Component.renderProp(ListItem);
