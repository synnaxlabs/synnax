import { device } from "@synnaxlabs/client";
import { z } from "zod";

export const connectionConfigZ = z.object({
  endpoint: z.string(),
  username: z.string().optional(),
  password: z.string().optional(),
});

export type ConnectionConfig = z.infer<typeof connectionConfigZ>;

export const nodeProperties = z.object({
  dataType: z.string(),
  name: z.string(),
  nodeId: z.string(),
});

export type NodeProperties = z.infer<typeof nodeProperties>;

export const propertiesZ = z.object({
  connection: connectionConfigZ,
  channels: nodeProperties.array(),
});

export type Properties = z.infer<typeof propertiesZ>;

export type Device = device.Device<Properties>;
