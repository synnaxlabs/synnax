import { useState, type ReactElement, useMemo, useEffect } from "react";

import { Align, Button, Channel, Header, Select, Status } from "@synnaxlabs/pluto";
import { Input } from "@synnaxlabs/pluto/input";
import { List } from "@synnaxlabs/pluto/list";
import { Text } from "@synnaxlabs/pluto/text";
import { useFieldArray, useFormContext, useWatch } from "react-hook-form";

import { CSS } from "@/css";
import {
  CHANNEL_TYPE_DISPLAY,
  type LinearScale,
  type NIChannel,
  type NITask,
} from "@/hardware/configure/ni/types";
import { type Configuration } from "@/hardware/device/new/types";

import "@/hardware/device/new/SoftwareTasksForm.css";

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
    <Align.Space direction="y" className={CSS.B("software-tasks")} grow>
      <Text.Text level="h2">
        Let's setup the first acquistiion & control tasks for your device.
      </Text.Text>
      <Align.Space direction="x" className={CSS.B("config")} empty>
        <TaskList selectedTask={selectedTask?.key} onSelectTask={handleSelectTask} />
        <ChannelList
          selectedChannels={selectedChannels}
          selectedTaskIndex={mostRecentSelected?.index ?? 0}
          onSelectChannels={handleSelectChannels}
        />
        {mostRecentSelected != null && (
          <Details selected={mostRecentSelected} taskIndex={selectedTask?.index} />
        )}
      </Align.Space>
    </Align.Space>
  );
};

interface TaskListProps {
  selectedTask: string | undefined;
  onSelectTask: (key: string, index: number) => void;
}

const TaskList = ({ selectedTask, onSelectTask }: TaskListProps): ReactElement => {
  const { fields } = useFieldArray<Configuration>({
    name: "softwarePlan.tasks",
  });
  return (
    <Align.Space className={CSS.B("tasks")} grow empty>
      <Header.Header level="h3">
        <Header.Title level="h3">Tasks</Header.Title>
      </Header.Header>
      <List.List<string, NITask> data={fields as unknown as NITask[]}>
        <List.Selector<string, NITask>
          allowMultiple={false}
          // eslint-disable-next-line @typescript-eslint/non-nullable-type-assertion-style
          value={selectedTask as string}
          onChange={([key], { clickedIndex }) =>
            clickedIndex != null && onSelectTask(key, clickedIndex)
          }
        />
        <List.Core<string, NITask>>{(props) => <TaskListItem {...props} />}</List.Core>
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
  selectedTaskIndex: selectedGroupIndex,
  onSelectChannels,
}: ChannelListProps): ReactElement => {
  const channels = useWatch<Configuration>({
    name: `softwarePlan.tasks.${selectedGroupIndex}.config.channels`,
  });
  return (
    <Align.Space className={CSS.B("channels")} grow empty>
      <Header.Header level="h3">
        <Header.Title weight={500}>Channels</Header.Title>
      </Header.Header>
      <List.List<string, NIChannel> data={channels as NIChannel[]}>
        <List.Selector<string, NIChannel>
          value={selectedChannels}
          allowNone={false}
          onChange={(keys, { clickedIndex }) =>
            clickedIndex != null && onSelectChannels(keys, clickedIndex)
          }
          replaceOnSingle
        />
        <List.Core<string, NIChannel> grow>
          {(props) => <ChannelListItem {...props} groupIndex={selectedGroupIndex} />}
        </List.Core>
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
  return (
    <List.ItemFrame {...props} justify="spaceBetween" align="center">
      <Align.Space direction="y" size="small">
        <Align.Space direction="x">
          <Text.Text level="small" weight={500} shade={6} style={{ width: "3rem" }}>
            {entry.port} {hasLine && `/${entry.line}`}
          </Text.Text>
          <Text.Text level="small" weight={500} shade={9}>
            {entry.name}
          </Text.Text>
        </Align.Space>
        <Text.Text level="small" shade={6}>
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
  console.log(selected, taskIndex);
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
  const [scaleType, setScaleType] = useState("none");
  const prefix = `softwarePlan.tasks.${taskIndex}.config.channels.${index}`;
  const channel = useWatch<NIChannel>({
    name: prefix,
  });
  const { getValues } = useFormContext<Configuration>();

  // We should only need to do this on first render, since the groups are static.
  const channelOptions = useMemo(
    () =>
      getValues(`physicalPlan.groups`)
        .map((g) => g.channels)
        .flat(),
    [],
  );

  const hasLine = "line" in channel;
  return (
    <Align.Space className={CSS.B("details")}>
      <Text.Text level="h3">Channel Properties</Text.Text>
      <Input.HFItem<number> label="Port" name={`${prefix}.port`}>
        {(p) => <Input.Numeric {...p} />}
      </Input.HFItem>
      {hasLine && (
        <Input.HFItem<number> label="Line" name={`${prefix}.line`}>
          {(p) => <Input.Numeric {...p} />}
        </Input.HFItem>
      )}
      <Input.HFItem<string> label="Channel" name={`${prefix}.channel`}>
        {(p) => <Channel.SelectSingle data={channelOptions} {...p} />}
      </Input.HFItem>
      <SelectScale value={scaleType} onChange={setScaleType} />
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
    renderKey="label"
    {...props}
  />
);

interface LinearTwoPointProps {
  name: string;
}

const LinearTwoPoint = ({ name }: LinearTwoPointProps): ReactElement => {
  console.log(name);
  const value = useWatch({
    name,
  }) as LinearScale;
  const isValid = value != null && value.type === "linear";
  const { setValue } = useFormContext<Configuration>();
  useEffect(() => {
    console.log(isValid);
    if (isValid) return;
    setValue(name, {
      type: "linear",
      one: { x: 0, y: 0 },
      two: { x: 1, y: 1 },
    });
  }, [isValid]);
  if (!isValid) return <></>;
  return (
    <Align.Space direction="y" grow>
      <Align.Space direction="x">
        <Input.HFItem label="Raw Min" name={`${name}.one.x`} grow>
          {(p) => <Input.Numeric {...p} />}
        </Input.HFItem>
        <Input.HFItem label="Raw Max" name={`${name}.two.x`} grow>
          {(p) => <Input.Numeric {...p} />}
        </Input.HFItem>
      </Align.Space>
      <Align.Space direction="x">
        <Input.HFItem label="Scaled Min" name={`${name}.one.y`} grow>
          {(p) => <Input.Numeric {...p} />}
        </Input.HFItem>
        <Input.HFItem label="Scaled Max" name={`${name}.two.y`} grow>
          {(p) => <Input.Numeric {...p} />}
        </Input.HFItem>
      </Align.Space>
    </Align.Space>
  );
};
