import { useState, type ReactElement, useCallback } from "react";

import { Channel, Form, Select, Device, Header } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Input } from "@synnaxlabs/pluto/input";
import { Text } from "@synnaxlabs/pluto/text";

import { CSS } from "@/css";
import { ChannelList } from "@/hardware/ni/ChannelList";
import {
  analogReadTaskConfigZ,
  DEFAULT_SCALES,
  ZERO_ANALOG_READ_TASK_CONFIG,
  type LinearScale,
  type LinearScaleType,
} from "@/hardware/ni/types";

import "@/hardware/configure/ni/AnalogReadTask.css";

export interface AnalogReadTaskProps {
  taskKey: string;
}

export const AnalogReadTask = ({
  taskKey,
}: AnalogReadTaskProps): ReactElement | null => {
  // const client = Synnax.use();
  // const { data } = useQuery({
  //   queryKey: [taskKey, client?.key],
  //   queryFn: async () => await client?.hardware.tasks.retrieve(taskKey),
  // });

  // if (data == null) return null;

  const methods = Form.use({
    values: analogReadTaskConfigZ.parse(ZERO_ANALOG_READ_TASK_CONFIG),
    schema: analogReadTaskConfigZ,
  });

  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);
  const [selectedChannelIndex, setSelectedChannelIndex] = useState<number | null>(null);

  return (
    <Align.Space className={CSS.B("ni-analog-read-task")} direction="y" grow empty>
      <Form.Form {...methods}>
        <Align.Space direction="x">
          <Form.Field<string> path="device" label="Device">
            {(p) => <Device.SelectSingle {...p} />}
          </Form.Field>
          <Form.Field<number> label="Sample Rate" path="sampleRate">
            {(p) => <Input.Numeric {...p} />}
          </Form.Field>
          <Form.Field<number> label="Stream Rate" path="streamRate">
            {(p) => <Input.Numeric {...p} />}
          </Form.Field>
        </Align.Space>
        <Align.Space direction="x" empty>
          <ChannelList
            path="channels"
            selected={selectedChannels}
            onSelect={useCallback(
              (v, i) => {
                setSelectedChannels(v);
                setSelectedChannelIndex(i);
              },
              [setSelectedChannels, setSelectedChannelIndex],
            )}
          />
          {selectedChannelIndex != null && (
            <ChannelForm path={`channels.${selectedChannelIndex}`} />
          )}
        </Align.Space>
      </Form.Form>
    </Align.Space>
  );
};

interface ChannelFormProps {
  path: string;
}

const ChannelForm = ({ path }: ChannelFormProps): ReactElement => {
  return (
    <Align.Space className={CSS.B("details")}>
      <Header.Header level="h3">
        <Header.Title weight={500}>Channel Properties</Header.Title>
      </Header.Header>
      <Align.Space direction="y" className="form">
        <Form.Field<number> label="Port" path={`${path}.port`}>
          {(p) => <Input.Numeric {...p} />}
        </Form.Field>
        <Form.Field<number>
          label="Line"
          path={`${path}.line`}
          hideIfNull
          visible={(fs) => fs.value !== 0}
        >
          {(p) => <Input.Numeric {...p} />}
        </Form.Field>
        <Form.Field<number> label="Channel" path={`${path}.channel`}>
          {(p) => <Channel.SelectSingle {...p} />}
        </Form.Field>
        <Form.Field<LinearScaleType>
          label="Scale Type"
          path={`${path}.scale.type`}
          onChange={(v, { set, get }) => {
            const { value: prev } = get<LinearScale>({ path: `${path}.scale` });
            if (prev.type === v) return;
            set({ path: `${path}.scale`, value: DEFAULT_SCALES[v] });
          }}
        >
          {(p) => <SelectScale {...p} />}
        </Form.Field>
        <ScaleForm path={`${path}.scale`} />
      </Align.Space>
    </Align.Space>
  );
};

interface ScaleEntry {
  key: LinearScaleType;
  label: string;
}

const SCALE_DATA: ScaleEntry[] = [
  {
    key: "none",
    label: "None",
  },
  {
    label: "Linear",
    key: "linear",
  },
];

const SelectScale = (
  props: Omit<Select.DropdownButtonProps<LinearScaleType, ScaleEntry>, "data">,
): ReactElement => (
  <Select.DropdownButton<LinearScaleType, ScaleEntry>
    columns={[
      {
        key: "label",
        name: "Scale",
      },
    ]}
    data={SCALE_DATA}
    entryRenderKey="label"
    {...props}
  />
);

interface ScaleFormProps {
  path: string;
}

const ScaleForm = ({ path }: ScaleFormProps): ReactElement | null => {
  const typeField = Form.useField<LinearScaleType>({
    path: `${path}.type`,
  });
  if (typeField.value === "none") return null;
  return (
    <Align.Space direction="y" grow>
      <Align.Space direction="x">
        <Form.Field<number> label="Raw Min" path={`${path}.one.x`} grow>
          {(p) => <Input.Numeric {...p} />}
        </Form.Field>
        <Form.Field<number> label="Raw Max" path={`${path}.two.x`} grow>
          {(p) => <Input.Numeric {...p} />}
        </Form.Field>
      </Align.Space>
      <Align.Space direction="x">
        <Form.Field<number> label="Scaled Min" path={`${path}.one.y`} grow>
          {(p) => <Input.Numeric {...p} />}
        </Form.Field>
        <Form.Field<number> label="Scaled Max" path={`${path}.two.y`} grow>
          {(p) => <Input.Numeric {...p} />}
        </Form.Field>
      </Align.Space>
    </Align.Space>
  );
};
