# Performance Profiling Dashboard & Automated Workflows

## Overview

The Console app (`/console/`) includes a performance profiling dashboard (`/console/src/perf/`) that monitors real-time metrics during user interactions. It tracks CPU, GPU, FPS, and heap memory usage, runs automated "workflows" (scripted UI interactions), and generates reports with severity labels (nominal/warning/error) that are saved to Synnax ranges.

The "automated workflow" provide way to create automated console actions to test specific components and enable us to establish known profiling metrics for the console. This differs from playwright integration tests, where the objective there is to test the console as if we are a real user.

A con of the playwright testing is that it uses the browser, so there is no access to resource use for CPU, GPU, and heap. It's also VERY slow. That fundementally limits what we can actually test for. Conversely, we cannot use playwright to test the desktop app (which means no DevTools available).

Saving profiling metrics to synnax (either as KV metadata or independent channels) allows us to store all metrics for the desktop app, which is the primary use-case of the console. The workflow/macros feature will allow us to test multiple different vectors and enhanced integration testing. Examples:

- Start the profiler in the setup() for each existing playwright integration test
- Playwright integration tests can trigger existing workflows/macros
- Run long-duration testing that is not suitable for integration testing
  - Repetitions of a single action (drop down components, new pages, etc)
  - Run workflows/actions in random order to test for unknown impacts
- There is likely a way to fully automate the workflows in CI. I have not explored this yet.



## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                         Dashboard.tsx                           │
│  (Main UI: Start/Stop/Pause, MetricSections, WorkflowPanel)     │
└─────────────────────────────────────────────────────────────────┘
                                │
         ┌──────────────────────┼──────────────────────┐
         ▼                      ▼                      ▼
┌─────────────────┐   ┌─────────────────┐   ┌─────────────────────┐
│   Collectors    │   │    Analyzers    │   │  useWorkflowExec    │
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
│  status, config, reports, workflowResults, rangeKey             │
└─────────────────────────────────────────────────────────────────┘
```

## Thresholds (from constants.ts)

| Metric | Warning | Error |
|--------|---------|-------|
| FPS (peak/min) | < 10 | < 5 |
| FPS (avg) | < 25 | < 10 |
| FPS Change | > 20% degradation | > 40% degradation |
| CPU (peak) | > 85% | > 95% |
| CPU (avg) | > 50% | > 75% |
| GPU (peak) | > 85% | > 95% |
| GPU (avg) | > 50% | > 75% |
| Heap Growth | > 20% | > 40% |

## Key Files

**Core Infrastructure:**
- `slice.ts` - Redux state: status (idle/running/paused/error), config, reports
- `types/v0.ts` - Versioned types for HarnessConfig, SliceState, MetricsConfig, WorkflowConfig
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

**Workflows:**
- `workflows/types.ts` - WorkflowType (branded string), WorkflowConfig, WorkflowResult, WorkflowContext
- `workflows/registry.ts` - WorkflowRegistry pattern for registering/retrieving workflows
- `workflows/runner.ts` - Executes workflows in configurable iterations
- `workflows/execution.ts` - WorkflowExecutionState, progress tracking types
- `workflows/lineplot.ts` - LinePlot workflow: create plot, snap to right, add channels, close
- `workflows/schematic.ts` - Schematic workflow: create schematic, add symbols, align/distribute, close

**Hooks:**
- `hooks/useCollectors.ts` - Manages all metric collectors, sampling interval
- `hooks/useProfilingSession.ts` - Orchestrates session lifecycle (start/pause/resume/stop)
- `hooks/useProfilingRange.ts` - Creates Synnax ranges, adds labels based on severity
- `hooks/useWorkflowExecution.ts` - Manages workflow execution state and runner lifecycle

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
- Only tracking basic matrics for now
- `hostname`, `platform`, `osVersion` - System info at session start
- `averages` - JSON object with running averages (`{cpu, fps, gpu}`)
- `peaks` - JSON object with worst-case values (`{cpu, fps, gpu, heap}`)
- Metadata updates every 5 seconds while running

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

## Workflow System (Dev only)
This tool should be hidden in production.
- Enables rapid testing of Console actions 
  - Faster than playwright integration tests
- Registry pattern for extensible workflow definitions
- WorkflowRunner executes steps with configurable delays
- WorkflowPanel UI for selecting and running workflows
- LinePlot and Schematic workflows are example implementations demonstrating the pattern
- Context provides: store, dispatch, placer, client, createdLayoutKeys

**WorkflowConfig:**
```typescript
interface WorkflowConfig {
  workflows: WorkflowType[];           // Which workflows to run
  iterations: number;                   // How many times to run each (-1 for unlimited)
  delayBetweenIterationsMs: number;    // Delay between iteration loops
  delayBetweenWorkflowsMs: number;     // Delay between individual workflows
}
```

**WorkflowDefinition:**
```typescript
interface WorkflowDefinition {
  type: WorkflowType;
  name: string;
  description: string;
  category: WorkflowCategory;
  factory: () => WorkflowStep[];
}
```

**WorkflowStep:**
```typescript
interface WorkflowStep {
  name: string;
  execute: (context: WorkflowContext) => Promise<void>;
  delayAfterMs?: number;  // Optional delay after step completes
}
```

**WorkflowContext** provides:
- `store` - Redux store
- `dispatch` - Redux dispatch
- `placer` - Layout.Placer for creating visualizations
- `client` - Synnax client (optional)
- `createdLayoutKeys` - Track layouts for cleanup
- `availableChannelKeys` - Channels to use

## Example Workflows

1. **linePlot** (`workflows/lineplot.ts`) - Creates a line plot, snaps to right, adds channels, then closes.
2. **schematic** (`workflows/schematic.ts`) - Creates a schematic, snaps to right, adds symbols, aligns/distributes, then closes.

## Adding New Workflows

Register workflows in their own file using `registerWorkflow()`:

```typescript
registerWorkflow({
  type: "myWorkflow",
  name: "My Workflow",
  description: "Does something",
  category: "general",
  factory: () => [
    {
      name: "Step 1",
      execute: async (ctx) => {
        ctx.placer(Layout.create({ ... }));
      },
      delayAfterMs: 500,
    },
  ],
});
```


## Workflow Execution Flow

```
Dashboard.tsx
    └── WorkflowPanel (Dev mode only)
            ├── WorkflowSelect (multi-select from registry)
            ├── WorkflowConfigInputs (iterations, delays)
            └── Run/Cancel button
                    │
                    ▼
            useWorkflowExecution.ts
                    │
                    ▼
            WorkflowRunner.run()
                    │
                    ├── For each iteration:
                    │   └── For each workflow:
                    │       └── For each step:
                    │           └── step.execute(context)
                    │
                    └── Progress updates via onWorkflowComplete callback
```

- **Registry pattern**: Workflows self-register via `registerWorkflow()`
- **Defined workflows**: `linePlot`, `schematic` (example implementations)
- **WorkflowContext**: Provides `store`, `dispatch`, `placer`, `client`
- **Future work**: Add more workflow examples for other Console features
