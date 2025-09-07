import { type channel } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Channel } from "@/channel";
import { Form as Core } from "@/form";

export const Form = (): ReactElement => (
  <Core.Field<channel.Key> path="channel">
    {({ value, onChange }) => (
      <Channel.SelectSingle value={value} onChange={onChange} />
    )}
  </Core.Field>
);
