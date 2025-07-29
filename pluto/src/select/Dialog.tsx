// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/select/Dialog.css";

import { type record, type status } from "@synnaxlabs/x";

import { CSS } from "@/css";
import { Dialog as CoreDialog } from "@/dialog";
import { List } from "@/list";
import { SearchInput, type SearchInputProps } from "@/select/SearchInput";
import { Status } from "@/status";

export interface DialogProps<K extends record.Key>
  extends Omit<CoreDialog.DialogProps, "children">,
    SearchInputProps,
    Pick<List.ItemsProps<K>, "emptyContent" | "children"> {
  status?: status.Status;
}

const DEFAULT_HEIGHT = 250;

const DefaultEmptyContent = () => (
  <Status.Text.Centered variant="disabled" style={{ height: DEFAULT_HEIGHT }}>
    No results
  </Status.Text.Centered>
);

export const Dialog = <K extends record.Key>({
  onSearch,
  children,
  emptyContent = <DefaultEmptyContent />,
  searchPlaceholder,
  style,
  status,
  actions,
  ...rest
}: DialogProps<K>) => {
  const { variant } = CoreDialog.useContext();
  if (status != null && status.variant !== "success")
    emptyContent = (
      <Status.Text.Centered
        variant={status?.variant}
        style={{ height: DEFAULT_HEIGHT }}
        description={status?.description}
      >
        {status?.message}
      </Status.Text.Centered>
    );
  else if (typeof emptyContent === "string")
    emptyContent = (
      <Status.Text.Centered variant="disabled" style={{ height: DEFAULT_HEIGHT }}>
        {emptyContent}
      </Status.Text.Centered>
    );
  return (
    <CoreDialog.Dialog
      {...rest}
      style={{ ...style }}
      className={CSS.BE("select", "dialog")}
    >
      {onSearch != null && (
        <SearchInput
          dialogVariant={variant}
          onSearch={onSearch}
          searchPlaceholder={searchPlaceholder}
          actions={actions}
        />
      )}
      <List.Items
        emptyContent={emptyContent}
        bordered
        borderShade={6}
        style={{ maxHeight: DEFAULT_HEIGHT }}
      >
        {children}
      </List.Items>
    </CoreDialog.Dialog>
  );
};
