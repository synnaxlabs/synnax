// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/select/Dialog.css";

import { caseconv, type record, type status } from "@synnaxlabs/x";
import { memo, type ReactElement, useMemo } from "react";

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
  resourceName?: string;
}

export interface DefaultEmptyContentProps extends Status.TextProps {
  resourceName?: string;
}

const DefaultEmptyContent = ({ resourceName = "result" }: DefaultEmptyContentProps) => (
  <Status.Text center variant="disabled" hideIcon>
    No {resourceName}s found
  </Status.Text>
);

export const Core = memo(
  <K extends record.Key>({
    onSearch,
    children,
    emptyContent,
    searchPlaceholder,
    status,
    resourceName = "result",
    actions,
    ...rest
  }: DialogProps<K>) => {
    emptyContent = useMemo(() => {
      if (status != null && status.variant !== "success")
        return (
          <Status.Text
            center
            variant={status?.variant}
            description={status?.description}
          >
            {status?.message}
          </Status.Text>
        );
      if (typeof emptyContent === "string")
        return (
          <Status.Text center variant="disabled">
            {emptyContent}
          </Status.Text>
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
            searchPlaceholder={searchPlaceholder}
            actions={actions}
          />
        )}
        <List.Items emptyContent={emptyContent} bordered borderColor={6} grow rounded>
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
