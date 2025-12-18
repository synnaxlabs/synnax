# Client Documentation Content Mapping

This document maps existing Python/TypeScript client documentation sections to the new
unified client documentation structure.

## Navigation Structure

```
/reference/client/
├── quick-start.mdx                     ✅ Complete
├── authentication.mdx                  ✅ Complete
├── fundamentals/
│   ├── channels.mdx                    ✅ Complete
│   ├── read-data.mdx                   ✅ Complete
│   └── write-data.mdx                  ✅ Complete
├── working-with-data/
│   ├── series-and-frames.mdx           🔄 In Progress
│   ├── ranges.mdx                      🔄 In Progress
│   ├── streaming-data.mdx              🔄 In Progress
│   └── iterators.mdx                   🔄 In Progress
├── advanced/
│   ├── writers.mdx                     🔄 In Progress
│   ├── delete-data.mdx                 🔄 In Progress
│   └── timestamps.mdx                  📝 5 sections (TS only)
└── resources/
    ├── examples.mdx                    📝 TBD
    └── build-device-driver.mdx         📝 6 sections (Python only)
```

## Sections Remaining to Transfer

Legend: 🔄 = in progress | 📝 = remaining

### Remaining Python Client (`/reference/python-client/`)

```
python-client/
├── read-data.mdx
│   └── Examples                        📝 → resources/examples
│
├── write-data.mdx
│
│
├── ranges.mdx
│
│
├── device-driver.mdx
│   ├── Setup and Installation          📝 → resources/build-device-driver
│   ├── Read-Only Driver                📝 → resources/build-device-driver
│   ├── Write-Only Driver               📝 → resources/build-device-driver
│   └── Read-Write Driver               📝 → resources/build-device-driver
│
├── examples.mdx
│   └── Examples                        📝 → resources/examples
│
└── troubleshooting.mdx                 📝 → TBD (migration strategy needed)
    ├── Installing Python
    ├── Incorrect Python Version
    └── Synnax Command Not Found
```

### Remaining TypeScript Client (`/reference/typescript-client/`)

```
typescript-client/
├── read-data.mdx
│
├── write-data.mdx
│
│
├── ranges.mdx
│
│
├── timestamps.mdx
│   ├── JavaScript's Limitations        📝 → advanced/timestamps
│   ├── TimeStamp                       📝 → advanced/timestamps
│   ├── TimeSpan                        📝 → advanced/timestamps
│   └── TimeRange                       📝 → advanced/timestamps
│
├── examples.mdx
│   └── Examples                        📝 → resources/examples
│
└── troubleshooting.mdx                 📝 → TBD (migration strategy needed)
    ├── Old Core Version
    └── Old Client Version
```

## Summary

| Source            | Remaining |
| ----------------- | --------- |
| Python Client     | 9         |
| TypeScript Client | 7         |

**Breakdown:**
- Shared sections (in both): ~14 (~~series/frames~~, ~~ranges~~, ~~streaming~~, ~~delete-data~~, ~~writers~~, examples)
- Python-only: ~13 (device-driver, ~~range reads/writes~~, ~~async streamer~~, troubleshooting)
- TypeScript-only: ~6 (timestamps, troubleshooting)

## Implementation Checklist

### Phase 1: Get Started (✅ COMPLETE)

- [x] Create Quick Start with actual content
- [x] Create Authentication page with actual content
- [x] Update navigation structure

### Phase 2: Fundamentals (✅ COMPLETE)

- [x] Create Channels with actual content
- [x] Create Read Data with actual content
- [x] Create Write Data with actual content

### Phase 3: Working with Data (🔄 IN PROGRESS)

- [x] Create all shell pages
- [🔄] Populate Series & Frames (content transferred, needs cleanup)
- [🔄] Populate Ranges (content transferred, needs cleanup)
- [🔄] Populate Streaming Data (content transferred, needs cleanup)
- [🔄] Populate Iterators (content transferred, needs cleanup)

### Phase 4: Advanced Topics (🔄 IN PROGRESS)

- [x] Create all shell pages
- [🔄] Populate Writers (content transferred, needs cleanup)
- [🔄] Populate Delete Data (content transferred, needs cleanup)
- [ ] Populate Timestamps

### Phase 5: Resources (📝 SHELLS COMPLETE)

- [x] Create all shell pages
- [ ] Populate Examples page
- [ ] Populate Build Device Driver

### Phase 6: Cleanup

- [ ] Add console tab content
- [ ] Final navigation review
- [ ] Remove/redirect old python-client and typescript-client pages

## Total Section Count

| Category          | Sections | Status            |
| ----------------- | -------- | ----------------- |
| Get Started       | 2        | ✅ Complete       |
| Fundamentals      | 3        | ✅ Complete       |
| Working with Data | 4        | 🔄 4 in progress  |
| Advanced Topics   | 3        | 🔄 2 in progress  |
| Resources         | 2        | 📝 Shells created |
| **TOTAL**         | **14**   | **5 complete, 6 in progress** |

## Notes

1. **Language Parity**: TypeScript is missing some Python features (rename, regex,
   conditional creation, range-based reads/writes)
2. **TypeScript-Specific**: Timestamps page addresses JavaScript precision issues
3. **Python-Specific**: Build Device Driver for hardware integration, async streamers
4. **Troubleshooting**: Migration strategy TBD - content is language-specific but needs
   a home in the new structure
5. **Console Tabs**: All console fragments currently empty, to be filled after refactor
