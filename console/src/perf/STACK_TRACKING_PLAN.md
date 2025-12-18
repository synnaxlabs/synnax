# ⚠️ IMPORTANT: PRE-REFACTOR PLAN ⚠️

```
╔═══════════════════════════════════════════════════════════════════════════╗
║                                                                           ║
║  🚨 THIS PLAN WAS WRITTEN BEFORE THE DASHBOARD CLEANUP/REFACTOR 🚨      ║
║                                                                           ║
║  File paths, line numbers, and implementation details MAY BE OUTDATED    ║
║  after the refactor completes. Use this as a REFERENCE for the overall   ║
║  approach and architecture, but verify all specifics against the         ║
║  refactored codebase before implementing.                                ║
║                                                                           ║
║  Created: 2025-12-18                                                     ║
║  Status: PENDING REFACTOR                                                ║
║                                                                           ║
╚═══════════════════════════════════════════════════════════════════════════╝
```

---

# Stack Tracking for Long Tasks Monitoring

## Goal
Add comprehensive stack trace capture to the Console performance profiling dashboard's long task monitoring system. Extract maximum contextual information to help identify what code is causing main thread blocking in production environments (Tauri desktop app, Chromium-based).

## User Requirements
- **Use case**: Production monitoring - extract ALL available information about long tasks
- **Platform**: Tauri desktop app (Chromium) with browser fallback
- **Scope**: Both native PerformanceObserver and RAF fallback modes
- **No manual instrumentation** (Phase 1 only - automatic capture)

## Current State Analysis (PRE-REFACTOR)

### Existing Files
- **`console/src/perf/metrics/longtasks.ts`** (lines 1-301)
  - Dual detection: PerformanceObserver (line 164) + RAF fallback (line 188)
  - Event attribution system (lines 82-107) correlates tasks with user events
  - Data: `LongTaskEntry { timestamp, duration, name }` (lines 13-17)
  - Display: 3 columns (name, age, duration) via `LONG_TASK_TABLE_COLUMNS` (lines 32-36)

- **`console/src/perf/Dashboard.tsx`** (lines 1-1076)
  - Section-based UI with collapsible groups (lines 380-433)
  - Long Tasks section (lines 612-630) shows live count + table when profiling
  - Uses `MetricTable` component (line 624) with tooltip support

- **`console/src/perf/components/MetricTable.tsx`** (lines 1-116)
  - Table component with `getTooltip` prop (line 82) for hover details
  - Pattern: `getNetworkTableTooltip` in network.ts shows how to display detailed info

### Key Patterns to Follow
1. **Network Collector Pattern** (network.ts): Aggregation, cleanup, tooltip with multi-line details
2. **Event Attribution** (longtasks.ts:82-107): Existing correlation system we can extend
3. **Tooltip Pattern** (MetricTable.tsx:49): `title` attribute for hover information

## Implementation Plan

### Phase 1: Comprehensive Stack Capture & Context Extraction

#### 1.1 Create Stack Utilities (`console/src/perf/utils/stack.ts`) - NEW FILE

**Purpose**: Capture, parse, and format stack traces with maximum information extraction

**Functions**:
```typescript
// Core capture
export function captureStack(): StackFrame[]
export function parseStackString(stack: string): StackFrame[]
export function parseStackLine(line: string): StackFrame | null

// Context extraction
export function extractComponentName(frames: StackFrame[]): string | null
export function extractSourceLocation(frame: StackFrame): string

// Filtering & cleanup
export function filterMonitoringFrames(frames: StackFrame[]): StackFrame[]
export function getTopFrames(frames: StackFrame[], count: number): StackFrame[]

// Deduplication & hashing
export function hashStack(frames: StackFrame[]): string
export function hashTopFrames(frames: StackFrame[], count: number): string

// Formatting
export function formatStackPreview(frames: StackFrame[]): string
export function formatStackFull(frames: StackFrame[]): string
export function formatFrame(frame: StackFrame): string
```

**Key Implementation Details**:
- **Cross-browser parsing**: Handle V8 (Chrome/Tauri), SpiderMonkey (Firefox), JSCore (Safari)
  - V8: `"    at functionName (file:line:col)"` or `"    at file:line:col"`
  - SpiderMonkey/JSCore: `"functionName@file:line:col"`
- **Smart filtering**: Remove internal monitoring frames (longtasks.ts, RAF, PerformanceObserver)
- **Component detection**: Search for React component names in stack (e.g., `Dashboard`, `MetricTable.render`)
- **Source location**: Extract file path, line, column for quick navigation
- **Hashing strategy**: Hash function names + file paths (not line numbers) for deduplication
- **Preview format**: `"functionName (file:line)"` - first meaningful frame only
- **Full format**: Multi-line with up to 10 frames, indented for readability

**Configuration**:
```typescript
const STACK_CONFIG = {
  MAX_DEPTH: 10,                    // Capture top 10 frames
  PREVIEW_FRAMES: 1,                 // Show 1 frame in table preview
  IGNORED_PATTERNS: [
    /longtasks\.ts/,
    /requestAnimationFrame/,
    /PerformanceObserver/,
  ],
};
```

#### 1.2 Enhance LongTaskCollector (`console/src/perf/metrics/longtasks.ts`) - MODIFY

**Data Structure Updates**:
```typescript
// Extend LongTaskEntry
interface LongTaskEntry {
  timestamp: number;
  duration: number;
  name: string;                      // Existing: event attribution
  stack?: StackFrame[];              // NEW: Parsed stack frames (top 10)
  stackHash?: string;                // NEW: For deduplication
  componentName?: string;            // NEW: React component if detected
  sourceLocation?: string;           // NEW: Primary source file:line
}

// Add StackFrame interface
interface StackFrame {
  functionName: string;              // Function name or "anonymous"
  fileName: string;                  // File path (may be minified)
  lineNumber?: number;               // Line number if available
  columnNumber?: number;             // Column number if available
}

// Extend LongTaskStats (for display)
export interface LongTaskStats {
  name: string;                      // Event attribution
  duration: number;
  timestamp: number;
  age: number;
  stackPreview?: string;             // NEW: "functionName (file:line)"
  stackFrames?: StackFrame[];        // NEW: Full stack for tooltip
  componentName?: string;            // NEW: React component name
  sourceLocation?: string;           // NEW: file:line:col
}
```

**Constructor Enhancement**:
```typescript
constructor(
  windowMs = 600_000,
  captureStacks = true,              // NEW: Enable/disable stack capture
) {
  this.windowMs = windowMs;
  this.captureStacks = captureStacks;
}
```

**New Method - Centralized Long Task Handler**:
```typescript
private handleLongTask(timestamp: number, duration: number): void {
  this.totalCount++;
  this.totalDurationMs += duration;

  const name = this.findEventForTask(timestamp);

  // Capture comprehensive context
  let stack: StackFrame[] | undefined;
  let stackHash: string | undefined;
  let componentName: string | undefined;
  let sourceLocation: string | undefined;

  if (this.captureStacks) {
    const rawStack = captureStack();           // From utils/stack.ts
    stack = filterMonitoringFrames(rawStack);
    if (stack.length > 0) {
      stackHash = hashTopFrames(stack, 3);     // Hash top 3 for grouping
      componentName = extractComponentName(stack);
      sourceLocation = extractSourceLocation(stack[0]);
    }
  }

  this.recentTasks.push({
    timestamp,
    duration,
    name,
    stack,
    stackHash,
    componentName,
    sourceLocation,
  });
}
```

**Update Native Observer**:
```typescript
this.observer = new PerformanceObserver((list) => {
  const now = performance.now();
  for (const entry of list.getEntries()) {
    this.handleLongTask(now, entry.duration);  // Use centralized handler
  }
});
```

**Update RAF Callback**:
```typescript
const rafCallback = (currentTime: number) => {
  if (this.lastRafTime !== null) {
    const delta = currentTime - this.lastRafTime;
    if (delta > this.LONG_TASK_THRESHOLD_MS) {
      this.handleLongTask(performance.now(), delta);  // Use centralized handler
    }
  }
  this.lastRafTime = currentTime;
  this.rafId = requestAnimationFrame(rafCallback);
};
```

**Enhance getTopLongTasks()**:
```typescript
getTopLongTasks(): { data: LongTaskStats[]; total: number; truncated: boolean } {
  const now = performance.now();
  const cutoff = now - this.windowMs;
  this.recentTasks = this.recentTasks.filter((t) => t.timestamp >= cutoff);

  const data = this.recentTasks
    .slice()
    .reverse()
    .map((task) => ({
      name: task.name,
      duration: task.duration,
      timestamp: task.timestamp,
      age: now - task.timestamp,
      stackPreview: task.stack ? formatStackPreview(task.stack) : undefined,  // NEW
      stackFrames: task.stack,                                                 // NEW
      componentName: task.componentName,                                       // NEW
      sourceLocation: task.sourceLocation,                                     // NEW
    }));

  return { data, total: data.length, truncated: false };
}
```

**Update Table Columns**:
```typescript
export const LONG_TASK_TABLE_COLUMNS: MetricTableColumn<LongTaskStats>[] = [
  { getValue: (task) => task.name, color: 7 as const },
  { getValue: (task) => formatAge(task.age) },
  { getValue: (task) => formatDuration(task.duration) },
  { getValue: (task) => task.stackPreview ?? '—', color: 8 as const },  // NEW
];
```

**Add Tooltip Generator**:
```typescript
// NEW: Tooltip with full stack trace and context
export const getLongTaskTableTooltip = (task: LongTaskStats): string | undefined => {
  if (!task.stackFrames || task.stackFrames.length === 0) return undefined;

  const parts: string[] = [];

  // Context header
  if (task.componentName) {
    parts.push(`Component: ${task.componentName}`);
  }
  if (task.sourceLocation) {
    parts.push(`Location: ${task.sourceLocation}`);
  }
  parts.push(`Duration: ${formatDuration(task.duration)}`);
  parts.push(`Event: ${task.name}`);
  parts.push('');  // Blank line

  // Full stack trace
  parts.push('Stack Trace:');
  parts.push(formatStackFull(task.stackFrames));

  return parts.join('\n');
};
```

#### 1.3 Add Formatting Utilities (`console/src/perf/utils/formatting.ts`) - MODIFY

**Add stack formatting functions**:
```typescript
// Append to existing file

export const formatStackPreview = (frames: StackFrame[]): string => {
  if (!frames || frames.length === 0) return 'No stack';
  const frame = frames[0];
  const fileName = frame.fileName.split('/').pop()?.split('?')[0] || frame.fileName;
  const location = frame.lineNumber
    ? `${fileName}:${frame.lineNumber}`
    : fileName;
  return frame.functionName !== 'anonymous'
    ? `${frame.functionName} (${location})`
    : location;
};

export const formatStackFull = (frames: StackFrame[]): string => {
  return frames
    .map((f, i) => {
      const fn = f.functionName || 'anonymous';
      const file = f.fileName.split('/').pop()?.split('?')[0] || f.fileName;
      const loc = f.lineNumber ? `:${f.lineNumber}` : '';
      return `  ${i + 1}. ${fn} (${file}${loc})`;
    })
    .join('\n');
};
```

#### 1.4 Update Dashboard Display (`console/src/perf/Dashboard.tsx`) - MODIFY

**Add tooltip to MetricTable**:
```typescript
<MetricTable
  result={topLongTasks}
  columns={LONG_TASK_TABLE_COLUMNS}
  getKey={getLongTaskTableKey}
  getTooltip={getLongTaskTableTooltip}  // NEW: Add tooltip support
/>
```

**Update imports**:
```typescript
import {
  getLongTaskTableKey,
  getLongTaskTableTooltip,           // NEW
  LongTaskCollector,
  LONG_TASK_TABLE_COLUMNS,
  type LongTaskStats,
} from "@/perf/metrics/longtasks";
```

## Memory & Performance Considerations

### Memory Management
- **Stack frame limit**: 10 frames max per capture (configurable)
- **Deduplication**: Hash-based grouping by top 3 frames reduces storage
- **Existing cleanup**: 600s window cleanup handles stack disposal
- **Pattern reuse**: Same memory management as existing NetworkCollector

### Performance Overhead
- **Stack capture**: ~0.1-0.5ms per capture (negligible vs 50ms threshold)
- **Parsing**: Simple regex, ~0.05ms
- **Filtering**: Linear scan, ~0.01ms
- **Total**: <1ms overhead per long task detection
- **Production safe**: Minimal impact, no blocking operations

### Configuration
```typescript
// Default: enabled
const collector = new LongTaskCollector(600_000, true);

// Disable if needed (not recommended for production monitoring)
const collector = new LongTaskCollector(600_000, false);
```

## Display Architecture

### Table View (4 columns)
```
| Event Attribution | Age     | Duration | Stack (Preview)           |
|-------------------|---------|----------|---------------------------|
| click (button)    | 2.3s    | 127ms    | handleClick (App.tsx:45)  |
| Unknown           | 5.1s    | 89ms     | render (Dashboard.tsx:12) |
| keydown (input)   | 8.7s    | 67ms     | processInput (Form.tsx:8) |
```

### Tooltip (hover for full details)
```
Component: Dashboard
Location: Dashboard.tsx:142:15
Duration: 127ms
Event: click (button)

Stack Trace:
  1. handleClick (App.tsx:45)
  2. processEvent (eventHandler.ts:112)
  3. syncUpdate (reconciler.ts:89)
  4. commitRoot (ReactFiberWorkLoop.ts:234)
  5. render (ReactDOM.ts:67)
  6. ... (5 more frames)
```

## Additional Context Extraction (Maximize Information)

Beyond stack traces, the implementation will capture:

1. **React Component Name**: Parse stack for component boundaries
   - Pattern: `functionName.render`, `ComponentName`, etc.
   - Helps identify which part of UI triggered the task

2. **Source Location**: Primary file:line:col
   - Quick navigation to source
   - Works with minified code (shows bundle location)

3. **Stack Hash**: Deduplication key
   - Groups identical call patterns
   - Future: Enable frequency analysis (not in Phase 1)

4. **Event Attribution**: Existing system (preserved)
   - User interaction that preceded task
   - Correlation window: 1000ms

5. **Timestamp & Duration**: Existing (preserved)
   - Temporal analysis
   - Pattern detection over time

## Testing Strategy

### Unit Tests (stack.ts)
```typescript
describe('Stack Parsing', () => {
  it('should parse V8 format');
  it('should parse SpiderMonkey format');
  it('should filter monitoring frames');
  it('should extract component names');
  it('should hash stacks consistently');
});
```

### Manual Testing
- [ ] Trigger long task on Tauri (Chromium)
- [ ] Verify stack appears in table
- [ ] Check tooltip shows full trace
- [ ] Confirm component name detection works
- [ ] Test on Safari (RAF fallback mode)
- [ ] Verify memory cleanup (run for 10+ minutes)

### Edge Cases
- **No stack available**: Display "—" gracefully
- **Parse failures**: Catch exceptions, log warning, continue
- **Minified code**: Accept bundled names (source maps not implemented)
- **Cross-browser**: Parser handles all major formats
- **Empty stacks**: Filter leaves no frames → display "No stack"

## File Change Summary

### New Files
1. **`console/src/perf/utils/stack.ts`** (~200 lines)
   - Stack capture, parsing, filtering, hashing, formatting

### Modified Files (⚠️ LINE NUMBERS MAY CHANGE POST-REFACTOR)
1. **`console/src/perf/metrics/longtasks.ts`** (~350 lines, +50 lines)
   - Add `StackFrame` interface
   - Extend `LongTaskEntry` and `LongTaskStats`
   - Add `handleLongTask()` method
   - Update native observer callback
   - Update RAF callback
   - Enhance `getTopLongTasks()`
   - Update `LONG_TASK_TABLE_COLUMNS`
   - Add `getLongTaskTableTooltip()`

2. **`console/src/perf/utils/formatting.ts`** (~20 lines added)
   - Add `formatStackPreview()`
   - Add `formatStackFull()`

3. **`console/src/perf/Dashboard.tsx`** (~2 lines changed)
   - Import `getLongTaskTableTooltip`
   - Pass `getTooltip` to MetricTable

## Implementation Sequence

1. **Create `utils/stack.ts`** (Core utilities) - 2-3 hours
2. **Modify `longtasks.ts`** (Collector integration) - 2-3 hours
3. **Update `formatting.ts`** (Display helpers) - 30 minutes
4. **Update `Dashboard.tsx`** (Wire tooltip) - 15 minutes
5. **Manual testing** (Cross-browser verification) - 1 hour

**Total Estimate**: 6-8 hours development + testing

## Success Criteria

1. ✅ Stack traces captured for both native and RAF modes
2. ✅ Table shows 4th column with stack preview
3. ✅ Tooltip displays full stack trace with context
4. ✅ Component names detected when present
5. ✅ Works across Chromium (Tauri), Safari, Firefox
6. ✅ <1ms performance overhead
7. ✅ Memory managed via existing 600s window cleanup
8. ✅ Graceful degradation (no crashes if stack unavailable)

## Future Enhancements (Not in Phase 1)

- **Stack-based aggregation**: Group long tasks by stack hash, show frequency
- **Source map support**: Decode minified stacks to original source
- **Manual marks integration**: Phase 2 instrumentation API
- **Export functionality**: Download stack traces for offline analysis
- **Pattern detection**: Identify recurring stack patterns automatically

---

## ⚠️ POST-REFACTOR CHECKLIST ⚠️

Before implementing this plan after the refactor, verify:

- [ ] `longtasks.ts` still exists at the same path
- [ ] `Dashboard.tsx` structure hasn't significantly changed
- [ ] `MetricTable` component still uses `getTooltip` pattern
- [ ] Event attribution system still uses same approach
- [ ] Line numbers updated to match refactored code
- [ ] New architecture patterns are compatible with this approach
- [ ] Configuration/initialization patterns haven't changed
- [ ] Testing infrastructure still supports manual testing approach

**If major changes occurred, re-evaluate this plan against the new architecture.**
