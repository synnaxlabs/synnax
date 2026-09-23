import { type channel, type TimeStamp } from "@synnaxlabs/client";
import { z } from "zod";
import { type Source } from "../../../telem/aether/telem";
export interface LogEntry {
    channelKey: channel.Key;
    timestamp: TimeStamp;
    value: string;
    continuation?: boolean;
}
export interface LogSource extends Source<LogEntry[]> {
    readonly evictedCount: number;
    setChannels?: (channels: channel.Key[]) => void;
}
export declare const logSourceSpecZ: z.ZodObject<{
    type: z.ZodString;
    props: z.ZodAny;
    transfer: z.ZodOptional<z.ZodArray<z.ZodCustom<ArrayBuffer, ArrayBuffer>>>;
    variant: z.ZodLiteral<"source">;
    valueType: z.ZodLiteral<"log">;
}, z.core.$strip>;
export type LogSourceSpec = z.infer<typeof logSourceSpecZ>;
//# sourceMappingURL=types.d.ts.map