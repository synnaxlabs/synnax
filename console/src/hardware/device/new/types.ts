import { z } from "zod";

const VENDORS = ["ni", "other"] as const;

export const vendorZ = z.enum(VENDORS);

export type Vendor = z.infer<typeof vendorZ>;

const channelZ = z.object({
  key: z.string(),
  port: z.number(),
  line: z.number(),
  name: z.string(),
  dataType: z.string(),
  isIndex: z.boolean(),
});

export type Channel = z.infer<typeof channelZ>;

const groupZ = z.object({
  key: z.string(),
  name: z.string(),
  channelPrefix: z.string(),
  channelSuffix: z.string(),
  channels: z.array(channelZ),
});

export type Group = z.infer<typeof groupZ>;

const moduleZ = z.object({
  key: z.string(),
  slot: z.number().min(1),
  model: z.string(),
  category: z.string(),
  busType: z.string(),
  analogInCount: z.number().min(1),
  analogOutCount: z.number().min(1),
  digitalInCount: z.number().min(1),
  digitalOutCount: z.number().min(1),
  groups: z.array(groupZ),
});

export type Module = z.infer<typeof moduleZ>;

const properties = z.object({
  vendor: vendorZ,
  name: z.string().min(3).max(32),
  model: z.string(),
  key: z.string(),
  identifier: z
    .string()
    .min(3)
    .max(6)
    .refine((s) => !s.includes(" ") && /^[a-zA-Z0-9]+$/.test(s), {
      message: "Only alphanumeric characters allowed",
    }),
  isChassis: z.boolean(),
  slotCount: z.number(),
});

export type Properties = z.infer<typeof properties>;

export const configurationZ = properties.extend({
  modules: moduleZ.array(),
});

export type Configuration = z.infer<typeof configurationZ>;

const sampleConfiguration: Configuration = {
  name: "GSE DAQ",
  vendor: "ni",
  model: "NI CDAQ 9178",
  key: "01A100CE",
  identifier: "gse",
};
