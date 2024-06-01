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
import { componentRenderProp } from "@/util/renderProp";

export type SelectListProps<
  K extends Key = Key,
  E extends Keyed<K> = Keyed<K>,
> = CoreList.SelectorProps<K, E> &
  Pick<Partial<CoreList.ColumnHeaderProps<K, E>>, "columns"> &
  Omit<Dropdown.DialogProps, "onChange" | "children"> &
  PropsWithChildren<{}> & {
    emptyContent?: ReactElement;
    hideColumnHeader?: boolean;
    data?: E[];
    omit?: K[];
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
  replaceOnSingle,
  omit,
  ...props
}: SelectListProps<K, E>): ReactElement => (
  <CoreList.List data={data} emptyContent={emptyContent} omit={omit}>
    {/* @ts-expect-error - selector compatibility with generic props */}
    <CoreList.Selector
      value={value}
      onChange={onChange}
      allowMultiple={allowMultiple}
      allowNone={allowNone}
      replaceOnSingle={replaceOnSingle}
    >
      <Dropdown.Dialog visible={visible} className={CSS.B("select")} {...props}>
        {children}
        <CoreList.Hover disabled={!visible}>
          <CoreList.Column.Header hide={hideColumnHeader} columns={columns}>
            <CoreList.Core.Virtual itemHeight={CoreList.Column.itemHeight}>
              {componentRenderProp(CoreList.Column.Item)}
            </CoreList.Core.Virtual>
          </CoreList.Column.Header>
        </CoreList.Hover>
      </Dropdown.Dialog>
    </CoreList.Selector>
  </CoreList.List>
);
