// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc } from "@synnaxlabs/client";
import {
  Flex,
  type Flux,
  Icon,
  Input,
  List as PList,
  Menu,
  Select,
} from "@synnaxlabs/pluto";
import { useCallback, useState } from "react";

import { ContextMenu } from "@/arc/ContextMenu";
import { useRename } from "@/arc/hooks";
import { Item, type ItemProps } from "@/arc/list/Item";

export interface ListProps
  extends
    Pick<
      Flux.UseListReturn<PList.PagerParams, arc.Key, arc.Arc>,
      "data" | "getItem" | "subscribe" | "retrieve"
    >,
    Pick<ItemProps, "textIdPrefix"> {
  enableSearch?: boolean;
}

export const List = ({
  data,
  getItem,
  subscribe,
  retrieve,
  enableSearch = false,
  textIdPrefix = "text",
}: ListProps) => {
  const { fetchMore, search } = PList.usePager({ retrieve });
  const [value, setValue] = useState<arc.Key[]>([]);
  const [searchTerm, setSearchTerm] = useState("");
  const menuProps = Menu.useContextMenu();
  const { update: handleRename } = useRename(getItem);

  const contextMenu = useCallback<NonNullable<Menu.ContextMenuProps["menu"]>>(
    (props) => <ContextMenu {...props} getItem={getItem} textIdPrefix={textIdPrefix} />,
    [getItem, textIdPrefix],
  );

  return (
    <Menu.ContextMenu menu={contextMenu} {...menuProps}>
      <Select.Frame<arc.Key, arc.Arc>
        multiple
        data={data}
        getItem={getItem}
        subscribe={subscribe}
        onChange={setValue}
        value={value}
        onFetchMore={fetchMore}
      >
        {enableSearch && (
          <Flex.Box x bordered style={{ padding: "2rem" }} background={1}>
            <Input.Text
              size="large"
              level="h4"
              variant="text"
              value={searchTerm}
              placeholder={
                <>
                  <Icon.Search />
                  Search Arcs...
                </>
              }
              onChange={(value) => {
                setSearchTerm(value);
                search(value);
              }}
            />
          </Flex.Box>
        )}
        <PList.Items<arc.Key> onContextMenu={menuProps.open}>
          {({ key, ...rest }) => (
            <Item
              key={key}
              {...rest}
              textIdPrefix={textIdPrefix}
              onRename={(name) => handleRename({ key, name })}
            />
          )}
        </PList.Items>
      </Select.Frame>
    </Menu.ContextMenu>
  );
};
