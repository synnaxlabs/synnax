// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { type Key, type KeyedRenderableRecord } from "@synnaxlabs/x";

import { type Dropdown } from "@/dropdown";
import { List as CoreList } from "@/list";
import { componentRenderProp } from "@/util/renderProp";

export interface SelectListProps<K extends Key, E extends KeyedRenderableRecord<K, E>>
  extends CoreList.SelectorProps<K, E>,
    Omit<CoreList.ColumnHeaderProps<K, E>, "hide">,
    Pick<Dropdown.DialogProps, "visible"> {
  hideColumnHeader?: boolean;
}

export const List = <K extends Key, E extends KeyedRenderableRecord<K, E>>({
  value,
  onChange,
  allowMultiple,
  visible,
  allowNone,
  hideColumnHeader = false,
  ...props
}: SelectListProps<K, E>): ReactElement => (
  <>
    <CoreList.Selector
      value={value}
      onChange={onChange}
      allowMultiple={allowMultiple}
      allowNone={allowNone}
    />
    {visible && <CoreList.Hover />}
    <CoreList.Column.Header hide={hideColumnHeader} {...props} />
    <CoreList.Core.Virtual itemHeight={CoreList.Column.itemHeight}>
      {componentRenderProp(CoreList.Column.Item)}
    </CoreList.Core.Virtual>
  </>
);
