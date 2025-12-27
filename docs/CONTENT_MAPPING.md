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
│   ├── streaming-data.mdx              ✅ Complete
│   └── iterators.mdx                   ✅ Complete
├── advanced/
│   ├── writers.mdx                     ✅ Complete
│   ├── delete-data.mdx                 ✅ Complete
│   └── time-types.mdx                  ✅ Complete
└── resources/
    ├── examples.mdx                    ✅ Complete
    ├── build-device-driver.mdx         ✅ Complete
    └── troubleshooting.mdx             ✅ Complete
```


## Implementation Checklist

### Phase 1: Get Started (✅ COMPLETE)

- [x] Quick Start
- [x] Authentication
- [x] Update navigation structure

### Phase 2: Fundamentals (✅ COMPLETE)

- [x] Channels
- [x] Read Data
- [x] Write Data

### Phase 3: Working with Data (✅ COMPLETE)

- [x] Create all shell pages
- [x] Series & Frames
- [x] Ranges
- [x] Streaming Data
- [x] Iterators

### Phase 4: Advanced Topics (✅ COMPLETE)

- [x] Create all shell pages
- [x] Writers
- [x] Delete Data
- [x] Time Types

### Phase 5: Resources (✅ COMPLETE)

- [x] Create all shell pages
- [x] Examples
- [x] Build Device Driver
- [x] Troubleshooting

### Phase 6: Cleanup

- [x] Final navigation review
- [x] Examples validation/review
- [x] Remove/redirect old python-client and typescript-client pages

## Total Section Count

| Category          | Sections | Status        |
| ----------------- | -------- | ------------- |
| Get Started       | 2        | ✅ Complete   |
| Fundamentals      | 3        | ✅ Complete   |
| Working with Data | 4        | ✅ Complete   |
| Advanced Topics   | 3        | ✅ Complete   |
| Resources         | 3        | ✅ Complete   |
| **TOTAL**         | **15**   | **15 complete** |

## Notes

1. **Language Parity**: Mirrored throughout docs
2. **TypeScript-Specific**: Timestamps page addresses JavaScript precision issues (mirrored to py)
3. **Python-Specific**: Build Device Driver for hardware integration, async streamers (mirrored to ts)
4. **Troubleshooting**: Combined Python and TypeScript troubleshooting into unified page
   with language tabs. Some sections are Python-only or TypeScript-only.

