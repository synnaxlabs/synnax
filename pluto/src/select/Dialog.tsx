// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";

import { Dialog as CoreDialog } from "@/dialog";
import { List } from "@/list";
import {
  SearchInput,
  type SearchInputProps,
  type SearchParams,
} from "@/select/SearchInput";

export interface DialogProps<K extends record.Key, P extends SearchParams>
  extends Omit<CoreDialog.DialogProps, "children">,
    SearchInputProps<P>,
    Pick<List.ItemsProps<K>, "emptyContent" | "children"> {}

export const Dialog = <K extends record.Key, P extends SearchParams>({
  onSearch,
  children,
  emptyContent,
  searchPlaceholder,
  ...rest
}: DialogProps<K, P>) => (
  <CoreDialog.Dialog {...rest}>
    {onSearch != null && (
      <SearchInput onSearch={onSearch} searchPlaceholder={searchPlaceholder} />
    )}
    <List.Items emptyContent={emptyContent} bordered borderShade={6}>
      {children}
    </List.Items>
  </CoreDialog.Dialog>
);
