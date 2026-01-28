// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type arc } from "@synnaxlabs/client";
import { Flex, type Flux, Icon, Input, List as PList, Select } from "@synnaxlabs/pluto";
import { useState } from "react";

import { Item, type ItemProps } from "@/arc/list/Item";

export interface ListProps
  extends
    Pick<
      Flux.UseListReturn<PList.PagerParams, arc.Key, arc.Arc>,
      "data" | "getItem" | "subscribe" | "retrieve"
    >,
    Pick<ItemProps, "showStatus"> {
  enableSearch?: boolean;
}

export const List = ({
  data,
  getItem,
  subscribe,
  retrieve,
  enableSearch = false,
  showStatus = true,
}: ListProps) => {
  const { fetchMore, search } = PList.usePager({ retrieve });
  const [value, setValue] = useState<arc.Key[]>([]);
  const [searchTerm, setSearchTerm] = useState("");
  return (
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
      <PList.Items<arc.Key>>
        {({ key, ...rest }) => <Item key={key} {...rest} showStatus={showStatus} />}
      </PList.Items>
    </Select.Frame>
  );
};
