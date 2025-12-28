# Arc Task Infrastructure Redesign

## Problem Statement

Arc programs currently have two conflicting execution models:

1. **Go Server Model**: Arc programs have a `Deploy` flag that tells the Arc service
   runtime coordinator to compile and execute the program. Lifecycle is declarative -
   setting `Deploy = true` starts execution, `Deploy = false` stops it.

2. **C++ Driver Model**: The driver uses a task-based model where tasks receive
   imperative commands (`start`, `stop`) via the `sy_task_cmd` channel.

These models are in contention:

- The Go runtime starts automatically when `Deploy = true`, with no task created
- The C++ driver expects explicit task creation and commands
- If both run the same program, state diverges and channel writes race
- No unified lifecycle control or status reporting

## Solution: Unified Task-Based Execution

We will unify on the **task-based model** for all Arc execution. Arc programs become
inert data; execution is controlled entirely through tasks and commands.

### Key Concepts

| Concept | Responsibility |
|---------|----------------|
| **Arc Program** | Pure data - the code/logic definition. No lifecycle state. |
| **Arc Task** | An execution instance. References an Arc program, assigned to a rack. |
| **Arc Executor** | Runtime that processes tasks. One in Go (core), one in C++ (driver). |

### Core Rack

The Go executor simply **creates a standard rack** on startup (just like the C++
embedded driver does). No special rack type needed - a rack is just an identifier
for a task executor. What backs it (Go or C++) is an implementation detail.

```
Node Architecture:
├── Core Rack               → Go task executor in core service (NEW)
├── Embedded Rack           → C++ driver (in-process)
└── External Rack(s)        → C++ driver(s) (separate processes)
```

The Core Rack is a regular rack. The Go executor creates it on startup and claims it.

### Execution Routing

Task keys encode the target rack:

```
Task Key (64 bits)
├── Rack Key (32 bits) → Determines which executor handles the task
│   ├── Node Key (16 bits)
│   └── Local Key (16 bits)
└── Local Task Key (32 bits)
```

- Task assigned to Core Rack → Go executor handles it
- Task assigned to Embedded/External Rack → C++ driver handles it

Same Arc program can run on different racks (different executors, different configs).

---

## Implementation Plan

### Phase 1: No Rack Model Changes Needed

The rack model stays as-is. A rack is just an identifier - what backs it is an
implementation detail of the executor.

---

### Phase 2: Arc Model Changes

#### 2.1 Remove Deploy flag from Arc model

**File**: `core/pkg/service/arc/arc.go`

Remove the `Deploy` field:

```go
type Arc struct {
    Key     uuid.UUID
    Name    string
    Graph   graph.Graph
    Text    text.Text
    Module  module.Module  // Computed, not stored
    Version string
    // Deploy field REMOVED
}
```

#### 2.2 Remove Deploy/Stop methods from Arc service

**File**: `core/pkg/service/arc/service.go`

Remove:
- `Deploy(ctx, key)` method
- `Stop(ctx, key)` method
- `handleChange` auto-deployment logic

The Arc service becomes purely CRUD for program definitions.

#### 2.3 Keep compilation logic

The Arc service should still handle compilation. Add/keep a method:

```go
func (s *Service) Compile(ctx context.Context, key uuid.UUID) (module.Module, error) {
    arc, err := s.Retrieve(ctx, key)
    if err != nil {
        return module.Module{}, err
    }
    return arc.CompileGraph(ctx, arc.Graph, arc.WithResolver(s.symbolResolver))
}
```

---

### Phase 3: Go Arc Executor (Core Executor)

#### 3.1 Create new executor package

**New directory**: `core/pkg/service/arc/executor/`

This package implements a Go-based task executor for Arc tasks.

#### 3.2 Executor structure

**File**: `core/pkg/service/arc/executor/executor.go`

```go
package executor

type Executor struct {
    cfg      Config
    rack     rack.Rack          // The rack this executor owns
    runtimes map[task.Key]*runtime.Runtime
    mu       sync.RWMutex

    streamer framer.Streamer
    shutdown chan struct{}
}

type Config struct {
    HostKey  cluster.NodeKey    // For naming the rack
    Rack     *rack.Service      // For creating the rack
    Arcs     *arc.Service       // For retrieving/compiling programs
    Tasks    *task.Service      // For retrieving task configs
    Framer   *framer.Service    // For streaming task channels
    Channel  *channel.Service   // For channel metadata
    Status   *status.Service    // For reporting task status
    L        *zap.Logger
}

func Open(ctx context.Context, cfg Config) (*Executor, error) {
    e := &Executor{cfg: cfg}

    // Create a rack for this executor (just like C++ drivers do)
    e.rack = rack.Rack{
        Name: fmt.Sprintf("Node %s Core", cfg.HostKey),
    }
    if err := cfg.Rack.NewWriter(nil).Create(ctx, &e.rack); err != nil {
        return nil, err
    }

    return e, nil
}
```

#### 3.3 Task channel subscription

The executor subscribes to the same channels as C++ drivers:

```go
func (e *Executor) Run(ctx context.Context) error {
    // Get task channel keys
    taskSetCh, _ := e.cfg.Channel.RetrieveByName(ctx, "sy_task_set")
    taskDeleteCh, _ := e.cfg.Channel.RetrieveByName(ctx, "sy_task_delete")
    taskCmdCh, _ := e.cfg.Channel.RetrieveByName(ctx, "sy_task_cmd")

    // Open streamer
    streamer, _ := e.cfg.Framer.NewStreamer(ctx, framer.StreamerConfig{
        Keys: []channel.Key{taskSetCh.Key, taskDeleteCh.Key, taskCmdCh.Key},
    })

    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case res := <-streamer.Responses():
            e.processFrame(ctx, res.Frame)
        }
    }
}
```

#### 3.4 Task routing (filter by rack)

Only process tasks assigned to this executor's rack:

```go
func (e *Executor) processTaskSet(ctx context.Context, taskKeys []task.Key) {
    for _, key := range taskKeys {
        if key.Rack() != e.rack.Key {
            continue  // Not for us - different rack
        }
        e.configureTask(ctx, key)
    }
}

func (e *Executor) processTaskCmd(ctx context.Context, cmd task.Command) {
    if cmd.Task.Rack() != e.rack.Key {
        return  // Not for us
    }
    e.executeCommand(ctx, cmd)
}
```

#### 3.5 Task configuration

```go
func (e *Executor) configureTask(ctx context.Context, key task.Key) error {
    // Retrieve task metadata
    tsk, err := e.cfg.Tasks.Retrieve(ctx, key)
    if err != nil {
        return err
    }

    // Only handle "arc" type tasks
    if tsk.Type != "arc" {
        return nil
    }

    // Parse config
    var cfg ArcTaskConfig
    if err := json.Unmarshal([]byte(tsk.Config), &cfg); err != nil {
        return err
    }

    // Compile Arc program
    mod, err := e.cfg.Arcs.Compile(ctx, cfg.ArcKey)
    if err != nil {
        e.reportStatus(key, status.Error, "Compilation failed: "+err.Error())
        return err
    }

    // Create runtime (but don't start yet)
    rt, err := runtime.Open(ctx, runtime.Config{
        Module:  mod,
        Framer:  e.cfg.Framer,
        Channel: e.cfg.Channel,
    })
    if err != nil {
        return err
    }

    e.mu.Lock()
    e.runtimes[key] = rt
    e.mu.Unlock()

    // Auto-start if configured
    if cfg.AutoStart {
        return e.start(ctx, key, "")
    }

    return nil
}
```

#### 3.6 Command execution

```go
type ArcTaskConfig struct {
    ArcKey    uuid.UUID `json:"arc_key"`
    AutoStart bool      `json:"auto_start"`
}

func (e *Executor) executeCommand(ctx context.Context, cmd task.Command) error {
    e.mu.RLock()
    rt, ok := e.runtimes[cmd.Task]
    e.mu.RUnlock()

    if !ok {
        return errors.New("task not configured")
    }

    switch cmd.Type {
    case "start":
        return e.start(ctx, cmd.Task, cmd.Key)
    case "stop":
        return e.stop(ctx, cmd.Task, cmd.Key)
    default:
        return errors.Errorf("unknown command: %s", cmd.Type)
    }
}

func (e *Executor) start(ctx context.Context, key task.Key, cmdKey string) error {
    rt := e.runtimes[key]
    if err := rt.Start(ctx); err != nil {
        e.reportStatus(key, status.Error, err.Error())
        return err
    }
    e.reportStatus(key, status.Success, "Running", task.StatusDetails{
        Running: true,
        CmdKey:  cmdKey,
    })
    return nil
}

func (e *Executor) stop(ctx context.Context, key task.Key, cmdKey string) error {
    rt := e.runtimes[key]
    if err := rt.Stop(); err != nil {
        return err
    }
    e.reportStatus(key, status.Success, "Stopped", task.StatusDetails{
        Running: false,
        CmdKey:  cmdKey,
    })
    return nil
}
```

#### 3.7 Integrate executor into Arc service

**File**: `core/pkg/service/arc/service.go`

Start the executor when the Arc service opens:

```go
func OpenService(ctx context.Context, configs ...ServiceConfig) (*Service, error) {
    // ... existing setup ...

    // Start the Go executor (it creates its own rack)
    exec, err := executor.Open(ctx, executor.Config{
        HostKey: cfg.HostProvider.HostKey(),
        Rack:    cfg.Rack,
        Arcs:    s,
        Tasks:   cfg.Task,
        Framer:  cfg.Framer,
        Channel: cfg.Channel,
        Status:  cfg.Status,
        L:       cfg.L.Named("executor"),
    })
    if err != nil {
        return nil, err
    }

    s.executor = exec
    go func() {
        if err := exec.Run(ctx); err != nil {
            cfg.L.Error("executor stopped", zap.Error(err))
        }
    }()

    return s, nil
}
```

---

### Phase 4: C++ Driver Updates

The C++ Arc task implementation already follows the task-based model. Minor updates
may be needed:

#### 4.1 Ensure command parity

**File**: `driver/arc/task.h`

Verify the C++ task handles the same commands as the Go executor:
- `start` - Begin execution
- `stop` - Stop execution

#### 4.2 Config structure alignment

**File**: `driver/arc/task.cpp`

Ensure `ArcTaskConfig` parses the same JSON structure:

```cpp
struct ArcTaskConfig {
    std::string arc_key;    // UUID as string
    bool auto_start;
    bool data_saving;

    // C++ specific (ignored by Go executor)
    std::string execution_mode;  // "high_rate", "event_driven", etc.
    std::vector<int> cpu_affinity;
    int rt_priority;
};
```

---

### Phase 5: API Updates

#### 5.1 Task creation for Arc

The client creates an Arc task by:

1. Creating the Arc program (just stores it)
2. Creating a task with `type: "arc"` and appropriate rack assignment
3. Sending `start` command when ready

**File**: `core/pkg/api/task.go`

No changes needed - existing task API works. Example request:

```json
{
    "tasks": [{
        "key": "0x0001000100000001",  // Encodes Core Rack
        "name": "Motor Control",
        "type": "arc",
        "config": "{\"arc_key\": \"uuid-here\", \"auto_start\": false}"
    }]
}
```

#### 5.2 Add convenience endpoint (optional)

Consider adding a convenience endpoint that:
- Creates task on Core Rack by default
- Auto-assigns task key
- Optionally starts immediately

```go
// POST /api/v1/arcs/{key}/deploy
// Body: {"rack": "core" | "embedded" | rack_key, "auto_start": true}
```

---

### Phase 6: Console UI Updates

The Console needs UI changes to reflect the new model:

#### 6.1 Arc program view

- Remove "Deploy" toggle
- Show list of tasks running this program
- Add "Create Task" button

#### 6.2 Task creation dialog

When creating an Arc task:
- Select Arc program
- Select target rack (Core Rack = "Run on Server", others = "Run on Driver")
- Configure auto-start, data saving, etc.
- For driver racks: configure execution mode, CPU affinity, etc.

#### 6.3 Task control

- Start/Stop buttons send commands via task API
- Status reflects command acknowledgment

---

## Migration

### Database Migration

1. Remove `deploy` column from `arc` table (or keep as deprecated)

### Existing Arc Programs

For Arc programs with `Deploy = true`:
1. Create corresponding task on the Core Rack
2. Set `auto_start: true` in task config
3. Clear `Deploy` flag

This can be a one-time migration script or handled in the upgrade path.

---

## Testing

### Unit Tests

1. **Arc executor**: Test rack creation, task configuration, command handling, status reporting
2. **Task routing**: Test that only tasks for executor's rack are handled

### Integration Tests

1. Create Arc program, create task on Core Rack, start/stop via commands
2. Create Arc program, create task on Embedded Rack, verify C++ driver handles it
3. Same Arc program on multiple racks simultaneously
4. Task status propagation to clients

---

## File Summary

### New Files
- `core/pkg/service/arc/executor/executor.go` - Go Arc executor
- `core/pkg/service/arc/executor/config.go` - Executor configuration

### Modified Files
- `core/pkg/service/arc/arc.go` - Remove `Deploy` field
- `core/pkg/service/arc/service.go` - Remove auto-deploy, integrate executor
- `driver/arc/task.h` - Verify command parity
- `driver/arc/task.cpp` - Verify config structure

### Console Changes
- Arc program view - remove Deploy toggle
- Task creation - add rack selector for Arc tasks
- Task list - show Arc tasks with start/stop controls
