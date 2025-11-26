# Synnax Client Documentation - Comprehensive Content Map

## Executive Summary

This document provides a complete analysis of all Python and TypeScript client
documentation pages, their sections, and organization. This structure enables accurate
documentation refactoring and ensures no content is missed when reorganizing the
documentation hierarchy.

**Total Pages:** 23 (12 Python + 11 TypeScript, with 10 pages shared across both)
**Shared Pages:** 10 (Get Started, Channels, Ranges, Read Data, Write Data, Stream Data,
Delete Data, Series and Frames, Examples, Troubleshooting) **Language-Specific Pages:**
3 (Python: Device Driver | TypeScript: Timestamps)

---

## Python Client Documentation

**Location:** `/docs/site/src/pages/reference/python-client/`

### Page 1: Get Started

**File:** `get-started.mdx` **Sections:**

- Installation
- Authenticating with a Core
  - The Synnax Login Command
  - Passing Credentials Directly

**Key Concepts:** Installation, credentials, authentication, CLI login

---

### Page 2: Channels

**File:** `channels.mdx` **Sections:**

- Creating Channels
  - Creating a Data Channel and its Index
  - Creating Multiple Channels
  - Only Create a Channel if it Doesn't Exist
- Retrieving Channels
  - Retrieving a Single Channel
  - Retrieving Multiple Channels
  - Retrieving a Channel Using a Range
  - Retrieving Channels Using Regular Expressions
- Renaming Channels
- Deleting Channels

**Key Concepts:** Channel CRUD, batching operations, regex retrieval

---

### Page 3: Ranges

**File:** `ranges.mdx` **Sections:**

- Range Configuration Reference
- Creating Ranges
  - Only Create a Range if it Doesn't Exist
- Creating Child Ranges
- Retrieving Ranges
  - Retrieving a Single Range
  - Retrieving Multiple Ranges
  - Retrieving Child Ranges
- Updating a Range
- Working with Channels
  - Accessing Channels
  - Accessing Multiple Channels
  - Aliasing Channels
- Attaching Metadata
  - Setting Metadata
  - Getting Metadata
  - Deleting Metadata
- Deleting Ranges

**Key Concepts:** Time range categorization, metadata, channel aliases, parent-child
relationships, TimeRange structure

---

### Page 4: Read Data

**File:** `read-data.mdx` **Sections:**

- Reading from a Channel
- Reading from Multiple Channels
- Reading Channel Data from a Range
- Reading the Latest Data
- Reading with Iterators
- Examples

**Key Concepts:** Direct reads, frame returns, Series objects, time ranges, server-side
iterators, lazy evaluation

---

### Page 5: Write Data

**File:** `write-data.mdx` **Sections:**

- Writing to a Channel
- Writing to a Range
- Using a Writer
  - Auto-Commit
    - When to Disable Auto-Commit
  - Write Authorities
    - Opening a writer with the same authority on all channels
    - Opening a writer with different authorities on each channel
    - Adjusting write authorities after open
  - Persistence/Streaming Mode
- Common Pitfalls
  - Using Many Individual Write Calls Instead of a Writer
  - Calling Commit on Every Write

**Key Concepts:** Direct writes, Writer objects, auto-commit, write authorities,
persistence modes, alignment, timestamps

---

### Page 6: Stream Data

**File:** `stream-data.mdx` **Sections:**

- Opening a Streamer
- Reading Frames
  - Specifying a Timeout
  - Downsampling
  - Handling Partial Frames
  - Using a For Loop
- Updating the Channel List
- Closing the Streamer
  - Using a Context Manager
- Using an Async Streamer

**Key Concepts:** Real-time streaming, frames, downsampling, partial frames, async
streaming, context managers, channel updates

---

### Page 7: Delete Data

**File:** `delete-data.mdx` **Sections:**

- Deleting Data From a Channel
- Limitations of Deletions

**Key Concepts:** Data deletion, idempotency, time range deletion, channel names vs keys

---

### Page 8: Series and Frames

**File:** `series-and-frames.mdx` **Sections:**

- Series
  - Constructing a Series
  - Interop with Numpy
  - The Time Range Property
- Frames
  - Constructing a Frame
  - Accessing Frame Data
    - Using the dictionary interface

**Key Concepts:** Series type, numpy integration, Frame type, time ranges, data access
patterns

---

### Page 9: Examples

**File:** `examples.mdx` **Sections:**

- Examples (links to GitHub implementations)
  - Basic Read and Write
  - Stream Write
  - Stream Read
  - Async Stream Read
  - Calculated Channels
  - Create Range
  - Create Channels
  - Simulation
  - Advanced Calculated Channels
  - Control
  - National Instruments
  - OPC UA
  - Arduino
  - Plot
  - Export to CSV

**Key Concepts:** Reference implementations, hardware integration, data processing
patterns

---

### Page 10: Troubleshooting

**File:** `troubleshooting.mdx` **Sections:**

- Installing Python
- Incorrect Python Version or Command Not Found
  - Python Command Is Under a Different Name
  - Python Is Not Available on Your PATH
- Synnax Command Not Found
  - Check Your pip Version
  - If Synnax Is Installed in a Virtual Environment

**Key Concepts:** Environment setup, PATH configuration, virtual environments, pip
issues

---

### Page 11: Build a Device Driver

**File:** `device-driver.mdx` **Sections:**

- Setup and Installation
  - Downloading the Arduino IDE
  - Installing Synnax (The Core, Console)
- Read-Only Driver (Arduino example)
- Write-Only Driver (Arduino example)
- Read-Write Driver (Arduino example)

**Key Concepts:** Custom hardware integration, driver patterns, Arduino integration,
Python client for hardware control

---

## TypeScript Client Documentation

**Location:** `/docs/site/src/pages/reference/typescript-client/`

### Page 1: Get Started

**File:** `get-started.mdx` **Sections:**

- Installation
- Authenticating with a Core

**Key Concepts:** npm installation, credentials, authentication

---

### Page 2: Channels

**File:** `channels.mdx` **Sections:**

- Creating Channels
  - Creating a Data Channel and its Index
  - Creating Multiple Channels
- Retrieving Channels
  - Retrieving a Single Channel
  - Retrieving Multiple Channels
- Deleting Channels

**Key Concepts:** Channel CRUD (fewer features than Python version - no rename, regex)

---

### Page 3: Ranges

**File:** `ranges.mdx` **Sections:** [Parallel structure to Python client]

**Key Concepts:** Time range categorization, metadata, channel access

---

### Page 4: Read Data

**File:** `read-data.mdx` **Sections:**

- Reading from a Channel
- Reading from Multiple Channels
- Reading the Latest Data
- Using Iterators

**Key Concepts:** Direct reads, frames, iterators, lazy evaluation

---

### Page 5: Write Data

**File:** `write-data.mdx` **Sections:**

- Writing to a Channel
- Using a Writer
  - Closing the Writer
  - Different Ways of Writing Data
  - Auto-Commit
    - When to Disable Auto-Commit
  - Write Authorities
    - Opening a writer with the same authority on all channels
    - Opening a writer with different authorities on each channel
    - Adjusting write authorities after open
  - Persistence/Streaming Mode
- Common Pitfalls
  - Using Many Individual Write Calls Instead of a Writer
  - Calling Commit on Every Write

**Key Concepts:** Direct writes, Writer objects, auto-commit, write authorities,
persistence modes

---

### Page 6: Stream Data

**File:** `stream-data.mdx` **Sections:** [Parallel structure to Python client]

**Key Concepts:** Real-time streaming, frames, downsampling, partial frames, async
support

---

### Page 7: Delete Data

**File:** `delete-data.mdx` **Sections:**

- Deleting Data From a Channel
- Limitations of Deletions

**Key Concepts:** Data deletion, idempotency, limitations

---

### Page 8: Series and Frames

**File:** `series-and-frames.mdx` **Sections:**

- Series
  - Constructing a Series
  - Accessing Data
    - The at method
    - The as method
    - Accessing a TypedArray
  - Converting to a Javascript Array
  - The Time Range Property
  - Other Useful Properties
    - Length
    - Data type
    - Max, min, and bounds
- Frames
  - Constructing a Frame
  - Accessing Frame Data
    - The get method
    - The at method

**Key Concepts:** Series type, access methods, TypedArray, Frame type, time ranges,
properties

---

### Page 9: Timestamps

**File:** `timestamps.mdx` **Sections:**

- JavaScript's Limitations
- TimeStamp
  - Constructing a TimeStamp
  - Converting to a Date
  - Arithmetic
  - Comparisons
  - Accessing the Underlying Value
- TimeSpan
  - Constructing a TimeSpan
  - Performing Arithmetic
  - Accessing the Underlying Value
- TimeRange
  - Constructing a TimeRange
  - Checking if a TimeStamp is in a TimeRange
  - Checking if Two TimeRanges Overlap
  - Getting the TimeSpan of a TimeRange

**Key Concepts:** JavaScript precision issues, nanosecond timestamps, bigint, time
utilities, arithmetic operations

---

### Page 10: Examples

**File:** `examples.mdx` **Sections:** [Parallel structure to Python client]

**Key Concepts:** Reference implementations, patterns

---

### Page 11: Troubleshooting

**File:** `troubleshooting.mdx` **Sections:** [Parallel structure to Python client]

**Key Concepts:** Setup issues, configuration

---

## Content Organization Matrix

| Topic               | Python | TypeScript | File                  |
| ------------------- | ------ | ---------- | --------------------- |
| Installation & Auth | Yes    | Yes        | get-started.mdx       |
| Channel CRUD        | Yes    | Yes        | channels.mdx          |
| Rename Channels     | Yes    | No         | channels.mdx          |
| Regex Retrieval     | Yes    | No         | channels.mdx          |
| Range Management    | Yes    | Yes        | ranges.mdx            |
| Metadata            | Yes    | Yes        | ranges.mdx            |
| Read Operations     | Yes    | Yes        | read-data.mdx         |
| Iterators           | Yes    | Yes        | read-data.mdx         |
| Write Operations    | Yes    | Yes        | write-data.mdx        |
| Writers             | Yes    | Yes        | write-data.mdx        |
| Write Authorities   | Yes    | Yes        | write-data.mdx        |
| Streaming           | Yes    | Yes        | stream-data.mdx       |
| Data Deletion       | Yes    | Yes        | delete-data.mdx       |
| Series              | Yes    | Yes        | series-and-frames.mdx |
| Frames              | Yes    | Yes        | series-and-frames.mdx |
| Numpy Interop       | Yes    | No         | series-and-frames.mdx |
| Time Utilities      | No     | Yes        | timestamps.mdx        |
| Examples            | Yes    | Yes        | examples.mdx          |
| Troubleshooting     | Yes    | Yes        | troubleshooting.mdx   |
| Device Drivers      | Yes    | No         | device-driver.mdx     |

---

## Major Content Themes

### 1. Fundamental Operations (6 pages)

- **Channels** - Creating, retrieving, managing channel definitions
- **Ranges** - Marking time periods and attaching metadata
- **Read Data** - Various read patterns and optimizations
- **Write Data** - Writing strategies and performance considerations
- **Stream Data** - Real-time data processing
- **Delete Data** - Data removal operations

### 2. Data Structures (3 pages)

- **Series and Frames** - Core data containers
- **Timestamps** (TypeScript) - Time handling utilities

### 3. Developer Support (3 pages)

- **Examples** - Real-world implementations
- **Troubleshooting** - Problem resolution
- **Device Driver** (Python) - Hardware integration guide

### 4. Foundational (1 page)

- **Get Started** - Setup and authentication

---

## Hierarchical Content Organization

```
Client Documentation
├── Foundational
│   └── Get Started
│       ├── Installation
│       └── Authentication
├── Core Concepts
│   ├── Channels
│   │   ├── Create
│   │   ├── Retrieve
│   │   ├── Rename (Python only)
│   │   └── Delete
│   └── Ranges
│       ├── Create
│       ├── Retrieve
│       ├── Update
│       ├── Child Ranges
│       └── Metadata
├── Data Operations
│   ├── Read Data
│   │   ├── Single Channel
│   │   ├── Multiple Channels
│   │   ├── From Ranges
│   │   ├── Latest Data
│   │   └── Iterators
│   ├── Write Data
│   │   ├── Direct Writes
│   │   ├── Writers
│   │   ├── Auto-Commit
│   │   ├── Write Authorities
│   │   └── Persistence Modes
│   ├── Stream Data
│   │   ├── Opening Streamers
│   │   ├── Reading Frames
│   │   ├── Downsampling
│   │   ├── Partial Frames
│   │   └── Async Streaming
│   └── Delete Data
│       ├── Channel Deletion
│       └── Data Range Deletion
├── Data Structures
│   ├── Series
│   │   ├── Construction
│   │   ├── Operations
│   │   └── Interop (Language-specific)
│   ├── Frames
│   │   ├── Construction
│   │   └── Access Patterns
│   └── Timestamps (TypeScript only)
│       ├── TimeStamp
│       ├── TimeSpan
│       └── TimeRange
└── Developer Resources
    ├── Examples
    ├── Troubleshooting
    └── Device Drivers (Python only)
```

---

## Key Features Documented

### Cross-Language Features

- Channel management (CRUD)
- Range management and metadata
- Data reading (single/multiple/range)
- Data writing and writers
- Data streaming
- Data deletion
- Series and Frame data structures
- Example implementations
- Troubleshooting guides

### Python-Specific

- Device driver development guide
- Channel rename operation
- Regex-based channel retrieval
- Numpy interoperability for Series
- Async/await streaming

### TypeScript-Specific

- TimeStamp, TimeSpan, TimeRange utilities
- JavaScript precision handling
- TypedArray access
- Bigint timestamp representation

---

## Navigation Sequences

### Recommended Learning Path for New Users

**Both Languages:**

1. Get Started (installation & auth)
2. Channels (core entity management)
3. Ranges (time categorization)
4. Read Data (retrieval patterns)
5. Write Data (data persistence)
6. Stream Data (real-time processing)

**Optional Advanced Topics:**

- Series and Frames (data structure details)
- Delete Data (cleanup operations)
- Timestamps (TypeScript) / Device Drivers (Python)
- Examples (reference implementations)

---

## File Locations Summary

### Python Client Files

- `/docs/site/src/pages/reference/python-client/get-started.mdx`
- `/docs/site/src/pages/reference/python-client/channels.mdx`
- `/docs/site/src/pages/reference/python-client/ranges.mdx`
- `/docs/site/src/pages/reference/python-client/read-data.mdx`
- `/docs/site/src/pages/reference/python-client/write-data.mdx`
- `/docs/site/src/pages/reference/python-client/stream-data.mdx`
- `/docs/site/src/pages/reference/python-client/delete-data.mdx`
- `/docs/site/src/pages/reference/python-client/series-and-frames.mdx`
- `/docs/site/src/pages/reference/python-client/examples.mdx`
- `/docs/site/src/pages/reference/python-client/troubleshooting.mdx`
- `/docs/site/src/pages/reference/python-client/device-driver.mdx`
- `/docs/site/src/pages/reference/python-client/_nav.ts` (navigation config)
- `/docs/site/src/pages/reference/python-client/index.astro` (landing page)

### TypeScript Client Files

- `/docs/site/src/pages/reference/typescript-client/get-started.mdx`
- `/docs/site/src/pages/reference/typescript-client/channels.mdx`
- `/docs/site/src/pages/reference/typescript-client/ranges.mdx`
- `/docs/site/src/pages/reference/typescript-client/read-data.mdx`
- `/docs/site/src/pages/reference/typescript-client/write-data.mdx`
- `/docs/site/src/pages/reference/typescript-client/stream-data.mdx`
- `/docs/site/src/pages/reference/typescript-client/delete-data.mdx`
- `/docs/site/src/pages/reference/typescript-client/series-and-frames.mdx`
- `/docs/site/src/pages/reference/typescript-client/timestamps.mdx`
- `/docs/site/src/pages/reference/typescript-client/examples.mdx`
- `/docs/site/src/pages/reference/typescript-client/troubleshooting.mdx`
- `/docs/site/src/pages/reference/typescript-client/_nav.ts` (navigation config)
- `/docs/site/src/pages/reference/typescript-client/index.astro` (landing page)

---

## Validation Checklist for Refactoring

When refactoring or reorganizing documentation, ensure:

- [ ] All 11 core pages exist (Get Started, Channels, Ranges, Read Data, Write Data,
      Stream Data, Delete Data, Series and Frames, Examples, Troubleshooting, + 1
      language-specific)
- [ ] Python Device Driver page is preserved
- [ ] TypeScript Timestamps page is preserved
- [ ] Navigation configuration files (\_nav.ts) updated with any structural changes
- [ ] All section headings preserved (## and ### levels)
- [ ] Cross-references between pages remain valid
- [ ] Example links point to correct GitHub locations
- [ ] Code examples remain accurate and executable
- [ ] Landing page (index.astro) reflects new structure

---

## Conclusion

The Synnax client documentation is well-structured with clear separation of concerns:

- **Foundational**: Get Started
- **Entity Management**: Channels, Ranges
- **Operations**: Read, Write, Stream, Delete
- **Details**: Series/Frames, Timestamps
- **Support**: Examples, Troubleshooting, Device Drivers

This structure should be maintained during any refactoring to preserve usability and
logical flow.
