import { z } from 'zod';

import { DataType, Density, Rate } from '../telem';

export const ChannelPayloadSchema = z.object({
  rate: z.number().transform((n) => new Rate(n)),
  dataType: z.string().transform((s) => new DataType(s)),
  key: z.string().default('').optional(),
  name: z.string().default('').optional(),
  nodeId: z.number().default(0).optional(),
  density: z
    .number()
    .default(0)
    .transform((n) => new Density(n))
    .optional(),
  index: z.string().default('').optional(),
  isIndex: z.boolean().default(false).optional(),
});

export type ChannelPayload = z.infer<typeof ChannelPayloadSchema>;
