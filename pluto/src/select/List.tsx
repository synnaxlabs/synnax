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

import { CSS } from "@/css";
import { Dropdown } from "@/dropdown";
import { List as CoreList } from "@/list";

export type SelectListProps<
  K extends Key = Key,
  E extends Keyed<K> = Keyed<K>,
> = CoreList.SelectorProps<K, E> &
  Pick<Partial<CoreList.ColumnHeaderProps<K, E>>, "columns"> &
  Omit<Dropdown.DialogProps, "onChange" | "children"> &
  Partial<Pick<CoreList.VirtualCoreProps<K, E>, "itemHeight">> &
  PropsWithChildren<{}> & {
    emptyContent?: ReactElement;
    hideColumnHeader?: boolean;
    data?: E[];
    omit?: K[];
    listItem?: CoreList.VirtualCoreProps<K, E>["children"];
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
  children,
  columns = DEFAULT_COLUMNS,
  visible,
  itemHeight = CoreList.Column.itemHeight,
  listItem = CoreList.Column.Item<K, E>,
  replaceOnSingle,
  omit,
  ...props
}: SelectListProps<K, E>): ReactElement => (
  <CoreList.List<K, E> data={data} emptyContent={emptyContent} omit={omit}>
    {/* @ts-expect-error - selector compatibility with generic props */}
    <CoreList.Selector<K, E>
      value={value}
      onChange={onChange}
      allowMultiple={allowMultiple}
      allowNone={allowNone}
      replaceOnSingle={replaceOnSingle}
    >
      <Dropdown.Dialog
        visible={visible}
        className={CSS.B("select")}
        keepMounted={false}
        {...props}
      >
        {children}
        <CoreList.Hover<K, E> disabled={!visible}>
          <CoreList.Column.Header
            hide={hideColumnHeader || listItem != null}
            columns={columns}
          >
            <CoreList.Core.Virtual<K, E> itemHeight={itemHeight}>
              {listItem}
            </CoreList.Core.Virtual>
          </CoreList.Column.Header>
        </CoreList.Hover>
      </Dropdown.Dialog>
    </CoreList.Selector>
  </CoreList.List>
);
