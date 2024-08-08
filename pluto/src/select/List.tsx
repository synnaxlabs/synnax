// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Key, type Keyed } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement } from "react";

import { Align } from "@/align";
import { CSS } from "@/css";
import { Dropdown } from "@/dropdown";
import { List as CoreList } from "@/list";
import { componentRenderProp } from "@/util/renderProp";

export type SelectListProps<K extends Key = Key, E extends Keyed<K> = Keyed<K>> = Omit<
  CoreList.SelectorProps<K, E>,
  "children"
> &
  Pick<Partial<CoreList.ColumnHeaderProps<K, E>>, "columns"> &
  Omit<Dropdown.DialogProps, "onChange" | "children"> &
  Partial<Pick<CoreList.VirtualCoreProps<K, E>, "itemHeight">> & {
    emptyContent?: ReactElement;
    hideColumnHeader?: boolean;
    data?: E[];
    omit?: K[];
    listItem?: CoreList.VirtualCoreProps<K, E>["children"];
    trigger?: ReactElement;
    extraDialogContent?: ReactElement;
  };

const DEFAULT_COLUMNS: CoreList.ColumnSpec[] = [];

export const Core = <K extends Key, E extends Keyed<K>>({
  data,
  emptyContent,
  value,
  onChange,
  allowMultiple,
  allowNone,
  hideColumnHeader = false,
  columns = DEFAULT_COLUMNS,
  visible,
  itemHeight = CoreList.Column.itemHeight,
  listItem = componentRenderProp(CoreList.Column.Item) as CoreList.VirtualCoreProps<
    K,
    E
  >["children"],
  replaceOnSingle,
  omit,
  autoSelectOnNone,
  trigger,
  extraDialogContent,
  variant,
  ...props
}: SelectListProps<K, E>): ReactElement => {
  let dialogContent = (
    <CoreList.Hover<K, E> disabled={!visible}>
      {extraDialogContent}
      <CoreList.Column.Header
        hide={hideColumnHeader || listItem != null}
        columns={columns}
      >
        <CoreList.Core.Virtual<K, E> itemHeight={itemHeight}>
          {listItem}
        </CoreList.Core.Virtual>
      </CoreList.Column.Header>
    </CoreList.Hover>
  );

  if (variant !== "connected") {
    dialogContent = (
      <Align.Pack direction="y" className={CSS.B("select-dialog-content")}>
        {dialogContent}
      </Align.Pack>
    );
  }

  return (
    <CoreList.List<K, E> data={data} emptyContent={emptyContent} omit={omit}>
      {/* @ts-expect-error - selector compatibility with generic props */}
      <CoreList.Selector<K, E>
        value={value}
        onChange={onChange}
        allowMultiple={allowMultiple}
        allowNone={allowNone}
        replaceOnSingle={replaceOnSingle}
        autoSelectOnNone={autoSelectOnNone}
      >
        <Dropdown.Dialog
          visible={visible}
          className={CSS.B("select")}
          keepMounted={false}
          variant={variant}
          {...props}
        >
          {trigger}
          {dialogContent}
        </Dropdown.Dialog>
      </CoreList.Selector>
    </CoreList.List>
  );
};
