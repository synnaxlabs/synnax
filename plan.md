# Plan: Add HTTP Scan Task to Console

## Context

The HTTP device connection form already exists at `console/src/hardware/http/device/`.
The HTTP scan task is **user-configurable** (unlike other drivers where scan tasks are
internal/automatic). It needs a full task configuration form with start/stop controls.

### Scan Task Config (from RFC)

```json
{
  "device": "device_key",
  "autoStart": true,
  "rate": 0.1,
  "path": "/health",
  "method": "GET",
  "response": {
    "field": "/status",
    "expectedValue": "ok"
  }
}
```

Fields: `device`, `autoStart`, `rate` (Hz), `path`, `method` (GET/POST), and optional
`response` validation (`response.field` JSON pointer + `response.expectedValue`).

## Files to Create

### 1. `console/src/hardware/http/task/types.ts`

- Define `PREFIX = "http"`
- Define `SCAN_TYPE = "http_scan"`, `scanTypeZ`, `scanConfigZ`, `scanStatusDataZ`
- Define `SCAN_SCHEMAS` (task.Schemas)
- Define `ZERO_SCAN_CONFIG` and `ZERO_SCAN_PAYLOAD`
- Zod schema for scan config:
  - `device` (string, device key)
  - `autoStart` (boolean)
  - `rate` (number, positive, health check frequency in Hz)
  - `path` (string, endpoint path like `/health`)
  - `method` (enum: `"GET"` | `"POST"`)
  - `response` (optional object with `field` string and `expectedValue` string)

### 2. `console/src/hardware/http/task/Scan.tsx`

- Use `Common.Task.wrapForm` pattern for the scan task form
- **Properties section**: Device select, Rate field, AutoStart toggle
- **Form section**: Path input, Method select (GET/POST), optional Response Validation
  section with Field (JSON pointer) and Expected Value inputs
- Export `SCAN_LAYOUT` (using `Common.Task.LAYOUT` base, type `SCAN_TYPE`,
  icon `"Logo.HTTP"`)
- Export `ScanSelectable` for the task selector

### 3. `console/src/hardware/http/task/palette.tsx`

- Create `CreateScanCommand` palette command using `Palette.createSimpleCommand`
- Export `COMMANDS` array

### 4. `console/src/hardware/http/task/external.ts`

- Aggregate task exports: COMMANDS, EXTRACTORS, FILE_INGESTERS, LAYOUTS, SELECTABLES,
  ZERO_LAYOUTS
- For extractors/ingesters: use `Common.Task.extract` for SCAN_TYPE

### 5. `console/src/hardware/http/task/index.ts`

- Barrel re-export from external.ts

## Files to Modify

### 6. `console/src/hardware/http/external.ts`

- Add `export * from "@/hardware/http/task"` alongside existing device exports

### 7. `console/src/hardware/http/index.ts`

- Ensure Task is re-exported (may need to add namespace re-export)

### 8. `console/src/hardware/task/external.ts`

- Import `HTTP` from `@/hardware/http`
- Add `...HTTP.Task.COMMANDS` to COMMANDS
- Add `...HTTP.Task.EXTRACTORS` to EXTRACTORS
- Add `...HTTP.Task.FILE_INGESTERS` to FILE_INGESTERS (if applicable)
- Add `...HTTP.Task.LAYOUTS` to LAYOUTS

### 9. `console/src/hardware/task/layouts.ts`

- Import `HTTP` from `@/hardware/http`
- Add `...HTTP.Task.ZERO_LAYOUTS` to ZERO_LAYOUTS

### 10. `console/src/hardware/http/device/services/ontology.tsx`

- Add scan task to the device context menu items (like Modbus has read/write items)

## Implementation Notes

- The `onConfigure` callback for `wrapForm` needs to look up the device to get the rack
  key, following the pattern used by other task types
- Method select should use `Select.Buttons` component (GET/POST) similar to how
  auth type is selected in the device connection form
- Response validation should be toggled - hidden by default, shown when user enables it
- Rate field: use the `Common.Task.Fields.SampleRate`-like pattern but labeled
  "Health Check Rate" with Hz units