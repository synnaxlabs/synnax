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

// Helper to create mock Log instance
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

    // Set up minimal required state
    (log as any)._state = {
        region: box.construct({ x: 0, y: 0 }, { width: 800, height: 600 }),
        font: "p",
        wheelPos: 0,
        scrolling: false,
        empty: false,
        visible: true,
        color: { r: 0, g: 0, b: 0, a: 0 },
        overshoot: { x: 0, y: 0 },
    };

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

    // Initialize charWidth (simulate what happens in render)
    (log as any).charWidth = 7; // Mock monospace width

    return log;
}

// Helper to create mock log data
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
        data: new Float64Array(data.map((_, i) => i)),
        dataType: DataType.STRING,
        timeRange: TimeRange.ZERO,
    });

    (series as any)._data = data;
    return new MultiSeries([series]);
}

describe("softWrapLog benchmarks - your implementation", () => {
    let log: Log;

    bench("wrap short line (no wrapping needed)", () => {
        const text = "Log entry 123";
        log["softWrapLog"](text, 800);
    }, {
        time: 1000,
        setup: () => {
            log = createMockLog();
        },
    });

    bench("wrap medium line (1-2 wraps)", () => {
        const text = "[2025-01-01T12:00:00.000Z] INFO: Processing request 456 with status code 200 and additional context";
        log["softWrapLog"](text, 400);
    }, {
        time: 1000,
        setup: () => {
            log = createMockLog();
        },
    });

    bench("wrap long line (multiple wraps)", () => {
        const text = "[2025-01-01T12:00:00.000Z] ERROR: Failed to connect to database server at 192.168.1.100:5432 after 3 retry attempts. Connection timeout: 30s. Last error: ECONNREFUSED. Request ID: abc123-def456-ghi789. Stack trace: Error at DatabaseConnection.connect (db.ts:45) at retry (retry.ts:12) at async main (index.ts:100)";
        log["softWrapLog"](text, 400);
    }, {
        time: 1000,
        setup: () => {
            log = createMockLog();
        },
    });

    bench("wrap very long word (character-level wrapping)", () => {
        const text = "a".repeat(500);
        log["softWrapLog"](text, 400);
    }, {
        time: 1000,
        setup: () => {
            log = createMockLog();
        },
    });

    bench("wrap line with many short words", () => {
        const text = "a b c d e f g h i j k l m n o p q r s t u v w x y z " +
            "a b c d e f g h i j k l m n o p q r s t u v w x y z";
        log["softWrapLog"](text, 400);
    }, {
        time: 1000,
        setup: () => {
            log = createMockLog();
        },
    });

    bench("wrap mixed content (words + long token)", () => {
        const text = "Normal words here then AVERYLONGTOKENWITHOUTSPACES and more normal words after";
        log["softWrapLog"](text, 400);
    }, {
        time: 1000,
        setup: () => {
            log = createMockLog();
        },
    });
});

describe("renderElements benchmarks - with your wrapping", () => {
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

    let log: Log;
    let series10: MultiSeries;
    let series50: MultiSeries;
    let series100: MultiSeries;

    bench("render 10 short lines (with wrapping)", () => {
        const dataArray = Array.from({ length: 10 }, (_, i) => (series10 as any).series[0]._data[i]);
        log["renderElements"](mockDraw2D, dataArray);
    }, {
        time: 1000,
        setup: () => {
            textCalls.length = 0;
            log = createMockLog();
            series10 = createLogSeries(10, "short");
            log.values = series10;
        },
    });

    bench("render 50 medium lines (with wrapping)", () => {
        const dataArray = Array.from({ length: 50 }, (_, i) => (series50 as any).series[0]._data[i]);
        log["renderElements"](mockDraw2D, dataArray);
    }, {
        time: 1000,
        setup: () => {
            textCalls.length = 0;
            log = createMockLog();
            series50 = createLogSeries(50, "medium");
            log.values = series50;
        },
    });

    bench("render 100 long lines (with wrapping)", () => {
        const dataArray = Array.from({ length: 100 }, (_, i) => (series100 as any).series[0]._data[i]);
        log["renderElements"](mockDraw2D, dataArray);
    }, {
        time: 1000,
        setup: () => {
            textCalls.length = 0;
            log = createMockLog();
            series100 = createLogSeries(100, "long");
            log.values = series100;
        },
    });

    bench("render 200 mixed lines (with wrapping)", () => {
        const series: string[] = [];
        for (let i = 0; i < 200; i++) {
            if (i % 3 === 0) series.push(`Short ${i}`);
            else if (i % 3 === 1) series.push(`[2025-01-01] Medium log entry ${i} with some context`);
            else series.push(`[2025-01-01] ERROR: Very long error message with stack trace and details for entry ${i}. Connection failed, timeout exceeded, retry logic exhausted.`);
        }

        const mockSeries = new MultiSeries([]);
        (mockSeries as any).length = series.length;
        (mockSeries as any).dataType = DataType.STRING;

        log.values = mockSeries;
        log["renderElements"](mockDraw2D, series as any);
    }, {
        time: 1000,
        setup: () => {
            textCalls.length = 0;
            log = createMockLog();
        },
    });
});