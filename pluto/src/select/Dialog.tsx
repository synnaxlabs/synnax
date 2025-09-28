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
import { memo, type ReactElement, useMemo } from "react";

import { CSS } from "@/css";
import { Dialog as CoreDialog } from "@/dialog";
import { List } from "@/list";
import { SearchInput, type SearchInputProps } from "@/select/SearchInput";
import { Status } from "@/status/core";
import { Text } from "@/text";

export interface DialogProps<K extends record.Key>
  extends Omit<CoreDialog.DialogProps, "children">,
    Omit<SearchInputProps, "searchPlaceholder">,
    Pick<List.ItemsProps<K>, "emptyContent" | "children"> {
  status?: status.Status;
  resourceName?: string;
}

export interface DefaultEmptyContentProps extends Status.SummaryProps {
  resourceName?: string;
}

const DefaultEmptyContent = ({ resourceName = "result" }: DefaultEmptyContentProps) => (
  <Text.Text center status="disabled">
    No {resourceName.toLowerCase()}s found
  </Text.Text>
);

export const Core = memo(
  <K extends record.Key>({
    onSearch,
    children,
    emptyContent,
    status,
    resourceName = "result",
    actions,
    ...rest
  }: DialogProps<K>) => {
    emptyContent = useMemo(() => {
      if (status != null && status.variant !== "success")
        return (
          <Status.Summary
            center
            variant={status?.variant}
            description={status?.description}
          >
            {status?.message}
          </Status.Summary>
        );
      if (typeof emptyContent === "string")
        return (
          <Status.Summary center variant="disabled">
            {emptyContent}
          </Status.Summary>
        );
      if (emptyContent == null)
        return <DefaultEmptyContent resourceName={resourceName} />;
      return emptyContent;
    }, [status?.key, emptyContent]);
    return (
      <CoreDialog.Dialog
        {...rest}
        className={CSS.BE("select", "dialog")}
        bordered={false}
      >
        {onSearch != null && (
          <SearchInput
            dialogVariant="floating"
            onSearch={onSearch}
            searchPlaceholder={`Search ${resourceName}s...`}
            actions={actions}
          />
        )}
        <List.Items
          emptyContent={emptyContent}
          bordered
          borderColor={6}
          grow
          rounded
          full="x"
        >
          {children}
        </List.Items>
      </CoreDialog.Dialog>
    );
  },
);
Core.displayName = "Select.Dialog";
export const Dialog = Core as <K extends record.Key>(
  props: DialogProps<K>,
) => ReactElement;
