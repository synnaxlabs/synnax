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
  Component,
  Flex,
  type Flux,
  Icon,
  Input,
  List as PList,
  Select,
  type state,
} from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { useCallback, useState } from "react";

import { EmptyAction } from "@/components";
import { Layout } from "@/layout";

export interface ExplorerProps<K extends record.Key, E extends record.Keyed<K>>
  extends Pick<
    Flux.UseListReturn<PList.PagerParams, K, E>,
    "data" | "getItem" | "subscribe" | "retrieve"
  > {
  enableSearch?: boolean;
  enableFilters?: boolean;
  initialRequest?: status.MultiRetrieveArgs;
}

const componentRenderProp = Component.renderProp(Item);

const EmptyContent = () => {
  const placeLayout = Layout.usePlacer();
  return (
    <EmptyAction
      message="No statuses found."
      action="Create a status"
      onClick={() => placeLayout(CREATE_LAYOUT)}
    />
  );
};

export const Explorer = <K extends record.Key, E extends record.Keyed<K>>({
  data,
  getItem,
  subscribe,
  retrieve,
  enableSearch = false,
  enableFilters = false,
  initialRequest = {},
}: ExplorerProps<K, E>) => {
  const [request, setRequest] = useState<status.MultiRetrieveArgs>(initialRequest);
  const [selected, setSelected] = useState<K[]>([]);

  const handleRequestChange = useCallback(
    (setter: state.SetArg<status.MultiRetrieveArgs>, opts?: Flux.AsyncListOptions) => {
      retrieve(setter, opts);
      setRequest(setter);
    },
    [retrieve],
  );

  const handleSearch = useCallback(
    (term: string) =>
      handleRequestChange((p: status.MultiRetrieveArgs) => PList.search(p, term)),
    [handleRequestChange],
  );

  const handleFetchMore = useCallback(
    () => handleRequestChange((r) => PList.page(r, 25), { mode: "append" }),
    [handleRequestChange],
  );

  return (
    <Flex.Box full="y" empty>
      <Select.Frame<K, E>
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
              placeholder={
                <>
                  <Icon.Search />
                  Search Statuses...
                </>
              }
              onChange={handleSearch}
            />
            <CreateButton />
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
        <PList.Items<status.Key>
          emptyContent={<EmptyContent />}
          displayItems={Infinity}
          grow
        >
          {componentRenderProp}
        </PList.Items>
      </Select.Frame>
    </Flex.Box>
  );
};
