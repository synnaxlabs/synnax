import { useState, type ReactElement, memo, useCallback } from "react";

import { Icon } from "@synnaxlabs/media";
import {
  Form,
  Haul,
  Select,
  type UseSelectOnChangeExtra,
  type UseSelectProps,
} from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Header } from "@synnaxlabs/pluto/header";
import { Input } from "@synnaxlabs/pluto/input";
import { List } from "@synnaxlabs/pluto/list";
import { Text } from "@synnaxlabs/pluto/text";
import { nanoid } from "nanoid";

import { CSS } from "@/css";
import {
  type PhysicalChannelPlan,
  type PhysicalGroupPlan,
} from "@/hardware/device/new/types";

import "@/hardware/device/new/PhysicalPlanForm.css";

interface MostRecentSelectedState {
  key: string;
  type: "group" | "channel";
  index: number;
}

interface SelectedGroupState {
  index: number;
  key: string;
}

export const PhysicalPlanForm = (): ReactElement => {
  const model = Form.useField<string>({ path: "properties.model" }).value;
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
      { clickedIndex, clicked }: UseSelectOnChangeExtra<string>,
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
  clearSelection: () => void;
}

const GroupList = ({
  selectedGroup,
  onSelectGroup,
  clearSelection,
}: GroupListProps): ReactElement => {
  const { push, value } = Form.useFieldArray<PhysicalGroupPlan>({
    path: "physicalPlan.groups",
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
      <List.List<string, PhysicalGroupPlan> data={value}>
        <List.Selector<string, PhysicalGroupPlan>
          allowMultiple={false}
          // eslint-disable-next-line @typescript-eslint/non-nullable-type-assertion-style
          value={selectedGroup as string}
          allowNone={false}
          onChange={(key, { clickedIndex }) =>
            clickedIndex != null && onSelectGroup(key, clickedIndex)
          }
        >
          <List.Core<string, PhysicalGroupPlan> grow>
            {(props) => <GroupListItem clearSelection={clearSelection} {...props} />}
          </List.Core>
        </List.Selector>
      </List.List>
    </Align.Space>
  );
};

export interface GroupListItemProps extends List.ItemProps<string, PhysicalGroupPlan> {
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
  } = props;

  const [draggingOver, setDraggingOver] = useState<boolean>(false);

  const ctx = Form.useContext();

  const drop = Haul.useDrop({
    type: "Device.Group",
    key: props.entry.key,
    canDrop: ({ source }) => source.type === "Device.Channel",
    onDrop: ({ items }) => {
      props.onSelect?.(props.entry.key);
      const path = `physicalPlan.groups.${index}.channels`;
      const v = ctx.get<PhysicalChannelPlan[]>(path, false);
      ctx.set(
        path,
        v.value
          .concat(items.map((i) => ({ ...(i.data as PhysicalChannelPlan) })))
          .sort((a, b) => a.port - b.port),
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
  onSelectChannels: UseSelectProps["onChange"];
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
  const channels = Form.useFieldArray<PhysicalChannelPlan[]>({
    path: `physicalPlan.groups.${selectedGroupIndex}.channels`,
  });

  return (
    <Align.Space className={CSS.B("channels")} grow empty>
      <Header.Header level="h3">
        <Header.Title weight={500}>Channels</Header.Title>
      </Header.Header>
      <List.List<string, PhysicalChannelPlan> data={channels.value}>
        <List.Selector<string, PhysicalChannelPlan>
          value={selectedChannels}
          allowNone={false}
          onChange={onSelectChannels}
          replaceOnSingle
        >
          <List.Column.Header<string, PhysicalChannelPlan>
            columns={CHANNEL_LIST_COLUMNS}
            hide
          >
            <List.Core<string, PhysicalChannelPlan> grow>
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
  }: List.ItemProps<string, PhysicalChannelPlan> & {
    groupIndex: number;
  }): ReactElement => {
    const { startDrag, onDragEnd } = Haul.useDrag({
      type: "Device.Channel",
      key: props.entry.key,
    });

    const groupChannels = `physicalPlan.groups.${groupIndex}.channels`;
    const prefix = `${groupChannels}.${props.index}`;

    const methods = Form.useContext();
    const [validPort, setValidPort] = useState<boolean>(
      methods.get(prefix, true)?.status.variant !== "error",
    );
    Form.useFieldListener(
      `physicalPlan.groups.${groupIndex}.channels.${props.index}.port`,
      (state) => {
        setValidPort(state.status.variant !== "error");
      },
    );

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
          .get<PhysicalChannelPlan[]>(groupChannels, false)
          .value.filter((c) => selected.includes(c.key));
        haulItems = channels.map((c) => ({
          key: c.key,
          type: "Device.Channel",
          data: { ...c },
        }));
      }
      startDrag(haulItems, ({ dropped }) => {
        const keys = dropped.map((d) => d.key);
        const channels = methods.get<PhysicalChannelPlan[]>(groupChannels, false).value;
        methods.set(
          groupChannels,
          channels.filter((c) => !keys.includes(c.key)),
        );
      });
    }, [startDrag, props.entry.key, groupIndex, getSelected, methods.get, methods.set]);

    const childValues = Form.useChildFieldValues<PhysicalChannelPlan>({
      path: prefix,
      allowNull: true,
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
  if (selected.type === "group") return <GroupForm index={selected.index} />;
  return <ChannelForm index={selected.index} groupIndex={groupIndex} />;
};

interface ChannelFormProps {
  groupIndex: number;
  index: number;
}

const ChannelForm = ({ index, groupIndex }: ChannelFormProps): ReactElement | null => {
  const prefix = `physicalPlan.groups.${groupIndex}.channels.${index}`;
  const ctx = Form.useContext();
  if (!ctx.has(prefix)) return null;

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
      <Form.Field<number> path={`${prefix}.dataType`} label="Data Type">
        {(p) => <Select.DataType {...p} allowNone={false} hideColumnHeader />}
      </Form.Field>
      <Form.Field<number>
        path={`${prefix}.port`}
        label="Port"
        visible={(fs) => fs.value !== 0}
      >
        {(p) => <Input.Numeric {...p} />}
      </Form.Field>
      <Form.Field<number>
        path={`${prefix}.line`}
        label="Line"
        visible={(fs) => fs.value !== 0}
      >
        {(p) => <Input.Numeric {...p} />}
      </Form.Field>
    </>
  );
};

interface GroupFormProps {
  index: number;
}

const GroupForm = ({ index }: GroupFormProps): ReactElement => {
  return <h2>{index}</h2>;
};
