// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type record } from "@synnaxlabs/x";
import pluralize from "pluralize";
import { type ReactElement, type ReactNode } from "react";
import { type z } from "zod";

import { CSS } from "@/css";
import { Flex } from "@/flex";
import { Items, type ItemsProps } from "@/select/Items";
import { List as SelectList } from "@/select/List";
import { Search, type SearchProps } from "@/select/Search";
import { Status } from "@/status";
import { Text } from "@/text";

/** Props for {@link Body}. */
export interface BodyProps<K extends record.Key>
  extends
    Pick<SearchProps, "onSearch" | "actions">,
    Pick<ItemsProps<K>, "emptyContent" | "children"> {
  status?: Status.Status<z.ZodNever>;
  resourceName: string;
  /** Pinned below the scrollable list; stays visible regardless of list length. */
  footer?: ReactNode;
  /** Fixed {@link Item}s listed above the data. */
  fixedItems?: ReactNode;
}

export const DefaultEmptyContent = ({ resourceName }: { resourceName: string }) => (
  <Text.Text center status="disabled">
    No {pluralize.plural(resourceName)} found
  </Text.Text>
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
  let empty: ReactNode = emptyContent;
  if (loading) empty = hasSearch ? null : <Status.Loading />;
  else if (status != null && status.variant !== "success")
    empty = (
      <Status.Summary center variant={status.variant} description={status.description}>
        {status.message}
      </Status.Summary>
    );
  else if (typeof emptyContent === "string")
    empty = (
      <Status.Summary center variant="disabled">
        {emptyContent}
      </Status.Summary>
    );
  else if (emptyContent == null)
    empty = <DefaultEmptyContent resourceName={resourceName} />;
  return (
    <>
      {hasSearch && (
        <Search
          onSearch={onSearch}
          placeholder={`Search ${pluralize.plural(resourceName)}...`}
          actions={actions}
          loading={loading}
        />
      )}
      {footer == null || footer === false ? (
        <SelectList bordered borderColor={6} grow rounded full="x">
          {fixedItems}
          <Items<K> emptyContent={empty}>{children}</Items>
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
            {fixedItems}
            <Items<K> emptyContent={empty}>{children}</Items>
          </SelectList>
          {footer}
        </Flex.Box>
      )}
    </>
  );
};
