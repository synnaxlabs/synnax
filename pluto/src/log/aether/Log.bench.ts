// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { theme } from "@synnaxlabs/lyra/theme";
import { box, TimeStamp } from "@synnaxlabs/x";
import { type Bench, test } from "vitest";

import { Log, logStateZ } from "@/log/aether/Log";
import {
  MockLogSource,
  mockLogSourceSpec,
  registerMockLogSource,
} from "@/log/aether/telem/mock";
import { type LogEntry } from "@/log/aether/telem/types";
import { renderAether } from "@/testutil/renderAether";
import { canvasTest } from "@/vis/render/testutil";

// Times the CPU side of one redraw of a full log, through a recording canvas.

const THEME: theme.Theme = theme.themeZ.parse(theme.SYNNAX_DARK);
const LINE_HEIGHT = THEME.typography.p.size * THEME.sizes.base;
const WIDTH = 1280;
const ROW_COUNTS = [25, 50, 100];
const RUN_URL = "https://github.com/synnaxlabs/synnax/actions/runs";

// The recording canvas gives no ink measures, and a link box needs them.
const TEXT_METRICS = {
  width: 8,
  fontBoundingBoxAscent: 10,
  actualBoundingBoxAscent: 8,
  actualBoundingBoxDescent: 2,
};

type Value = (i: number) => string;

const PLAIN: Value = (i) =>
  `job ${i} finished with status ok after 1234 ms on runner mac-mini-07`;
const LINK: Value = (i) => `job ${i} finished, see ${RUN_URL}/${i} for the full log`;
const THREE_LINKS: Value = (i) =>
  `job ${i} ${RUN_URL}/${i} on 10.0.0.${i % 250}:9090 wrote /var/log/synnax/run-${i}.log`;
const MARKDOWN_LINK: Value = (i) =>
  `job ${i} finished, see [run ${i}](${RUN_URL}/${i}) for the full log`;
const WRAPPED_LINK: Value = (i) => `job ${i} ${RUN_URL}/${i}/${"a".repeat(400)}`;

let mounted = 0;

// Mounts a log that holds more entries than it shows, and returns its redraw.
const mount = (rows: number, value: Value): (() => void) => {
  const id = `log-bench-${++mounted}`;
  const source = new MockLogSource();
  registerMockLogSource(id, source);
  const recorder = canvasTest.record();
  recorder.lower2d.measureText = () => TEXT_METRICS;
  const state = logStateZ.parse({
    // One more line of height holds the padding around the rows.
    region: box.construct(0, 0, WIDTH, (rows + 1) * LINE_HEIGHT),
    wheelPos: 0,
    scrolling: false,
    empty: true,
    visible: true,
    telem: mockLogSourceSpec(id),
  });
  const handle = renderAether(Log, {
    state,
    theming: { theme: THEME, fontURLs: [] },
    render: recorder,
  });
  const entries = Array.from({ length: rows * 3 }, (_, i): LogEntry => ({
    channelKey: 1,
    timestamp: TimeStamp.milliseconds(i * 1000),
    value: value(i),
  }));
  source.push(...entries);
  // The log reads its source only on an update.
  handle.setState(() => state);
  const log = handle.component;
  // The first redraw sends the rows and the links to the main thread.
  log.render();
  const shown = log.state.rowEntries.length;
  if (shown < rows) throw new Error(`log shows ${shown} rows of ${rows}`);
  return () => {
    recorder.clear();
    log.render();
  };
};

const compareRowCounts = async (bench: Bench, value: Value): Promise<void> => {
  const redraws = ROW_COUNTS.map((rows) => bench(`${rows} rows`, mount(rows, value)));
  await bench.compare(...redraws);
};

test("no links", async ({ bench }) => {
  await compareRowCounts(bench, PLAIN);
});

test("one link in each row", async ({ bench }) => {
  await compareRowCounts(bench, LINK);
});

test("three links in each row", async ({ bench }) => {
  await compareRowCounts(bench, THREE_LINKS);
});

test("one Markdown link in each row", async ({ bench }) => {
  await compareRowCounts(bench, MARKDOWN_LINK);
});

test("one link that wraps in each entry", async ({ bench }) => {
  await compareRowCounts(bench, WRAPPED_LINK);
});
