# Dynamic Task Management for EtherCAT CyclicEngine

## Executive Summary

The current EtherCAT CyclicEngine implementation has a fundamental limitation: **PDO
registration must occur before the engine starts, and new PDOs cannot be added while
running**. This prevents users from dynamically starting/stopping read and write tasks
on the same EtherCAT network.

This document explores the changes required to support dynamic task addition/removal
with automatic engine restart.

---

## Current Architecture

### Component Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                          Factory                                 │
│  ┌─────────────────────────────────────────────────────────┐    │
│  │  engines: map<interface_name, shared_ptr<CyclicEngine>> │    │
│  └─────────────────────────────────────────────────────────┘    │
│                              │                                   │
│            ┌─────────────────┴─────────────────┐                │
│            ▼                                   ▼                │
│    ┌───────────────┐                   ┌───────────────┐        │
│    │  ReadTask A   │                   │  WriteTask B  │        │
│    │  (Source)     │                   │  (Sink)       │        │
│    └───────┬───────┘                   └───────┬───────┘        │
│            │                                   │                │
│            └─────────────┬─────────────────────┘                │
│                          ▼                                      │
│              ┌─────────────────────┐                            │
│              │    CyclicEngine     │                            │
│              │  ┌───────────────┐  │                            │
│              │  │ input_pdos_   │  │                            │
│              │  │ output_pdos_  │  │                            │
│              │  │ task_count_   │  │                            │
│              │  └───────────────┘  │                            │
│              └──────────┬──────────┘                            │
│                         ▼                                       │
│              ┌─────────────────────┐                            │
│              │       Master        │                            │
│              │  (SOEM or IgH)      │                            │
│              └─────────────────────┘                            │
└─────────────────────────────────────────────────────────────────┘
```

### Current Task Startup Flow

```cpp
// ReadTaskSource::start()
xerrors::Error start() override {
    // 1. Resolve slave positions from serial numbers
    for (auto &ch : config_.channels) {
        ch->slave_position = serial_to_position[ch->slave_serial];

        // 2. Register PDOs with engine (FAILS if running_)
        auto [reg_index, err] = engine_->register_input_pdo(ch->to_pdo_entry(true));
        if (err) return err;  // ← Error: "cannot register PDO while running"
    }

    // 3. Add task (starts engine on first task)
    if (auto err = engine_->add_task(); err) return err;

    // 4. Resolve actual buffer offsets
    for (size_t i = 0; i < config_.channels.size(); ++i)
        config_.channels[i]->buffer_offset = engine_->get_actual_input_offset(i);

    return xerrors::NIL;
}
```

### The Problem

1. Task A starts → registers PDOs → calls `add_task()` → engine starts
   (`running_ = true`)
2. Task B starts → tries to register PDOs → **FAILS** because `running_ == true`

---

## Proposed Solution: Automatic Engine Restart

### High-Level Design

When a new task needs to register PDOs while the engine is running:

1. Signal all running tasks that a reconfiguration is pending
2. Stop the cycle thread
3. Deactivate the master (transitions slaves to INIT)
4. Register the new PDOs
5. Reactivate the master (transitions slaves back to OP)
6. Restart the cycle thread
7. Notify tasks to resume with updated offsets

### Key Challenges

#### 1. **Offset Stability**

When the IOmap is reconfigured, PDO offsets may change. Existing tasks have cached
`buffer_offset` values that become invalid.

**Options:**

- A) Tasks re-query offsets after restart via callback
- B) Engine maintains offset mapping and translates on behalf of tasks
- C) Offsets are stable by design (maintain registration order)

#### 2. **Data Continuity During Restart**

During reconfiguration (typically 100-500ms), no cyclic exchange occurs.

**Options:**

- A) Accept data gap - tasks see timeout waiting for inputs
- B) Hold last values - maintain output state across restart
- C) Signal tasks with special "reconfiguring" state

#### 3. **Thread Synchronization**

Multiple tasks may be blocked in `wait_for_inputs()` when restart begins.

**Requirements:**

- Wake all waiting tasks before stopping cycle thread
- Prevent new `wait_for_inputs()` calls during restart
- Coordinate PDO registration during restart window

#### 4. **Output State Preservation**

Write tasks may have pending output values that should persist across restart.

**Options:**

- A) Preserve `output_buffer_` across restart
- B) Tasks re-apply state after restart
- C) Zero outputs on restart (safe but disruptive)

#### 5. **Task Removal and PDO Deregistration**

When a task stops, should its PDOs be removed from the IOmap?

**Options:**

- A) Never remove PDOs - IOmap only grows (simplest)
- B) Remove PDOs and restart if other tasks still running
- C) Mark PDOs as inactive but keep offsets stable

#### 6. **Error Handling During Restart**

What if reactivation fails (e.g., slave disconnected)?

**Options:**

- A) Propagate error to all tasks, stop everything
- B) Retry with exponential backoff
- C) Continue without the problematic slave

---

## Detailed Design Options

### Option A: Blocking Restart (Simplest)

```cpp
std::pair<size_t, xerrors::Error> CyclicEngine::register_input_pdo(const PDOEntry &entry) {
    std::lock_guard lock(registration_mutex_);

    if (running_) {
        // Stop, reconfigure, restart
        auto err = restart_for_reconfiguration();
        if (err) return {0, err};
    }

    // Normal registration...
    input_pdos_.push_back({entry, relative_offset, 0});
    return {registration_index, xerrors::NIL};
}

xerrors::Error CyclicEngine::restart_for_reconfiguration() {
    // 1. Signal restart pending
    restarting_ = true;

    // 2. Wake all waiting tasks
    input_cv_.notify_all();

    // 3. Stop cycle thread
    running_ = false;
    if (cycle_thread_.joinable()) cycle_thread_.join();

    // 4. Deactivate master (slaves → INIT)
    master_->deactivate();

    // 5. Reinitialize master (rescan slaves)
    if (auto err = master_->initialize(); err) return err;

    // 6. Reactivate master (slaves → OP)
    if (auto err = master_->activate(); err) return err;

    // 7. Resolve new offsets
    resolve_pdo_offsets();

    // 8. Resize buffers
    auto *active = master_->active_domain();
    input_snapshot_.resize(active->input_size(), 0);
    output_buffer_.resize(active->output_size(), 0);

    // 9. Restart cycle thread
    running_ = true;
    restarting_ = false;
    cycle_thread_ = std::thread(&CyclicEngine::cycle_loop, this);

    return xerrors::NIL;
}
```

**Pros:**

- Simple implementation
- Clear semantics

**Cons:**

- All tasks blocked during restart
- Existing tasks may have stale offsets

### Option B: Callback-Based Notification

Tasks register callbacks to be notified of restarts:

```cpp
class CyclicEngine {
    // ... existing members ...

    using RestartCallback = std::function<void(RestartPhase phase)>;
    std::vector<std::pair<void*, RestartCallback>> restart_callbacks_;

public:
    void register_restart_callback(void* owner, RestartCallback cb);
    void unregister_restart_callback(void* owner);
};

// In task:
engine_->register_restart_callback(this, [this](RestartPhase phase) {
    if (phase == RestartPhase::BEFORE_RESTART) {
        // Prepare for restart
    } else if (phase == RestartPhase::AFTER_RESTART) {
        // Re-resolve offsets
        for (size_t i = 0; i < registration_indices_.size(); ++i)
            config_.channels[i]->buffer_offset =
                engine_->get_actual_input_offset(registration_indices_[i]);
    }
});
```

**Pros:**

- Tasks can react to restarts
- Offsets always up-to-date

**Cons:**

- More complex API
- Tasks must handle restart lifecycle

### Option C: Stable Offset Guarantees

Design the system so offsets never change:

```cpp
struct PDORegistration {
    PDOEntry entry;
    size_t registration_index;  // Stable across restarts
    size_t actual_offset;       // May change on restart
    bool active;                // False if task stopped
};
```

Engine maintains stable registration indices. Tasks always use registration index,
engine translates to actual offset internally.

**Pros:**

- Tasks don't need to know about restarts
- Simplest task-side API

**Cons:**

- Extra indirection
- Complexity in engine

---

## Interface Changes Required

### CyclicEngine Changes

```cpp
class CyclicEngine {
public:
    // NEW: Registration returns stable handle, not offset
    struct PDOHandle {
        size_t index;
        bool valid;
    };

    [[nodiscard]] std::pair<PDOHandle, xerrors::Error> register_input_pdo(
        const PDOEntry &entry
    );

    [[nodiscard]] std::pair<PDOHandle, xerrors::Error> register_output_pdo(
        const PDOEntry &entry
    );

    // NEW: Deregister PDOs when task stops
    void unregister_pdo(PDOHandle handle);

    // NEW: Restart notification
    enum class RestartPhase { BEFORE, AFTER };
    using RestartCallback = std::function<void(RestartPhase)>;
    void on_restart(void* owner, RestartCallback cb);
    void off_restart(void* owner);

    // CHANGED: Read via handle instead of raw offset
    [[nodiscard]] xerrors::Error read_input(
        PDOHandle handle,
        void* buffer,
        size_t length
    );

    // CHANGED: Write via handle instead of raw offset
    void write_output(PDOHandle handle, const void* data, size_t length);

private:
    // NEW: State for coordinated restart
    std::atomic<bool> restarting_;
    std::condition_variable restart_cv_;

    // NEW: Internal restart coordination
    xerrors::Error restart_for_reconfiguration();
    void notify_restart(RestartPhase phase);
};
```

### Master Interface Changes

```cpp
class Master {
public:
    // POTENTIAL NEW: Reset without full close/reopen
    [[nodiscard]] virtual xerrors::Error reset() = 0;

    // POTENTIAL NEW: Check if reinitialization needed
    [[nodiscard]] virtual bool needs_reinitialize() const = 0;
};
```

### Task Changes

```cpp
class ReadTaskSource : public common::Source {
    std::vector<CyclicEngine::PDOHandle> pdo_handles_;  // NEW: Store handles

    xerrors::Error start() override {
        // Register restart callback
        engine_->on_restart(this, [this](auto phase) {
            handle_restart(phase);
        });

        // Register PDOs (may trigger restart if engine running)
        for (auto &ch : config_.channels) {
            auto [handle, err] = engine_->register_input_pdo(ch->to_pdo_entry(true));
            if (err) return err;
            pdo_handles_.push_back(handle);
        }

        // ...
    }

    xerrors::Error stop() override {
        engine_->off_restart(this);
        for (auto handle : pdo_handles_)
            engine_->unregister_pdo(handle);
        engine_->remove_task();
        return xerrors::NIL;
    }

    void handle_restart(CyclicEngine::RestartPhase phase) {
        // Handle restart notification
    }
};
```

---

## State Transitions

### Engine States

```
┌──────────────────────────────────────────────────────────────────┐
│                                                                  │
│  ┌─────────┐    add_task()     ┌─────────┐                      │
│  │  IDLE   │ ─────────────────►│ RUNNING │◄─────────────┐       │
│  │         │                   │         │              │       │
│  └─────────┘                   └────┬────┘              │       │
│       ▲                             │                   │       │
│       │                     register_pdo()              │       │
│       │                     (while running)             │       │
│       │                             │                   │       │
│       │                             ▼                   │       │
│       │                    ┌─────────────────┐          │       │
│       │                    │  RESTARTING     │          │       │
│       │                    │  (stop→reconf   │──────────┘       │
│       │                    │   →restart)     │                  │
│       │                    └─────────────────┘                  │
│       │                                                         │
│       │    remove_task()                                        │
│       │    (last task)                                          │
│       └─────────────────────────────────────────────────────────┘
```

### Task States During Restart

```
┌────────────────┐
│   RUNNING      │ ◄── Normal operation
└───────┬────────┘
        │ Engine signals restart
        ▼
┌────────────────┐
│   PAUSED       │ ◄── wait_for_inputs() returns special error
└───────┬────────┘
        │ Engine completes restart
        ▼
┌────────────────┐
│   RESUMING     │ ◄── Task re-resolves offsets
└───────┬────────┘
        │
        ▼
┌────────────────┐
│   RUNNING      │ ◄── Normal operation resumes
└────────────────┘
```

---

## Timing Considerations

### Restart Duration Breakdown

| Phase              | Typical Duration | Notes                     |
| ------------------ | ---------------- | ------------------------- |
| Stop cycle thread  | 1-2ms            | Wait for current cycle    |
| Deactivate master  | 10-50ms          | Transition slaves to INIT |
| Reinitialize       | 50-200ms         | Rescan bus                |
| Reactivate         | 100-500ms        | Transition slaves to OP   |
| Start cycle thread | <1ms             | Thread creation           |
| **Total**          | **160-750ms**    | Data gap during restart   |

### Impact on Real-Time Operation

- **1kHz cycle rate**: 160-750 missed cycles
- **Output behavior**: Slaves hold last output during restart (hardware-dependent)
- **Input behavior**: No new data until restart complete

---

## Questions for Clarification

### 1. Offset Stability Strategy

Should we:

- A) Require tasks to re-resolve offsets after restart (callback-based)?
- B) Guarantee stable offsets via registration handles (indirection)?
- C) Accept that offsets may change and tasks must adapt?

### 2. Data Gap Handling

During the ~200-500ms restart window:

- A) Should tasks receive a special "reconfiguring" error from `wait_for_inputs()`?
- B) Should the engine buffer the last good data and return that?
- C) Should tasks simply timeout and retry?

### 3. Output Preservation

When restarting:

- A) Should output values be preserved (maintain `output_buffer_`)?
- B) Should outputs be zeroed (safer but disruptive)?
- C) Should tasks be required to re-send outputs after restart?

### 4. PDO Deregistration on Task Stop

When a task stops while others are running:

- A) Keep its PDOs registered (simplest, wastes IOmap space)?
- B) Deregister and restart (consistent, but disrupts other tasks)?
- C) Mark as inactive, compact on next restart?

### 5. Error Recovery During Restart

If master reactivation fails:

- A) Propagate error to all tasks and stop?
- B) Retry with backoff?
- C) Continue without failed slaves?

### 6. Concurrent Restart Requests

If two tasks try to register PDOs simultaneously:

- A) Serialize with mutex (second waits for first restart to complete)?
- B) Batch registrations and do single restart?
- C) Queue registrations and restart once?

### 7. Minimum Restart Interval

Should there be a minimum time between restarts?

- A) Yes, with configurable cooldown (e.g., 5 seconds)?
- B) No, restart as often as needed?
- C) Batch pending registrations with short delay (e.g., 100ms)?

### 8. Restart Notification API

How should tasks be notified of restarts?

- A) Callback-based (`on_restart(callback)`)?
- B) Error-based (`wait_for_inputs()` returns `RECONFIGURING_ERROR`)?
- C) Polling-based (`engine->is_restarting()`)?

### 9. IgH-Specific Considerations

Does IgH master handle restart differently than SOEM?

- A) Same deactivate/reactivate pattern works?
- B) IgH has specific restart API?
- C) Need to test on actual hardware?

### 10. Testing Strategy

How should we test this?

- A) MockMaster with simulated restart timing?
- B) Integration tests with real hardware?
- C) Both, with specific test scenarios for each edge case?

---

## Design Decisions

Based on discussion and analysis, the following decisions were made:

| #   | Question                 | Decision                             | Rationale                                                                                                                |
| --- | ------------------------ | ------------------------------------ | ------------------------------------------------------------------------------------------------------------------------ |
| 1   | Offset Stability         | **Handle indirection**               | Tasks use stable `PDOHandle`, engine translates to actual offset internally. Simplest task-side API.                     |
| 2   | Data Gap Handling        | **Return `ENGINE_RESTARTING` error** | `wait_for_inputs()` returns special error during restart. Tasks retry until restart completes.                           |
| 3   | Output Preservation      | **Preserve `output_buffer_`**        | Maintain output state across restart. Hardware holds last value anyway; this ensures consistency when cyclic resumes.    |
| 4   | PDO Deregistration       | **Keep PDOs registered**             | When task stops, PDOs remain in IOmap. Simplest approach; revisit if IOmap grows too large.                              |
| 5   | Error Recovery           | **Retry with breaker**               | Use existing `breaker` pattern for exponential backoff on restart failures. Consistent with other driver error handling. |
| 6   | Concurrent Registrations | **Serialize with mutex**             | Second registration waits for first restart to complete. Simplest approach, acceptable given restart is infrequent.      |
| 7   | Restart Cooldown         | **No cooldown**                      | Allow restarts as needed. Batching adds complexity for minimal benefit.                                                  |
| 8   | Restart Notification     | **Error-only**                       | No callbacks. Tasks handle `ENGINE_RESTARTING` error from `wait_for_inputs()`. Simpler API.                              |
| 9   | IgH vs SOEM              | **Same abstract pattern**            | Both masters use `deactivate() → re-register → activate()`. Interface abstraction handles differences internally.        |
| 10  | Testing                  | **MockMaster with simulated timing** | Fast, deterministic unit tests. Hardware tests can be added later for validation.                                        |

---

## Final Design

### CyclicEngine Interface Changes

```cpp
class CyclicEngine {
public:
    /// Stable handle for PDO access (index never changes across restarts)
    struct PDOHandle {
        size_t index;
        bool valid() const { return index != SIZE_MAX; }
    };

    /// Register input PDO. May trigger engine restart if already running.
    /// Returns stable handle that remains valid across restarts.
    [[nodiscard]] std::pair<PDOHandle, xerrors::Error> register_input_pdo(
        const PDOEntry &entry
    );

    /// Register output PDO. May trigger engine restart if already running.
    [[nodiscard]] std::pair<PDOHandle, xerrors::Error> register_output_pdo(
        const PDOEntry &entry
    );

    /// Read input data via handle (translates to actual offset internally)
    [[nodiscard]] xerrors::Error read_input(
        PDOHandle handle,
        void* buffer,
        size_t length
    );

    /// Write output data via handle (translates to actual offset internally)
    void write_output(PDOHandle handle, const void* data, size_t length);

    /// Wait for inputs. Returns ENGINE_RESTARTING error during reconfiguration.
    [[nodiscard]] xerrors::Error wait_for_inputs(
        std::vector<uint8_t> &buffer,
        std::atomic<bool> &breaker
    );

private:
    std::atomic<bool> restarting_{false};
    breaker::Breaker restart_breaker_;  // For retry with backoff

    /// Internal restart when PDO registered while running
    xerrors::Error restart_for_reconfiguration();
};
```

### Restart Flow

```cpp
xerrors::Error CyclicEngine::restart_for_reconfiguration() {
    // 1. Signal restart pending
    restarting_ = true;

    // 2. Wake all waiting tasks (they'll get ENGINE_RESTARTING error)
    input_cv_.notify_all();

    // 3. Stop cycle thread
    running_ = false;
    if (cycle_thread_.joinable()) cycle_thread_.join();

    // 4. Deactivate master
    master_->deactivate();

    // 5. Retry activation with breaker
    while (!restart_breaker_.stopped()) {
        if (auto err = master_->initialize(); err) {
            if (!restart_breaker_.wait(err))
                return err;  // Max retries exceeded
            continue;
        }

        if (auto err = master_->activate(); err) {
            master_->deactivate();
            if (!restart_breaker_.wait(err))
                return err;
            continue;
        }

        break;  // Success
    }

    // 6. Resolve new offsets (handles remain stable)
    resolve_pdo_offsets();

    // 7. Resize buffers (preserve output_buffer_ contents)
    auto *active = master_->active_domain();
    input_snapshot_.resize(active->input_size(), 0);

    std::vector<uint8_t> old_output = std::move(output_buffer_);
    output_buffer_.resize(active->output_size(), 0);
    std::memcpy(output_buffer_.data(), old_output.data(),
                std::min(old_output.size(), output_buffer_.size()));

    // 8. Restart cycle thread
    running_ = true;
    restarting_ = false;
    cycle_thread_ = std::thread(&CyclicEngine::cycle_loop, this);

    return xerrors::NIL;
}
```

### Task Usage Pattern

```cpp
xerrors::Error ReadTaskSource::start() {
    for (auto &ch : config_.channels) {
        // Registration may trigger restart if engine already running
        auto [handle, err] = engine_->register_input_pdo(ch->to_pdo_entry(true));
        if (err) return err;
        pdo_handles_.push_back(handle);
    }

    return engine_->add_task();
}

std::pair<Frame, xerrors::Error> ReadTaskSource::read(breaker::Breaker &breaker) {
    while (!breaker.stopped()) {
        auto err = engine_->wait_for_inputs(buffer_, breaker.stopped_);

        if (err.matches(ENGINE_RESTARTING)) {
            // Engine is reconfiguring, retry
            std::this_thread::sleep_for(std::chrono::milliseconds(10));
            continue;
        }

        if (err) return {{}, err};

        // Read via handles (offset translation is internal)
        for (size_t i = 0; i < pdo_handles_.size(); ++i) {
            engine_->read_input(pdo_handles_[i], &values[i], sizeof(values[i]));
        }

        return {build_frame(values), xerrors::NIL};
    }
    return {{}, breaker::STOPPED};
}
```

---

## Implementation Phases

### Phase 1: Basic Restart Capability

- Add `restarting_` flag and `ENGINE_RESTARTING` error
- Implement `restart_for_reconfiguration()` with breaker retry
- Update `wait_for_inputs()` to return error during restart
- Update `register_*_pdo()` to trigger restart when running

### Phase 2: Handle-Based Access

- Add `PDOHandle` struct
- Implement `read_input(handle, ...)` and `write_output(handle, ...)`
- Update tasks to use handles instead of raw offsets
- Ensure handles remain stable across restarts

### Phase 3: Output Preservation

- Preserve `output_buffer_` contents across restart
- Handle buffer size changes (copy what fits)

### Phase 4: Testing

- Add MockMaster restart simulation
- Unit tests for:
  - Single task start/stop
  - Second task starting while first running
  - Restart failure and recovery
  - Concurrent registration attempts
  - Output preservation across restart

---

## Next Steps

1. Implement Phase 1 (basic restart)
2. Write unit tests with MockMaster
3. Implement Phase 2 (handle-based access)
4. Update ReadTask and WriteTask to use new API
5. Test on actual EtherCAT hardware
6. Document user-facing behavior changes
