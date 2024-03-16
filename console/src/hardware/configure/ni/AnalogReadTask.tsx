import { useState, type ReactElement, useEffect, useMemo } from "react";

import { Channel, Form, Select, Synnax } from "@synnaxlabs/pluto";
import { Align } from "@synnaxlabs/pluto/align";
import { Input } from "@synnaxlabs/pluto/input";
import { Text } from "@synnaxlabs/pluto/text";
import { useQuery } from "@tanstack/react-query";

import { ChannelList } from "@/hardware/configure/ni/ChannelList";
import {
  type LinearScale,
  analogReadTaskConfigZ,
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
    queryKey: [taskKey],
    queryFn: async () => await client?.hardware.tasks.retrieve(taskKey),
  });

  if (data == null) return null;

  const methods = Form.use({
    values: analogReadTaskConfigZ.parse(data.config),
    schema: analogReadTaskConfigZ,
  });

  const [selectedChannels, setSelectedChannels] = useState<string[]>([]);

  return (
    <Form.Form {...methods}>
      <Align.Space>
        <ChannelList
          path="channels"
          selected={selectedChannels}
          onSelect={setSelectedChannels}
        />
      </Align.Space>
    </Form.Form>
  );
};

interface ChannelFormProps {
  path: string;
}

const ChannelForm = ({ path }: ChannelFormProps): ReactElement => {
  const { get } = Form.useContext();

  return (
    <Align.Space className={CSS.B("details")}>
      <Text.Text level="h3">Channel Properties</Text.Text>
      <Form.Field<number> label="Port" path={`${path}.port`}>
        {(p) => <Input.Numeric {...p} />}
      </Form.Field>
      {/* <Form.Field<number>
        label="Line"
        path={`${prefix}.line`}
        visible={(fs) => fs.value !== 0}
      >
        {(p) => <Input.Numeric {...p} />}
      </Form.Field> */}
      <Form.Field<number> label="Channel" path={`${path}.channel`}>
        {(p) => <Channel.SelectSingle {...p} />}
      </Form.Field>
      <Form.Field<LinearScaleType> label="Scale Type" path={`${path}.scale.type`}>
        {(p) => <SelectScale {...p} />}
      </Form.Field>
      <SelectScale value={scaleType} onChange={setScaleType} />
      {scaleType === "two-point-linear" && <LinearTwoPoint path={`${prefix}.scale`} />}
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

const SelectScale = (props: Omit<Select.ButtonProps<string>, "data">): ReactElement => (
  <Select.DropdownButton<string, ScaleEntry>
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
  path: string;
}

const LinearTwoPoint = ({ path }: LinearTwoPointProps): ReactElement => {
  const value = Form.useField({ path });
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
