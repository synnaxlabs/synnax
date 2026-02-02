# Sift Integration Plan

## Goal

Upload historical Synnax data (from ranges) to Sift for analysis and visualization.

## Sift API Overview

Based on [Sift Go SDK](https://pkg.go.dev/github.com/sift-stack/sift/go) and
[Sift Docs](https://docs.siftstack.com):

### Core Concepts

| Concept | Description |
|---------|-------------|
| **Asset** | Top-level container (e.g., "rocket-1", "test-stand-3") |
| **Ingestion Config** | Schema definition with flows and channels, identified by `client_key` |
| **Flow** | Named group of channels sampled together |
| **Channel** | Single data stream with name and data type |
| **Run** | Time-bounded session grouping data |

### Synnax → Sift Mapping

| Synnax | Sift | Notes |
|--------|------|-------|
| Range | Run | Range name → Run name |
| Channel | Channel | Channel name → Channel name |
| (user-specified) | Flow | Specified in upload config |
| (user-specified) | Asset | Specified in upload config |

### Ingestion Flow

```
1. Connect to Sift (URI + API key)
2. Create/retrieve IngestionConfig (asset + client_key + flows)
3. Create Run (from range name)
4. Open IngestWithConfigDataStream
5. Send timestamped channel values
6. Close stream
```

---

## Decided Architecture

### Device Configuration (Minimal)

Device properties store only connection info:

```go
type DeviceProperties struct {
    URI    string `json:"uri"`     // e.g., "api.siftstack.com:443"
    APIKey string `json:"api_key"` // Sift API key
}
```

### Task Configuration (Per-Upload)

Each upload specifies all details:

```go
type TaskConfig struct {
    DeviceKey string          `json:"device_key"` // Reference to Sift device
    AssetName string          `json:"asset_name"` // Sift asset name
    FlowName  string          `json:"flow_name"`  // Sift flow name
    RunName   string          `json:"run_name"`   // Sift run name
    Channels  []channel.Key   `json:"channels"`   // Channels to upload
    TimeRange telem.TimeRange `json:"time_range"` // Time range to upload
}
```

### Task Lifecycle

- **Auto-start:** Upload begins immediately when task is created
- **No commands:** Task accepts no commands via Exec
- **Cancellation:** User deletes the task → calls `Stop()` → cancels upload
- **Ephemeral:** Task deletes itself on successful completion

### Ingestion Config Client Key

Generate client key by hashing the task key. This ensures each upload gets a fresh
ingestion config while remaining deterministic.

### Timestamp Handling

Use actual index channel values for timestamps, not interpolated values from
`series.TimeRange`.

---

## File Structure

```
core/pkg/service/sift/
├── sift.go           # Package docs, constants
├── config.go         # DeviceProperties, TaskConfig
├── client.go         # Sift gRPC client wrapper + pool
├── upload.go         # Upload task implementation
├── mapping.go        # Data type conversions
├── factory.go        # Task factory
└── sift_test.go      # Tests (with mocked Sift client)
```

---

## Implementation Details

### Client Interface

```go
type Client interface {
    GetOrCreateIngestionConfig(ctx context.Context, asset string, flows []FlowConfig) (*IngestionConfig, error)
    CreateRun(ctx context.Context, name string) (*Run, error)
    OpenIngestStream(ctx context.Context, configID string) (*IngestStream, error)
    Close() error
}
```

### Upload Task Flow

```
1. Factory.ConfigureTask called with task config
2. Parse TaskConfig, retrieve device properties
3. Create uploadTask, immediately start upload goroutine
4. Upload goroutine:
   a. Set status "Starting upload"
   b. Retrieve channel metadata
   c. Get/create Sift client from pool
   d. Build FlowConfig from channels (name + data type)
   e. GetOrCreateIngestionConfig on Sift (client_key = hash of task key)
   f. CreateRun using config.RunName
   g. OpenIngestStream
   h. Open Synnax iterator for config.TimeRange
   i. For each frame:
      - Read index channel for actual timestamps
      - Convert data to Sift channel values
      - Send via stream
      - Update progress status
   j. Close stream
   k. Set status "Completed" or "Failed"
   l. Delete task
```

### Task Interface

```go
func (u *uploadTask) Exec(ctx context.Context, cmd task.Command) error {
    return driver.ErrUnsupportedCommand // No commands accepted
}

func (u *uploadTask) Stop() error {
    u.cancel() // Cancel context to stop upload goroutine
    return nil
}
```

---

## Testing Strategy (TDD)

**Development approach:** Test-Driven Development. Write tests first with a mock Sift
backend, then implement the production code.

**Coverage requirement:** Always run tests with `ginkgo --cover` and verify all code
paths are covered.

### Mock Sift Backend

Define interfaces that can be mocked:

```go
// Client is the interface for Sift operations (mockable)
type Client interface {
    GetOrCreateIngestionConfig(ctx context.Context, req CreateIngestionConfigRequest) (*IngestionConfig, error)
    CreateRun(ctx context.Context, name string) (*Run, error)
    OpenIngestStream(ctx context.Context, configID string) (IngestStream, error)
    Close() error
}

// IngestStream is the interface for streaming data (mockable)
type IngestStream interface {
    Send(flow string, ts telem.TimeStamp, values []ChannelValue) error
    Close() error
}

// MockClient for testing
type MockClient struct {
    IngestionConfig *IngestionConfig
    Run             *Run
    Stream          *MockIngestStream
    Err             error
}

// MockIngestStream captures sent data for verification
type MockIngestStream struct {
    Requests []IngestRequest
    Closed   bool
    Err      error
}
```

### Test Cases

1. **Successful upload** - verify all data sent correctly
2. **Stop mid-upload** - verify Stop() cancels cleanly
3. **Connection failure** - verify error handling
4. **Invalid data type** - verify graceful handling
5. **Empty time range** - verify no-op behavior
6. **Data type mapping** - verify all Synnax types map correctly

### Integration Tests (Later)

- Requires real Sift credentials (SIFT_API_KEY env var)
- Test full upload to actual Sift instance
- Verify data appears in Sift UI

---

## Implementation Steps (TDD)

### Phase 1: Core Upload (MVP)

1. [ ] Define interfaces (Client, IngestStream) in client.go
2. [ ] Create mock implementations for testing
3. [ ] Write tests for data type mapping
4. [ ] Implement mapping.go - pass tests
5. [ ] Write tests for upload task (success, cancel, errors)
6. [ ] Implement config.go - DeviceProperties, TaskConfig
7. [ ] Implement upload.go - pass tests
8. [ ] Implement factory.go
9. [ ] Implement real Sift client (wraps gRPC)
10. [ ] Update layer.go integration
11. [ ] Manual testing with real Sift

### Phase 2: Polish

1. [ ] Progress reporting with percentage
2. [ ] Batching for performance
3. [ ] Better error messages
4. [ ] Integration tests with real Sift

### Phase 3: Console UI

1. [ ] Device configuration form
2. [ ] Upload trigger UI
3. [ ] Progress display

---

## Open Questions

1. **Channel name validation:** Should we validate/sanitize channel names before sending
   to Sift, or let Sift reject invalid names?

2. **Concurrent uploads:** Allow multiple uploads to same asset simultaneously?

---

## References

- [Sift Go SDK](https://pkg.go.dev/github.com/sift-stack/sift/go)
- [Sift GitHub Examples](https://github.com/sift-stack/sift/tree/main/go/examples/ingestion)
