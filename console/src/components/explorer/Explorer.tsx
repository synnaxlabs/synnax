// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Button,
  type Component,
  Flex,
  type Flux,
  Icon,
  Input,
  List as PList,
  Select,
  type state,
} from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { plural } from "pluralize";
import { useCallback, useState } from "react";

import { EmptyAction } from "@/components/EmptyAction";

export interface ExplorerProps<K extends record.Key, E extends record.Keyed<K>>
  extends Pick<
    Flux.UseListReturn<PList.PagerParams, K, E>,
    "data" | "getItem" | "subscribe" | "retrieve"
  > {
  enableSearch?: boolean;
  item: Component.RenderProp<PList.ItemProps<K>>;
  initialRequest?: PList.PagerParams;
  resourceType: string;
  filters?: React.FC<FiltersProps>;
  onCreate: () => void;
  hideToolbar?: boolean;
  initialEditable?: boolean;
}

export interface FiltersProps {
  request: PList.PagerParams;
  onRequestChange: state.Setter<PList.PagerParams>;
}

export const Explorer = <K extends record.Key, E extends record.Keyed<K>>({
  data,
  getItem,
  subscribe,
  filters,
  retrieve,
  enableSearch = false,
  initialRequest = {},
  initialEditable = false,
  resourceType,
  hideToolbar = false,
  onCreate,
  item,
}: ExplorerProps<K, E>) => {
  const [request, setRequest] = useState<PList.PagerParams>(initialRequest);
  const [editable, setEditable] = useState(initialEditable);
  const [selected, setSelected] = useState<K[]>([]);

  const handleRequestChange = useCallback(
    (setter: state.SetArg<PList.PagerParams>, opts?: Flux.AsyncListOptions) => {
      retrieve(setter, opts);
      setRequest(setter);
    },
    [retrieve],
  );

  const handleSearch = useCallback(
    (term: string) => handleRequestChange((p) => PList.search(p, term)),
    [handleRequestChange],
  );

  const handleFetchMore = useCallback(
    () => handleRequestChange((r) => PList.page(r, 25), { mode: "append" }),
    [handleRequestChange],
  );

  const showTopToolbar = !hideToolbar && (enableSearch || filters != null) && editable;

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
        {showTopToolbar && (
          <Flex.Box
            x
            bordered
            style={{ padding: "1.5rem" }}
            background={1}
            justify="between"
            align="center"
          >
            {filters != null && (
              <>{filters({ request, onRequestChange: handleRequestChange })}</>
            )}
            {enableSearch && (
              <Input.Text
                size="small"
                level="h5"
                variant="text"
                value={request.searchTerm ?? ""}
                placeholder={`Search ${plural(resourceType)}...`}
                onChange={handleSearch}
              />
            )}
            <CreateButton onCreate={onCreate} resourceType={resourceType} />
          </Flex.Box>
        )}
        <Flex.Box x bordered>
          <Button.Button onClick={() => setEditable((editable) => !editable)}>
            {editable ? <Icon.EditOff /> : <Icon.Edit />}
          </Button.Button>
        </Flex.Box>
        <PList.Items<K>
          emptyContent={
            <EmptyContent onCreate={onCreate} resourceType={resourceType} />
          }
          displayItems={Infinity}
          grow
        >
          {item}
        </PList.Items>
      </Select.Frame>
    </Flex.Box>
  );
};

interface CreateButtonProps {
  onCreate: () => void;
  resourceType: string;
}

const CreateButton = ({ onCreate, resourceType }: CreateButtonProps) => (
  <Button.Button onClick={onCreate} title={`Create a ${resourceType}`}>
    <Icon.Add />
  </Button.Button>
);

interface EmptyContentProps {
  onCreate: () => void;
  resourceType: string;
}

const EmptyContent = ({ onCreate, resourceType }: EmptyContentProps) => (
  <EmptyAction
    message={`No ${plural(resourceType)} created.`}
    action={`Create a ${resourceType}`}
    onClick={onCreate}
  />
);
