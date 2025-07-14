// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record, type status } from "@synnaxlabs/x";

import { Dialog as CoreDialog } from "@/dialog";
import { List } from "@/list";
import { SearchInput, type SearchInputProps } from "@/select/SearchInput";
import { Status } from "@/status";

export interface DialogProps<K extends record.Key>
  extends Omit<CoreDialog.DialogProps, "children">,
    SearchInputProps,
    Pick<List.ItemsProps<K>, "emptyContent" | "children"> {
  status?: Pick<status.Status, "variant" | "message">;
}

export const Dialog = <K extends record.Key>({
  onSearch,
  children,
  emptyContent,
  searchPlaceholder,
  style,
  status,
  ...rest
}: DialogProps<K>) => {
  if (status?.variant !== "success")
    emptyContent = (
      <Status.Text variant={status?.variant}>{status?.message}</Status.Text>
    );
  return (
    <CoreDialog.Dialog {...rest} style={{ ...style }}>
      {onSearch != null && (
        <SearchInput onSearch={onSearch} searchPlaceholder={searchPlaceholder} />
      )}
      <List.Items emptyContent={emptyContent} bordered borderShade={6}>
        {children}
      </List.Items>
    </CoreDialog.Dialog>
  );
};
