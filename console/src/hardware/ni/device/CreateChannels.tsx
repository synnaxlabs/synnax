// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/ni/device/CreateChannels.css";

import { Icon } from "@synnaxlabs/media";
import { Button, CSS as PCSS, Form, Haul, Menu, Select } from "@synnaxlabs/pluto";
import { Note } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Header } from "@synnaxlabs/pluto/header";
import { Input } from "@synnaxlabs/pluto/input";
import { List } from "@synnaxlabs/pluto/list";
import { Text } from "@synnaxlabs/pluto/text";
import { nanoid } from "nanoid/non-secure";
import { memo, type ReactElement, useCallback, useState } from "react";

import { CSS } from "@/css";
import { type ChannelConfig, type GroupConfig } from "@/hardware/ni/device/types";

interface MostRecentSelectedState {
  key: string;
  type: "group" | "channel";
  index: number;
}

interface SelectedGroupState {
  index: number;
  keys: string[];
}

const channelRoleDocumentation: Record<string, ReactElement> = {
  analogInput: (
    <Text.Text level="p">
      This channel will store data from an analog input on your device. We recommend
      using 32-bit floating points numbers for these channels, as they'll perform much
      better than their 64-bit alternatives.
    </Text.Text>
  ),
  index: (
    <Text.Text level="p">
      This channel will store a timestamp every time a sample is collected from this
      group. This channel is called an 'index' channel because it is used to keep track
      of (or 'index') the time at which each sample was collected. You can use this
      channel for time-based lookups of data across multiple channels.
    </Text.Text>
  ),
};

const groupRoleDocumentation: Record<string, ReactElement> = {
  analogInput: (
    <Text.Text level="p">
      This group of channels will store data from the analog inputs on your device. The
      first channel (the 'index' channel) will store timestamps for each sample
      collected from the other channels in the group.
    </Text.Text>
  ),
  digitalOutputCommand: (
    <Text.Text level="p">
      By default, this group has two channels. The command channel will store and send
      commands to the corresponding digital output on the device. The timestamp (or
      'index') channel will store timestamps for when the command was send.
    </Text.Text>
  ),
};

export interface CreateChannelsProps {
  applyDefaultGroups: () => void;
}

export const CreateChannels = ({
  applyDefaultGroups,
}: CreateChannelsProps): ReactElement => {
  const model = Form.useField<string>({ path: "properties.model" }).value;
  const [mostRecentSelected, setMostRecentSelected] =
    useState<MostRecentSelectedState | null>(null);
  const [selectedGroup, setSelectedGroup] = useState<SelectedGroupState | undefined>(
    undefined,
  );
  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);

  const handleGroupSelect = useCallback(
    (keys: string[], index: number): void => {
      if (keys.length > 0) {
        setMostRecentSelected({ key: keys[0], type: "group", index });
        setSelectedGroup({ index, keys });
      } else {
        setMostRecentSelected(null);
        setSelectedGroup(undefined);
      }
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
    setSelectedGroup(undefined);
    setMostRecentSelected(null);
    setSelectedChannels([]);
  }, [setMostRecentSelected, setSelectedChannels]);

  return (
    <Align.Space
      direction="x"
      grow
      className={CSS(CSS.B("physical-plan"), CSS.B("ni"))}
      empty
    >
      <Align.Space
        direction="y"
        className={CSS.B("description")}
        justify="spaceBetween"
      >
        <Align.Space direction="y">
          <Text.Text level="h2" weight={600}>
            Here are the channels we'll create for your {model} device
          </Text.Text>
          <Align.Space className="description" direction="y" size="small">
            <Text.Text level="p">
              These channels will store data from the inputs and send commands to the
              outputs of your device. We'll just focus on creating them for now, and you
              can define parameters like calibration and sampling rate later.
            </Text.Text>
            <Text.Text level="p">
              They are separated into indepedent sampling groups. We've automatically
              identified the channel groupings we think would work well for your device.
            </Text.Text>
            <Text.Text level="p">
              <b>All channels in a group must be sampled together</b>. What does this
              mean? Let's say you have two analog input channels (ai_1 and ai_2) that
              are part of the same group. It's not possible to sample ai_1 at 1 kHz in
              one acquisition task and ai_2 at 10 Hz in another. If you need to sample
              them at different rates, split them into separate groups.
            </Text.Text>
            <Text.Text level="p">
              Click on a group to see what its purpose is, and split any groups that
              need to have indepedendent sampling rates.{" "}
            </Text.Text>
            <Text.Text level="p">
              Don't worry, you can reconfigure the channels for the device later if you
              need to.
            </Text.Text>
          </Align.Space>
        </Align.Space>
        <Align.Space direction="x" size="small">
          <Button.Button onClick={applyDefaultGroups} variant="outlined">
            Reset to Default Groups
          </Button.Button>
        </Align.Space>
      </Align.Space>
      <Align.Space direction="y" bordered className={CSS.B("form")} grow empty>
        <Align.Space direction="x" empty className={PCSS(PCSS.bordered("bottom"))} grow>
          <GroupList
            selectedGroups={selectedGroup?.keys ?? []}
            selectedGroupIndex={selectedGroup?.index}
            onSelectGroup={handleGroupSelect}
            clearSelection={clearSelection}
          />
          <Align.Space className={CSS.B("channels")} grow empty>
            <Header.Header level="h3">
              <Header.Title weight={500}>Channels</Header.Title>
            </Header.Header>
            {selectedGroup != null && (
              <ChannelList
                key={selectedGroup.index}
                selectedGroupIndex={selectedGroup.index}
                selectedChannels={selectedChannels}
                onSelectChannels={handleChannelSelect}
              />
            )}
          </Align.Space>
        </Align.Space>
        <Align.Space className={CSS.B("details")} grow empty>
          {mostRecentSelected != null && (
            <Details selected={mostRecentSelected} groupIndex={selectedGroup?.index} />
          )}
        </Align.Space>
      </Align.Space>
    </Align.Space>
  );
};

interface GroupListProps {
  selectedGroups: string[];
  selectedGroupIndex?: number;
  onSelectGroup: (keys: string[], index: number) => void;
  clearSelection: () => void;
}

const GroupList = ({
  selectedGroups,
  onSelectGroup,
  clearSelection,
  selectedGroupIndex,
}: GroupListProps): ReactElement => {
  const { value, remove, add } = Form.useFieldArray<GroupConfig>({
    path: "groups",
  });
  const menuProps = Menu.useContextMenu();
  return (
    <Align.Space className={CSS(CSS.B("groups"))} grow empty>
      <Header.Header level="h3">
        <Header.Title weight={500}>Sampling Groups</Header.Title>
        <Header.Actions>
          {[
            {
              onClick: () => {
                const key = nanoid();
                onSelectGroup([key], value.length);
                add(
                  {
                    key,
                    name: "New Group",
                    channels: [],
                    channelPrefix: "",
                    channelSuffix: "",
                    role: "unknown",
                  },
                  (selectedGroupIndex ?? 0) + 1,
                );
              },
              children: <Icon.Add />,
              size: "large",
            },
          ]}
        </Header.Actions>
      </Header.Header>
      <Menu.ContextMenu
        menu={({ keys }) => {
          const handleSelect = (key: string) => {
            switch (key) {
              case "remove": {
                const indices = keys.map((k) => value.findIndex((v) => v.key === k));
                remove(indices);
                const newSelectedGroup = value.findIndex((v) => !keys.includes(v.key));
                if (newSelectedGroup >= 0)
                  onSelectGroup([value[newSelectedGroup].key], newSelectedGroup);
                else clearSelection();
              }
            }
          };
          return (
            <Menu.Menu onChange={handleSelect} level="small" iconSpacing="small">
              <Menu.Item itemKey="remove" startIcon={<Icon.Close />}>
                Remove
              </Menu.Item>
            </Menu.Menu>
          );
        }}
        {...menuProps}
      >
        <List.List<string, GroupConfig> data={value}>
          <List.Selector<string, GroupConfig>
            value={selectedGroups as string[]}
            allowMultiple
            replaceOnSingle
            onChange={(
              keys: string[],
              { clickedIndex }: { clickedIndex: number | null },
            ) => clickedIndex != null && onSelectGroup(keys, clickedIndex)}
          >
            <List.Core<string, GroupConfig> grow>
              {(props) => <GroupListItem clearSelection={clearSelection} {...props} />}
            </List.Core>
          </List.Selector>
        </List.List>
      </Menu.ContextMenu>
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
    entry: { channels },
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
      const v = ctx.get<ChannelConfig[]>(path);
      ctx.set(
        path,
        v.value
          .concat(items.map((i) => ({ ...(i.data as ChannelConfig) })))
          .sort((a, b) => a.port - b.port),
      );
      setDraggingOver(false);
      return items;
    },
    onDragOver: () => setDraggingOver(true),
  });

  const name = Form.useFieldValue<string>(`groups.${index}.name`, true);
  const channelsAreValid = Form.useFieldValid(`groups.${index}.channels`);

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
        <Text.Text
          level="p"
          weight={500}
          color={!channelsAreValid && "var(--pluto-error-z)"}
        >
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
  onSelectChannels: List.UseSelectMultipleProps<string, ChannelConfig>["onChange"];
}

const CHANNEL_LIST_COLUMNS: Array<List.ColumnSpec<string, ChannelConfig>> = [
  {
    key: "port",
    name: "Port",
    visible: true,
    width: 40,
    render: ({ entry }) => (
      <Text.Text
        level="p"
        className={CSS.B("port")}
        style={{ marginLeft: 10, width: 30, minWidth: 30 }}
        shade={7}
      >
        {entry.port < 0 && entry.line < 0 ? "N/A" : entry.port}
        {entry.line >= 0 && `/${entry.line}`}
      </Text.Text>
    ),
  },
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
    render: ({ entry: { isIndex } }) => {
      return isIndex ? (
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
  const channels = Form.useFieldArray<ChannelConfig>({
    path: `groups.${selectedGroupIndex}.channels`,
  });

  return (
    <List.List<string, ChannelConfig> data={channels.value}>
      <List.Selector<string, ChannelConfig>
        value={selectedChannels}
        allowNone={false}
        onChange={onSelectChannels}
        replaceOnSingle
      >
        <List.Column.Header<string, ChannelConfig> columns={CHANNEL_LIST_COLUMNS} hide>
          <List.Core<string, ChannelConfig> grow>
            {(props) => <ChannelListItem {...props} groupIndex={selectedGroupIndex} />}
          </List.Core>
        </List.Column.Header>
      </List.Selector>
    </List.List>
  );
};

export const ChannelListItem = memo(
  ({
    groupIndex,
    ...props
  }: List.ItemProps<string, ChannelConfig> & {
    groupIndex: number;
  }): ReactElement | null => {
    const { startDrag, onDragEnd } = Haul.useDrag({
      type: "Device.Channel",
      key: props.entry.key,
    });

    const groupChannels = `groups.${groupIndex}.channels`;
    const prefix = `${groupChannels}.${props.index}`;

    const methods = Form.useContext();
    const [validPort, setValidPort] = useState<boolean>(
      methods.get(prefix, { optional: true })?.status.variant !== "error",
    );
    Form.useFieldListener({
      path: `groups.${groupIndex}.channels.${props.index}.port`,
      onChange: (state) => setValidPort(state.status.variant !== "error"),
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
          .get<ChannelConfig[]>(groupChannels)
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
  groupIndex?: number;
}

const Details = ({ selected, groupIndex }: DetailsProps): ReactElement | null => {
  if (groupIndex == null) return null;
  if (selected.type === "group")
    return <GroupForm key={selected.index} index={selected.index} />;
  return <ChannelForm index={selected.index} groupIndex={groupIndex} />;
};

interface ChannelFormProps {
  groupIndex: number;
  index: number;
}

const ChannelForm = ({ index, groupIndex }: ChannelFormProps): ReactElement | null => {
  const prefix = `groups.${groupIndex}.channels.${index}`;
  const ctx = Form.useContext();
  if (!ctx.has(prefix)) return null;

  const role = ctx.get<string>({ path: `${prefix}.role` }).value;

  return (
    <>
      <Form.Field<string> path={`${prefix}.name`} label="Name" showLabel={false}>
        {(p) => (
          <Input.Text
            variant="natural"
            level="h2"
            placeholder="Range Name"
            autoFocus
            {...p}
          />
        )}
      </Form.Field>
      <Align.Space direction="x" grow>
        <Form.Field<number> path={`${prefix}.dataType`} label="Data Type" grow>
          {(p) => (
            <Select.DataType {...p} location="top" allowNone={false} hideColumnHeader />
          )}
        </Form.Field>
        <Form.Field<number>
          path={`${prefix}.port`}
          label="Port"
          visible={(fs) => fs.value >= 0}
        >
          {(p) => <Input.Numeric {...p} />}
        </Form.Field>
      </Align.Space>
      <Form.Field<number>
        path={`${prefix}.line`}
        label="Line"
        visible={(fs) => fs.value >= 0}
      >
        {(p) => <Input.Numeric {...p} />}
      </Form.Field>
      <Note.Note variant="info">{channelRoleDocumentation[role]}</Note.Note>
    </>
  );
};

interface GroupFormProps {
  index: number;
}

const GroupForm = ({ index }: GroupFormProps): ReactElement => {
  const prefix = `groups.${index}`;
  const role = Form.useField<string>({ path: `${prefix}.role` }).value;
  return (
    <>
      <Form.Field<string>
        path={`${prefix}.name`}
        label="Name"
        showLabel={false}
        hideIfNull
      >
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
      <Note.Note variant="info">{groupRoleDocumentation[role]}</Note.Note>
    </>
  );
};
