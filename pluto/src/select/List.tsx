// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { forwardRef, type PropsWithChildren, type ReactElement } from "react";

import { type Key, type KeyedRenderableRecord } from "@synnaxlabs/x";

import { CSS } from "@/css";
import { Dropdown } from "@/dropdown";
import { List as CoreList } from "@/list";
import { componentRenderProp } from "@/util/renderProp";

export interface SelectListProps<
  K extends Key = Key,
  E extends KeyedRenderableRecord<K, E> = KeyedRenderableRecord<K>,
> extends CoreList.SelectorProps<K, E>,
    Pick<CoreList.ColumnHeaderProps<K, E>, "columns">,
    Omit<Dropdown.DialogProps, "onChange" | "children">,
    PropsWithChildren<{}> {
  data?: E[];
  emtpyContent?: ReactElement;
  hideColumnHeader?: boolean;
}

const CoreBase = forwardRef<HTMLDivElement, SelectListProps>(
  (
    {
      data,
      emtpyContent,
      value,
      onChange,
      allowMultiple,
      allowNone,
      hideColumnHeader = false,
      children,
      columns,
      visible,
      replaceOnSingle,
      ...props
    }: SelectListProps,
    ref,
  ): ReactElement => (
    <CoreList.List data={data} emptyContent={emtpyContent}>
      <CoreList.Selector
        value={value}
        onChange={onChange}
        allowMultiple={allowMultiple}
        allowNone={allowNone}
        replaceOnSingle={replaceOnSingle}
      >
        <Dropdown.Dialog
          ref={ref}
          visible={visible}
          className={CSS.B("select")}
          {...props}
          matchTriggerWidth
        >
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
  ),
);
CoreBase.displayName = "Select.Core";

export const Core = CoreBase as <K extends Key, E extends KeyedRenderableRecord<K, E>>(
  props: SelectListProps<K, E> & { ref?: React.Ref<HTMLDivElement> },
) => ReactElement;
