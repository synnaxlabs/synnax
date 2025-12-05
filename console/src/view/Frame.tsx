// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/view/View.css";

import { type ontology } from "@synnaxlabs/client";
import {
  Button,
  Flex,
  type Flux,
  Icon,
  List,
  Select,
  Status,
  View as PView,
} from "@synnaxlabs/pluto";
import { location, type record } from "@synnaxlabs/x";
import {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useMemo,
  useState,
} from "react";

import { Controls } from "@/components";
import { CSS } from "@/css";
import { Modals } from "@/modals";
import { Provider } from "@/view/context";
import { type Query, type UseQueryReturn } from "@/view/useQuery";

export interface FrameProps<
  K extends record.Key,
  E extends record.Keyed<K>,
  Q extends Query,
> extends PropsWithChildren,
    Pick<Flux.UseListReturn<Q, K, E>, "data" | "getItem" | "subscribe">,
    Pick<UseQueryReturn<Q>, "query" | "onQueryChange"> {
  resourceType: ontology.ResourceType;
  onCreate: () => void;
}

export const Frame = <
  K extends record.Key,
  E extends record.Keyed<K>,
  Q extends Query,
>({
  children,
  query,
  onQueryChange,
  onCreate,
  resourceType,
  data,
  getItem,
  subscribe,
}: FrameProps<K, E, Q>): ReactElement => {
  const [selected, setSelected] = useState<K[]>([]);
  const [editable, setEditable] = useState(true);
  const handleFetchMore = useCallback(
    () => onQueryChange((q) => ({ ...q, ...List.page(q, 25) }), { mode: "append" }),
    [onQueryChange],
  );
  const { update: create } = PView.useCreate();
  const handleError = Status.useErrorHandler();
  const renameModal = Modals.useRename();
  const handleCreateView = () =>
    handleError(async () => {
      const name = await renameModal(
        { initialValue: `View for ${resourceType}` },
        { icon: "View", name: "View.Create" },
      );
      if (name == null) return;
      create({ name, type: resourceType, query });
    }, "Failed to create view");
  const contextValue = useMemo(
    () => ({ editable, resourceType }),
    [editable, resourceType],
  );
  return (
    <Flex.Box full="y" empty className={CSS.B("view")}>
      <Controls x>
        <Button.Button
          onClick={onCreate}
          size="small"
          tooltipLocation={location.BOTTOM_LEFT}
          tooltip={`Create a ${resourceType}`}
        >
          <Icon.Add />
        </Button.Button>
        <Button.Toggle
          size="small"
          value={editable}
          onChange={() => setEditable((e) => !e)}
          tooltipLocation={location.BOTTOM_LEFT}
          tooltip={`${editable ? "Disable" : "Enable"} editing`}
        >
          {editable ? <Icon.EditOff /> : <Icon.Edit />}
        </Button.Toggle>
        <Button.Button
          size="small"
          onClick={handleCreateView}
          tooltipLocation={location.BOTTOM_LEFT}
          tooltip="Create a view"
        >
          <Icon.View />
        </Button.Button>
      </Controls>
      <Select.Frame
        multiple
        data={data}
        getItem={getItem}
        subscribe={subscribe}
        onChange={setSelected}
        value={selected}
        onFetchMore={handleFetchMore}
      >
        <Provider value={contextValue}>{children}</Provider>
      </Select.Frame>
    </Flex.Box>
  );
};
