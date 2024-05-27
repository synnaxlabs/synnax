// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";
import { z } from "zod";

// VENDOR

const VENDORS = ["NI", "other"] as const;

export const vendorZ = z.enum(VENDORS);

export type Vendor = z.infer<typeof vendorZ>;

// DEVICE DIGEST

const propertiesDigestZ = z.object({
  vendor: vendorZ,
  key: z.string(),
  location: z.string(),
  model: z.string(),
});

export type PropertiesDigest = z.infer<typeof propertiesDigestZ>;

// DEVICE ENRICHED

const enrichedPropertiesDigestZ = propertiesDigestZ.extend({
  name: z.string(),
  identifier: z
    .string()
    .min(3)
    .max(6)
    .refine((s) => !s.includes(" ") && /^[a-zA-Z0-9]+$/.test(s), {
      message: "Only alphanumeric characters allowed",
    }),
  analogInput: z.object({
    portCount: z.number(),
  }),
  analogOutput: z.object({
    portCount: z.number(),
  }),
  digitalInputOutput: z.object({
    portCount: z.number(),
    lineCounts: z.number().array(),
  }),
  digitalInput: z.object({
    portCount: z.number(),
    lineCounts: z.number().array(),
  }),
  digitalOutput: z.object({
    portCount: z.number(),
    lineCounts: z.number().array(),
  }),
});

export type EnrichedProperties = z.infer<typeof enrichedPropertiesDigestZ>;

// PHYSICAL PLAN

const channelConfigZ = z.object({
  key: z.string(),
  synnaxChannel: z.number().optional(),
  port: z.number(),
  line: z.number(),
  name: z.string().min(1),
  dataType: z.string(),
  isIndex: z.boolean(),
  role: z.string(),
});

export type ChannelConfig = z.infer<typeof channelConfigZ>;

const groupConfigZ = z.object({
  key: z.string(),
  name: z.string(),
  channelPrefix: z.string(),
  channelSuffix: z.string(),
  channels: z.array(channelConfigZ),
  role: z.string(),
});

export type GroupConfig = z.infer<typeof groupConfigZ>;

const groupsZ = z.array(groupConfigZ).superRefine((groups, ctx) => {
  const portLineRoleCombos = new Map<string, number>();

  groups.forEach((group) => {
    group.channels.forEach((channel) => {
      const key = `${channel.role}/${channel.port}/${channel.line}`;
      portLineRoleCombos.set(key, (portLineRoleCombos.get(key) ?? 0) + 1);
    });
  });

  groups.forEach((group, i) => {
    group.channels.forEach((channel, j) => {
      const key = `${channel.role}/${channel.port}/${channel.line}`;
      if ((portLineRoleCombos.get(key) ?? 0) > 1) {
        const [, port, line] = key.split("/").map(Number);

        if (line >= 0) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: [i, "channels", j, "line"],
            message: `Line ${channel.line} has already been used on port ${port}`,
          });
        } else if (port >= 0) {
          ctx.addIssue({
            code: z.ZodIssueCode.custom,
            path: [i, "channels", j, "port"],
            message: `Port ${channel.port} has already been used`,
          });
        }
      }
    });
  });
});

export const softwarePlanZ = z.object({
  tasks: z.array(task.taskZ),
});

export type SoftwarePlan = z.infer<typeof softwarePlanZ>;

export const configurationZ = z.object({
  properties: enrichedPropertiesDigestZ,
  groups: groupsZ,
});

export type Configuration = z.infer<typeof configurationZ>;
