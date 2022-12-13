import { z } from 'zod';

export const UserPayloadSchema = z.object({
  key: z.string(),
  username: z.string(),
});

export type UserPayload = z.infer<typeof UserPayloadSchema>;
