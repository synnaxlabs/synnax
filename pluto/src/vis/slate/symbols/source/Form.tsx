import { type channel } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Channel } from "@/channel";
import { Form as Core } from "@/form";

export const Form = (): ReactElement => (
  <Core.Field<channel.Key[]> path="channels">
    {(p) => <Channel.SelectMultiple {...p} variant="connected" />}
  </Core.Field>
);
