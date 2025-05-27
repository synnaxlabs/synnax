import { type channel } from "@synnaxlabs/client";
import { type ReactElement } from "react";

import { Channel } from "@/channel";
import { Form as Core } from "@/form";

export const Form = (): ReactElement => (
  <Core.Field<channel.Key> path="channel">
    {(p) => <Channel.SelectSingle {...p} />}
  </Core.Field>
);
