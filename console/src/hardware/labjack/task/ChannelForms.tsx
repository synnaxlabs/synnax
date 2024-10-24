import { Align, Form, Select } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { ModelDirectory, ModelKey } from "../device/types";

interface ChannelFormProps {
  selectedChannelIndex: number;
}

const locationSelector = (model: ModelKey, type: "AIN" | "DIO") => {
  const pinoutInfo = ModelDirectory[model];
  const locations: string[] = [];
  const regType = pinoutInfo[type];
  for (let i = 0; i < regType; i++) locations.push(`${type}${i}`);
  for (let i = 0; i < pinoutInfo.FIO; i++) locations.push(`FIO${i}`);

  return (
    <Select.Single columns={deviceColumns} entryRenderKey={"name"} data={locations} />
  );
};

const ChannelForm = ({ selectedChannelIndex }: ChannelFormProps): ReactElement => {
  const prefix = `config.channels.${selectedChannelIndex}`; //datatype, location, range, channel type
  return (
    <Align.Space direction="x" grow>
      <Form.TextField path={`${prefix}.location`} label="Location" grow />
      <Form.TextField path={`${prefix}.dataType`} label="Data Type" grow />
      <Form.NumericField path={`${prefix}.range`} label="Range" grow />
      <Form.TextField
        path={`${prefix}.channelTypes`}
        label="Channel Type"
        optional
        grow
      />
    </Align.Space>
  );
};
