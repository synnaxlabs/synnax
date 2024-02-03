import { useState, type ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";
import { Haul, Select } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Header } from "@synnaxlabs/pluto/header";
import { Input } from "@synnaxlabs/pluto/input";
import { List } from "@synnaxlabs/pluto/list";
import { Text } from "@synnaxlabs/pluto/text";
import { nanoid } from "nanoid";
import {
  useController,
  useFieldArray,
  useFormContext,
  useWatch,
} from "react-hook-form";

import { CSS } from "@/css";
import {
  type PhysicalChannelPlan,
  type PhysicalGroupPlan,
  type Configuration,
} from "@/hardware/device/new/types";

import "@/hardware/device/new/PhysicalPlanForm.css";

interface MostRecentSelectedState {
  type: "group" | "channel";
  index: number;
}

interface SelectedGroupState {
  index: number;
  key: string;
}

export const PhysicalPlanForm = (): ReactElement => {
  const model = useWatch<Configuration>({ name: "properties.model" });
  const [mostRecentSelected, setMostRecentSelected] =
    useState<MostRecentSelectedState | null>(null);
  const [selectedGroup, setSelectedGroup] = useState<SelectedGroupState | undefined>(
    undefined,
  );
  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);

  const handleGroupSelect = (key: string, index: number): void => {
    setSelectedGroup({ index, key });
    setMostRecentSelected({ type: "group", index });
  };

  const handleChannelSelect = (keys: string[], index: number): void => {
    setSelectedChannels(keys);
    setMostRecentSelected({ type: "channel", index });
  };

  return (
    <Align.Space direction="y" className={CSS.B("module-form")} empty>
      <Align.Space direction="y" className={CSS.B("header")}>
        <Text.Text level="h2" weight={600}>
          Here are the channels we'll create for your {model} device
        </Text.Text>
        <Align.Space className="description" direction="y" size="small">
          <Text.Text level="p" shade={8}>
            These channels will store data from the inputs and send commands to the
            outputs of your device.
          </Text.Text>
          <Text.Text level="p" shade={8}>
            They are separated into indepedent sampling groups.{" "}
            <b>All channels in a group must be sampled together</b>, although Each group
            has a special{" "}
            <span style={{ color: "var(--pluto-secondary-z)" }}>index </span>
            channel that stores the timestamps for the samples emitted by the rest of
            the channels in the group.
          </Text.Text>
          <Text.Text level="p" shade={8}>
            We've automatically identified the channel groupings we think would work
            well for your device. Click on a group to see what its purpose is, and split
            any groups that need to have indepedendent sampling rates.
          </Text.Text>
        </Align.Space>
      </Align.Space>
      <Align.Space direction="x" className={CSS.B("config")} grow empty>
        <GroupList
          selectedGroup={selectedGroup?.key}
          onSelectGroup={handleGroupSelect}
        />
        {selectedGroup != null && (
          <ChannelList
            key={selectedGroup.key}
            selectedGroupIndex={selectedGroup.index}
            selectedChannels={selectedChannels}
            onSelectChannels={handleChannelSelect}
          />
        )}
        <Align.Space className={CSS.B("details")} grow>
          {mostRecentSelected != null && (
            <Details selected={mostRecentSelected} groupIndex={selectedGroup?.index} />
          )}
        </Align.Space>
      </Align.Space>
    </Align.Space>
  );
};

interface GroupListProps {
  selectedGroup: string | undefined;
  onSelectGroup: (key: string, index: number) => void;
}

const GroupList = ({ selectedGroup, onSelectGroup }: GroupListProps): ReactElement => {
  const { prepend, fields } = useFieldArray<Configuration>({
    name: "physicalPlan.groups",
  });
  return (
    <Align.Space className={CSS.B("groups")} grow empty>
      <Header.Header level="h3">
        <Header.Title weight={500}>Groups</Header.Title>
        <Header.Actions>
          {[
            {
              icon: "add",
              label: "Add Group",
              onClick: () => {
                prepend({
                  key: nanoid(),
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
      <List.List<string, PhysicalGroupPlan> data={fields as PhysicalGroupPlan[]}>
        <List.Selector<string, PhysicalGroupPlan>
          allowMultiple={false}
          // eslint-disable-next-line @typescript-eslint/non-nullable-type-assertion-style
          value={selectedGroup as string}
          allowNone={false}
          onChange={([key], { clickedIndex }) =>
            clickedIndex != null && onSelectGroup(key, clickedIndex)
          }
        />
        <List.Core<string, PhysicalGroupPlan> grow>
          {(props) => <GroupListItem {...props} />}
        </List.Core>
      </List.List>
    </Align.Space>
  );
};

const GroupListItem = (
  props: List.ItemProps<string, PhysicalGroupPlan>,
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
      const v = ctx.getValues(`physicalPlan.groups.${index}.channels`);
      ctx.setValue(
        `physicalPlan.groups.${index}.channels`,
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

interface ChannelListProps {
  selectedGroupIndex: number;
  selectedChannels: string[];
  onSelectChannels: (keys: string[], index: number) => void;
}

const CHANNEL_LIST_COLUMNS: Array<List.ColumnSpec<string, PhysicalChannelPlan>> = [
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
];

const ChannelList = ({
  selectedChannels,
  selectedGroupIndex,
  onSelectChannels,
}: ChannelListProps): ReactElement => {
  const channels = useWatch<Configuration>({
    name: `physicalPlan.groups.${selectedGroupIndex}.channels`,
  });
  return (
    <Align.Space className={CSS.B("channels")} grow empty>
      <Header.Header level="h3">
        <Header.Title weight={500}>Channels</Header.Title>
      </Header.Header>
      <List.List<string, PhysicalChannelPlan> data={channels as PhysicalChannelPlan[]}>
        <List.Selector<string, PhysicalChannelPlan>
          value={selectedChannels}
          allowNone={false}
          onChange={(keys, { clickedIndex }) =>
            clickedIndex != null && onSelectChannels(keys, clickedIndex)
          }
          replaceOnSingle
        />
        <List.Column.Header<string, PhysicalChannelPlan>
          columns={CHANNEL_LIST_COLUMNS}
          hide
        />
        <List.Core<string, PhysicalChannelPlan> grow>
          {(props) => <ChannelListItem {...props} groupIndex={selectedGroupIndex} />}
        </List.Core>
      </List.List>
    </Align.Space>
  );
};

export const ChannelListItem = ({
  groupIndex,
  ...props
}: List.ItemProps<string, PhysicalChannelPlan> & {
  groupIndex: number;
}): ReactElement => {
  const { startDrag, onDragEnd } = Haul.useDrag({
    type: "Device.Channel",
    key: props.entry.key,
  });

  const arr = useFieldArray<Configuration>({
    name: `physicalPlan.groups.${groupIndex}.channels`,
  });

  const {
    fieldState: { invalid: invalidPort },
  } = useController({
    name: `physicalPlan.groups.${groupIndex}.channels.${props.index}.port`,
  });

  const { getValues } = useFormContext<Configuration>();

  const {
    select: { value },
  } = List.useContext();

  return (
    <List.Column.Item
      {...props}
      draggable
      className={invalidPort ? CSS.B("bad-port") : ""}
      onDragStart={() => {
        const channels = getValues(`physicalPlan.groups.${groupIndex}.channels`);
        const haulItems = channels
          .filter((k) => value.includes(k.key))
          .map((c) => ({ key: c.key, type: "Device.Channel", data: c }));
        haulItems.push({
          key: props.entry.key,
          type: "Device.Channel",
          data: props.entry,
        });
        startDrag(haulItems, ({ dropped }) =>
          arr.remove(dropped.map((i) => channels.findIndex((f) => f.key === i.key))),
        );
      }}
      onDragEnd={onDragEnd}
    />
  );
};

export interface DetailsProps {
  selected: MostRecentSelectedState;
  groupIndex?: number;
}

const Details = ({ selected, groupIndex }: DetailsProps): ReactElement | null => {
  if (groupIndex == null) return null;
  if (selected.type === "group") return <GroupForm index={selected.index} />;
  return (
    <ChannelForm key={selected.index} index={selected.index} groupIndex={groupIndex} />
  );
};

interface ChannelFormProps {
  groupIndex: number;
  index: number;
}

const ChannelForm = ({ index, groupIndex }: ChannelFormProps): ReactElement => {
  const { watch } = useFormContext();
  const prefix = `physicalPlan.groups.${groupIndex}.channels.${index}`;
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
          alsoValidate={[`groups.${groupIndex}.channels`]}
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
}

const GroupForm = ({ index: Key }: GroupFormProps): ReactElement => {
  return <></>;
};
