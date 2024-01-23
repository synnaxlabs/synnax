import { useState, type ReactElement } from "react";

import { Icon } from "@synnaxlabs/media";
import { Haul, Header, Text, componentRenderProp } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { List } from "@synnaxlabs/pluto/list";
import { nanoid } from "nanoid";
import {
  useWatch,
  type useForm,
  useFieldArray,
  useFormState,
  useFormContext,
} from "react-hook-form";

import { CSS } from "@/css";
import {
  type Configuration,
  type Module,
  type Channel,
  type Group,
} from "@/hardware/device/new/types";

import "@/hardware/device/new/Hardware.css";

const CATEGORIES_TITLES = {
  "multifunction-io": "Multifunction I/O",
  voltage: "Analog Voltage",
  current: "Analog Current",
};

export interface HardwareProps
  extends Pick<ReturnType<typeof useForm<Configuration>>, "control"> {}

export const Hardware = (): ReactElement => {
  const [selectedModule, setSelectedModule] = useState<string | undefined>(undefined);
  const [selectedGroup, setSelectedGroup] = useState<string | undefined>(undefined);
  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);

  const handleSelectModule = (key: string) => {
    setSelectedModule(key);
    setSelectedGroup(undefined);
    setSelectedChannels([]);
  };

  const modules = useWatch({ name: "modules" });
  const selectedModuleIndex = modules?.findIndex((m) => m.key === selectedModule);
  const groups = modules?.find((m) => m.key === selectedModule)?.groups ?? [];
  const channels = groups?.find((g) => g.key === selectedGroup)?.channels ?? [];
  const groupFieldArr = useFieldArray({
    name: `modules.${selectedModuleIndex}.groups`,
  });

  console.log(modules, groups, channels);

  return (
    <Align.Space direction="y" className={CSS.B("hardware")} empty>
      <Align.Space direction="y" className={CSS.B("header")}>
        <Text.Text level="h2" weight={600}>
          Here's how we'll setup your hardware
        </Text.Text>
        <Text.Text level="p" shade={8}>
          We've automatically detected modules and placed their channels into groups for
          you.
        </Text.Text>
      </Align.Space>
      <Align.Space direction="x" className={CSS.B("config")} grow empty>
        <Align.Space direction="y" className={CSS.B("modules")} grow empty>
          <Header.Header level="h3">
            <Header.Title weight={500}>Modules</Header.Title>
          </Header.Header>
          <List.List<string, Module> data={modules}>
            <List.Selector<string, Module>
              allowMultiple={false}
              value={selectedModule}
              allowNone={false}
              onChange={([key]) => handleSelectModule(key)}
            />

            <List.Core<string, Module> grow>
              {componentRenderProp(ModuleListItem)}
            </List.Core>
          </List.List>
        </Align.Space>
        <Align.Space className={CSS.B("groups")} grow empty>
          <Header.Header level="h3">
            <Header.Title weight={500}>Groups</Header.Title>
            <Header.Actions>
              {[
                {
                  icon: "add",
                  label: "Add Group",
                  onClick: () => {
                    groupFieldArr.prepend({
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
              onChange={([key]) => setSelectedGroup(key)}
            />
            <List.Core<string, Group> grow>
              {(props) => (
                <GroupListItem {...props} moduleIndex={selectedModuleIndex ?? -1} />
              )}
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
              onChange={setSelectedChannels}
              replaceOnSingle
            />
            <List.Core<string, Channel> grow>
              {(props) => (
                <ChannelListItem
                  {...props}
                  moduleIndex={selectedModuleIndex ?? -1}
                  groupIndex={groups.findIndex((g) => g.key === selectedGroup)}
                />
              )}
            </List.Core>
          </List.List>
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
        <Align.Space direction="x">
          <Text.Text level="p">{channels.length}</Text.Text>
          <Text.Text level="p">Channels</Text.Text>
        </Align.Space>
      </Align.Space>
    </List.ItemFrame>
  );
};

export const ChannelListItem = (
  props: List.ItemProps<string, Channel> & { moduleIndex: number; groupIndex: number },
): ReactElement => {
  const {
    entry: { name, dataType },
  } = props;
  const { startDrag, onDragEnd } = Haul.useDrag({
    type: "Device.Channel",
    key: props.entry.key,
  });

  const formCtx = useFormContext();
  const arr = useFieldArray({
    name: `modules.${props.moduleIndex}.groups.${props.groupIndex}.channels`,
  });

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

  console.log(props.hovered);

  return (
    <List.ItemFrame
      {...props}
      draggable
      onDragStart={() =>
        startDrag(selected, ({ dropped }) => {
          const indexes = dropped.map((i) =>
            sourceData.findIndex((f) => f.key === i.key),
          );
          arr.remove(indexes);
        })
      }
      onDragEnd={onDragEnd}
    >
      <Align.Space direction="x" size="large">
        <Text.Text level="p" shade={7}>
          {dataType}
        </Text.Text>
        <Text.Text level="p" weight={500}>
          {name}
        </Text.Text>
      </Align.Space>
    </List.ItemFrame>
  );
};
