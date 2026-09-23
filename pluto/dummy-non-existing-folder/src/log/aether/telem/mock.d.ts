import { type channel } from "@synnaxlabs/client";
import { type destructor, observe } from "@synnaxlabs/x";
import { type LogEntry, type LogSource, type LogSourceSpec } from "./types";
export declare class MockLogSource implements LogSource {
    static readonly TYPE = "test-source";
    private _entries;
    private _evictedCount;
    private activeChannels;
    private readonly observable;
    get evictedCount(): number;
    set evictedCount(count: number);
    value(): LogEntry[];
    setChannels(channels: channel.Key[]): void;
    onChange(handler: observe.Handler<void>): destructor.Destructor;
    push(...entries: LogEntry[]): void;
    setEntries(entries: LogEntry[]): void;
    notify(): void;
    cleanup(): void;
}
export declare const registerMockLogSource: (testId: string, source: MockLogSource) => destructor.Destructor;
export declare const mockLogSourceSpec: (testId: string) => LogSourceSpec;
//# sourceMappingURL=mock.d.ts.map