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

const DEFAULT_HEIGHT = 250;

const defaultEmptyContent = (
  <Status.Text.Centered variant="disabled" style={{ height: DEFAULT_HEIGHT }}>
    No results
  </Status.Text.Centered>
);

export const Dialog = <K extends record.Key>({
  onSearch,
  children,
  emptyContent = defaultEmptyContent,
  searchPlaceholder,
  style,
  status,
  ...rest
}: DialogProps<K>) => {
  if (status?.variant !== "success")
    emptyContent = (
      <Status.Text.Centered
        variant={status?.variant}
        style={{ height: DEFAULT_HEIGHT }}
      >
        {status?.message}
      </Status.Text.Centered>
    );
  return (
    <CoreDialog.Dialog {...rest} style={{ ...style }}>
      {onSearch != null && (
        <SearchInput onSearch={onSearch} searchPlaceholder={searchPlaceholder} />
      )}
      <List.Items
        emptyContent={emptyContent}
        bordered
        borderShade={6}
        grow
        style={{ maxHeight: DEFAULT_HEIGHT }}
      >
        {children}
      </List.Items>
    </CoreDialog.Dialog>
  );
};
