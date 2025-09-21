// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ranger } from "@synnaxlabs/client";
import {
  Flex,
  type Flux,
  Icon,
  Input,
  List as PList,
  Select,
  type state,
} from "@synnaxlabs/pluto";
import { useState } from "react";

import { Item, type ItemProps } from "@/range/list/Item";
import { Filters, SelectFilters } from "@/range/list/SelectFilters";

export interface ListProps
  extends Pick<
      Flux.UseListReturn<PList.PagerParams, ranger.Key, ranger.Range>,
      "data" | "getItem" | "subscribe" | "retrieve"
    >,
    Pick<ItemProps, "showParent" | "showLabels" | "showTimeRange" | "showFavorite"> {
  enableSearch?: boolean;
  enableFilters?: boolean;
  initialRequest?: ranger.RetrieveRequest;
}

export const List = ({
  data,
  getItem,
  subscribe,
  retrieve,
  enableSearch = false,
  enableFilters = false,
  showParent = true,
  showLabels = true,
  showTimeRange = true,
  showFavorite = true,
  initialRequest = {},
}: ListProps) => {
  const [request, setRequest] = useState<ranger.RetrieveRequest>(initialRequest);
  const [selected, setSelected] = useState<ranger.Key[]>([]);
  const handleRequestChange = (setter: state.SetArg<ranger.RetrieveRequest>) => {
    retrieve(setter);
    setRequest(setter);
  };
  const handleSearch = (term: string) =>
    handleRequestChange((p: ranger.RetrieveRequest) => PList.search(p, term));
  const handleFetchMore = () => handleRequestChange(PList.page);
  return (
    <Select.Frame<ranger.Key, ranger.Range>
      multiple
      data={data}
      virtual
      getItem={getItem}
      subscribe={subscribe}
      onChange={setSelected}
      value={selected}
      onFetchMore={handleFetchMore}
    >
      {enableSearch && (
        <Flex.Box
          x
          bordered
          style={{ padding: "1.5rem" }}
          background={1}
          justify="between"
        >
          <Input.Text
            size="small"
            level="h5"
            variant="text"
            value={request.searchTerm ?? ""}
            placeholder={
              <>
                <Icon.Search />
                Search Ranges...
              </>
            }
            onChange={handleSearch}
          />
        </Flex.Box>
      )}
      {enableFilters && (
        <Flex.Box
          x
          bordered
          style={{ padding: "1rem 2rem", borderTop: "none" }}
          background={1}
          justify="between"
        >
          <SelectFilters request={request} onRequestChange={handleRequestChange} />
          <Filters request={request} onRequestChange={handleRequestChange} />
        </Flex.Box>
      )}
      <PList.Items<string>>
        {({ key, ...rest }) => (
          <Item
            key={key}
            {...rest}
            showParent={showParent}
            showLabels={showLabels}
            showTimeRange={showTimeRange}
            showFavorite={showFavorite}
          />
        )}
      </PList.Items>
    </Select.Frame>
  );
};
