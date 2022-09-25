import { z } from 'zod';

import { DataType, Density, Rate } from '../telem';

export const ChannelPayloadSchema = z.object({
  rate: z.number().transform((n) => new Rate(n)),
  dataType: z.string().transform((s) => new DataType(s)),
  key: z.string().default('').optional(),
  name: z.string().default('').optional(),
  nodeID: z.number().default(0).optional(),
  density: z
    .number()
    .default(0)
    .transform((n) => new Density(n))
    .optional(),
});

export type ChannelPayload = z.infer<typeof ChannelPayloadSchema>;
