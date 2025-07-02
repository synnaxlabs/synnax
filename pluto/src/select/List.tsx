// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Align } from "@/align";
import { CSS } from "@/css";
import { Dropdown } from "@/dropdown";
import { List as CoreList } from "@/list/v2";
import { componentRenderProp } from "@/util/renderProp";

export type SelectListProps<
  K extends record.Key = record.Key,
  E extends record.Keyed<K> = record.Keyed<K>,
> = {
  emptyContent?: ReactElement;
  hideColumnHeader?: boolean;
  data: K[];
  listItem?: CoreList.Item;
  trigger?: ReactElement;
  extraDialogContent?: ReactElement;
};

export const Core = <K extends record.Key, E extends record.Keyed<K>>({
  data,
  emptyContent,
  value,
  onChange,
  allowMultiple,
  allowNone,
  visible,
  itemHeight = CoreList.Column.itemHeight,
  listItem = componentRenderProp(CoreList.Column.Item) as CoreList.VirtualCoreProps<
    K,
    E
  >["children"],
  replaceOnSingle,
  autoSelectOnNone,
  trigger,
  extraDialogContent,
  variant = "connected",
  ...rest
}: SelectListProps<K, E>): ReactElement => {
  let dialogContent = (
    <CoreList.Hover disabled={!visible}>
      {extraDialogContent}
      <CoreList.Items<K> data={data} itemHeight={itemHeight}>
        {listItem}
      </CoreList.Items>
    </CoreList.Hover>
  );

  if (variant !== "connected")
    dialogContent = (
        {dialogContent}
      </Align.Pack>
    );
  return (
    <Dropdown.Dialog
      visible={visible}
      className={CSS.B("select")}
      keepMounted={false}
      variant={variant}
      {...rest}
    >
      {trigger}
      {dialogContent}
    </Dropdown.Dialog>
  );
};
