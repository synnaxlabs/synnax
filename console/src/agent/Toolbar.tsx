// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type agent } from "@synnaxlabs/client";
import {
  Agent as AgentFlux,
  Button,
  Flex,
  Icon,
  List,
  Select,
  Status,
  Text,
} from "@synnaxlabs/pluto";
import { type status } from "@synnaxlabs/x";
import { useCallback, useState } from "react";

import { useCreateModal } from "@/agent/CreateModal";
import { create as createEditor } from "@/agent/editor/Editor";
import { EmptyAction, Toolbar } from "@/components";
import { type Layout } from "@/layout";
import { usePlacer } from "@/layout/usePlacer";

const STATUS_VARIANT: Record<string, status.Variant> = {
  running: "success",
  error: "error",
  generating: "loading",
  stopped: "disabled",
};

interface AgentListItemProps extends List.ItemProps<agent.Key> {}

const AgentListItem = ({ ...rest }: AgentListItemProps) => {
  const a = List.useItem<agent.Key, agent.Agent>(rest.itemKey);
  const variant = STATUS_VARIANT[a?.state ?? "stopped"] ?? "disabled";
  return (
    <Select.ListItem {...rest} justify="between" align="center">
      <Flex.Box y gap="small" grow>
        <Flex.Box x align="center" gap="small">
          <Status.Indicator
            variant={variant}
            style={{ fontSize: "2rem", minWidth: "2rem" }}
          />
          <Text.Text level="p" weight={500} overflow="ellipsis">
            {a?.name ?? "Agent"}
          </Text.Text>
        </Flex.Box>
        <Text.Text level="small" status={variant}>
          {a?.state ?? "stopped"}
        </Text.Text>
      </Flex.Box>
    </Select.ListItem>
  );
};

interface EmptyContentProps {
  onCreate: () => void;
}

const EmptyContent = ({ onCreate }: EmptyContentProps) => (
  <EmptyAction message="No existing Agents." action="Create an Agent" onClick={onCreate} />
);

const Content = () => {
  const [selected, setSelected] = useState<agent.Key[]>([]);
  const placeLayout = usePlacer();
  const createModal = useCreateModal();
  const handleError = Status.useErrorHandler();

  const { data, getItem, subscribe } = AgentFlux.useList({});
  const { fetchMore } = List.usePager({ retrieve: AgentFlux.useList({}).retrieve, pageSize: 1e3 });

  const handleEdit = useCallback(
    (key: agent.Key) => {
      const a = getItem(key);
      placeLayout(createEditor({ key, name: a?.name ?? "Agent" }));
    },
    [getItem, placeLayout],
  );

  const handleCreate = useCallback(() => {
    handleError(async () => {
      const result = await createModal({});
      if (result == null) return;
      placeLayout(createEditor({ key: result.agent.key, name: result.agent.name }));
    }, "Failed to create Agent");
  }, [createModal, handleError, placeLayout]);

  return (
    <Toolbar.Content>
      <Toolbar.Header padded>
        <Toolbar.Title icon={<Icon.Auto />}>Agents</Toolbar.Title>
        <Toolbar.Actions>
          <Toolbar.Action onClick={handleCreate}>
            <Icon.Add />
          </Toolbar.Action>
        </Toolbar.Actions>
      </Toolbar.Header>
      <Select.Frame
        multiple
        data={data}
        getItem={getItem}
        subscribe={subscribe}
        value={selected}
        onChange={setSelected}
        onFetchMore={fetchMore}
        replaceOnSingle
      >
        <List.Items<agent.Key, agent.Agent>
          full="y"
          emptyContent={<EmptyContent onCreate={handleCreate} />}
        >
          {({ key, ...p }) => (
            <AgentListItem
              key={key}
              {...p}
              onDoubleClick={() => handleEdit(key)}
            />
          )}
        </List.Items>
      </Select.Frame>
    </Toolbar.Content>
  );
};

export const TOOLBAR: Layout.NavDrawerItem = {
  key: "agent",
  icon: <Icon.Auto />,
  content: <Content />,
  trigger: ["G"],
  tooltip: "Agents",
  initialSize: 300,
  minSize: 225,
  maxSize: 400,
};
