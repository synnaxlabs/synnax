# Performance Profiling Dashboard & Automated Macros

## Overview

The Console app (`/console/`) includes a performance profiling dashboard (`/console/src/perf/`) that monitors real-time metrics during user interactions. It tracks CPU, GPU, FPS, and heap memory usage, runs automated "macros" (scripted UI interactions), and generates reports with severity labels (nominal/warning/error) that are saved to Synnax ranges.

The "automated macros" provide a way to create automated console actions to test specific components and enable us to establish known profiling metrics for the console. This differs from playwright integration tests, where the objective there is to test the console as if we are a real user.

A con of the playwright testing is that it uses the browser, so there is no access to resource use for CPU, GPU, and heap. It's also VERY slow. That fundamentally limits what we can actually test for. Conversely, we cannot use playwright to test the desktop app (which means no DevTools available).

Saving profiling metrics to synnax (either as KV metadata or independent channels) allows us to store all metrics for the desktop app, which is the primary use-case of the console. The macros feature will allow us to test multiple different vectors and enhanced integration testing. Examples:

- Start the profiler in the setup() for each existing playwright integration test
- Playwright integration tests can trigger existing macros
- Run long-duration testing that is not suitable for integration testing
  - Repetitions of a single action (drop down components, new pages, etc)
  - Run macros/actions in random order to test for unknown impacts
- There is likely a way to fully automate the macros in CI. I have not explored this yet.



## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Dashboard.tsx                           │
│  (Main UI: Start/Stop/Pause, MetricSections, MacroPanel)        │
└─────────────────────────────────────────────────────────────────┘
                                │
         ┌──────────────────────┼──────────────────────┐
         ▼                      ▼                      ▼
┌─────────────────┐   ┌─────────────────┐   ┌─────────────────────┐
│   Collectors    │   │    Analyzers    │   │  useMacroExecution  │
│ CPU, GPU, FPS,  │   │ FPS, CPU, GPU,  │   │ (Executes scripted  │
│ Heap, Network,  │   │ Heap (leak)     │   │  UI interactions)   │
│ LongTasks, Logs │   │                 │   │                     │
└─────────────────┘   └─────────────────┘   └─────────────────────┘
         │                      │                      │
         └──────────┬───────────┘                      │
                    ▼                                  │
         ┌─────────────────────┐                       │
         │   Report Compiler   │                       │
         │ (Verdict, Issues,   │                       │
         │  MetricsReport)     │                       │
         └─────────────────────┘                       │
                    │                                  │
                    ▼                                  ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Redux Slice (perf/slice.ts)                  │
│  status, config, reports, macroResults, rangeKey                │
└─────────────────────────────────────────────────────────────────┘
```

## Thresholds (from constants.ts)

| Metric | Warning | Error |
|--------|---------|-------|
| FPS (peak/min) | < 10 | < 1 |
| FPS (avg) | < 25 | < 10 |
| FPS Change | > 20% degradation | > 50% degradation |
| CPU (peak) | > 90% | > 99% |
| CPU (avg) | > 50% | > 75% |
| CPU Change | > 80% | > 98% |
| GPU (peak) | > 90% | > 99% |
| GPU (avg) | > 50% | > 75% |
| GPU Change | > 80% | > 98% |
| Heap Growth | > 20% | > 40% |

## Key Files

**Core Infrastructure:**
- `slice.ts` - Redux state: status (idle/running/paused/error), config, reports
- `types/v0.ts` - Versioned types for HarnessConfig, SliceState, MetricsConfig, MacroConfig
- `constants.ts` - Thresholds, metric names, label colors, buffer sizes

**Metrics Collection:**
- `metrics/buffer.ts` - SampleBuffer with baseline + circular recent buffer, aggregates
- `metrics/polling-collector.ts` - Base class for CPU/GPU/Heap collectors
- `metrics/cpu.ts`, `gpu.ts`, `heap.ts`, `fps.ts` - Core metric collectors
- `metrics/network.ts` - Network request tracking (PerformanceObserver)
- `metrics/longtasks.ts` - Browser long task detection (jank > 50ms)
- `metrics/console.ts` - Console log capture (errors, warnings)

**Analysis:**
- `analyzer/resource-analyzer.ts` - Base class for resource analysis (peak/avg severity)
- `analyzer/cpu-analyzer.ts`, `gpu-analyzer.ts` - CPU/GPU analysis extending resource-analyzer
- `analyzer/fps-analyzer.ts` - FPS analysis with peak (minFps) and avg severity
- `analyzer/heap-analyzer.ts` - Memory leak detection via linear regression trend analysis

**Report:**
- `report/types.ts` - FinalReport, DetectedIssue, Verdict, MetricsReport types
- `report/compiler.ts` - Compiles samples + analysis results into final report

**Macros:**
- `macros/types.ts` - MacroType (branded string), MacroConfig, MacroResult, MacroContext
- `macros/registry.ts` - MacroRegistry pattern for registering/retrieving macros
- `macros/runner.ts` - Executes macros in configurable iterations
- `macros/execution.ts` - MacroExecutionState, progress tracking types
- `macros/lineplot.ts` - LinePlot macro: create plot, snap to right, add channels, close
- `macros/schematic.ts` - Schematic macro: create schematic, add symbols, align/distribute, close

**Hooks:**
- `hooks/useCollectors.ts` - Manages all metric collectors, sampling interval
- `hooks/useProfilingSession.ts` - Orchestrates session lifecycle (start/pause/resume/stop)
- `hooks/useProfilingRange.ts` - Creates Synnax ranges, adds labels based on severity
- `hooks/useMacroExecution.ts` - Manages macro execution state and runner lifecycle
- `hooks/useCapturedValues.ts` - Tracks initial/final metric values for delta analysis
- `hooks/useProfilingAnalyzers.ts` - Composes all analyzers into single analysis function
- `hooks/useElapsedSeconds.ts` - Tracks elapsed profiling time with pause support


## Command Palette Integration

> **Note:** This functionality is nice-to-have but not required. Deferring to higher opinion on whether to keep/expand.

The profiling dashboard registers commands in the Console command palette (`palette.tsx`):

| Command | Visibility | Action |
|---------|------------|--------|
| Start Console Profiling | `status === "idle"` | Opens dashboard and starts profiling |
| Open Console Profiling Monitor | Always | Opens dashboard (doesn't start profiling) |
| Pause Console Profiling | `status === "running"` | Pauses active session |
| Resume Console Profiling | `status === "paused"` | Resumes paused session |
| Reset Console Profiling | `status === "paused" \|\| "error"` | Resets session state |

Commands are conditionally visible based on current profiling status, preventing invalid state transitions.

## Range & Label System

When profiling starts, a Synnax range is created to track the session. Labels are dynamically added/removed based on metric severity:

**Label Hierarchy:**
- `Nominal` - Default label, removed when any issue is detected
- `{Metric} Warn` - Added when warning threshold exceeded (e.g., "FPS Warn", "CPU Warn")
- `{Metric} Error` - Added when error threshold exceeded, supersedes warning label

**Latching Behavior:**
- **Peak-triggered labels** (latched): Permanent once added, cannot be removed
- **Avg-triggered labels** (transient): Can be removed if avg improves back to nominal
- Once a metric is latched, further changes for that metric are ignored

**Range Metadata (KV store):**
- Only tracking basic metrics for now
- `hostname`, `platform`, `osVersion` - System info at session start
- `username` - Logged-in user who started the session
- `version` - Synnax Console version
- `startValues` - JSON object with live values at session start (`{fps, cpu, gpu, heap}`)
- `stopValues` - JSON object with live values at session stop (`{fps, cpu, gpu, heap}`)
- `averages` - JSON object with running averages (`{cpu, fps, gpu}`)
- `peaks` - JSON object with worst-case values (`{cpu, fps, gpu, heap}`)
- Metadata updates every 5 seconds while running
- On resume, `stopValues` is cleared (only present after a stop)

**Finalization Flow:**
1. Session stops → `finalizeRange()` called
2. End time updated on the range
3. For each metric with issues: add appropriate warn/error label
4. Error labels supersede and remove warning labels for same metric
5. Nominal label removed if any warn/errors present

## Report System

When a profiling session ends, the report compiler aggregates all collected data into a structured `FinalReport`:

```
Collectors (samples) ─┐
                      ├──► Report Compiler ──► FinalReport
Analyzers (results) ──┘
```

**FinalReport Structure:**
```typescript
interface FinalReport {
  summary: {
    verdict: "Passed" | "Failed";  // Failed if any critical issue
    durationMs: number;
    totalSamples: number;
    issueCount: number;
  };
  metrics: {
    fps: { avg, min, max, changePercent };
    cpu: { avg, max, changePercent };
    gpu: { avg, max, changePercent };
    memory: { minHeapMB, maxHeapMB, growthPercent };
  };
  issues: DetectedIssue[];  // List of threshold violations
}
```

**Issue Detection:**
- Compares aggregates against thresholds from `constants.ts`
- Issues are categorized: `fps`, `cpu`, `gpu`, `memory`
- Severity levels: `warning` (threshold exceeded) or `critical` (error threshold exceeded)
- Verdict is `Failed` if any critical issue exists, otherwise `Passed`

## Macro System (Dev only)
This tool should be hidden in production.
- Enables rapid testing of Console actions
  - Faster than playwright integration tests
- Registry pattern for extensible macro definitions
- MacroRunner executes steps with configurable delays
- MacroPanel UI for selecting and running macros
- LinePlot and Schematic macros are example implementations demonstrating the pattern
- Context provides: store, dispatch, placer, client, createdLayoutKeys

**Examples:**
1. **linePlot** (`macros/lineplot.ts`) - Creates a line plot, snaps to right, adds channels, then closes.
2. **schematic** (`macros/schematic.ts`) - Creates a schematic, snaps to right, adds symbols, aligns/distributes, then closes.
**MacroConfig:**
```typescript
interface MacroConfig {
  macros: MacroType[];                 // Which macros to run
  iterations: number;                   // How many times to run each (-1 for unlimited)
  delayBetweenMacrosMs: number;        // Delay between individual macros
  delayBetweenStepsMs: number;         // Delay between steps within a macro
}
```

**MacroDefinition:**
```typescript
interface MacroDefinition {
  type: MacroType;
  name: string;
  description: string;
  category: MacroCategory;
  factory: () => MacroStep[];
}
```

**MacroStep:**
```typescript
interface MacroStep {
  name: string;
  execute: (context: MacroContext) => Promise<void>;
}
```

**MacroContext** provides:
- `store` - Redux store
- `dispatch` - Redux dispatch
- `placer` - Layout.Placer for creating visualizations
- `client` - Synnax client (optional)
- `createdLayoutKeys` - Track layouts for cleanup
- `availableChannelKeys` - Channels to use



## Macro Execution Flow

```
Dashboard.tsx
    └── MacroPanel (Dev mode only)
            ├── MacroSelect (multi-select from registry)
            ├── MacroConfigInputs (iterations, delays)
            └── Run/Cancel button
                    │
                    ▼
            useMacroExecution.ts
                    │
                    ▼
            MacroRunner.run()
                    │
                    ├── For each iteration:
                    │   └── For each macro:
                    │       └── For each step:
                    │           └── step.execute(context)
                    │
                    └── Progress updates via onMacroComplete callback
```

## Profiler Implementation Patterns

### Circular Dependency Resolution (Dashboard.tsx)

The Dashboard has a circular dependency between hooks:
1. `useCollectors` needs `onSample` callback to report collected samples
2. `useProfilingSession` needs `collectors` and `sampleBuffer` from `useCollectors`
3. `useProfilingSession` returns `handleSample` which IS the `onSample` callback

**Solution: Ref + Stable Wrapper Pattern**
```typescript
// 1. Create ref to hold the eventual function
const handleSampleRef = useRef<HandleSampleFn | undefined>(undefined);

// 2. Create stable wrapper that delegates to ref (empty deps = stable identity)
const onSample = useCallback(
  (sample, buffer) => handleSampleRef.current?.(sample, buffer),
  [],
);

// 3. Pass stable wrapper to useCollectors
const { collectors, sampleBuffer } = useCollectors({ status, onSample });

// 4. Get real function from useProfilingSession
const { handleSample } = useProfilingSession({ collectors, sampleBuffer });

// 5. Update ref synchronously during render
handleSampleRef.current = handleSample;
```

This works because:
- The wrapper's identity is stable (empty deps), so `useCollectors` doesn't re-run
- The ref is updated synchronously during render, before any effects run
- When `useCollectors` calls `onSample`, it will use the latest `handleSample`

### Status Ref Optimization (useCollectors.ts)

To avoid recreating collectors on every status change, `useCollectors` uses a ref pattern:

```typescript
// Store status in ref to avoid recreating collectors
const statusRef = useRef(status);
statusRef.current = status;

useEffect(() => {
  // Collectors created once on mount
  const c = collectorsRef.current;
  c.cpu = new CpuCollector();
  // ...

  const updateInterval = setInterval(() => {
    // Read from ref instead of closure
    if (statusRef.current === "running") {
      sampleBufferRef.current.push(sample);
    }
  }, SAMPLE_INTERVAL_MS);

  return () => clearInterval(updateInterval);
}, [collectSample]); // status NOT in deps
```

### Async Cleanup (useProfilingRange.ts)

Async operations use an AbortController to prevent state updates after unmount:

```typescript
const abortRef = useRef<AbortController | null>(null);

useEffect(() => {
  abortRef.current = new AbortController();
  return () => abortRef.current?.abort();
}, []);

const createRange = useCallback(() => {
  const create = async () => {
    // ... async operations
  };

  create().catch((error: Error) => {
    // Skip error logging if component unmounted
    if (abortRef.current?.signal.aborted) return;
    console.error("Failed to create profiling range:", error);
  });
}, [client, dispatch, getRangeName]);
```

The abort check is placed only in catch handlers to minimize boilerplate while still preventing console noise from expected unmount errors.

### Macro Execution Error Handling (useMacroExecution.ts)

The macro runner uses proper promise handling with `.catch()` and `.finally()`:

```typescript
void runner
  .run()
  .catch((e) => console.error("Macro runner error:", e))
  .finally(() => {
    setState((prev) =>
      prev.status === "running"
        ? { ...prev, status: "idle", progress: ZERO_EXECUTION_STATE.progress }
        : prev,
    );
    runnerRef.current = null;
  });
```
