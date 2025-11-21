// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, DataType, MultiSeries, Series, TimeRange } from "@synnaxlabs/x";
import { bench, describe } from "vitest";

import { Log } from "./Log";

function createMockLog(): Log {
    const mockProps = {
        key: "test-log",
        type: Log.TYPE,
        sender: (() => { }) as any,
        instrumentation: {
            child: () => ({
                child: () => ({}),
            }),
        } as any,
        parentCtxValues: new Map(),
    };

    const log = new Log(mockProps);

    const mockState = {
        region: box.construct({ x: 0, y: 0 }, { width: 800, height: 600 }),
        font: "p",
        wheelPos: 0,
        scrolling: false,
        empty: false,
        visible: true,
        color: { r: 0, g: 0, b: 0, a: 0 },
        overshoot: { x: 0, y: 0 },
    };

    (log as any)._state = mockState;

    Object.defineProperty(log, 'state', {
        get: () => mockState,
        configurable: true
    });

    (log as any)._internalState = {
        theme: {
            typography: { p: { size: 12 } },
            sizes: { base: 1 },
            colors: {
                gray: {
                    l11: { r: 255, g: 255, b: 255, a: 1 },
                    l6: { r: 100, g: 100, b: 100, a: 1 }
                }
            },
        },
    };

    (log as any).charWidth = 7;

    return log;
}

function createLogSeries(count: number, lineLength: "short" | "medium" | "long"): MultiSeries {
    const data: string[] = [];

    for (let i = 0; i < count; i++) {
        let line: string;
        switch (lineLength) {
            case "short":
                line = `Log entry ${i}`;
                break;
            case "medium":
                line = `[${new Date().toISOString()}] INFO: Processing request ${i} with status code 200`;
                break;
            case "long":
                line = `[${new Date().toISOString()}] ERROR: Failed to connect to database server at 192.168.1.100:5432 after 3 retry attempts. Connection timeout: 30s. Last error: ECONNREFUSED. Request ID: ${i}-${Math.random().toString(36)}`;
                break;
        }
        data.push(line);
    }

    const series = new Series({
        data,
        dataType: DataType.STRING,
        timeRange: TimeRange.ZERO,
    });

    return new MultiSeries([series]);
}

describe("softWrapLog benchmarks - your implementation", () => {
    let log: any;

    bench("wrap short line (no wrapping needed)", () => {
        const text = "Log entry 123";
        log.softWrapLog(text, 800);
    }, {
        time: 1000,
        setup: () => {
            log = createMockLog();
        },
    });

    bench("wrap medium line (1-2 wraps)", () => {
        const text = "[2025-01-01T12:00:00.000Z] INFO: Processing request 456 with status code 200 and additional context";
        log.softWrapLog(text, 400);
    }, {
        time: 1000,
        setup: () => {
            log = createMockLog();
        },
    });

    bench("wrap long line (multiple wraps)", () => {
        const text = "[2025-01-01T12:00:00.000Z] ERROR: Failed to connect to database server at 192.168.1.100:5432 after 3 retry attempts. Connection timeout: 30s. Last error: ECONNREFUSED. Request ID: abc123-def456-ghi789. Stack trace: Error at DatabaseConnection.connect (db.ts:45) at retry (retry.ts:12) at async main (index.ts:100)";
        log.softWrapLog(text, 400);
    }, {
        time: 1000,
        setup: () => {
            log = createMockLog();
        },
    });

    bench("wrap very long word (character-level wrapping)", () => {
        const text = "a".repeat(500);
        log.softWrapLog(text, 400);
    }, {
        time: 1000,
        setup: () => {
            log = createMockLog();
        },
    });

    bench("wrap line with many short words", () => {
        const text = "a b c d e f g h i j k l m n o p q r s t u v w x y z " +
            "a b c d e f g h i j k l m n o p q r s t u v w x y z";
        log.softWrapLog(text, 400);
    }, {
        time: 1000,
        setup: () => {
            log = createMockLog();
        },
    });

    bench("wrap mixed content (words + long token)", () => {
        const text = "Normal words here then AVERYLONGTOKENWITHOUTSPACES and more normal words after";
        log.softWrapLog(text, 400);
    }, {
        time: 1000,
        setup: () => {
            log = createMockLog();
        },
    });
});

describe("renderElements benchmarks - with wrapping and caching", () => {
    const textCalls: any[] = [];
    const mockDraw2D = {
        text: (props: any) => {
            textCalls.push({
                text: props.text,
                position: { ...props.position },
                level: props.level,
            });
            const _ = props.text.split(' ').join('-');
        },
    } as any;

    bench("render 10 short lines (with cache)", () => {
        const { log, series } = this as any;
        const dataArray = Array.from({ length: series.length }, (_, i) => series.at(i));
        log.renderElements(mockDraw2D, dataArray, 0);
    }, {
        time: 1000,
        setup() {
            textCalls.length = 0;
            const log: any = createMockLog();
            const series = createLogSeries(10, "short");
            log.values = series;
            log.rebuildWrapCache();
            (this as any).log = log;
            (this as any).series = series;
        },
    });

    bench("render 50 medium lines (with cache)", () => {
        const { log, series } = this as any;
        const dataArray = Array.from({ length: series.length }, (_, i) => series.at(i));
        log.renderElements(mockDraw2D, dataArray, 0);
    }, {
        time: 1000,
        setup() {
            textCalls.length = 0;
            const log: any = createMockLog();
            const series = createLogSeries(50, "medium");
            log.values = series;
            log.rebuildWrapCache();
            (this as any).log = log;
            (this as any).series = series;
        },
    });

    bench("render 100 long lines (with cache)", () => {
        const { log, series } = this as any;
        const dataArray = Array.from({ length: series.length }, (_, i) => series.at(i));
        log.renderElements(mockDraw2D, dataArray, 0);
    }, {
        time: 1000,
        setup() {
            textCalls.length = 0;
            const log: any = createMockLog();
            const series = createLogSeries(100, "long");
            log.values = series;
            log.rebuildWrapCache();
            (this as any).log = log;
            (this as any).series = series;
        },
    });

    bench("cache rebuild - 100 long lines", () => {
        const { log } = this as any;
        log.rebuildWrapCache();
    }, {
        time: 1000,
        setup() {
            const log = createMockLog();
            const series = createLogSeries(100, "long");
            log.values = series;
            (this as any).log = log;
        },
    });

    bench("cache rebuild - 500 long lines", () => {
        const { log } = this as any;
        log.rebuildWrapCache();
    }, {
        time: 1000,
        setup() {
            const log = createMockLog();
            const series = createLogSeries(500, "long");
            log.values = series;
            (this as any).log = log;
        },
    });
});