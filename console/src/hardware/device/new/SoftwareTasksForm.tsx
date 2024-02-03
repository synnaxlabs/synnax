import { useState, type ReactElement } from "react";

import { Align, Button, Header, Select, Status } from "@synnaxlabs/pluto";
import { Input } from "@synnaxlabs/pluto/input";
import { List } from "@synnaxlabs/pluto/list";
import { Text } from "@synnaxlabs/pluto/text";
import { useFieldArray, useWatch } from "react-hook-form";

import { CSS } from "@/css";
import { type NIChannel, type NITask } from "@/hardware/configure/ni/types";
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
        <Align.Space className={CSS.B("details")} grow>
          {mostRecentSelected != null && (
            <Details selected={mostRecentSelected} taskIndex={selectedTask?.index} />
          )}
        </Align.Space>
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
      <Align.Space direction="y">
        <Align.Space direction="x">
          <Text.Text level="small" weight={500} shade={6} style={{ width: "3rem" }}>
            {entry.port} {hasLine && `/${entry.line}`}
          </Text.Text>
          <Text.Text level="small" weight={500} shade={9}>
            {entry.name}
          </Text.Text>
        </Align.Space>
        <Text.Text level="small" shade={6}>
          {entry.type}
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
  return <ChannelForm taskIndex={taskIndex} index={selected.index} />;
};

interface ChannelFormProps {
  taskIndex: number;
  index: number;
}

const ChannelForm = ({ taskIndex, index }: ChannelFormProps): ReactElement => {
  const [scaleType, setScaleType] = useState("none");
  const prefix = `softwarePlan.tasks.${taskIndex}.config.channels.${index}`;
  console.log("DIG");
  return (
    <Align.Space className={CSS.B("properties")}>
      <Text.Text level="h3">Channel Properties</Text.Text>
      <Input.HFItem<number> label="Port" name={`${prefix}.port`}>
        {(p) => <Input.Numeric {...p} />}
      </Input.HFItem>
      {/* <Input.HFItem<number> label="Line" name={`${prefix}.line`}>
        {(p) => <Input.Numeric {...p} />}
      </Input.HFItem>
      <Input.Item label="Scale">
        <SelectScale value={scaleType} onChange={setScaleType} />
      </Input.Item> */}
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
  <Select.Button<string> entryRenderKey="label" data={SCALE_DATA} {...props} />
);
