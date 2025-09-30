import { type channel } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Channel } from "@/channel";
import { Flex } from "@/flex";
import { Form as Core } from "@/form";

export const Form = (): ReactElement => (
  <Flex.Box x>
    <Core.Field<channel.Key> path="channel">
      {({ value, onChange }) => (
        <Channel.SelectSingle value={value} onChange={onChange} />
      )}
    </Core.Field>
    <Core.NumericField path="value" grow />
  </Flex.Box>
);
