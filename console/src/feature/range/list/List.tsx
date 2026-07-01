// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/range/list/List.css";

import { ranger } from "@synnaxlabs/client";
import {
  Access,
  Button,
  Component,
  Flex,
  type Flux,
  Icon,
  Input,
  List as PList,
  Menu,
  Select,
  type state,
} from "@synnaxlabs/pluto";
import { type ReactElement, type ReactNode, useCallback, useState } from "react";

import { ContextMenu } from "@/feature/range/list/ContextMenu";
import { Empty } from "@/platform/empty";
import { Range } from "@/platform/range";
import { type ItemProps } from "@/platform/range/list/Item";

export interface ListProps
  extends
    Pick<
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
  const openCreate = Range.useCreateModal();
  const hasCreatePermission = Access.useCreateGranted(ranger.TYPE_ONTOLOGY_ID);
  return (
    <Empty.Action
      message="No ranges found."
      action={hasCreatePermission ? "Create a range" : undefined}
      onClick={() => openCreate()}
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
  const menuProps = Menu.useContextMenu();
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
                <Range.SelectFilters
                  request={request}
                  onRequestChange={handleRequestChange}
                />
                <Range.Filters
                  request={request}
                  onRequestChange={handleRequestChange}
                />
              </>
            )}
          </Flex.Box>
        )}
        <Menu.ContextMenu menu={contextMenu} {...menuProps} />
        <PList.Items<string>
          emptyContent={emptyContent}
          grow
          onContextMenu={menuProps.open}
        >
          {({ key, ...rest }) => (
            <Range.Item
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

const contextMenu = Component.renderProp<Menu.ContextMenuMenuProps>((p) => (
  <ContextMenu {...p} />
));

const AddButton = (): ReactElement | null => {
  const openCreate = Range.useCreateModal();
  const hasCreatePermission = Access.useCreateGranted(ranger.TYPE_ONTOLOGY_ID);
  if (!hasCreatePermission) return null;
  return (
    <Button.Button tooltip="Create Range" onClick={() => openCreate()}>
      <Icon.Add />
    </Button.Button>
  );
};
