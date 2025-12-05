// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import {
  Access,
  Component,
  Flex,
  type Flux,
  Icon,
  Input,
  List as PList,
  Select,
  type state,
} from "@synnaxlabs/pluto";
import { useCallback, useState } from "react";

import { EmptyAction } from "@/components";
import { Layout } from "@/layout";
import { CREATE_LAYOUT } from "@/status/Create";
import { Item } from "@/status/list/Item";
import { Filters, SelectFilters } from "@/status/list/SelectFilters";
import { CreateButton } from "@/status/Select";

export interface ListProps
  extends Pick<
    Flux.UseListReturn<PList.PagerParams, status.Key, status.Status>,
    "data" | "getItem" | "subscribe" | "retrieve"
  > {
  enableSearch?: boolean;
  enableFilters?: boolean;
  initialRequest?: status.MultiRetrieveArgs;
}

const componentRenderProp = Component.renderProp(Item);

const EmptyContent = () => {
  const placeLayout = Layout.usePlacer();
  const canEdit = Access.useEditGranted(status.TYPE_ONTOLOGY_ID);
  return (
    <EmptyAction
      message="No statuses found."
      action={canEdit ? "Create a status" : undefined}
      onClick={canEdit ? () => placeLayout(CREATE_LAYOUT) : undefined}
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
  initialRequest = {},
}: ListProps) => {
  const [request, setRequest] = useState<status.MultiRetrieveArgs>(initialRequest);
  const [selected, setSelected] = useState<status.Key[]>([]);

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
        <PList.Items<status.Key> emptyContent={<EmptyContent />} grow>
          {componentRenderProp}
        </PList.Items>
      </Select.Frame>
    </Flex.Box>
  );
};
