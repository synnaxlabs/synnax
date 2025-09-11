// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/ranger/Select.css";

import { type ranger } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Component } from "@/component";
import { type Dialog } from "@/dialog";
import { type Flux } from "@/flux";
import { Icon } from "@/icon";
import { List } from "@/list";
import { ListItem } from "@/ranger/ListItem";
import { type ListParams, useList } from "@/ranger/queries";
import { HAUL_TYPE } from "@/ranger/types";
import { Select as Core } from "@/select";

const listItemRenderProp = Component.renderProp(ListItem);

export interface SelectProps
  extends Omit<
      Core.SingleProps<ranger.Key, ranger.Payload | undefined>,
      "resourceName" | "data" | "getItem" | "subscribe" | "children"
    >,
    Flux.UseListArgs<ListParams, ranger.Key, ranger.Payload> {}

const DIALOG_PROPS: Dialog.DialogProps = { style: { width: 800 } };

export const Select = ({
  onChange,
  value,
  filter,
  allowNone,
  emptyContent,
  initialParams,
  ...rest
}: SelectProps): ReactElement => {
  const { data, retrieve, subscribe, getItem, status } = useList({
    filter,
    initialParams,
  });
  const { fetchMore, search } = List.usePager({ retrieve });
  return (
    <Core.Single<ranger.Key, ranger.Payload | undefined>
      resourceName="Range"
      variant="modal"
      value={value}
      onChange={onChange}
      data={data}
      allowNone={allowNone}
      haulType={HAUL_TYPE}
      onFetchMore={fetchMore}
      getItem={getItem}
      subscribe={subscribe}
      status={status}
      onSearch={search}
      emptyContent={emptyContent}
      icon={<Icon.Range />}
      itemHeight={45}
      dialogProps={DIALOG_PROPS}
      {...rest}
    >
      {listItemRenderProp}
    </Core.Single>
  );
};
