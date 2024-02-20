import { type ReactElement } from "react";

import { DataType } from "@synnaxlabs/x";
import { z } from "zod";

const groupZ = z.object({
  name: z.string(),
  channelCount: z.number(),
  dataType: z.number(),
});

export const Channels = (): ReactElement => {};
