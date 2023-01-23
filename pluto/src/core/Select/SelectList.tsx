// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { RenderableRecord } from "@synnaxlabs/x";

import { List, ListColumnHeaderProps, ListSelectorProps } from "@/core/List";

export interface SelectListProps<E extends RenderableRecord<E>>
  extends ListSelectorProps<E>,
    ListColumnHeaderProps<E> {}

export const SelectList = <E extends RenderableRecord>({
  value,
  onChange,
  allowMultiple,
  ...props
}: SelectListProps<E>): JSX.Element => (
  <>
    <List.Selector value={value} onChange={onChange} allowMultiple={allowMultiple} />
    <List.Column.Header {...props} />
    <List.Core.Virtual itemHeight={List.Column.itemHeight}>
      {List.Column.Item}
    </List.Core.Virtual>
  </>
);
