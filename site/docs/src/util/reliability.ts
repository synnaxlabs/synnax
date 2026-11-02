// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export type State = "passed" | "flaky" | "failed" | "skipped" | "unknown";

/** Worst first. A test takes the worst state of its runs. */
export const RANK: Record<State, number> = {
  failed: 0,
  flaky: 1,
  unknown: 2,
  skipped: 3,
  passed: 4,
};

export interface Suite {
  job: string;
  product: string;
  lang: string;
  kind: "unit" | "system";
  tags: string[];
  os: string;
  conclusion: string | null;
  url: string;
  count: number;
}

/** One run of a test, on one operating system. */
export interface Run {
  suite: Suite;
  name: string;
  state: State;
  message: string | null;
  duration: number | null;
  /** Repository path of the test's source file. */
  path: string;
  /** GitHub URL of the test's source at the run's commit. */
  source: string;
  /** What kind of test it is and what it needs, as in "playwright" or "no-driver". */
  tags: string[];
  /** Skipped in CI, then run by hand for the release, and passed. */
  passedByHand: boolean;
}

/** A test and its runs, one per operating system it ran on. */
export interface Test {
  job: string;
  kind: Suite["kind"];
  name: string;
  /** The worst state of its runs. */
  state: State;
  path: string;
  source: string;
  tags: string[];
  /** Every skipped run was run by hand for the release, and passed. */
  passedByHand: boolean;
  runs: Run[];
}

export interface Subsystem {
  name: string;
  count: number;
}

export interface Product {
  key: string;
  name: string;
  tests: Test[];
  suites: Suite[];
  counts: Record<State, number>;
  subsystems: Record<string, Subsystem>;
}

interface Raw {
  run: {
    id: number;
    url: string;
    sha: string;
    version: string;
    branch: string;
    started_at: string;
    ended_at: string;
    jobs: number;
    /** Skip reasons whose tests were run by hand, and all passed. */
    manual_passes: string[];
    /** Why the manual_passes tests were run by hand. */
    manual_note: string | null;
  };
  products: { key: string; name: string }[];
  suites: Suite[];
  tests: [
    number,
    string,
    State,
    string | null,
    number | null,
    string,
    tags?: string[],
  ][];
}

// `scripts/reliability.py` uploads the data. Pages that read it are prerendered, so
// only the docs build fetches it. The origin URL skips the CDN's cached copy.
const DATA_URL = "https://synnax.nyc3.digitaloceanspaces.com/docs/reliability/run.json";

const res = await fetch(DATA_URL);
if (!res.ok) throw new Error(`fetch ${DATA_URL}: ${res.status} ${res.statusText}`);
const data = (await res.json()) as Raw;

export const run = data.run;

/** GitHub URL prefix for a repository path at the run's commit. */
export const blob = `https://github.com/synnaxlabs/synnax/blob/${data.run.sha}/`;

const OS_NAMES: Record<string, string> = {
  ubuntu: "Ubuntu",
  windows: "Windows",
  macos: "macOS",
};

const allSuites: Suite[] = data.suites.map((s) => {
  const os = OS_NAMES[s.os];
  if (os == null) throw new Error(`unknown OS ${s.os}`);
  return { ...s, os };
});

const emptyCounts = (): Record<State, number> =>
  Object.fromEntries(Object.keys(RANK).map((s) => [s, 0])) as Record<State, number>;

const allRuns: Run[] = data.tests.map(
  ([s, name, state, message, duration, source, tags = []]) => ({
    suite: allSuites[s],
    name,
    state,
    message,
    duration,
    path: source.split("#")[0],
    source: blob + source,
    tags: [...allSuites[s].tags, ...tags],
    passedByHand: state === "skipped" && data.run.manual_passes.includes(message!),
  }),
);

// Unit test path -> subsystem, per product. The first match wins, and every path must
// match one. System tests are always "e2e".
const SUBSYSTEMS: Record<string, [key: string, name: string, match: RegExp][]> = {
  core: [
    ["api", "API", /^core\/pkg\/(api|transport|server)\//],
    ["distribution", "Distribution", /^core\/pkg\/distribution\//],
    ["services", "Services", /^core\//],
    ["aspen", "Aspen", /^aspen\//],
    ["cesium", "Cesium", /^cesium\//],
    ["freighter", "Freighter", /^freighter\//],
    ["x", "X", /^((x|alamos)\/go|gorp)\//],
  ],
  arc: [
    ["parser", "Parser", /^arc\/go\/(parser|text|literal)\//],
    ["analyzer", "Analyzer", /^arc\/(go\/(analyzer|types|symbol)|cpp\/types)\//],
    [
      "compiler",
      "Compiler",
      /^arc\/(go\/(compiler|ir|graph|stratifier|program)|cpp\/(ir|program))\//,
    ],
    ["runtime-go", "Go runtime", /^arc\/go\/runtime\//],
    ["runtime-cpp", "C++ runtime", /^arc\/cpp\/runtime\//],
    ["stl-go", "Go standard library", /^arc\/go\/stl\//],
    ["stl-cpp", "C++ standard library", /^arc\/cpp\/stl\//],
    ["tools", "LSP and formatter", /^arc\/go\/(lsp|formatter)\//],
    ["programs", "Whole programs", /^arc\/go\/[^/]+$/],
  ],
  driver: [
    ["ethercat", "EtherCAT", /^(driver\/ethercat|vendor\/soem)\//],
    ["http", "HTTP", /^driver\/http\//],
    ["opcua", "OPC UA", /^driver\/opcua\//],
    ["ni", "NI", /^driver\/ni\//],
    ["modbus", "Modbus", /^driver\/modbus\//],
    ["labjack", "LabJack", /^driver\/labjack\//],
    ["arc", "Arc runtime", /^driver\/arc\//],
    ["tasks", "Task pipeline", /^driver\//],
    ["foundation", "C++ foundation", /^(x|freighter)\/cpp\//],
  ],
  console: [
    ["console", "Console app", /^console\//],
    ["pluto", "Pluto", /^pluto\//],
    ["lyra", "Lyra", /^lyra\//],
    ["drift", "Drift", /^drift\//],
    ["x", "X", /^(x|alamos)\/ts\//],
  ],
  clients: [
    ["ts", "TypeScript", /^(client|freighter)\/ts\//],
    ["py", "Python", /^(client|freighter|x|alamos)\/py\//],
    ["cpp", "C++", /^client\/(cpp|clib)\//],
  ],
  toolchain: [
    ["codegen", "Code generators", /^oracle\/(plugin|codegen)\//],
    [
      "analysis",
      "Schema analysis",
      /^oracle\/(analyzer|resolution|domain|check|parser|paths|versions)\//,
    ],
    ["tools", "Language tools", /^oracle\//],
    ["scripts", "Repo scripts", /^(scripts|\.github\/scripts)\//],
    ["docs", "Docs site", /^site\/docs\//],
    ["vite", "Vite plugin", /^configs\/vite\//],
    ["eslint", "Lint rules", /^configs\/eslint\//],
  ],
};

/** Identifies a test across the operating systems it ran on. */
export const id = (r: Run): string => `${r.suite.job} ${r.source} ${r.name}`;

/** Groups runs into tests, in the order of each test's first run. */
const group = (runs: Run[]): Test[] => {
  const tests = new Map<string, Test>();
  for (const r of runs) {
    const t = tests.get(id(r));
    if (t == null)
      tests.set(id(r), {
        job: r.suite.job,
        kind: r.suite.kind,
        name: r.name,
        state: r.state,
        path: r.path,
        source: r.source,
        tags: r.tags,
        passedByHand: false,
        runs: [r],
      });
    else {
      t.runs.push(r);
      if (RANK[r.state] < RANK[t.state]) t.state = r.state;
    }
  }
  for (const t of tests.values())
    t.passedByHand =
      t.state === "skipped" &&
      t.runs.every((r) => r.state !== "skipped" || r.passedByHand);
  return [...tests.values()];
};

const count = (tests: Test[]): Record<State, number> => {
  const counts = emptyCounts();
  for (const t of tests) counts[t.state]++;
  return counts;
};

const subsystems = (product: string, own: Test[]): Record<string, Subsystem> => {
  const out: Record<string, Subsystem> = {};
  for (const t of own) {
    let key = "e2e";
    let name = "End-to-end";
    if (t.kind === "unit") {
      const hit = SUBSYSTEMS[product].find(([, , match]) => match.test(t.path));
      if (hit == null) throw new Error(`no ${product} subsystem for ${t.path}`);
      [key, name] = hit;
    }
    out[key] ??= { name, count: 0 };
    out[key].count++;
  }
  return out;
};

// The size of each job's largest suite, which orders jobs within a product.
const jobSizes = new Map<string, number>();
for (const s of allSuites)
  jobSizes.set(s.job, Math.max(jobSizes.get(s.job) ?? 0, s.count));

const ORDER = ["core", "console", "arc", "clients", "driver", "toolchain"];

/**
 * Products in page order. Within a product, system tests come first, then unit jobs
 * from largest to smallest, so the tests of each job are one contiguous run of cells.
 */
export const products: Product[] = data.products
  .map(({ key, name }) => {
    const suites = allSuites
      .filter((s) => s.product === key && s.count > 0)
      .sort(
        (a, b) =>
          Number(a.kind === "unit") - Number(b.kind === "unit") ||
          jobSizes.get(b.job)! - jobSizes.get(a.job)! ||
          a.job.localeCompare(b.job),
      );
    const order = new Map(suites.map((s, i) => [s, i]));
    const own = group(
      allRuns
        .filter((r) => r.suite.product === key)
        .sort((a, b) => order.get(a.suite)! - order.get(b.suite)!),
    );
    return {
      key,
      name,
      tests: own,
      suites,
      counts: count(own),
      subsystems: subsystems(key, own),
    };
  })
  .filter((p) => p.tests.length > 0)
  .sort((a, b) => ORDER.indexOf(a.key) - ORDER.indexOf(b.key));

export const ordered: Test[] = products.flatMap((p) => p.tests);

export const runs: Run[] = ordered.flatMap((t) => t.runs);

/** Display names of the operating systems the run tested on, in a fixed order. */
export const systems = Object.values(OS_NAMES).filter((os) =>
  allSuites.some((s) => s.os === os),
);

export const totals = count(ordered);
