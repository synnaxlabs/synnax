// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import { type ReactElement, type ReactNode } from "react";
import { type z } from "zod";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Items, type ItemsProps } from "@/select/Items";
import { List as SelectList } from "@/select/List";
import { plural, searchPlaceholder } from "@/select/placeholder";
import { Search, type SearchProps } from "@/select/Search";
import { Status } from "@/status";

/** Props for {@link Body}. */
export interface BodyProps<K extends record.Key>
  extends
    Pick<SearchProps, "onSearch" | "actions">,
    Pick<ItemsProps<K>, "emptyContent" | "children"> {
  status?: Status.Status<z.ZodNever>;
  resourceName: string;
  /**
   * Pinned below the scrollable list. A fixed {@link Item} in it is a pinned option,
   * which the search never hides.
   */
  footer?: ReactNode;
  /** Fixed {@link Item}s listed above the data. */
  fixedItems?: ReactNode;
}

/** Props for {@link EmptyContent}. */
export interface EmptyContentProps {
  resourceName: string;
  /** A string renders as gray text. Unset, it says no items were found. */
  emptyContent?: ReactNode;
}

/** The content a selection shows when it has no options to list. */
export const EmptyContent = ({
  resourceName,
  emptyContent,
}: EmptyContentProps): ReactNode => {
  if (emptyContent != null && typeof emptyContent !== "string") return emptyContent;
  return (
    <Status.Summary center variant="disabled" hideIcon>
      {emptyContent ?? `No ${plural(resourceName)} found`}
    </Status.Summary>
  );
};

/** Props for {@link BorderedList}. */
export interface BorderedListProps {
  /** Pinned below the list, inside its border. */
  footer?: ReactNode;
  children: ReactNode;
}

/** The option list of a select, with the footer pinned under it. */
export const BorderedList = ({ footer, children }: BorderedListProps): ReactElement =>
  footer == null || footer === false ? (
    <SelectList bordered borderColor={6} grow rounded full="x">
      {children}
    </SelectList>
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
      <SelectList grow full="x">
        {children}
      </SelectList>
      {footer}
    </Flex.Box>
  );

/**
 * The search field, list, and empty and error content of a data-driven selection. Use
 * it inside `Select.Dialog` when a select needs its own trigger.
 */
export const Body = <K extends record.Key>({
  onSearch,
  children,
  emptyContent,
  status,
  resourceName,
  actions,
  footer,
  fixedItems,
}: BodyProps<K>): ReactElement => {
  const loading = status?.variant === "loading";
  const hasSearch = onSearch != null;
  let empty: ReactNode;
  if (loading) empty = hasSearch ? null : <Status.Loading />;
  else if (status != null && status.variant !== "success")
    empty = (
      <Status.Summary center variant={status.variant} description={status.description}>
        {status.message}
      </Status.Summary>
    );
  else empty = <EmptyContent resourceName={resourceName} emptyContent={emptyContent} />;
  return (
    <>
      {hasSearch && (
        <Search
          onSearch={onSearch}
          placeholder={searchPlaceholder(resourceName)}
          actions={actions}
          loading={loading}
        />
      )}
      <BorderedList footer={footer}>
        {fixedItems}
        <Items<K> emptyContent={empty}>{children}</Items>
      </BorderedList>
    </>
  );
};
