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
  // Runs checks on ports:
  //
  // 1. Checks that ports + line combos are unique
  // 2. Checks that port/line combos don't exceed the max number of channels
  //
  //
  .superRefine((mod, ctx) => {
    const portLineCombos = new Map<string, number>();

    mod.groups.forEach((group) => {
      group.channels.forEach((channel) => {
        const key = `${channel.port}/${channel.line}`;
        portLineCombos.set(key, (portLineCombos.get(key) ?? 0) + 1);
      });
    });

    mod.groups.forEach((group, i) => {
      group.channels.forEach((channel, j) => {
        const key = `${channel.port}/${channel.line}`;
        if (portLineCombos.get(key) !== 1) {
          const [port, line] = key.split("/").map(Number);

          if (line === 0) {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              path: ["groups", i, "channels", j, "port"],
              message: `Port ${channel.port} has already been used`,
            });
          } else {
            ctx.addIssue({
              code: z.ZodIssueCode.custom,
              path: ["groups", i, "channels", j, "line"],
              message: `Line ${channel.line} has already been used on port ${port}`,
            });
          }
        }
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
