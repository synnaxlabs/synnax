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
import { type ReactElement, type ReactNode, useMemo } from "react";
import { type z } from "zod";

import { memo } from "@/component/memo";
import { CSS } from "@/css";
import { Dialog as BaseDialog } from "@/dialog";
import { Flex } from "@/flex";
import { List } from "@/list";
import { SearchInput, type SearchInputProps } from "@/select/SearchInput";
import { Status } from "@/status/base";
import { Text } from "@/text";

/** Props for {@link Dialog}. */
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

/** Props for the content shown when a selection has nothing to offer. */
export interface DefaultEmptyContentProps extends Status.SummaryProps {
  resourceName: string;
}

const DefaultEmptyContent = ({ resourceName }: DefaultEmptyContentProps) => (
  <Text.Text center status="disabled">
    No {plural(resourceName)} found
  </Text.Text>
);

/* Height the list may take, leaving the search input its share of the dialog's cap in
   Dialog.css. Rows are only ever whole, so this is a budget rather than a limit. */
const LIST_BUDGET = 220;

/**
 * useDisplayItems returns how many rows fit the dialog at the enclosing list's row
 * height. Counting rows rather than pixels gives the list a definite height, which is
 * what lets its growth animate.
 */
const useDisplayItems = (): number => {
  const itemHeight = List.useItemHeight();
  return useMemo(
    () => (itemHeight == null ? 1 : Math.max(1, Math.floor(LIST_BUDGET / itemHeight))),
    [itemHeight],
  );
};

const Base = <K extends record.Key>({
  onSearch,
  children,
  emptyContent,
  status,
  resourceName,
  actions,
  footer,
  className,
  ...rest
}: DialogProps<K>): ReactElement => {
  const loading = status?.variant === "loading";
  const hasSearch = onSearch != null;
  const displayItems = useDisplayItems();
  emptyContent = useMemo(() => {
    if (loading) return hasSearch ? null : <Status.Loading />;
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
  }, [status?.key, emptyContent, loading, hasSearch]);
  return (
    <BaseDialog.Dialog
      {...rest}
      className={CSS.cx(CSS.BE("select", "dialog"), className)}
      bordered={false}
    >
      {hasSearch && (
        <SearchInput
          dialogVariant="floating"
          onSearch={onSearch}
          searchPlaceholder={`Search ${plural(resourceName)}...`}
          actions={actions}
          loading={loading}
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
          displayItems={displayItems}
          animateHeight
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
          <List.Items
            emptyContent={emptyContent}
            grow
            full="x"
            displayItems={displayItems}
            animateHeight
          >
            {children}
          </List.Items>
          {footer}
        </Flex.Box>
      )}
    </BaseDialog.Dialog>
  );
};
Base.displayName = "Select.Dialog";

/**
 * The dropdown of a selection: its search field, its list, and its empty and error
 * content. It sizes the list to whole rows, so its growth animates.
 */
export const Dialog = memo(Base);
