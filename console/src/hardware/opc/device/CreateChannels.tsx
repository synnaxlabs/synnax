// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState, type ReactElement, memo, useCallback } from "react";

import { Icon } from "@synnaxlabs/media";
import { Form, Haul, Select, CSS as PCSS } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Header } from "@synnaxlabs/pluto/header";
import { Input } from "@synnaxlabs/pluto/input";
import { List } from "@synnaxlabs/pluto/list";
import { Text } from "@synnaxlabs/pluto/text";
import { nanoid } from "nanoid/non-secure";

import { CSS } from "@/css";
import { type ChannelConfig, type GroupConfig } from "@/hardware/ni/device/types";
import { SelectNode } from "@/hardware/opc/SelectNode";
import { type DeviceProperties } from "@/hardware/opc/types";

import "@/hardware/ni/device/CreateChannels.css";

interface MostRecentSelectedState {
  key: string;
  type: "group" | "channel";
  index: number;
}

interface SelectedGroupState {
  index: number;
  key: string;
}

export interface CreateChannelsProps {
  deviceProperties: DeviceProperties;
}

export const CreateChannels = ({
  deviceProperties,
}: CreateChannelsProps): ReactElement => {
  const [mostRecentSelected, setMostRecentSelected] =
    useState<MostRecentSelectedState | null>(null);
  const [selectedGroup, setSelectedGroup] = useState<SelectedGroupState | undefined>(
    undefined,
  );
  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);

  const handleGroupSelect = useCallback(
    (key: string, index: number): void => {
      setSelectedGroup({ index, key });
      setMostRecentSelected({ key, type: "group", index });
    },
    [setMostRecentSelected, setSelectedGroup],
  );

  const handleChannelSelect = useCallback(
    (
      keys: string[],
      { clickedIndex, clicked }: List.UseSelectOnChangeExtra<string>,
    ): void => {
      if (clickedIndex == null || clicked == null) return;
      setSelectedChannels(keys);
      setMostRecentSelected({ type: "channel", index: clickedIndex, key: clicked });
    },
    [setMostRecentSelected, setSelectedChannels],
  );

  const clearSelection = useCallback((): void => {
    setMostRecentSelected(null);
    setSelectedChannels([]);
  }, [setMostRecentSelected, setSelectedChannels]);

  return (
    <Align.Space direction="x" grow className={CSS.B("physical-plan")} size={10}>
      <Align.Space direction="y" className={CSS.B("description")}>
        <Text.Text level="h2" weight={600}>
          Here are the channels we'll create for your OPC UA server
        </Text.Text>
        <Align.Space className="description" direction="y" size="small">
          <Text.Text level="p">
            These channels will store data from the inputs and send commands to the
            outputs of your device. We'll just focus on creating them for now, and you
            can define parameters like calibration and sampling rate later.
          </Text.Text>
          <Text.Text level="p">
            They are separated into independent sampling groups. We've automatically
            identified the channel groupings we think would work well for your device.
          </Text.Text>
          <Text.Text level="p">
            <b>All channels in a group must be sampled together</b>. What does this
            mean? Let's say you have two analog input channels (ai_1 and ai_2) that are
            part of the same group. It's not possible to sample ai_1 at 1 kHz in one
            acquisition task and ai_2 at 10 Hz in another. If you need to sample them at
            different rates, split them into separate groups.
          </Text.Text>
          <Text.Text level="p">
            Click on a group to see what its purpose is, and split any groups that need
            to have independent sampling rates.{" "}
          </Text.Text>
          <Text.Text level="p">
            Don't worry, you can reconfigure the channels for the device later if you
            need to.
          </Text.Text>
          <Text.Text level="p"></Text.Text>
        </Align.Space>
      </Align.Space>
      <Align.Space direction="y" bordered className={CSS.B("form")} grow empty>
        <Align.Space direction="x" empty className={PCSS(PCSS.bordered("bottom"))}>
          <GroupList
            selectedGroup={selectedGroup?.key}
            onSelectGroup={handleGroupSelect}
            clearSelection={clearSelection}
          />
          {selectedGroup != null && (
            <ChannelList
              key={selectedGroup.key}
              selectedGroupIndex={selectedGroup.index}
              selectedChannels={selectedChannels}
              onSelectChannels={handleChannelSelect}
            />
          )}
        </Align.Space>
        <Align.Space className={CSS.B("details")} grow empty>
          {mostRecentSelected != null && (
            <Details
              selected={mostRecentSelected}
              groupIndex={selectedGroup?.index}
              deviceProperties={deviceProperties}
            />
          )}
        </Align.Space>
      </Align.Space>
    </Align.Space>
  );
};

interface GroupListProps {
  selectedGroup: string | undefined;
  onSelectGroup: (key: string, index: number) => void;
  clearSelection: () => void;
}

const GroupList = ({
  selectedGroup,
  onSelectGroup,
  clearSelection,
}: GroupListProps): ReactElement => {
  const { push, value } = Form.useFieldArray<GroupConfig>({ path: "groups" });
  return (
    <Align.Space className={CSS.B("groups")} grow empty>
      <Header.Header level="h3">
        <Header.Title weight={500}>Sampling Groups</Header.Title>
        <Header.Actions>
          {[
            {
              onClick: () => {
                const key = nanoid();
                onSelectGroup(key, value.length);
                push({
                  key,
                  name: "New Group",
                  channels: [],
                  channelPrefix: "",
                  channelSuffix: "",
                  role: "unknown",
                });
              },
              children: <Icon.Add />,
            },
          ]}
        </Header.Actions>
      </Header.Header>
      <List.List<string, GroupConfig> data={value}>
        <List.Selector<string, GroupConfig>
          allowMultiple={false}
          // eslint-disable-next-line @typescript-eslint/non-nullable-type-assertion-style
          value={selectedGroup as string}
          allowNone={false}
          onChange={(key, { clickedIndex }) =>
            clickedIndex != null && onSelectGroup(key, clickedIndex)
          }
        >
          <List.Core<string, GroupConfig> grow>
            {(props) => <GroupListItem clearSelection={clearSelection} {...props} />}
          </List.Core>
        </List.Selector>
      </List.List>
    </Align.Space>
  );
};

export interface GroupListItemProps extends List.ItemProps<string, GroupConfig> {
  clearSelection: () => void;
}
const GroupListItem = ({
  clearSelection,
  ...props
}: GroupListItemProps): ReactElement => {
  const {
    index,
    entry: { name, channels },
    hovered,
    selected,
  } = props;

  const [draggingOver, setDraggingOver] = useState<boolean>(false);

  const ctx = Form.useContext();

  const drop = Haul.useDrop({
    type: "Device.Group",
    key: props.entry.key,
    canDrop: ({ source }) => source.type === "Device.Channel",
    onDrop: ({ items }) => {
      props.onSelect?.(props.entry.key);
      const path = `groups.${index}.channels`;
      const v = ctx.get<ChannelConfig[]>({ path });
      ctx.set({
        path,
        value: v.value
          .concat(items.map((i) => ({ ...(i.data as ChannelConfig) })))
          .sort((a, b) => a.port - b.port),
      });
      setDraggingOver(false);
      return items;
    },
    onDragOver: () => setDraggingOver(true),
  });

  return (
    <List.ItemFrame
      {...props}
      {...drop}
      hovered={hovered || draggingOver}
      onDragLeave={() => setDraggingOver(false)}
      direction="x"
      justify="spaceBetween"
    >
      <Align.Space direction="y" size="small">
        <Text.Text level="p" weight={500}>
          {name}
        </Text.Text>
        <Align.Space direction="x" size="small">
          <Text.Text level="p" shade={7}>
            {channels.length}
          </Text.Text>
          <Text.Text level="p" shade={7}>
            Channels
          </Text.Text>
        </Align.Space>
      </Align.Space>
      {selected && <Text.WithIcon level="p" startIcon={<Icon.Caret.Right />} />}
    </List.ItemFrame>
  );
};

interface ChannelListProps {
  selectedGroupIndex: number;
  selectedChannels: string[];
  onSelectChannels: List.UseSelectProps["onChange"];
}

const CHANNEL_LIST_COLUMNS: Array<List.ColumnSpec<string, ChannelConfig>> = [
  {
    key: "name",
    name: "Name",
    visible: true,
  },
  {
    key: "dataType",
    name: "Data Type",
    visible: true,
    shade: 7,
  },
  {
    key: "index",
    name: "Index",
    visible: true,
    width: 20,
    render: ({ entry: { role } }) => {
      return role === "index" ? (
        <Text.Text
          level="p"
          color="var(--pluto-secondary-z)"
          style={{ marginLeft: 10, width: 20 }}
        >
          Index
        </Text.Text>
      ) : null;
    },
  },
];

const ChannelList = ({
  selectedChannels,
  selectedGroupIndex,
  onSelectChannels,
}: ChannelListProps): ReactElement => {
  const channels = Form.useFieldArray<ChannelConfig[]>({
    path: `groups.${selectedGroupIndex}.channels`,
  });

  return (
    <Align.Space className={CSS.B("channels")} grow empty>
      <Header.Header level="h3">
        <Header.Title weight={500}>Channels</Header.Title>
      </Header.Header>
      <List.List<string, ChannelConfig> data={channels.value}>
        <List.Selector<string, ChannelConfig>
          value={selectedChannels}
          allowNone={false}
          onChange={onSelectChannels}
          replaceOnSingle
        >
          <List.Column.Header<string, ChannelConfig>
            columns={CHANNEL_LIST_COLUMNS}
            hide
          >
            <List.Core<string, ChannelConfig> grow>
              {(props) => (
                <ChannelListItem {...props} groupIndex={selectedGroupIndex} />
              )}
            </List.Core>
          </List.Column.Header>
        </List.Selector>
      </List.List>
    </Align.Space>
  );
};

export const ChannelListItem = memo(
  ({
    groupIndex,
    ...props
  }: List.ItemProps<string, ChannelConfig> & {
    groupIndex: number;
  }): ReactElement => {
    const { startDrag, onDragEnd } = Haul.useDrag({
      type: "Device.Channel",
      key: props.entry.key,
    });

    const groupChannels = `groups.${groupIndex}.channels`;
    const prefix = `${groupChannels}.${props.index}`;

    const methods = Form.useContext();
    const [validPort, setValidPort] = useState<boolean>(
      methods.get({ path: prefix, optional: true })?.status.variant !== "error",
    );
    Form.useFieldListener({
      path: `groups.${groupIndex}.channels.${props.index}.port`,
      onChange: (state) => {
        console.log(state);
        setValidPort(state.status.variant !== "error");
      },
    });

    const { getSelected } = List.useSelectionUtils();
    const handleDragStart = useCallback(() => {
      const selected = getSelected();
      let haulItems = [
        {
          key: props.entry.key,
          type: "Device.Channel",
          data: props.entry,
        },
      ];
      if (selected.includes(props.entry.key)) {
        const channels = methods
          .get<ChannelConfig[]>({ path: groupChannels })
          .value.filter((c) => selected.includes(c.key));
        haulItems = channels.map((c) => ({
          key: c.key,
          type: "Device.Channel",
          data: { ...c },
        }));
      }
      startDrag(haulItems, ({ dropped }) => {
        const keys = dropped.map((d) => d.key);
        const channels = methods.get<ChannelConfig[]>({
          path: groupChannels,
        }).value;
        methods.set({
          path: groupChannels,
          value: channels.filter((c) => !keys.includes(c.key)),
        });
      });
    }, [startDrag, props.entry.key, groupIndex, getSelected, methods.get, methods.set]);

    const childValues = Form.useChildFieldValues<ChannelConfig>({
      path: prefix,
      optional: true,
    });
    if (childValues == null) return null;
    return (
      <List.Column.Item
        {...props}
        entry={childValues}
        draggable
        className={!validPort ? CSS.B("bad-port") : ""}
        onDragStart={handleDragStart}
        onDragEnd={onDragEnd}
      />
    );
  },
);
ChannelListItem.displayName = "ChannelListItem";

export interface DetailsProps {
  selected: MostRecentSelectedState;
  deviceProperties: DeviceProperties;
  groupIndex?: number;
}

const Details = ({
  selected,
  groupIndex,
  deviceProperties,
}: DetailsProps): ReactElement | null => {
  console.log(selected, groupIndex);
  if (groupIndex == null) return null;
  if (selected.type === "group") return <GroupForm index={selected.index} />;
  return (
    <ChannelForm
      index={selected.index}
      groupIndex={groupIndex}
      deviceProperties={deviceProperties}
    />
  );
};

interface ChannelFormProps {
  groupIndex: number;
  index: number;
  deviceProperties: DeviceProperties;
}

const ChannelForm = ({
  index,
  groupIndex,
  deviceProperties,
}: ChannelFormProps): ReactElement | null => {
  const prefix = `groups.${groupIndex}.channels.${index}`;
  const ctx = Form.useContext();
  if (!ctx.has(prefix)) return null;

  return (
    <>
      <Form.Field<string> path={`${prefix}.name`} label="Name" showLabel={false}>
        {(p) => (
          <Input.Text
            variant="natural"
            level="h2"
            placeholder="Channel Name"
            autoFocus
            {...p}
          />
        )}
      </Form.Field>
      <Form.Field<string> path={`${prefix}.dataType`} label="Data Type">
        {(p) => (
          <Select.DataType {...p} location="top" allowNone={false} hideColumnHeader />
        )}
      </Form.Field>
      <Form.Field<string> path={`${prefix}.nodeId`} label="Node ID" hideIfNull>
        {(p) => (
          <SelectNode {...p} data={deviceProperties.channels} allowNone={false} />
        )}
      </Form.Field>
    </>
  );
};

interface GroupFormProps {
  index: number;
}

const GroupForm = ({ index }: GroupFormProps): ReactElement => {
  const prefix = `groups.${index}`;
  return (
    <>
      <Form.Field<string> path={`${prefix}.name`} label="Name" showLabel={false}>
        {(p) => (
          <Input.Text
            variant="natural"
            level="h2"
            placeholder="Group Name"
            autoFocus
            {...p}
          />
        )}
      </Form.Field>
    </>
  );
};
