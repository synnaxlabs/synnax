import { type destructor, type observe } from "@synnaxlabs/x";
import { type LogEntry, type LogSource, type LogSourceSpec } from "./types";
export declare class NoopLogSource implements LogSource {
    static readonly TYPE = "noop-log-source";
    readonly evictedCount = 0;
    value(): LogEntry[];
    cleanup(): void;
    onChange(_handler: observe.Handler<void>): destructor.Destructor;
}
export declare const noopLogSourceSpec: LogSourceSpec;
//# sourceMappingURL=noop.d.ts.map