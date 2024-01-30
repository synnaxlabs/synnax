import { useState, type ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";
import { Haul, Header, Text, Select } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Input } from "@synnaxlabs/pluto/input";
import { List } from "@synnaxlabs/pluto/list";
import { nanoid } from "nanoid";
import {
  useController,
  useFieldArray,
  useFormContext,
  useWatch,
} from "react-hook-form";

import { CSS } from "@/css";
import {
  type Configuration,
  type Module,
  type Channel,
  type Group,
} from "@/hardware/device/new/types";

import "@/hardware/device/new/ModuleForm.css";

const CATEGORIES_TITLES = {
  "multifunction-io": "Multifunction I/O",
  voltage: "Analog Voltage",
  current: "Analog Current",
};

export interface ModuleFormProps {
  moduleIndex: number;
}

interface MostRecentSelectedState {
  type: "group" | "channel";
  index: number;
}

export const ModuleForm = ({ moduleIndex }: ModuleFormProps): ReactElement => {
  const [mostRecentSelected, setMostRecentSelected] =
    useState<MostRecentSelectedState | null>(null);
  const [selectedGroup, setSelectedGroup] = useState<string | undefined>(undefined);
  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);

  const mod = useWatch<Configuration>({
    name: `modules.${moduleIndex}`,
  }) as Module;
  const groups = mod.groups;
  const groupsFieldArr = useFieldArray<Configuration>({
    name: `modules.${moduleIndex}.groups`,
  });
  const channels = groups.find((g) => g.key === selectedGroup)?.channels ?? [];

  const handleGroupSelect = (key: string): void => {
    setSelectedGroup(key);
    setMostRecentSelected({
      type: "group",
      index: groups.findIndex((g) => g.key === key),
    });
  };

  const handleChannelSelect = (keys: string[]): void => {
    setSelectedChannels(keys);
    setMostRecentSelected({
      type: "channel",
      index:
        groups
          .find((g) => g.key === selectedGroup)
          ?.channels.findIndex((c) => c.key === keys[0]) ?? 0,
    });
  };

  return (
    <Align.Space direction="y" className={CSS.B("module-form")} empty>
      <Align.Space direction="y" className={CSS.B("header")}>
        <Text.Text level="h2" weight={600}>
          Here's how we'll setup your {mod.model} Module
        </Text.Text>
        <Text.Text level="p" shade={8}>
          We've automatically categorized the input and output channels into logical
          groups for you.
        </Text.Text>
      </Align.Space>
      <Align.Space direction="x" className={CSS.B("config")} grow empty>
        <Align.Space className={CSS.B("groups")} grow empty>
          <Header.Header level="h3">
            <Header.Title weight={500}>Groups</Header.Title>
            <Header.Actions>
              {[
                {
                  icon: "add",
                  label: "Add Group",
                  onClick: () => {
                    groupsFieldArr.prepend({
                      key: nanoid(),
                      name: "New Group",
                      channels: [],
                    });
                  },
                  children: <Icon.Add />,
                },
              ]}
            </Header.Actions>
          </Header.Header>
          <List.List<string, Group> data={groups}>
            <List.Selector<string, Group>
              allowMultiple={false}
              value={selectedGroup}
              allowNone={false}
              onChange={([key]) => handleGroupSelect(key)}
            />
            <List.Core<string, Group> grow>
              {(props) => <GroupListItem {...props} moduleIndex={moduleIndex ?? -1} />}
            </List.Core>
          </List.List>
        </Align.Space>
        <Align.Space className={CSS.B("channels")} grow empty>
          <Header.Header level="h3">
            <Header.Title weight={500}>Channels</Header.Title>
          </Header.Header>
          <List.List<string, Channel> data={channels}>
            <List.Selector<string, Channel>
              value={selectedChannels}
              allowNone={false}
              onChange={handleChannelSelect}
              replaceOnSingle
            />
            <List.Column.Header
              columns={[
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
                      {entry.port === 0 && entry.line === 0 ? "N/A" : entry.port}
                      {entry.line !== 0 && `/${entry.line}`}
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
              ]}
              hide
            />
            <List.Core<string, Channel> grow>
              {(props) => (
                <ChannelListItem
                  {...props}
                  moduleIndex={moduleIndex ?? -1}
                  groupIndex={groupsFieldArr.fields.findIndex(
                    (g) => g.key === selectedGroup,
                  )}
                />
              )}
            </List.Core>
          </List.List>
        </Align.Space>
        <Align.Space className={CSS.B("details")} grow>
          {mostRecentSelected != null && (
            <Details
              selected={mostRecentSelected}
              moduleIndex={moduleIndex ?? -1}
              groupIndex={groupsFieldArr.fields.findIndex(
                (g) => g.key === selectedGroup,
              )}
            />
          )}
        </Align.Space>
      </Align.Space>
    </Align.Space>
  );
};

export const ModuleListItem = (props: List.ItemProps<string, Module>): ReactElement => {
  const {
    entry: { slot, model, category },
  } = props;
  return (
    <List.ItemFrame {...props}>
      <Align.Space direction="y" size="small">
        <Align.Space direction="x" size="small">
          <Text.Text weight={600} level="p">
            Slot {slot}
          </Text.Text>
        </Align.Space>
        <Text.Text level="p">
          {model} {CATEGORIES_TITLES[category]} Module
        </Text.Text>
      </Align.Space>
    </List.ItemFrame>
  );
};

export const GroupListItem = (
  props: List.ItemProps<string, Group> & { moduleIndex: number },
): ReactElement => {
  const {
    index,
    entry: { name, channels },
    hovered,
  } = props;

  const [draggingOver, setDraggingOver] = useState<boolean>(false);

  const ctx = useFormContext();

  const drop = Haul.useDrop({
    type: "Device.Group",
    key: props.entry.key,
    canDrop: ({ source }) => source.type === "Device.Channel",
    onDrop: ({ items }) => {
      const v = ctx.getValues(`modules.${props.moduleIndex}.groups.${index}.channels`);
      ctx.setValue(
        `modules.${props.moduleIndex}.groups.${index}.channels`,
        v.concat(items.map((i) => i.data)),
      );
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
    </List.ItemFrame>
  );
};

export const ChannelListItem = (
  props: List.ItemProps<string, Channel> & { moduleIndex: number; groupIndex: number },
): ReactElement => {
  const { startDrag, onDragEnd } = Haul.useDrag({
    type: "Device.Channel",
    key: props.entry.key,
  });

  const arr = useFieldArray({
    name: `modules.${props.moduleIndex}.groups.${props.groupIndex}.channels`,
  });

  const badPort = useController({
    name: `modules.${props.moduleIndex}.groups.${props.groupIndex}.channels.${props.index}.port`,
  }).fieldState.invalid;

  const {
    select: { value },
    sourceData,
  } = List.useContext();

  const selected: Haul.Item[] = sourceData
    .filter((c) => value.includes(c.key) && c.key !== props.entry.key)
    .map((v) => ({
      key: v.key,
      type: "Device.Channel",
      data: v,
    }));
  selected.push({
    key: props.entry.key,
    type: "Device.Channel",
    data: props.entry,
  });

  return (
    <List.Column.Item
      {...props}
      draggable
      className={badPort ? CSS.B("bad-port") : ""}
      onDragStart={() =>
        startDrag(selected, ({ dropped }) => {
          const indexes = dropped.map((i) =>
            sourceData.findIndex((f) => f.key === i.key),
          );
          arr.remove(indexes);
        })
      }
      onDragEnd={onDragEnd}
    />
  );
};

export interface DetailsProps {
  selected: MostRecentSelectedState;
  moduleIndex: number;
  groupIndex: number;
}

const Details = ({ selected, moduleIndex, groupIndex }: DetailsProps): ReactElement => {
  if (selected.type === "group")
    return <GroupForm index={selected.index} moduleIndex={moduleIndex} />;
  return (
    <ChannelForm
      key={selected.index}
      index={selected.index}
      moduleIndex={moduleIndex}
      groupIndex={groupIndex}
    />
  );
};

interface ChannelFormProps {
  moduleIndex: number;
  groupIndex: number;
  index: number;
}

const ChannelForm = ({
  index,
  moduleIndex,
  groupIndex,
}: ChannelFormProps): ReactElement => {
  const { watch, control: c } = useFormContext();
  const prefix = `modules.${moduleIndex}.groups.${groupIndex}.channels.${index}`;
  const line = watch(`${prefix}.line` as const);
  const port = watch(`${prefix}.port` as const);
  const isIndex = watch(`${prefix}.isIndex` as const);

  return (
    <>
      <Input.HFItem<string> name={`${prefix}.name`} label="Name" showLabel={false}>
        {(props) => <Input.TextArea {...props} />}
      </Input.HFItem>
      <Input.HFItem<number> name={`${prefix}.dataType`} label="Data Type">
        {(props) => (
          <Select.DataType
            {...props}
            allowNone={false}
            hideColumnHeader
            disabled={isIndex}
          />
        )}
      </Input.HFItem>
      {port !== 0 && (
        <Input.HFItem<number>
          alsoValidate={[`modules.${moduleIndex}.groups.${groupIndex}.channels`]}
          name={`${prefix}.port`}
          label="Port"
        >
          {(props) => <Input.Numeric {...props} />}
        </Input.HFItem>
      )}
      {line !== 0 && (
        <Input.HFItem<number> name={`${prefix}.line`} label="Line">
          {(props) => <Input.Numeric {...props} />}
        </Input.HFItem>
      )}
    </>
  );
};

interface GroupFormProps {
  index: number;
  moduleIndex: number;
}

const GroupForm = ({ index: Key }: GroupFormProps): ReactElement => {
  return <></>;
};
