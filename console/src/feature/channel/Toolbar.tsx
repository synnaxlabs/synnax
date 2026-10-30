// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { channel, group } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { type Select } from "@synnaxlabs/lyra/select";
import { Access, Channel, Telem } from "@synnaxlabs/pluto";
import { DataType } from "@synnaxlabs/x";
import { type ReactElement, useState } from "react";

import { Channel as PlatformChannel } from "@/platform/channel";
import { Empty } from "@/platform/empty";
import { type Nav } from "@/platform/nav";
import { Toolbar } from "@/platform/toolbar";
import { Tree } from "@/platform/tree";

// The filter button is the select's own trigger, so one click opens the list.
const TRIGGER_PROPS: Select.SingleTriggerProps = {
  iconOnly: true,
  icon: <Icon.Filter />,
  variant: "text",
  size: "tiny",
  tooltip: "Filter",
};

// A group fails, so it shows only while one of its channels passes.
const dataTypeFilter = (dataType: string): Tree.Filter => {
  const target = new DataType(dataType);
  return ({ id, data }) =>
    id.type === "channel" && target.equals((data as channel.Payload).dataType);
};

interface DataTypeFilterProps {
  value: string | null;
  onChange: (dataType: string | null) => void;
}

const DataTypeFilter = ({ value, onChange }: DataTypeFilterProps): ReactElement => (
  <Telem.SelectDataType
    value={value ?? undefined}
    onChange={onChange}
    allowNone
    triggerProps={TRIGGER_PROPS}
  />
);

const Content = (): ReactElement => {
  const openCreate = PlatformChannel.useCreateModal();
  const openCalculated = PlatformChannel.useCalculatedModal();
  const hasCreatePermission = Access.useCreateGranted(channel.TYPE_ONTOLOGY_ID);
  const [searching, setSearching] = useState(false);
  const [dataType, setDataType] = useState<string | null>(null);
  return (
    <Toolbar.Content>
      <Toolbar.Header>
        <Toolbar.Title>
          <Icon.Channel />
          Channels
        </Toolbar.Title>
        <Toolbar.Actions>
          <Toolbar.SearchAction value={searching} onChange={setSearching} />
          {hasCreatePermission && (
            <>
              <Toolbar.Action
                onClick={() => openCalculated()}
                tooltip="Create calculated channel"
                variant="filled"
              >
                <Channel.CreateCalculatedIcon />
              </Toolbar.Action>
              <Toolbar.Action
                onClick={() => openCreate()}
                tooltip="Create channel"
                variant="filled"
              >
                <Icon.Add />
              </Toolbar.Action>
            </>
          )}
        </Toolbar.Actions>
      </Toolbar.Header>
      <Toolbar.Body>
        <Body
          search={searching}
          filter={searching && dataType != null ? dataTypeFilter(dataType) : undefined}
          bar={
            searching ? (
              <DataTypeFilter value={dataType} onChange={setDataType} />
            ) : undefined
          }
        />
      </Toolbar.Body>
    </Toolbar.Content>
  );
};

interface BodyProps extends Pick<Tree.TreeProps, "search" | "filter" | "bar"> {}

const Body = (props: BodyProps): ReactElement => {
  const g = Channel.useGroup({});
  const root = group.ontologyID(g.key);
  return <Tree.Tree root={root} emptyContent={<EmptyContent />} {...props} />;
};

const EmptyContent = (): ReactElement => {
  const openCreate = PlatformChannel.useCreateModal();
  const hasCreatePermission = Access.useCreateGranted(channel.TYPE_ONTOLOGY_ID);
  return (
    <Empty.Action
      message="No channels"
      action={hasCreatePermission ? "Create channel" : undefined}
      onClick={() => openCreate()}
    />
  );
};

export const TOOLBAR: Nav.Toolbar = {
  key: "channel",
  icon: <Icon.Channel />,
  content: <Content />,
  tooltip: "Channels",
  trigger: ["C"],
  initialSize: 300,
  sizeBounds: { lower: 175, upper: 400 },
  useVisible: () => Access.useRetrieveGranted(channel.TYPE_ONTOLOGY_ID),
};
