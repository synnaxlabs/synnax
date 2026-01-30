# EtherCAT Integration Research

> **Status**: Research Phase **Last Updated**: 2026-01-27 **Target**: Multi-backend
> EtherCAT support (IgH + SOEM)

## Executive Summary

Adding EtherCAT support to Synnax using a **multi-backend architecture**:

- **IgH EtherCAT Master** - Best real-time performance, Linux only, kernel module
- **SOEM** - Cross-platform (Linux/Windows/macOS), userspace, good performance

Key challenges:

1. **IgH kernel dependency** - Requires separate installation via DKMS
2. **Real-time constraints** - PREEMPT_RT recommended for IgH
3. **Abstraction layer** - Common interface over two different APIs
4. **Distribution** - Apt package for simplified IgH installation

## Multi-Backend Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                     Synnax Driver Layer                             │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌────────────┐ │
│  │ ReadTask    │  │ WriteTask   │  │ ScanTask    │  │ Factory    │ │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘  └─────┬──────┘ │
│         └────────────────┴────────────────┴────────────────┘        │
│                                   │                                  │
│                                   ▼                                  │
│  ┌──────────────────────────────────────────────────────────────┐   │
│  │              EtherCAT Interface (Abstract)                    │   │
│  │  class Master / class Domain / class SlaveConfig             │   │
│  └──────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
                                   │
                ┌──────────────────┼──────────────────┐
                │                  │                  │
                ▼                  ▼                  ▼
┌───────────────────────┐ ┌───────────────────────┐ ┌───────────────────┐
│   IgH Backend         │ │   SOEM Backend        │ │   Mock Backend    │
│   (Linux only)        │ │   (Cross-platform)    │ │   (Testing)       │
│                       │ │                       │ │                   │
│ - Best RT performance │ │ - Windows/macOS/Linux │ │ - Unit tests      │
│ - Kernel module req.  │ │ - Userspace only      │ │ - No hardware     │
│ - ecrt.h API          │ │ - Vendored in repo    │ │                   │
└───────────────────────┘ └───────────────────────┘ └───────────────────┘
```

### Backend Selection

```cpp
// Automatic backend selection
std::unique_ptr<Master> create_master(uint32_t index, const std::string& backend) {
    if (backend == "auto") {
#ifdef __linux__
        if (igh_available()) return make_igh_master(index);
#endif
        return make_soem_master(index);
    }
    // ... explicit selection
}
```

### Trade-offs

| Aspect       | IgH                 | SOEM                  |
| ------------ | ------------------- | --------------------- |
| Platform     | Linux only          | Linux, Windows, macOS |
| Performance  | Best (kernel space) | Good (userspace)      |
| Installation | Complex (DKMS)      | Simple (vendored)     |
| Real-time    | Excellent           | Good with RT patches  |
| DC Support   | Full                | Partial               |
| Maintenance  | User installs       | We vendor             |

## LOC Estimates

Based on existing integrations (with abstraction layer):

| Component                   | Estimated LOC   | Notes                              |
| --------------------------- | --------------- | ---------------------------------- |
| Abstract Interfaces         | 200-300         | master.h, domain.h, slave_config.h |
| CyclicEngine                | 300-400         | Cycle coordination, thread, sync   |
| Read/Write Tasks            | 400-500         | Source/Sink with PDO mapping       |
| IgH Backend                 | 400-500         | ecrt.h wrapper implementation      |
| SOEM Backend                | 400-500         | SOEM wrapper implementation        |
| Factory + Backend Selection | 150-200         | Task routing + create_master()     |
| Channel Types               | 150-200         | PDO entry mappings                 |
| Error Handling              | 200-300         | EtherCAT-specific error codes      |
| Mock Backend                | 150-200         | For testing                        |
| **Subtotal**                | **2,350-3,100** | Core implementation                |
| Tests                       | 1,000-1,400     | ~40-50% of implementation          |
| **Total**                   | **3,350-4,500** | Complete integration               |

For comparison:

- **Modbus**: ~3,868 LOC (simpler protocol, but similar architecture)
- **OPC UA**: ~7,226 LOC (more complex type system and pooling)

## IgH EtherCAT API Overview

The IgH EtherCAT Master API is C-based with these core structures:

```cpp
#include "ecrt.h"

ec_master_t *master;           // Master handle
ec_domain_t *domain;           // Process data domain
ec_slave_config_t *sc;         // Slave configuration
ec_pdo_entry_reg_t *pdo_regs;  // PDO entry registrations
```

### Typical Workflow

```cpp
// 1. Request master
master = ecrt_request_master(0);

// 2. Create domain for process data
domain = ecrt_master_create_domain(master);

// 3. Configure slaves
sc = ecrt_master_slave_config(master, alias, position, vendor_id, product_code);

// 4. Register PDO entries
ecrt_slave_config_reg_pdo_entry(sc, index, subindex, domain, &offset);

// 5. Activate master (transitions to operational)
ecrt_master_activate(master);

// 6. Get domain process data pointer
uint8_t *domain_pd = ecrt_domain_data(domain);

// 7. Cyclic task (in real-time context)
while (running) {
    ecrt_master_receive(master);
    ecrt_domain_process(domain);

    // Read inputs
    uint16_t value = EC_READ_U16(domain_pd + offset);

    // Write outputs
    EC_WRITE_U16(domain_pd + output_offset, new_value);

    ecrt_domain_queue(domain);
    ecrt_master_send(master);
}
```

### Key API Functions

| Function                            | Purpose                         |
| ----------------------------------- | ------------------------------- |
| `ecrt_request_master()`             | Request master by index         |
| `ecrt_master_create_domain()`       | Create process data domain      |
| `ecrt_master_slave_config()`        | Configure a slave device        |
| `ecrt_slave_config_reg_pdo_entry()` | Register PDO entry in domain    |
| `ecrt_master_activate()`            | Transition to operational state |
| `ecrt_domain_data()`                | Get pointer to process data     |
| `ecrt_master_receive()`             | Receive datagrams               |
| `ecrt_domain_process()`             | Process received data           |
| `ecrt_domain_queue()`               | Queue domain for sending        |
| `ecrt_master_send()`                | Send queued datagrams           |
| `ecrt_domain_state()`               | Check domain/process data state |

## Challenges & Issues

### 1. Kernel Module Dependency

IgH EtherCAT runs as a **Linux kernel module** (`ec_master.ko`), not a userspace
library.

**Issues:**

- Must match kernel version exactly
- Cross-compilation challenges
- Requires kernel headers at build time
- Module loading requires root privileges

**Impact on Synnax:**

- Cannot bundle as a simple shared library
- User must install kernel module separately
- Driver binary links against userspace library (`libethercat.so`)

### 2. Real-Time Kernel Requirement

For reliable EtherCAT timing, requires PREEMPT_RT patched kernel.

**Issues:**

- Standard Linux kernels have poor real-time performance
- Need to isolate CPU cores for EtherCAT task
- `ksoftirqd` priority conflicts
- 1ms cycle times require careful tuning

**Mitigations:**

```bash
# Isolate CPU core
isolcpus=1

# Set real-time priority for softirqs
chrt -f -p 10 $(pgrep ksoftirqd/1)
```

### 3. NIC Driver Compatibility

IgH supports specific NIC drivers:

| Driver            | Status           |
| ----------------- | ---------------- |
| `e1000e` (Intel)  | Native support   |
| `r8169` (Realtek) | Native support   |
| `igb` (Intel)     | Native support   |
| `generic`         | Works but slower |

**Issues:**

- Not all NICs supported
- Newer kernels may lack native driver support
- Generic driver adds latency

### 4. Platform Limitation

**Linux only** - no Windows or macOS support.

This requires Bazel `select()` exclusion:

```python
cc_library(
    name = "ethercat",
    srcs = select({
        "@platforms//os:linux": glob(["*.cpp"]),
        "//conditions:default": [],  # Excluded on Windows/macOS
    }),
)
```

### 5. Single Master Per NIC

Each network interface can only have **one master**. Unlike Modbus/OPC UA where multiple
connections are possible, EtherCAT is a dedicated network.

**Impact:**

- No connection pooling needed
- Must prevent multiple tasks from conflicting
- Need exclusive master access pattern

## Build System Considerations

### Bazel Integration

IgH provides `libethercat.so` (userspace) after kernel module install.

**Recommended: System Library Approach**

```python
cc_library(
    name = "ethercat",
    srcs = ["ethercat.cpp"],
    hdrs = ["ethercat.h"],
    linkopts = select({
        "@platforms//os:linux": ["-lethercat"],
        "//conditions:default": [],
    }),
    deps = select({
        "@platforms//os:linux": [":ethercat_impl"],
        "//conditions:default": [],
    }),
)
```

### User Installation Requirements

Users would need to:

1. Install PREEMPT_RT kernel (optional but recommended)
2. Build and install IgH EtherCAT kernel module
3. Load modules: `modprobe ec_master`, `modprobe ec_generic`
4. Configure network interface for EtherCAT
5. Run Synnax driver with appropriate permissions

## Configuration Design

### Device Configuration (stored in Synnax cluster)

```json
{
  "connection": {
    "master_index": 0,
    "cycle_time_us": 1000
  },
  "slaves": [
    {
      "alias": 0,
      "position": 0,
      "vendor_id": "0x00000002",
      "product_code": "0x044c2c52",
      "name": "EL3002"
    }
  ]
}
```

### Read Task Configuration

```json
{
  "type": "ethercat_read",
  "device": "ethercat-device-key",
  "sample_rate": 1000,
  "stream_rate": 25,
  "channels": [
    {
      "type": "pdo_input",
      "slave_position": 0,
      "index": "0x6000",
      "subindex": "0x11",
      "data_type": "int16",
      "synnax_channel": 12345
    }
  ]
}
```

### Write Task Configuration

```json
{
  "type": "ethercat_write",
  "device": "ethercat-device-key",
  "channels": [
    {
      "type": "pdo_output",
      "slave_position": 1,
      "index": "0x7000",
      "subindex": "0x01",
      "data_type": "uint16",
      "synnax_channel": 12346
    }
  ]
}
```

## Architecture Design

### Recommended Approach

1. **Master Lifecycle**: Create master once at task startup, share across read/write
2. **Domain Strategy**: Single domain for all PDO entries (simpler) vs multiple domains
   (more flexible)
3. **Timing**: Use hardware-timed cycle with `nanosleep()` or RT timer
4. **Thread Model**: Dedicated RT thread for EtherCAT cycle, separate from Synnax
   pipeline

### Proposed Code Structure

```
driver/ethercat/
├── BUILD                         # Main build, selects backends
├── ethercat.h                    # Integration constants + INTEGRATION_NAME
├── factory.cpp                   # Task factory
├── cyclic_engine.h               # Shared cycle coordinator + task lifecycle
├── cyclic_engine.cpp
├── channels.h                    # PDO entry channel configs
├── read_task.h                   # common::Source implementation
├── write_task.h                  # common::Sink implementation
├── scan_task.h                   # Slave discovery
├── master.h                      # Abstract Master interface
│
├── igh/                          # IgH backend (Linux only)
│   ├── BUILD
│   └── master.cpp                # ecrt_* wrapper
│
├── soem/                         # SOEM backend (cross-platform)
│   ├── BUILD
│   └── master.cpp                # ec_* wrapper
│
├── mock/                         # Mock backend (testing)
│   ├── BUILD
│   └── master.cpp                # Mock for unit tests
│
└── errors/
    └── errors.h                  # EtherCAT error codes
```

## Read/Write Task Implementation

### Key Difference: EtherCAT is Cyclic

Unlike Modbus/OPC (request → response), EtherCAT continuously exchanges process data:

```
Modbus:                          EtherCAT:
┌────────┐    request    ┌───────┐    ┌────────┐  continuous  ┌────────┐
│ Master │ ───────────▶  │ Slave │    │ Master │ ◀──────────▶ │ Slaves │
│        │ ◀───────────  │       │    │        │   exchange   │        │
└────────┘   response    └───────┘    └────────┘              └────────┘
                                           │
                                      Fixed cycle
                                      (e.g., 1ms)
```

This means:

1. **One master per network** - shared across read/write tasks
2. **Synchronized cycle** - receive/send happens once per cycle, not per-channel
3. **Memory-mapped I/O** - read/write directly to process data buffer

### Architecture: Shared Cyclic Engine

```
┌─────────────────────────────────────────────────────────────────┐
│                     EtherCAT Device                             │
│  ┌─────────────────────────────────────────────────────────┐   │
│  │              Cyclic Engine (runs at cycle_time)          │   │
│  │  while running:                                          │   │
│  │      master.receive()                                    │   │
│  │      domain.process()                                    │   │
│  │      notify_read_tasks()   ──▶  ReadTask gets data      │   │
│  │      wait_for_write_tasks() ◀── WriteTask sets data     │   │
│  │      domain.queue()                                      │   │
│  │      master.send()                                       │   │
│  └─────────────────────────────────────────────────────────┘   │
│         ▲                              ▲                        │
│         │                              │                        │
│  ┌──────┴──────┐              ┌───────┴───────┐                │
│  │  ReadTask   │              │  WriteTask    │                │
│  │  (Source)   │              │  (Sink)       │                │
│  └─────────────┘              └───────────────┘                │
└─────────────────────────────────────────────────────────────────┘
```

### CyclicEngine

The CyclicEngine manages the EtherCAT master lifecycle and coordinates tasks:

```cpp
class CyclicEngine {
    std::unique_ptr<Master> master_;
    std::unique_ptr<Domain> domain_;
    std::atomic<bool> running_{false};
    std::thread cycle_thread_;

    // Synchronization
    std::condition_variable input_ready_;   // Signals read tasks
    std::condition_variable output_done_;   // Signals cycle can continue

    // Process data snapshot for tasks
    std::vector<uint8_t> input_snapshot_;
    std::vector<uint8_t> output_buffer_;

public:
    /// Register a PDO entry, returns offset for read/write
    std::pair<size_t, xerrors::Error> register_pdo(
        uint16_t slave_pos, uint16_t index,
        uint8_t subindex, size_t bit_length
    );

    /// Called by ReadTask - blocks until new input data available
    void wait_for_inputs(uint64_t& last_cycle);

    /// Called by WriteTask - write to output buffer
    void write_output(size_t offset, const void* data, size_t len);

private:
    void cycle_loop() {
        while (running_) {
            master_->receive();
            domain_->process();

            // Snapshot inputs for read tasks
            memcpy(input_snapshot_.data(), domain_->data(), input_size_);
            input_ready_.notify_all();

            // Wait for write tasks (with timeout)
            output_done_.wait_for(lock, cycle_period_ / 2);

            // Copy outputs and send
            memcpy(domain_->data() + input_size_, output_buffer_.data(),
                   output_buffer_.size());
            domain_->queue();
            master_->send();
        }
    }
};
```

### ReadTaskSource

```cpp
class ReadTaskSource final : public common::Source {
    ReadTaskConfig config_;
    std::shared_ptr<CyclicEngine> engine_;
    uint64_t last_cycle_ = 0;

public:
    ReadTaskSource(std::shared_ptr<CyclicEngine> engine, ReadTaskConfig config)
        : engine_(std::move(engine)), config_(std::move(config)) {
        // Register PDOs and store offsets
        for (auto& ch : config_.channels) {
            auto [offset, err] = engine_->register_pdo(
                ch.slave_position, ch.pdo_index,
                ch.pdo_subindex, ch.data_type.density() * 8
            );
            ch.offset = offset;
        }
    }

    common::ReadResult read(breaker::Breaker& breaker, telem::Frame& fr) override {
        // Wait for new cycle data
        engine_->wait_for_inputs(last_cycle_);
        if (breaker.stopped()) return {};

        const uint8_t* data = engine_->input_data();

        // Extract values from process data buffer
        for (const auto& ch : config_.channels) {
            copy_pdo_value(data + ch.offset, ch.data_type, fr.series->at(i));
        }

        return {};
    }
};
```

### WriteTaskSink

```cpp
class WriteTaskSink final : public common::Sink {
    WriteTaskConfig config_;
    std::shared_ptr<CyclicEngine> engine_;

public:
    WriteTaskSink(std::shared_ptr<CyclicEngine> engine, WriteTaskConfig config)
        : engine_(std::move(engine)), config_(std::move(config)) {
        // Register output PDOs
        for (auto& ch : config_.channels) {
            auto [offset, err] = engine_->register_pdo(
                ch.slave_position, ch.pdo_index,
                ch.pdo_subindex, ch.data_type.density() * 8
            );
            ch.offset = offset;
        }
    }

    xerrors::Error write(const telem::Frame& fr) override {
        for (const auto& ch : config_.channels) {
            auto it = fr.find(ch.synnax_key);
            if (it == fr.end()) continue;

            // Write last value to output buffer
            const auto& series = it->second;
            engine_->write_output(ch.offset, series.last_value_ptr(),
                                  ch.data_type.density());
        }

        engine_->signal_output_done();
        return xerrors::NIL;
    }
};
```

### PDO Memory Layout

EtherCAT process data is a flat byte buffer with offsets:

```
Domain Process Data Buffer:
┌────────┬────────┬────────┬────────┬────────┬────────┐
│ IN 0   │ IN 1   │ IN 2   │ OUT 0  │ OUT 1  │ OUT 2  │
│ 2 bytes│ 2 bytes│ 1 byte │ 2 bytes│ 2 bytes│ 1 byte │
└────────┴────────┴────────┴────────┴────────┴────────┘
offset: 0      2        4        5        7        9
```

### Design Rationale

| Pattern             | Why                                                    |
| ------------------- | ------------------------------------------------------ |
| Shared CyclicEngine | One EtherCAT master per network, multiple Synnax tasks |
| Snapshot for inputs | Decouple tight cycle timing from Synnax pipeline       |
| Condition variables | Coordinate read/write tasks with cycle thread          |
| Offset-based access | EtherCAT process data is flat memory buffer            |

### Comparison to Modbus

| Aspect      | Modbus                    | EtherCAT           |
| ----------- | ------------------------- | ------------------ |
| Timing      | On-demand request         | Fixed cycle        |
| Data access | Read registers by address | Memory-mapped PDOs |
| Connection  | Per-task                  | Shared engine      |
| Sync        | None needed               | Cycle coordination |
| Multi-slave | Separate connections      | Single domain      |

## API Alignment Review

### Synnax Internal APIs

#### common::Source Interface (for ReadTask)

```cpp
struct ReadResult {
    xerrors::Error error;
    std::string warning;  // Non-fatal issues
};

struct Source {
    // Required methods
    [[nodiscard]] virtual synnax::WriterConfig writer_config() const = 0;
    [[nodiscard]] virtual std::vector<synnax::Channel> channels() const = 0;
    virtual ReadResult read(breaker::Breaker &breaker, telem::Frame &data) = 0;

    // Optional lifecycle
    virtual xerrors::Error start() { return xerrors::NIL; }
    virtual xerrors::Error stop() { return xerrors::NIL; }
};
```

#### common::Sink Interface (for WriteTask)

```cpp
class Sink : public pipeline::Sink, public pipeline::Source {
    // Command channels (receive from Synnax)
    std::vector<synnax::ChannelKey> cmd_channels;

    // State channels (write back to Synnax)
    std::unordered_map<synnax::ChannelKey, synnax::Channel> state_channels;
    std::set<synnax::ChannelKey> state_indexes;
    telem::Rate state_rate;

    // Current output state
    std::unordered_map<synnax::ChannelKey, telem::SampleValue> chan_state;

    // Required methods
    [[nodiscard]] synnax::StreamerConfig streamer_config() const;
    [[nodiscard]] synnax::WriterConfig writer_config() const;
    virtual xerrors::Error write(telem::Frame &frame) = 0;
    xerrors::Error read(breaker::Breaker&, telem::Frame&);  // State feedback

    // Lifecycle
    virtual xerrors::Error start() { return xerrors::NIL; }
    virtual xerrors::Error stop() { return xerrors::NIL; }
};
```

### IgH EtherCAT API

| Function                          | Signature                                                  | Purpose                    |
| --------------------------------- | ---------------------------------------------------------- | -------------------------- |
| `ecrt_request_master`             | `ec_master_t* (unsigned int index)`                        | Request master access      |
| `ecrt_release_master`             | `void (ec_master_t*)`                                      | Release master             |
| `ecrt_master_create_domain`       | `ec_domain_t* (ec_master_t*)`                              | Create process data domain |
| `ecrt_master_slave_config`        | `ec_slave_config_t* (master, alias, pos, vendor, product)` | Configure slave            |
| `ecrt_slave_config_reg_pdo_entry` | `int (sc, index, subindex, domain, &bit_pos)`              | Register PDO, get offset   |
| `ecrt_master_activate`            | `int (ec_master_t*)`                                       | Start cyclic operation     |
| `ecrt_master_deactivate`          | `int (ec_master_t*)`                                       | Stop cyclic operation      |
| `ecrt_domain_data`                | `uint8_t* (ec_domain_t*)`                                  | Get process data pointer   |
| `ecrt_master_receive`             | `int (ec_master_t*)`                                       | Receive datagrams          |
| `ecrt_domain_process`             | `int (ec_domain_t*)`                                       | Process received data      |
| `ecrt_domain_queue`               | `int (ec_domain_t*)`                                       | Queue for sending          |
| `ecrt_master_send`                | `int (ec_master_t*)`                                       | Send datagrams             |

### SOEM API

| Function                               | Purpose                       |
| -------------------------------------- | ----------------------------- |
| `ec_init(ifname)`                      | Initialize, bind to NIC       |
| `ec_close()`                           | Cleanup                       |
| `ec_config_init(FALSE)`                | Discover and configure slaves |
| `ec_config_map(&IOmap)`                | Map I/O to memory buffer      |
| `ec_configdc()`                        | Configure distributed clocks  |
| `ec_statecheck(slave, state, timeout)` | Wait for state transition     |
| `ec_send_processdata()`                | Send process data frame       |
| `ec_receive_processdata(timeout)`      | Receive and return WKC        |
| `ec_slave[n].inputs`                   | Pointer to slave input data   |
| `ec_slave[n].outputs`                  | Pointer to slave output data  |

### API Differences Requiring Abstraction

| Aspect        | IgH                     | SOEM                           |
| ------------- | ----------------------- | ------------------------------ |
| Master handle | Explicit `ec_master_t*` | Global state                   |
| Domain        | Explicit `ec_domain_t*` | Implicit `IOmap[]`             |
| Slave config  | `ec_slave_config_t*`    | `ec_slave[]` array             |
| PDO offset    | From `reg_pdo_entry()`  | Calculate from `Obytes/Ibytes` |
| Process data  | `ecrt_domain_data()`    | `ec_slave[n].inputs/outputs`   |

### Corrected ReadTaskSource

```cpp
class ReadTaskSource final : public common::Source {
    ReadTaskConfig config_;
    std::shared_ptr<CyclicEngine> engine_;
    uint64_t last_cycle_ = 0;

public:
    explicit ReadTaskSource(
        std::shared_ptr<CyclicEngine> engine,
        ReadTaskConfig config
    ) : engine_(std::move(engine)), config_(std::move(config)) {
        for (auto& ch : config_.channels) {
            auto [offset, err] = engine_->register_pdo(
                ch.slave_position, ch.pdo_index,
                ch.pdo_subindex, ch.data_type.density() * 8
            );
            if (err) throw std::runtime_error(err.message());
            ch.offset = offset;
        }
    }

    [[nodiscard]] synnax::WriterConfig writer_config() const override {
        std::vector<synnax::ChannelKey> keys;
        for (const auto& ch : config_.channels)
            keys.push_back(ch.synnax_key);
        for (const auto& idx : config_.index_keys)
            keys.push_back(idx);
        return synnax::WriterConfig{
            .channels = keys,
            .mode = common::data_saving_writer_mode(config_.data_saving),
        };
    }

    [[nodiscard]] std::vector<synnax::Channel> channels() const override {
        std::vector<synnax::Channel> result;
        for (const auto& ch : config_.channels)
            result.push_back(ch.sy_channel);
        return result;
    }

    xerrors::Error start() override {
        return engine_->add_task();
    }

    xerrors::Error stop() override {
        engine_->remove_task();
        return xerrors::NIL;
    }

    common::ReadResult read(breaker::Breaker& breaker, telem::Frame& fr) override {
        common::ReadResult result;

        engine_->wait_for_inputs(last_cycle_);
        if (breaker.stopped()) return result;

        const uint8_t* data = engine_->input_data();

        // Initialize frame if needed
        if (fr.empty()) {
            fr.reserve(config_.channels.size() + config_.index_keys.size());
            for (const auto& ch : config_.channels)
                fr.emplace(ch.synnax_key, telem::Series(ch.data_type, 1));
            for (const auto& idx : config_.index_keys)
                fr.emplace(idx, telem::Series(telem::TIMESTAMP_T, 1));
        }

        // Extract values, set warning if slave not operational
        size_t i = 0;
        for (const auto& ch : config_.channels) {
            if (!engine_->slave_operational(ch.slave_position)) {
                result.warning = "Slave " + std::to_string(ch.slave_position) +
                                 " not operational";
            }
            copy_pdo_value(data + ch.offset, ch.data_type, fr.series->at(i++));
        }

        // Add timestamps
        auto now = telem::TimeStamp::now();
        for (size_t j = 0; j < config_.index_keys.size(); j++)
            fr.series->at(i + j).write(now);

        return result;
    }
};
```

### Corrected WriteTaskSink

```cpp
class WriteTaskSink final : public common::Sink {
    WriteTaskConfig config_;
    std::shared_ptr<CyclicEngine> engine_;

public:
    WriteTaskSink(
        std::shared_ptr<CyclicEngine> engine,
        WriteTaskConfig config
    ) : Sink(
            config.state_rate,
            config.state_index_keys,
            config.state_channels,
            config.cmd_keys(),
            config.data_saving
        ),
        engine_(std::move(engine)),
        config_(std::move(config)) {
        // Register output PDOs
        for (auto& ch : config_.channels) {
            auto [offset, err] = engine_->register_pdo(
                ch.slave_position, ch.pdo_index,
                ch.pdo_subindex, ch.data_type.density() * 8
            );
            if (err) throw std::runtime_error(err.message());
            ch.offset = offset;
        }
    }

    xerrors::Error start() override {
        return engine_->add_task();
    }

    xerrors::Error stop() override {
        engine_->remove_task();
        return xerrors::NIL;
    }

    xerrors::Error write(telem::Frame& frame) override {
        for (const auto& ch : config_.channels) {
            if (!frame.contains(ch.cmd_key)) continue;

            const auto& series = frame.at(ch.cmd_key);
            if (series.empty()) continue;

            // Write to engine's output buffer
            engine_->write_output(
                ch.offset,
                series.last_value_ptr(),
                ch.data_type.density()
            );
        }

        // Update state for feedback to Synnax
        this->set_state(frame);

        return xerrors::NIL;
    }
};
```

### CyclicEngine Task Lifecycle

```cpp
class CyclicEngine {
    std::atomic<int> active_task_count_{0};
    std::mutex lifecycle_mu_;

public:
    xerrors::Error add_task() {
        std::lock_guard lock(lifecycle_mu_);
        if (active_task_count_++ == 0) {
            // First task - start the cycle
            auto err = activate();
            if (err) {
                active_task_count_--;
                return err;
            }
            start_cycle_thread();
        }
        return xerrors::NIL;
    }

    void remove_task() {
        std::lock_guard lock(lifecycle_mu_);
        if (--active_task_count_ == 0) {
            // Last task - stop the cycle
            stop_cycle_thread();
            deactivate();
        }
    }
};
```

### Bazel Build Configuration

```python
# driver/ethercat/BUILD
cc_library(
    name = "ethercat",
    srcs = ["factory.cpp"],
    hdrs = [
        "ethercat.h",
        "master.h",
        "domain.h",
        "slave_config.h",
        "channels.h",
        "read_task.h",
        "write_task.h",
    ],
    deps = [
        "//driver/ethercat/soem",      # Always available (vendored)
        "//driver/pipeline",
        "//driver/task/common",
    ] + select({
        "@platforms//os:linux": ["//driver/ethercat/igh"],
        "//conditions:default": [],
    }),
)

# driver/ethercat/igh/BUILD
cc_library(
    name = "igh",
    srcs = glob(["*.cpp"]),
    target_compatible_with = ["@platforms//os:linux"],
    linkopts = ["-lethercat"],
    deps = ["//driver/ethercat:headers"],  # Just the interface headers
)

# driver/ethercat/soem/BUILD
cc_library(
    name = "soem",
    srcs = glob(["*.cpp"]),
    deps = [
        "//driver/ethercat:headers",
        "//vendor/soem",
    ],
)
```

## Risk Assessment

| Risk                        | Severity | Mitigation                                  |
| --------------------------- | -------- | ------------------------------------------- |
| Kernel module compatibility | High     | DKMS package auto-rebuilds on kernel update |
| Real-time performance       | Medium   | Provide tuning guide, optional RT kernel    |
| NIC driver issues           | Medium   | Document supported NICs, test with generic  |
| Build complexity            | Medium   | Apt package for IgH, vendored SOEM          |
| Platform limitation         | Low      | SOEM provides Windows/macOS fallback        |
| Maintenance burden          | Medium   | IgH is mature, stable API                   |

## Distribution Strategy

### IgH Installation via DKMS Apt Package

We can automate IgH installation using DKMS (Dynamic Kernel Module Support):

```
┌──────────────────────────────────────────────────────────┐
│  User runs: sudo apt install synnax-ethercat-dkms        │
└──────────────────────┬───────────────────────────────────┘
                       │
                       ▼
┌──────────────────────────────────────────────────────────┐
│  DKMS automatically:                                     │
│  1. Installs source to /usr/src/ethercat-<version>       │
│  2. Compiles kernel modules for current kernel           │
│  3. Installs ec_master.ko, ec_generic.ko                 │
│  4. Rebuilds automatically on kernel updates             │
└──────────────────────────────────────────────────────────┘
```

### User Experience

```bash
# One-time: add Synnax apt repository
curl -fsSL https://apt.synnaxlabs.com/gpg | sudo gpg --dearmor -o /etc/apt/keyrings/synnax.gpg
echo "deb [signed-by=/etc/apt/keyrings/synnax.gpg] https://apt.synnaxlabs.com stable main" \
  | sudo tee /etc/apt/sources.list.d/synnax.list

# Install EtherCAT support (builds kernel module via DKMS)
sudo apt update
sudo apt install synnax-ethercat-dkms

# Configure NIC (still manual - hardware specific)
sudo nano /etc/ethercat/ethercat.conf
# Set: MASTER0_DEVICE="eth1"

# Start service
sudo systemctl enable --now ethercat

# Done - Synnax driver can now use IgH backend
```

### DKMS Package Contents

```
synnax-ethercat-dkms/
├── DEBIAN/
│   ├── control              # Package metadata
│   ├── postinst             # dkms add && dkms build && dkms install
│   └── prerm                # dkms remove
├── usr/src/ethercat-1.5.2/
│   ├── dkms.conf            # DKMS configuration
│   ├── master/              # Kernel module source
│   ├── devices/             # NIC driver source
│   └── ...
├── lib/
│   └── libethercat.so       # Userspace library
└── etc/ethercat/
    └── ethercat.conf.example
```

### Infrastructure Required

| Component       | Effort         | Notes                             |
| --------------- | -------------- | --------------------------------- |
| DKMS package    | 2-3 days       | Package IgH source with dkms.conf |
| Systemd service | 1 day          | Module loading, config parsing    |
| Apt repository  | 1-2 days       | Host on apt.synnaxlabs.com        |
| CI/CD pipeline  | 2-3 days       | Auto-build on IgH version updates |
| **Total**       | **~1-2 weeks** | One-time infrastructure           |

### SOEM: No Installation Required

SOEM is vendored directly in the Synnax repository:

```
vendor/soem/
├── BUILD
├── soem/
│   ├── ethercatbase.c
│   ├── ethercatmain.c
│   └── ...
└── osal/
    ├── linux/
    ├── win32/
    └── macosx/
```

Users on Windows/macOS (or Linux without IgH) get SOEM automatically - no extra
installation steps.

## Alternatives Considered

| Feature     | IgH EtherCAT  | SOEM      | TwinCAT    |
| ----------- | ------------- | --------- | ---------- |
| License     | GPL v2        | GPL v2    | Commercial |
| Platform    | Linux         | Linux/Win | Windows    |
| Real-time   | Kernel module | Userspace | Windows RT |
| Complexity  | Higher        | Lower     | Medium     |
| Performance | Best          | Good      | Best       |

**SOEM (Simple Open EtherCAT Master)** is an alternative - userspace-only and
cross-platform, but has slightly worse real-time performance.

## Testing Strategy

### Mock Backend for Unit Tests

Since we have an abstract interface, we can mock everything above the hardware layer:

```
┌─────────────────────────────────────────────────────────────┐
│  Unit Tests                                                 │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐         │
│  │ ReadTask    │  │ WriteTask   │  │ ScanTask    │         │
│  │ Tests       │  │ Tests       │  │ Tests       │         │
│  └──────┬──────┘  └──────┬──────┘  └──────┬──────┘         │
│         └────────────────┴────────────────┘                 │
│                          │                                  │
│                          ▼                                  │
│  ┌──────────────────────────────────────────────────────┐  │
│  │              Mock Backend                             │  │
│  │  - MockMaster (tracks API calls)                      │  │
│  │  - MockDomain (simulates process data)                │  │
│  │  - MockSlave  (configurable responses)                │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
        │
        │  Real backends only used in integration tests
        ▼
   ┌─────────┐  ┌──────────┐
   │   IgH   │  │   SOEM   │
   └─────────┘  └──────────┘
```

### Mock Implementation

```cpp
// driver/ethercat/mock/master.h

namespace ethercat::mock {

/// Records all API calls for verification
struct CallRecord {
    std::string method;
    std::vector<std::variant<int, uint32_t, std::string>> args;
};

class MockMaster : public ethercat::Master {
    std::vector<CallRecord> calls_;
    std::vector<std::unique_ptr<MockSlave>> slaves_;
    State state_ = State::Init;
    bool should_fail_activate_ = false;

public:
    // Configure mock behavior
    void add_slave(uint32_t vendor_id, uint32_t product_code,
                   std::vector<PDOEntry> inputs,
                   std::vector<PDOEntry> outputs);
    void set_fail_on_activate(bool fail) { should_fail_activate_ = fail; }
    void set_slave_state(size_t pos, State state);

    // Interface implementation - records calls and returns mock data
    std::unique_ptr<Domain> create_domain() override {
        calls_.push_back({"create_domain", {}});
        return std::make_unique<MockDomain>(this);
    }

    xerrors::Error activate() override {
        calls_.push_back({"activate", {}});
        if (should_fail_activate_)
            return xerrors::Error("Mock activation failure");
        state_ = State::Op;
        return xerrors::NIL;
    }

    // Test assertions
    bool was_called(const std::string& method) const;
    size_t call_count(const std::string& method) const;
    const std::vector<CallRecord>& calls() const { return calls_; }
};

class MockDomain : public ethercat::Domain {
    std::vector<uint8_t> process_data_;

public:
    uint8_t* data() override { return process_data_.data(); }

    // Test helpers - set inputs, read outputs
    void set_input_value(size_t offset, int16_t value);

    template<typename T>
    T get_output_value(size_t offset) const;
};

} // namespace ethercat::mock
```

### Example Unit Tests

```cpp
TEST_F(ReadTaskTest, ReadsProcessDataCorrectly) {
    ReadTaskSource source(master.get(), make_test_config());

    // Set mock input values
    domain->set_input_value(0, 1234);   // Channel 1
    domain->set_input_value(2, -5678);  // Channel 2

    breaker::Breaker brk;
    telem::Frame frame;
    auto result = source.read(brk, frame);

    ASSERT_NIL(result.error);
    EXPECT_EQ(frame.series->at(0).at<int16_t>(0), 1234);
    EXPECT_EQ(frame.series->at(1).at<int16_t>(0), -5678);
}

TEST_F(ReadTaskTest, CallsCyclicMethodsInOrder) {
    ReadTaskSource source(master.get(), make_test_config());

    breaker::Breaker brk;
    telem::Frame frame;
    source.read(brk, frame);

    auto& calls = master->calls();
    EXPECT_EQ(calls[0].method, "receive");
    EXPECT_EQ(calls[1].method, "process");
    EXPECT_EQ(calls[2].method, "queue");
    EXPECT_EQ(calls[3].method, "send");
}

TEST_F(ReadTaskTest, HandlesSlaveDisconnect) {
    ReadTaskSource source(master.get(), make_test_config());
    master->set_slave_state(0, State::Init);  // Simulate disconnect

    breaker::Breaker brk;
    telem::Frame frame;
    auto result = source.read(brk, frame);

    EXPECT_OCCURRED_AS(result.error, ethercat::SLAVE_NOT_OPERATIONAL);
}

TEST_F(WriteTaskTest, WritesOutputsCorrectly) {
    WriteTaskSink sink(master.get(), make_write_config());

    telem::Frame frame;
    frame.emplace(channel_key, telem::Series(std::vector<uint16_t>{0xABCD}));

    auto err = sink.write(frame);

    ASSERT_NIL(err);
    EXPECT_EQ(domain->get_output_value<uint16_t>(0), 0xABCD);
}
```

### Test Coverage

| Category         | What It Verifies                               |
| ---------------- | ---------------------------------------------- |
| PDO Registration | Correct index/subindex/size passed to domain   |
| Cyclic Order     | receive → process → queue → send sequence      |
| Data Parsing     | Bytes correctly extracted from process data    |
| Data Writing     | Values correctly written to output offsets     |
| State Machine    | Proper handling of Init/PreOp/SafeOp/Op states |
| Error Handling   | Graceful handling of failures and disconnects  |
| Configuration    | Parsing of JSON config, channel mapping        |

### What Unit Tests Don't Cover

- Actual EtherCAT wire protocol (trust IgH/SOEM)
- Real timing/jitter performance
- Hardware-specific quirks
- Backend API correctness

These require integration tests with real hardware or an EtherCAT software simulator.

### Integration Testing (Future)

Options for hardware-level testing:

| Approach            | Pros               | Cons                         |
| ------------------- | ------------------ | ---------------------------- |
| Real hardware in CI | Most accurate      | Expensive, complex setup     |
| EtherCAT simulator  | No hardware needed | May not catch all edge cases |
| Manual test suite   | Flexible           | Not automated                |

Potential simulators:

- **EtherCAT Slave Stack Code (SSC)** - Beckhoff's slave simulator
- **SOEM simple_test** - Basic loopback testing
- **Virtual EtherCAT** - Software-only network simulation

## Key Implementation Notes

1. **WriteTask is bidirectional**: The `common::Sink` class extends both
   `pipeline::Sink` (receive commands) AND `pipeline::Source` (send state feedback).
   This means write tasks have TWO pipelines: Control (commands in) and Acquisition
   (state out).

2. **ReadResult has warnings**: Don't just return errors - use the `warning` field for
   non-fatal issues like "slave not operational" that shouldn't stop the task.

3. **CyclicEngine lifecycle**: Use reference counting (`add_task`/`remove_task`) so the
   EtherCAT cycle only runs when at least one task is active.

4. **State feedback**: Write tasks must track current output values in `chan_state` map
   and periodically send them back to Synnax via the state acquisition pipeline.

## Open Questions

- [x] Should we support SOEM as an alternative/fallback? **Yes - multi-backend**
- [ ] What's the minimum supported kernel version for IgH?
- [ ] How to handle distributed clocks (DC) synchronization?
- [ ] CoE (CANopen over EtherCAT) support priority?
- [ ] How to expose slave configuration in Console UI?
- [ ] Should backend selection be per-device or global?
- [ ] Priority: DKMS package vs manual IgH install docs first?

## Implementation Order

### Overview

```
Phase 1: Foundation (testable without hardware)
├── Step 1: Abstract interface + Mock backend
├── Step 2: ReadTaskSource with mock
└── Step 3: WriteTaskSink with mock

Phase 2: SOEM Backend (testable with hardware)
├── Step 4: SOEM Master implementation
├── Step 5: Integration test with real slave
└── Step 6: Full read/write validation

Phase 3: IgH Backend (Linux only)
├── Step 7: IgH Master implementation
├── Step 8: Backend switching
└── Step 9: Performance comparison

Phase 4: Polish
├── Step 10: ScanTask for discovery
├── Step 11: Factory + registration
└── Step 12: Distribution (DKMS)
```

### Step 1: Abstract Interface + Mock Backend

**Files:**

```
driver/ethercat/
├── master.h          # Abstract Master interface
├── mock/
│   └── master.cpp    # MockMaster implementation
└── BUILD
```

**Validation tests:**

```cpp
TEST(MockMaster, InitializesSuccessfully) {
    auto master = ethercat::mock::create_master();
    ASSERT_NIL(master->initialize());
}

TEST(MockMaster, RegistersPDOAndReturnsOffset) {
    auto master = ethercat::mock::create_master();
    auto [offset, err] = master->register_input_pdo(0, 0x6000, 0x11, 16);
    ASSERT_NIL(err);
    EXPECT_EQ(offset, 0);

    auto [offset2, err2] = master->register_input_pdo(0, 0x6010, 0x11, 16);
    EXPECT_EQ(offset2, 2);  // 16 bits = 2 bytes
}

TEST(MockMaster, CycleUpdatesInputData) {
    auto master = ethercat::mock::create_master();
    master->set_input_value(0, int16_t{1234});
    master->receive();
    master->process();
    EXPECT_EQ(master->read_input<int16_t>(0), 1234);
}
```

**Done when:** All mock tests pass, interface is stable.

### Step 2: ReadTaskSource with Mock

**Files:**

```
driver/ethercat/
├── cyclic_engine.h
├── cyclic_engine.cpp
├── read_task.h
├── channels.h
└── read_task_test.cpp
```

**Validation tests:**

```cpp
TEST_F(ReadTaskTest, ImplementsSourceInterface) {
    auto source = make_read_source(mock_engine_, test_config_);
    auto writer_cfg = source->writer_config();
    EXPECT_FALSE(writer_cfg.channels.empty());
    EXPECT_EQ(source->channels().size(), 2);
}

TEST_F(ReadTaskTest, ReadsProcessDataIntoFrame) {
    mock_engine_->set_input_value(0, int16_t{1000});
    mock_engine_->trigger_cycle();

    breaker::Breaker brk;
    telem::Frame frame;
    auto result = source_->read(brk, frame);

    ASSERT_NIL(result.error);
    EXPECT_EQ(frame.at<int16_t>(channel1_key_, 0), 1000);
}

TEST_F(ReadTaskTest, StartAddsTaskToEngine) {
    EXPECT_EQ(mock_engine_->active_task_count(), 0);
    source_->start();
    EXPECT_EQ(mock_engine_->active_task_count(), 1);
    source_->stop();
    EXPECT_EQ(mock_engine_->active_task_count(), 0);
}

TEST_F(ReadTaskTest, SetsWarningWhenSlaveNotOperational) {
    mock_engine_->set_slave_state(0, State::PreOp);
    mock_engine_->trigger_cycle();

    breaker::Breaker brk;
    telem::Frame frame;
    auto result = source_->read(brk, frame);

    EXPECT_NIL(result.error);
    EXPECT_FALSE(result.warning.empty());
}
```

**Done when:** ReadTaskSource passes all unit tests.

### Step 3: WriteTaskSink with Mock

**Files:**

```
driver/ethercat/
├── write_task.h
└── write_task_test.cpp
```

**Validation tests:**

```cpp
TEST_F(WriteTaskTest, ImplementsSinkInterface) {
    auto sink = make_write_sink(mock_engine_, test_config_);
    EXPECT_FALSE(sink->streamer_config().channels.empty());
    EXPECT_FALSE(sink->writer_config().channels.empty());
}

TEST_F(WriteTaskTest, WritesCommandToOutputBuffer) {
    telem::Frame frame;
    frame.emplace(cmd_key_, telem::Series(uint16_t{0xABCD}));

    auto err = sink_->write(frame);

    ASSERT_NIL(err);
    EXPECT_EQ(mock_engine_->read_output<uint16_t>(0), 0xABCD);
}

TEST_F(WriteTaskTest, UpdatesStateAfterWrite) {
    telem::Frame cmd_frame;
    cmd_frame.emplace(cmd_key_, telem::Series(uint16_t{100}));
    sink_->write(cmd_frame);

    breaker::Breaker brk;
    telem::Frame state_frame;
    sink_->read(brk, state_frame);

    EXPECT_EQ(state_frame.at<uint16_t>(state_key_, 0), 100);
}
```

**Done when:** WriteTaskSink passes all unit tests.

### Step 4: SOEM Master Implementation

**Files:**

```
vendor/soem/              # Vendored SOEM library
driver/ethercat/soem/
└── master.cpp
```

**Validation (requires hardware):**

```cpp
TEST(SOEMMaster, DiscoversSlaves) {
    auto master = ethercat::soem::create_master("eth1");
    ASSERT_NIL(master->initialize());
    EXPECT_GT(master->slave_count(), 0);
}

TEST(SOEMMaster, TransitionsToOperational) {
    auto master = ethercat::soem::create_master("eth1");
    master->initialize();
    master->configure_slaves();
    ASSERT_NIL(master->activate());
    EXPECT_EQ(master->state(), State::Op);
}
```

**Done when:** SOEM discovers slaves and reaches operational state.

### Step 5: Integration Test with Real Slave

**Validation (with e.g., Beckhoff EL3002):**

```cpp
TEST(Integration, ReadsAnalogInput) {
    auto master = ethercat::soem::create_master("eth1");
    master->initialize();

    auto [offset, err] = master->register_input_pdo(0, 0x6000, 0x11, 16);
    master->activate();

    for (int i = 0; i < 10; i++) {
        master->receive();
        master->process();
        int16_t value = master->read_input<int16_t>(offset);
        std::cout << "Analog input: " << value << std::endl;
        master->queue();
        master->send();
        std::this_thread::sleep_for(1ms);
    }
}
```

**Done when:** Can read real sensor values from hardware.

### Step 6: Full Read/Write Validation

**Validation:**

```cpp
TEST(Integration, FullReadWriteCycle) {
    auto engine = std::make_shared<CyclicEngine>(
        ethercat::soem::create_master("eth1"),
        telem::Rate(1000)
    );

    auto read_source = std::make_unique<ReadTaskSource>(engine, read_config);
    auto write_sink = std::make_unique<WriteTaskSink>(engine, write_config);

    read_source->start();
    write_sink->start();

    telem::Frame cmd;
    cmd.emplace(output_key, telem::Series(uint16_t{500}));
    write_sink->write(cmd);

    breaker::Breaker brk;
    telem::Frame input;
    read_source->read(brk, input);
}
```

**Done when:** Can write outputs and read inputs with real hardware.

### Step 7-9: IgH Backend

Same validation pattern with IgH:

```cpp
auto master = ethercat::igh::create_master(0);
```

Additional: Compare jitter/latency between SOEM and IgH backends.

### Step 10: ScanTask for Discovery

**Validation:**

```cpp
TEST(ScanTask, DiscoversSlaves) {
    mock_engine_->add_mock_slave(0x2, 0x0bba3052, "EL3002");
    mock_engine_->add_mock_slave(0x2, 0x07d43052, "EL2004");

    auto devices = scanner->scan();

    EXPECT_EQ(devices.size(), 2);
    EXPECT_EQ(devices[0].name, "EL3002");
}
```

### Step 11: Factory + Registration

**Validation:**

```cpp
TEST(Factory, ConfiguresReadTask) {
    auto factory = ethercat::Factory(mock_engine_);

    synnax::Task task;
    task.type = "ethercat_read";
    task.config = R"({"device": "...", "channels": [...]})"_json;

    auto [result, handled] = factory.configure_task(ctx, task);

    EXPECT_TRUE(handled);
    EXPECT_NE(result.task, nullptr);
}
```

**Manual validation:**

```bash
./synnax-driver --integrations ethercat
# Device appears in Console, tasks can be created
```

### Step 12: Distribution (DKMS)

**Validation:**

```bash
sudo dpkg -i synnax-ethercat-dkms_1.0.0_all.deb
lsmod | grep ec_master
./synnax-driver --integrations ethercat
# Auto-detects IgH backend
```

### Validation Summary

| Step | Method        | Hardware    |
| ---- | ------------- | ----------- |
| 1    | Unit tests    | No          |
| 2    | Unit tests    | No          |
| 3    | Unit tests    | No          |
| 4    | Manual test   | Yes         |
| 5    | Manual test   | Yes         |
| 6    | Manual test   | Yes         |
| 7-9  | Manual test   | Yes (Linux) |
| 10   | Unit tests    | No          |
| 11   | Unit + manual | Yes         |
| 12   | Manual        | Yes (Linux) |

### Checklist

**Phase 1: Foundation**

- [ ] Abstract Master interface
- [ ] Mock backend
- [ ] CyclicEngine
- [ ] ReadTaskSource
- [ ] WriteTaskSink
- [ ] All unit tests passing

**Phase 2: SOEM Backend**

- [ ] Vendor SOEM in `/vendor/soem/`
- [ ] SOEM Master implementation
- [ ] Hardware integration tests
- [ ] Full read/write cycle working

**Phase 3: IgH Backend**

- [ ] IgH Master implementation
- [ ] Backend auto-detection
- [ ] Performance comparison

**Phase 4: Polish**

- [ ] ScanTask
- [ ] Factory registration
- [ ] DKMS package
- [ ] Documentation

## References

### IgH EtherCAT Master

- [IgH EtherCAT Master Documentation](https://docs.etherlab.org/ethercat/1.6/doxygen/group__ApplicationInterface.html)
- [EtherLab GitLab Repository](https://gitlab.com/etherlab.org/ethercat)
- [IgH Implementation Guide](https://github.com/veysiadn/IgHEtherCATImplementation)
- [Intel ECI EtherCAT Stack](https://eci.intel.com/docs/3.3/components/ethercat.html)
- [API Usage Notes](https://docs.etherlab.org/ethercat/1.6/doxygen/apiusage.html)
- [Official PDF Documentation](https://docs.etherlab.org/ethercat/1.5/pdf/ethercat_doc.pdf)

### SOEM

- [SOEM GitHub Repository](https://github.com/OpenEtherCATsociety/SOEM)
- [SOEM Documentation](https://openethercatsociety.github.io/doc/soem/)

### DKMS

- [DKMS Documentation](https://github.com/dell/dkms)
- [Debian DKMS Packaging Guide](https://wiki.debian.org/Packaging/Dkms)
