// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import {
  Flex,
  type Flux,
  Input,
  List as PList,
  Select,
  type state,
} from "@synnaxlabs/pluto";
import { useState } from "react";

import { Item } from "@/status/list/Item";

export interface ListProps
  extends Pick<
    Flux.UseListReturn<PList.PagerParams, status.Key, status.Status>,
    "data" | "getItem" | "subscribe" | "retrieve"
  > {
  enableSearch?: boolean;
}

export const List = ({
  data,
  getItem,
  subscribe,
  retrieve,
  enableSearch = false,
}: ListProps) => {
  const [request, setRequest] = useState<status.MultiRetrieveArgs>({});
  const [selected, setSelected] = useState<status.Key[]>([]);

  const handleRequestChange = (setter: state.SetArg<status.MultiRetrieveArgs>) => {
    retrieve(setter);
    setRequest(setter);
  };

  const handleSearch = (term: string) =>
    handleRequestChange((p: status.MultiRetrieveArgs) => PList.search(p, term));

  const handleFetchMore = () => handleRequestChange(PList.page);

  return (
    <Select.Frame<status.Key, status.Status>
      multiple
      data={data}
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
            placeholder="Search"
            onChange={handleSearch}
          />
        </Flex.Box>
      )}
      <PList.Items<status.Key>>
        {({ key, ...rest }) => <Item key={key} {...rest} />}
      </PList.Items>
    </Select.Frame>
  );
};