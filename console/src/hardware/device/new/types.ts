import { z } from "zod";

const VENDORS = ["ni", "other"] as const;

export const vendorZ = z.enum(VENDORS);

export type Vendor = z.infer<typeof vendorZ>;

const channelZ = z.object({
  key: z.string(),
  port: z.number(),
  line: z.number(),
  name: z.string().min(1),
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

const moduleZ = z
  .object({
    key: z.string(),
    slot: z.number().min(1),
    model: z.string(),
    category: z.string(),
    busType: z.string(),
    analogInCount: z.number().min(0),
    analogOutCount: z.number().min(0),
    digitalInCount: z.number().min(0),
    digitalOutCount: z.number().min(0),
    groups: z.array(groupZ),
  })
  .superRefine((mod, ctx) => {
    // Check that all ports and lines are unique
    const ports = new Map<number, number>();
    const lines = new Map<number, number>();

    mod.groups.forEach((group, i) => {
      group.channels.forEach((channel, j) => {
        ports.set(channel.port, (ports.get(channel.port) ?? 0) + 1);
        lines.set(channel.line, (lines.get(channel.line) ?? 0) + 1);
      });
    });

    mod.groups.forEach((group, i) => {
      group.channels.forEach((channel, j) => {
        if (ports.get(channel.port) !== 1) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: ["groups", i, "channels", j, "port"],
            message: `Port ${channel.port} is not unique`,
          });
        }
        // if (lines.get(channel.line) !== 1) {
        //   ctx.addIssue({
        //     code: z.ZodIssueCode.custom,
        //     path: ["groups", i, "channels", j, "line"],
        //     message: `Line ${channel.line} is not unique`,
        //   });
        // }
      });
    });
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
