// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/opc/device/CreateChannels.css";

import { Icon } from "@synnaxlabs/media";
import { CSS as PCSS, Form, Haul, Menu, Note, Select } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Header } from "@synnaxlabs/pluto/header";
import { Input } from "@synnaxlabs/pluto/input";
import { List } from "@synnaxlabs/pluto/list";
import { Text } from "@synnaxlabs/pluto/text";
import { nanoid } from "nanoid/non-secure";
import { memo, type ReactElement, useCallback, useState } from "react";

import { CSS } from "@/css";
import { SelectNode } from "@/hardware/opc/device/SelectNode";
import { type ChannelConfig, type GroupConfig } from "@/hardware/opc/device/types";
import { Properties } from "@/hardware/opc/device/types";

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
  deviceProperties: Properties;
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
      if (clickedIndex == null || clicked == null || clickedIndex === -1) return;
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
        <Align.Space
          direction="x"
          empty
          className={PCSS(PCSS.bordered("bottom"), CSS.B("list"))}
        >
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
  const { push, value, remove } = Form.useFieldArray<GroupConfig>({ path: "groups" });
  const menuProps = Menu.useContextMenu();
  return (
    <Align.Space className={CSS.B("groups")} grow empty>
      <Header.Header level="h3">
        <Header.Title weight={500}>Sampling Groups</Header.Title>
        <Header.Actions>
          {[
            {
              size: "large",
              onClick: () => {
                const key = nanoid();
                onSelectGroup(key, value.length);
                push({
                  key,
                  name: `Group ${value.length + 1}`,
                  channels: [
                    {
                      key: nanoid(),
                      name: `group_1_${value.length + 1}_time`,
                      dataType: "timestamp",
                      isIndex: true,
                      isArray: false,
                      nodeId: "",
                    },
                  ],
                });
              },
              children: <Icon.Add />,
            },
          ]}
        </Header.Actions>
      </Header.Header>
      <Menu.ContextMenu
        menu={({ keys }: Menu.ContextMenuMenuProps): ReactElement => {
          const handleSelect = (key: string) => {
            switch (key) {
              case "remove": {
                const indices = keys.map((k) => value.findIndex((g) => g.key === k));
                remove(indices);
                // find the first group whose key is not in keys
                const newSelectedGroup = value.findIndex((g) => !keys.includes(g.key));
                if (newSelectedGroup >= 0)
                  onSelectGroup(value[newSelectedGroup].key, newSelectedGroup);
                break;
              }
            }
          };
          return (
            <Menu.Menu onChange={handleSelect} level="small">
              {value.length > 1 && (
                <Menu.Item itemKey="remove" startIcon={<Icon.Close />}>
                  Remove
                </Menu.Item>
              )}
            </Menu.Menu>
          );
        }}
        {...menuProps}
      >
        <List.List<string, GroupConfig> data={value}>
          <List.Selector<string, GroupConfig>
            allowMultiple={false}
            value={selectedGroup as string}
            allowNone={false}
            onChange={(
              key: string,
              { clickedIndex }: { clickedIndex: number | null },
            ) => clickedIndex != null && onSelectGroup(key, clickedIndex)}
          >
            <List.Core<string, GroupConfig> grow>
              {({ key, ...props }) => (
                <GroupListItem key={key} clearSelection={clearSelection} {...props} />
              )}
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
      const v = ctx.get<ChannelConfig[]>(path);
      ctx.set(
        path,
        v.value.concat(items.map((i) => ({ ...(i.data as ChannelConfig) }))),
      );
      setDraggingOver(false);
      return items;
    },
    onDragOver: () => setDraggingOver(true),
  });

  const isValid = Form.useFieldValid(`groups.${index}.channels`);

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
        <Text.Text level="p" weight={500} color={!isValid && "var(--pluto-error-z)"}>
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

const ChannelList = ({
  selectedChannels,
  selectedGroupIndex,
  onSelectChannels,
}: ChannelListProps): ReactElement => {
  const channels = Form.useFieldArray<ChannelConfig>({
    path: `groups.${selectedGroupIndex}.channels`,
  });
  const menuProps = Menu.useContextMenu();
  return (
    <Align.Space className={CSS.B("channels")} grow empty>
      <Header.Header level="h3" style={{ borderBottom: "none" }}>
        <Header.Title weight={500}>Channels</Header.Title>
      </Header.Header>

      <List.List<string, ChannelConfig> data={channels.value}>
        <List.Filter>
          {(p) => (
            <Input.Text
              placeholder="Search Channels"
              selectOnFocus
              style={{ border: "none", borderBottom: "var(--pluto-border)" }}
              {...p}
            />
          )}
        </List.Filter>
        <List.Selector<string, ChannelConfig>
          value={selectedChannels}
          allowNone={false}
          onChange={onSelectChannels}
          replaceOnSingle
        >
          <Menu.ContextMenu
            menu={({ keys }: Menu.ContextMenuMenuProps): ReactElement => {
              const handleSelect = (key: string) => {
                const indices = keys.map((k) =>
                  channels.value.findIndex((c) => c.key === k),
                );
                switch (key) {
                  case "remove":
                    channels.remove(indices);
                    break;
                  case "keep": {
                    const idxIndex = channels.value.findIndex(
                      (c) => c.isIndex === true,
                    );
                    channels.keepOnly([idxIndex, ...indices]);
                    break;
                  }
                }
              };
              return (
                <Menu.Menu onChange={handleSelect} level="small">
                  <Menu.Item itemKey="remove" startIcon={<Icon.Close />}>
                    Remove
                  </Menu.Item>
                  <Menu.Item itemKey="keep" startIcon={<Icon.Check />}>
                    Keep Only Selected
                  </Menu.Item>
                </Menu.Menu>
              );
            }}
            {...menuProps}
          >
            <List.Core<string, ChannelConfig> grow>
              {({ key, ...props }) => (
                <ChannelListItem key={key} {...props} groupIndex={selectedGroupIndex} />
              )}
            </List.Core>
          </Menu.ContextMenu>
        </List.Selector>
      </List.List>
    </Align.Space>
  );
};

export const ChannelListItem = memo(
  ({
    groupIndex,
    sourceIndex,
    ...props
  }: List.ItemProps<string, ChannelConfig> & {
    groupIndex: number;
  }): ReactElement | null => {
    const { startDrag, onDragEnd } = Haul.useDrag({
      type: "Device.Channel",
      key: props.entry.key,
    });
    const groupChannels = `groups.${groupIndex}.channels`;
    const prefix = `${groupChannels}.${sourceIndex}`;
    const methods = Form.useContext();
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
        const channels = methods.get<ChannelConfig[]>(groupChannels).value;
        methods.set(
          groupChannels,
          channels.filter((c) => !keys.includes(c.key)),
        );
      });
    }, [startDrag, props.entry.key, groupIndex, getSelected, methods.get, methods.set]);

    const isValid = Form.useFieldValid(prefix);

    const childValues = Form.useChildFieldValues<ChannelConfig>({
      path: prefix,
      optional: true,
    });
    const isIndex = childValues?.isIndex ?? false;

    if (childValues == null) return null;
    return (
      <List.ItemFrame
        {...props}
        entry={childValues}
        draggable
        onDragStart={handleDragStart}
        onDragEnd={onDragEnd}
        direction="x"
        justify="spaceBetween"
        style={{ padding: "1.25rem 1.5rem" }}
        align="center"
      >
        <Align.Space direction="y" grow size="small">
          <Text.Text level="p" weight={500} color={!isValid && "var(--pluto-error-z)"}>
            {childValues.name}
          </Text.Text>
          {childValues.nodeId != null && (
            <Text.Text level="small" shade={7}>
              {childValues.nodeId}
            </Text.Text>
          )}
        </Align.Space>
        <Align.Space direction="y" empty align="end">
          <Text.Text level="p" shade={7}>
            {childValues.dataType}
            {childValues.isArray ? "[]" : ""}
          </Text.Text>
          {isIndex && (
            <Text.Text level="p" shade={7} color="var(--pluto-secondary-z)">
              {isIndex ? "Index" : ""}
            </Text.Text>
          )}
        </Align.Space>
      </List.ItemFrame>
    );
  },
);
ChannelListItem.displayName = "ChannelListItem";

export interface DetailsProps {
  selected: MostRecentSelectedState;
  deviceProperties: Properties;
  groupIndex?: number;
}

const Details = ({
  selected,
  groupIndex,
  deviceProperties,
}: DetailsProps): ReactElement | null => {
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
  deviceProperties: Properties;
}

const ChannelForm = ({
  index,
  groupIndex,
  deviceProperties,
}: ChannelFormProps): ReactElement | null => {
  const prefix = `groups.${groupIndex}.channels.${index}`;
  const fieldState = Form.useFieldState(prefix, true);
  const isIndex = Form.useFieldValue(`${prefix}.isIndex`, true);
  if (fieldState == null || isIndex == null) return null;
  return (
    <Align.Space direction="y" size="small">
      <Form.Field<string> path={`${prefix}.name`} label="Name">
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
      <Align.Space direction="x" size="small" grow>
        <Form.Field<string> path={`${prefix}.dataType`} label="Data Type" grow>
          {(p) => (
            <Select.DataType {...p} location="top" allowNone={false} hideColumnHeader />
          )}
        </Form.Field>
        <Form.SwitchField label="Is Index" path={`${prefix}.isIndex`}>
          {(p) => <Input.Switch {...p} />}
        </Form.SwitchField>
      </Align.Space>
      <Form.Field<string> path={`${prefix}.nodeId`} label="Node ID" optional>
        {({ onChange, ...props }) => (
          <SelectNode
            {...props}
            data={deviceProperties.channels}
            onChange={(value: string) => onChange(value ?? "")}
            allowNone={isIndex}
          />
        )}
      </Form.Field>
      {fieldState?.status.variant === "error" && (
        <Note.Note
          variant="error"
          style={{
            padding: "2rem",
            margin: 0,
          }}
        >
          <Text.Text level="p">{fieldState.status.message}</Text.Text>
        </Note.Note>
      )}
    </Align.Space>
  );
};

interface GroupFormProps {
  index: number;
}

const GroupForm = ({ index }: GroupFormProps): ReactElement => {
  const prefix = `groups.${index}`;
  const fs = Form.useFieldState(`groups.${index}.channels`);
  return (
    <Align.Space direction="y" size="small">
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
      {fs?.status.variant === "error" && (
        <Note.Note
          variant="error"
          style={{
            padding: "2rem",
            margin: 0,
          }}
        >
          <Text.Text level="p">{fs.status.message}</Text.Text>
        </Note.Note>
      )}
    </Align.Space>
  );
};
