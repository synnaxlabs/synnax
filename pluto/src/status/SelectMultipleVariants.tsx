// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/x/status";
import { Component } from "@synnaxlabs/lyra/component";
import { List } from "@synnaxlabs/lyra/list";
import { Select } from "@synnaxlabs/lyra/select";
import { Status } from "@synnaxlabs/lyra/status";

import { type ReactElement } from "react";

import { VARIANT_DATA } from "@/status/variantData";

type Entry = Select.StaticEntry<status.Variant>;

const listItem = Component.renderProp((p: List.ItemProps<string>) => {
  const { itemKey } = p;
  const item = List.useItem<string, Entry>(itemKey);
  if (item == null) return null;
  const { name, icon } = item;
  return (
    <Select.ListItem {...p}>
      {icon}
      {name}
    </Select.ListItem>
  );
});

export interface SelectMultipleVariantProps extends Omit<
  Select.MultipleProps<status.Variant, Entry>,
  "data" | "getItem" | "subscribe" | "children" | "resourceName" | "onSearch"
> {}

export const SelectMultipleVariants = (
  props: SelectMultipleVariantProps,
): ReactElement => {
  const { retrieve, ...listProps } = List.useStaticData<status.Variant, Entry>({
    data: VARIANT_DATA,
  });
  const { search } = List.usePager({ retrieve });
  return (
    <Select.Multiple<status.Variant, Entry>
      {...props}
      {...listProps}
      onSearch={search}
      virtual={false}
      resourceName="variant"
      icon={icon}
    >
      {listItem}
    </Select.Multiple>
  );
};

const icon = <Status.Indicator variant="success" />;
