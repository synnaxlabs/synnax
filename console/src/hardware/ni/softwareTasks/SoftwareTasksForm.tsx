// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState, type ReactElement, useMemo, useEffect } from "react";

import {
  Align,
  Button,
  Channel,
  Form,
  Header,
  Select,
  Status,
  CSS as PCSS,
} from "@synnaxlabs/pluto";
import { Input } from "@synnaxlabs/pluto/input";
import { List } from "@synnaxlabs/pluto/list";
import { Text } from "@synnaxlabs/pluto/text";

import { CSS } from "@/css";
import {
  CHANNEL_TYPE_DISPLAY,
  type LinearScale,
  type NIChannel,
  type NITask,
} from "@/hardware/ni/types";
import { type GroupConfig } from "@/hardware/ni/device/types";

import "@/hardware/device/new/softwareTasks/SoftwareTasksForm.css";

interface MostRecentSelectedState {
  type: "task" | "channel";
  index: number;
}

interface SelectedTaskState {
  index: number;
  key: string;
}

export const SoftwareTasksForm = (): ReactElement => {
  const [mostRecentSelected, setMostRecentSelected] =
    useState<MostRecentSelectedState | null>();
  const [selectedTask, setSelectedTask] = useState<SelectedTaskState | null>();
  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);

  const handleSelectTask = (key: string, index: number): void => {
    setSelectedTask({ key, index });
    setMostRecentSelected({ type: "task", index });
  };

  const handleSelectChannels = (keys: string[], index: number): void => {
    setSelectedChannels(keys);
    setMostRecentSelected({ type: "channel", index });
  };

  return (
    <Align.Space direction="x" className={CSS.B("software-tasks")} size={10} grow>
      <Align.Space className={CSS.B("description")} direction="y" size="small">
        <Text.Text level="h2">
          Let's setup the first acquistiion & control tasks for your device.
        </Text.Text>
      </Align.Space>
      <Align.Space direction="y" className={CSS.B("form")} grow empty>
        <Align.Space direction="x" empty>
          <TaskList selectedTask={selectedTask?.key} onSelectTask={handleSelectTask} />
          {selectedTask != null && (
            <ChannelList
              selectedChannels={selectedChannels}
              selectedTaskIndex={selectedTask?.index}
              onSelectChannels={handleSelectChannels}
            />
          )}
        </Align.Space>
        {mostRecentSelected != null && (
          <Details selected={mostRecentSelected} taskIndex={selectedTask?.index} />
        )}
      </Align.Space>
    </Align.Space>
  );
};

interface TaskListProps {
  selectedTask: string | null;
  onSelectTask: (key: string, index: number) => void;
}

const TaskList = ({ selectedTask, onSelectTask }: TaskListProps): ReactElement => {
  const { value } = Form.useFieldArray<NITask>({
    path: "softwarePlan.tasks",
  });
  return (
    <Align.Space className={CSS.B("tasks")} grow empty>
      <Header.Header level="h3">
        <Header.Title level="h3">Tasks</Header.Title>
      </Header.Header>
      <List.List<string, NITask> data={value}>
        <List.Selector<string, NITask>
          allowMultiple={false}
          allowNone={false}
          value={selectedTask}
          onChange={(key, { clickedIndex }) =>
            clickedIndex != null && onSelectTask(key, clickedIndex)
          }
        >
          <List.Core<string, NITask>>
            {(props) => <TaskListItem {...props} />}
          </List.Core>
        </List.Selector>
      </List.List>
    </Align.Space>
  );
};

interface TaskListItemProps extends List.ItemProps<string, NITask> {}

const TaskListItem = (props: TaskListItemProps): ReactElement => {
  const {
    entry: { name },
  } = props;
  return (
    <List.ItemFrame {...props}>
      <Align.Space direction="y" size="small">
        <Text.Text level="p" weight={500}>
          {name}
        </Text.Text>
      </Align.Space>
    </List.ItemFrame>
  );
};

interface ChannelListProps {
  selectedTaskIndex: number;
  selectedChannels: string[];
  onSelectChannels: (keys: string[], index: number) => void;
}

const ChannelList = ({
  selectedChannels,
  selectedTaskIndex,
  onSelectChannels,
}: ChannelListProps): ReactElement => {
  const { value } = Form.useFieldArray<NIChannel>({
    path: `softwarePlan.tasks.${selectedTaskIndex}.config.channels`,
  });
  return (
    <Align.Space className={CSS.B("channels")} grow empty>
      <Header.Header level="h3">
        <Header.Title weight={500}>Channels</Header.Title>
      </Header.Header>
      <List.List<string, NIChannel> data={value}>
        <List.Selector<string, NIChannel>
          value={selectedChannels}
          allowNone={false}
          onChange={(keys, { clickedIndex }) =>
            clickedIndex != null && onSelectChannels(keys, clickedIndex)
          }
          replaceOnSingle
        >
          <List.Core<string, NIChannel> grow>
            {(props) => <ChannelListItem {...props} groupIndex={selectedTaskIndex} />}
          </List.Core>
        </List.Selector>
      </List.List>
    </Align.Space>
  );
};

export const ChannelListItem = ({
  groupIndex,
  ...props
}: List.ItemProps<string, NIChannel> & {
  groupIndex: number;
}): ReactElement => {
  const { entry } = props;
  const hasLine = "line" in entry;
  const childValues = Form.useChildFieldValues<NIChannel>({
    path: `softwarePlan.tasks.${groupIndex}.config.channels.${props.index}`,
    optional: true,
  });
  if (childValues == null) return <></>;
  return (
    <List.ItemFrame
      {...props}
      entry={childValues}
      justify="spaceBetween"
      align="center"
    >
      <Align.Space direction="y" size={0.5}>
        <Align.Space direction="x">
          <Text.Text level="p" weight={500} shade={6} style={{ width: "3rem" }}>
            {childValues.port} {hasLine && `/${entry.line}`}
          </Text.Text>
          <Text.Text level="p" weight={500} shade={9}>
            {childValues.name}
          </Text.Text>
        </Align.Space>
        <Text.Text level="p" shade={6}>
          {CHANNEL_TYPE_DISPLAY[entry.type]}
        </Text.Text>
      </Align.Space>
      <Button.Toggle
        checkedVariant="text"
        uncheckedVariant="text"
        value={entry.enabled}
        size="small"
        onClick={(e) => e.stopPropagation()}
        tooltip={
          <Text.Text level="small" style={{ maxWidth: 300 }}>
            Data acquisition for this channel is{" "}
            {entry.enabled ? "enabled" : "disabled"}. Click to
            {entry.enabled ? " disable" : " enable"} it.
          </Text.Text>
        }
      >
        <Status.Text
          variant={entry.enabled ? "success" : "disabled"}
          level="small"
          align="center"
        >
          {entry.enabled ? "Enabled" : "Disabled"}
        </Status.Text>
      </Button.Toggle>
    </List.ItemFrame>
  );
};

interface DetailsPorps {
  selected: MostRecentSelectedState;
  taskIndex?: number;
}

const Details = ({ selected, taskIndex }: DetailsPorps): ReactElement | null => {
  if (taskIndex == null) return null;
  if (selected.type === "task") {
    return <></>;
    // return <TaskForm index={selected.index} />;
  }
  return (
    <ChannelForm
      key={`${taskIndex}-${selected.index}`}
      taskIndex={taskIndex}
      index={selected.index}
    />
  );
};

interface ChannelFormProps {
  taskIndex: number;
  index: number;
}

const ChannelForm = ({ taskIndex, index }: ChannelFormProps): ReactElement => {
  const { get } = Form.useContext();
  const [scaleType, setScaleType] = useState(
    get<LinearScale>({
      path: `softwarePlan.tasks.${taskIndex}.config.channels.${index}.scale`,
      optional: true,
    })?.value?.type ?? "none",
  );
  const prefix = `softwarePlan.tasks.${taskIndex}.config.channels.${index}`;

  // We should only need to do this on first render, since the groups are static.
  const channelOptions = useMemo(
    () =>
      get<GroupConfig[]>({ path: `groups`, optional: false })
        .value.map((g) => g.channels)
        .flat(),
    [],
  );

  return (
    <Align.Space className={CSS.B("details")} grow empty>
      <Text.Text level="h3">Channel Properties</Text.Text>
      <Form.Field<number> label="Port" path={`${prefix}.port`}>
        {(p) => <Input.Numeric {...p} />}
      </Form.Field>
      {/* <Form.Field<number>
        label="Line"
        path={`${prefix}.line`}
        visible={(fs) => fs.value !== 0}
      >
        {(p) => <Input.Numeric {...p} />}
      </Form.Field> */}
      <Form.Field<string> label="Channel" path={`${prefix}.channel`}>
        {(p) => <Channel.SelectSingle data={channelOptions} {...p} />}
      </Form.Field>
      {/* <SelectScale value={scaleType} onChange={setScaleType} /> */}
      {scaleType === "two-point-linear" && <LinearTwoPoint name={`${prefix}.scale`} />}
    </Align.Space>
  );
};

const SCALE_DATA = [
  {
    key: "none",
    label: "None",
  },
  {
    key: "slope-offset-linear",
    label: "Linear Slope Intercept",
  },
  {
    key: "two-point-linear",
    label: "Linear Two Point",
  },
];

const SelectScale = (props: Omit<Select.ButtonProps<string>, "data">): ReactElement => (
  <Select.DropdownButton<string>
    entryRenderKey="label"
    columns={[
      {
        key: "label",
        name: "Scale",
      },
    ]}
    data={SCALE_DATA}
    {...props}
  />
);

interface LinearTwoPointProps {
  name: string;
}

const LinearTwoPoint = ({ name }: LinearTwoPointProps): ReactElement => {
  const value = Form.useField({
    path: name,
    optional: true,
  });
  console.log(value);
  const isValid = value.value != null && value.value.type === "linear";
  useEffect(() => {
    if (isValid) return;
    value?.onChange({
      type: "linear",
      one: { x: 0, y: 0 },
      two: { x: 1, y: 1 },
    });
  }, [isValid]);
  if (!isValid) return <></>;
  return (
    <Align.Space direction="y" grow>
      <Align.Space direction="x">
        <Form.Field<number> label="Raw Min" path={`${name}.one.x`} grow>
          {(p) => <Input.Numeric {...p} />}
        </Form.Field>
        <Form.Field<number> label="Raw Max" path={`${name}.two.x`} grow>
          {(p) => <Input.Numeric {...p} />}
        </Form.Field>
      </Align.Space>
      <Align.Space direction="x">
        <Form.Field<number> label="Scaled Min" path={`${name}.one.y`} grow>
          {(p) => <Input.Numeric {...p} />}
        </Form.Field>
        <Form.Field<number> label="Scaled Max" path={`${name}.two.y`} grow>
          {(p) => <Input.Numeric {...p} />}
        </Form.Field>
      </Align.Space>
    </Align.Space>
  );
};
