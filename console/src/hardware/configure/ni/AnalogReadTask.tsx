import { useState, type ReactElement, useCallback } from "react";

import { Channel, Form, Select, Synnax } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Input } from "@synnaxlabs/pluto/input";
import { Text } from "@synnaxlabs/pluto/text";
import { useQuery } from "@tanstack/react-query";

import { CSS } from "@/css";
import { ChannelList } from "@/hardware/configure/ni/ChannelList";
import {
  analogReadTaskConfigZ,
  DEFAULT_SCALES,
  type LinearScale,
  type LinearScaleType,
} from "@/hardware/configure/ni/types";

export interface AnalogReadTaskProps {
  taskKey: string;
}

export const AnalogReadTask = ({
  taskKey,
}: AnalogReadTaskProps): ReactElement | null => {
  const client = Synnax.use();
  const { data } = useQuery({
    queryKey: [taskKey, client?.key],
    queryFn: async () => await client?.hardware.tasks.retrieve(taskKey),
  });

  if (data == null) return null;

  const methods = Form.use({
    values: analogReadTaskConfigZ.parse(ZERO_ANA),
    schema: analogReadTaskConfigZ,
  });

  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);
  const [selecedChanneIndex, setSelectedChannelIndex] = useState<number>(0);

  return (
    <Form.Form {...methods}>
      <Align.Space>
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
      </Align.Space>
      <ChannelForm path={`channels.${selecedChanneIndex}`} />
    </Form.Form>
  );
};

interface ChannelFormProps {
  path: string;
}

const ChannelForm = ({ path }: ChannelFormProps): ReactElement => {
  return (
    <Align.Space className={CSS.B("details")}>
      <Text.Text level="h3">Channel Properties</Text.Text>
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
