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
│   ├── series-and-frames.mdx           ✅ Complete
│   ├── ranges.mdx                      ✅ Complete
│   ├── streaming-data.mdx              🔄 In Progress
│   └── iterators.mdx                   🔄 In Progress
├── advanced/
│   ├── writers.mdx                     🔄 In Progress
│   ├── delete-data.mdx                 🔄 In Progress
│   └── timestamps.mdx                  🔄 In Progress (TS only)
└── resources/
    ├── examples.mdx                    🔄 In Progress
    ├── build-device-driver.mdx         🔄 In Progress (Python only)
    └── troubleshooting.mdx             🔄 In Progress
```


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
- [x] Series & Frames
- [x] Ranges (content transferred, needs cleanup)
- [🔄] Streaming Data (content transferred, needs cleanup)
- [🔄] Iterators (content transferred, needs cleanup)

### Phase 4: Advanced Topics (🔄 IN PROGRESS)

- [x] Create all shell pages
- [🔄] Writers (content transferred, needs cleanup)
- [🔄] Delete Data (content transferred, needs cleanup)
- [🔄] Timestamps (content transferred, needs cleanup)

### Phase 5: Resources (🔄 IN PROGRESS)

- [x] Create all shell pages
- [🔄] Examples page (content transferred, needs cleanup)
- [🔄] Build Device Driver (content transferred, needs cleanup)
- [🔄] Troubleshooting (content transferred, needs cleanup)

### Phase 6: Cleanup

- [ ] Add console tab content
- [ ] Final navigation review
- [ ] Remove/redirect old python-client and typescript-client pages

## Total Section Count

| Category          | Sections | Status            |
| ----------------- | -------- | ----------------- |
| Get Started       | 2        | ✅ Complete       |
| Fundamentals      | 3        | ✅ Complete       |
| Working with Data | 4        | 🔄 2 in progress  |
| Advanced Topics   | 3        | 🔄 3 in progress  |
| Resources         | 3        | 🔄 3 in progress  |
| **TOTAL**         | **15**   | **6 complete, 9 in progress** |

## Notes

1. **Language Parity**: TypeScript is missing some Python features (rename, regex,
   conditional creation, range-based reads/writes)
2. **TypeScript-Specific**: Timestamps page addresses JavaScript precision issues
3. **Python-Specific**: Build Device Driver for hardware integration, async streamers
4. **Troubleshooting**: Combined Python and TypeScript troubleshooting into unified page
   with language tabs. Some sections are Python-only or TypeScript-only.
5. **Console Tabs**: All console fragments currently empty, to be filled after refactor
