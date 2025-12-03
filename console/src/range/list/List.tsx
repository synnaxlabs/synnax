// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/range/list/List.css";

import { type ranger } from "@synnaxlabs/client";
import {
  Button,
  Flex,
  type Flux,
  Icon,
  Input,
  List as PList,
  Select,
  type state,
} from "@synnaxlabs/pluto";
import { type ReactElement, type ReactNode, useCallback, useState } from "react";

import { EmptyAction } from "@/components";
import { Layout } from "@/layout";
import { CREATE_LAYOUT } from "@/range/Create";
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
  enableAddButton?: boolean;
  initialRequest?: ranger.RetrieveRequest;
  emptyContent?: ReactNode;
}

const EmptyContent = () => {
  const placeLayout = Layout.usePlacer();
  return (
    <EmptyAction
      message="No ranges found."
      action="Create a range"
      onClick={() => placeLayout(CREATE_LAYOUT)}
    />
  );
};

export const List = ({
  data,
  getItem,
  subscribe,
  retrieve,
  enableSearch = false,
  enableFilters = false,
  enableAddButton = false,
  showParent = true,
  showLabels = true,
  showTimeRange = true,
  showFavorite = true,
  initialRequest = {},
  emptyContent = <EmptyContent />,
}: ListProps) => {
  const [request, setRequest] = useState<ranger.RetrieveRequest>(initialRequest);
  const [selected, setSelected] = useState<ranger.Key[]>([]);
  const handleRequestChange = useCallback(
    (setter: state.SetArg<ranger.RetrieveRequest>, opts?: Flux.AsyncListOptions) => {
      retrieve(setter, opts);
      setRequest(setter);
    },
    [retrieve],
  );
  const handleSearch = useCallback(
    (term: string) =>
      handleRequestChange((p: ranger.RetrieveRequest) => PList.search(p, term)),
    [handleRequestChange],
  );
  const handleFetchMore = useCallback(
    () => handleRequestChange((r) => PList.page(r, 25), { mode: "append" }),
    [handleRequestChange],
  );
  return (
    <Flex.Box full="y" empty>
      <Select.Frame<ranger.Key, ranger.Range>
        multiple
        data={data}
        virtual
        getItem={getItem}
        subscribe={subscribe}
        onChange={setSelected}
        value={selected}
        onFetchMore={handleFetchMore}
        itemHeight={45}
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
            {enableAddButton && <AddButton />}
          </Flex.Box>
        )}
        {(enableFilters || enableAddButton) && (
          <Flex.Box
            x
            bordered
            style={{ padding: "1rem 2rem", borderTop: "none" }}
            background={1}
            justify="between"
          >
            {enableFilters && (
              <>
                <SelectFilters
                  request={request}
                  onRequestChange={handleRequestChange}
                />
                <Filters request={request} onRequestChange={handleRequestChange} />
              </>
            )}
          </Flex.Box>
        )}
        <PList.Items<string> emptyContent={emptyContent} grow>
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
    </Flex.Box>
  );
};

const AddButton = (): ReactElement => {
  const placeLayout = Layout.usePlacer();
  return (
    <Button.Button tooltip="Create Range" onClick={() => placeLayout(CREATE_LAYOUT)}>
      <Icon.Add />
    </Button.Button>
  );
};
