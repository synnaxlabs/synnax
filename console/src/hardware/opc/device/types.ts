import { device } from "@synnaxlabs/client";
import { z } from "zod";

export const connectionConfigZ = z.object({
  endpoint: z.string(),
  username: z.string().optional(),
  password: z.string().optional(),
});

export type ConnectionConfig = z.infer<typeof connectionConfigZ>;

export const deviceNodeProperties = z.object({
  dataType: z.string(),
  name: z.string(),
  nodeId: z.string(),
});

export type DeviceNodeProperties = z.infer<typeof deviceNodeProperties>;

export const devicePropertiesZ = z.object({
  connection: connectionConfigZ,
  channels: deviceNodeProperties.array(),
});

export type DeviceProperties = z.infer<typeof devicePropertiesZ>;

export type Device = device.Device<DeviceProperties>;
