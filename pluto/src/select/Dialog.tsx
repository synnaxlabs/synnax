// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/select/Dialog.css";

import { type status } from "@synnaxlabs/client";
import { type record } from "@synnaxlabs/x";
import { plural } from "pluralize";
import { memo, type ReactElement, type ReactNode, useMemo } from "react";
import { type z } from "zod";

import { CSS } from "@/css";
import { Dialog as BaseDialog } from "@/dialog";
import { Flex } from "@/flex";
import { List } from "@/list";
import { SearchInput, type SearchInputProps } from "@/select/SearchInput";
import { Status } from "@/status/base";
import { Text } from "@/text";

export interface DialogProps<K extends record.Key>
  extends
    Omit<BaseDialog.DialogProps, "children">,
    Omit<SearchInputProps, "searchPlaceholder">,
    Pick<List.ItemsProps<K>, "emptyContent" | "children"> {
  status?: status.Status<z.ZodNever>;
  resourceName: string;
  /** Pinned below the scrollable list; stays visible regardless of list length. */
  footer?: ReactNode;
}

export interface DefaultEmptyContentProps extends Status.SummaryProps {
  resourceName: string;
}

const DefaultEmptyContent = ({ resourceName }: DefaultEmptyContentProps) => (
  <Text.Text center status="disabled">
    No {plural(resourceName)} found
  </Text.Text>
);

const Base = memo(
  <K extends record.Key>({
    onSearch,
    children,
    emptyContent,
    status,
    resourceName,
    actions,
    footer,
    className,
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
      <BaseDialog.Dialog
        {...rest}
        className={CSS(CSS.BE("select", "dialog"), className)}
        bordered={false}
      >
        {onSearch != null && (
          <SearchInput
            dialogVariant="floating"
            onSearch={onSearch}
            searchPlaceholder={`Search ${plural(resourceName)}...`}
            actions={actions}
          />
        )}
        {footer == null || footer === false ? (
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
        ) : (
          <Flex.Box
            y
            empty
            grow
            className={CSS.BE("select", "body")}
            bordered
            borderColor={6}
            rounded
            full="x"
          >
            <List.Items emptyContent={emptyContent} grow full="x">
              {children}
            </List.Items>
            {footer}
          </Flex.Box>
        )}
      </BaseDialog.Dialog>
    );
  },
);
Base.displayName = "Select.Dialog";
export const Dialog = Base as <K extends record.Key>(
  props: DialogProps<K>,
) => ReactElement;
