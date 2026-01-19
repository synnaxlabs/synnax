# Integration Test Implementation Plan

This document outlines the plan for implementing automated integration tests to cover
the QA steps defined in `.github/PULL_REQUEST_TEMPLATE/rc.md`. Tests are organized by
feature to enable incremental implementation across multiple Claude sessions.

---

## Development Workflow (CRITICAL)

### Incremental Test Development

**Every session MUST follow this pattern:**

1. **Write ONE test method** (or one helper method)
2. **Run in headed mode** to verify it passes
3. **Fix any issues** before proceeding
4. **Commit** the working test
5. **Repeat** for the next test

**DO NOT write multiple tests before running them.**

### Running Tests in Headed Mode

The test conductor runs tests via JSON sequence files. Always run in headed mode during development:

```bash
cd integration

# Run a specific sequence using Bazel-like target path
uv run tc //console/general/... --headed true

# Run tests matching a substring in a specific sequence
uv run tc //console/access/user --headed true

# Run all sequences in a test file
uv run tc //console/... --headed true

# Legacy syntax (still supported)
uv run tc -s console --headed true
```

### Target Path Syntax

Format: `//test_file/sequence/case_filter`

| Component | Description | Examples |
|-----------|-------------|----------|
| `test_file` | JSON file name (without `_tests.json`) | `console`, `driver` |
| `sequence` | Sequence name or `...` for all | `general`, `access`, `...` |
| `case_filter` | Substring to match in case path, or `...` for all | `channel`, `user`, `ni` |

The `case_filter` uses substring matching against the full case path (e.g., `channel` matches `console/channel/channel_operations`).

### Creating a Test Sequence JSON

Each feature needs a `*_tests.json` file in `tests/`:

```json
{
  "sequences": [
    {
      "sequence_name": "Channel_Tests",
      "sequence_order": "sequential",
      "tests": [
        {"case": "console/channel/channel_operations"}
      ]
    }
  ]
}
```

**File path mapping**: `{"case": "console/channel/channel_operations"}` → `tests/console/channel/channel_operations.py`

### Test Command Reference

```bash
cd integration

# Run specific sequence (Bazel-like syntax)
uv run tc //console/general/...

# Run tests matching a substring in a specific file/sequence
uv run tc //console/tasks/ni

# Run all tests in a file
uv run tc //console/...

# Filter across ALL test files (auto-discovers all *_tests.json)
uv run tc -f channel
uv run tc -f user

# Run in headed mode for development
uv run tc //console/general/... --headed true

# Legacy: Run single sequence
uv run tc -s console

# Legacy: Run multiple sequences
uv run tc -s console,driver

# Auto-discover all *_tests.json files
uv run tc
```

### Session Checklist

Before ending a session, verify:

- [ ] All new tests pass in headed mode
- [ ] No regressions in existing tests
- [ ] Code follows existing patterns in the codebase
- [ ] New test file added to appropriate `*_tests.json`

---

## Current State Analysis

### Existing Test Files

| Test File | Coverage |
|-----------|----------|
| `console/channel_lifecycle.py` | Channel create, rename, delete |
| `console/pages/open_close.py` | Create/close pages via command palette and (+) button |
| `console/pages/snap_to_grid.py` | Mosaic layout positioning |
| `console/schematic/alignment.py` | Symbol alignment, distribution, rotation |
| `console/schematic/edit_props.py` | Schematic/symbol properties, control acquire/release |
| `console/schematic/set_output.py` | Setpoint value control |
| `console/schematic/setpoint_press_user.py` | Valve control with setpoints, absolute control |
| `console/schematic/simple_press_valves.py` | Valve actuation sequences |
| `console/line-plot/download_csv.py` | CSV download from plot |
| `console/task/ni_analog_read_forms.py` | NI analog read task forms |
| `console/task/ni_analog_write_forms.py` | NI analog write task forms |
| `console/task/ni_counter_read_forms.py` | NI counter read task forms |
| `console/task/no_device.py` | Task status without device |

### Existing Console Automation Helpers

| Module | Methods |
|--------|---------|
| `console/console.py` | command_palette, create_page, close_page, select_from_dropdown, fill_input_field, click, check_for_notifications, screenshot |
| `console/channels.py` | create, rename, delete, list_all, existing_channel |
| `console/page.py` | ConsolePage base class with move, screenshot, close, get_value |
| `console/plot.py` | Plot class with add_channels, add_ranges, download_csv, set_axis |
| `console/log.py` | Log class with set_channel |
| `console/schematic/schematic.py` | Schematic class with create_symbol, align, distribute, rotate, acquire_control, release_control, set_authority, set_properties |
| `console/schematic/symbol.py` | Base Symbol class |
| `console/schematic/valve.py`, `button.py`, `setpoint.py`, `value.py` | Symbol implementations |
| `console/task/analog_read.py` | AnalogRead task page |

---

## QA Coverage Gap Analysis

The following sections map each RC QA step to existing/needed tests.

### Legend
- **[DONE]** - Covered by existing test
- **[PARTIAL]** - Partially covered
- **[TODO]** - Needs implementation

---

## ⚠️ ALL SESSIONS: Development Rules

**Every session MUST:**

1. **Create JSON sequence file first** (e.g., `tests/channel_tests.json`)
2. Run tests in **headed mode** with `--headed true`
3. Write **ONE test at a time**, verify it passes before writing the next
4. Use the test command:
   ```bash
   cd integration
   # Run specific sequence
   uv run tc //console/general/... --headed true

   # Run tests matching a substring
   uv run tc //console/access/user --headed true
   ```
5. Never write multiple untested methods/tests in one go

---

## Session 1: Channels Feature

### QA Steps Status

| QA Step | Status | Existing Test |
|---------|--------|---------------|
| Create channel from command palette | [DONE] | `channel_lifecycle.py` |
| Create multiple channels with "Create More" flag | [TODO] | - |
| Open channel plot by double-clicking | [TODO] | - |
| Drag channel onto line plot | [TODO] | - |
| Rename channel | [DONE] | `channel_lifecycle.py` |
| Delete channel | [DONE] | `channel_lifecycle.py` |
| Group multiple channels | [TODO] | - |
| Create calculated channel | [TODO] | - |
| Set channel alias under range | [TODO] | - |
| Copy link to channel | [TODO] | - |

### Step-by-Step Implementation

Follow this exact order, running tests after each step:

#### Step 1: Create test sequence JSON
```bash
# Create tests/channel_tests.json
```
```json
{
  "sequences": [
    {
      "sequence_name": "Channel_Operations",
      "sequence_order": "sequential",
      "tests": [
        {"case": "console/channel/channel_operations"}
      ]
    }
  ]
}
```

#### Step 2: Create shared context menu helper
```bash
# Create console/context_menu.py with ContextMenu class
# No test needed - this is infrastructure
```

#### Step 3: Create test file and add first test
```python
# Create tests/console/channel/channel_operations.py
class ChannelOperations(ConsoleCase):
    def run(self) -> None:
        self.test_open_channel_plot()

    def test_open_channel_plot(self) -> None:
        # Create channel, double-click to open plot, verify plot opened
```
```bash
# Run to verify:
cd integration
uv run tc //channel/Channel_Operations/... --headed true
```

#### Step 4: Add `open_plot()` to channels.py
```python
# In console/channels.py, add:
def open_plot(self, name: str) -> None:
    """Open channel plot by double-clicking channel in list."""
```
```bash
# Run to verify test passes:
uv run tc //channel/Channel_Operations/... --headed true
```

#### Step 5: Add `group()` method + test
```python
# In console/channels.py, add:
def group(self, names: list[str], group_name: str) -> None:
    """Group multiple channels via context menu."""

# In channel_operations.py, add to run():
def test_group_channels(self) -> None:
    # Create multiple channels, select them, group via context menu
```
```bash
# Run to verify:
uv run tc //channel/Channel_Operations/... --headed true
```

#### Continue pattern for remaining methods...

### Methods to Add to `console/channels.py`

| Method | Test After |
|--------|------------|
| `open_plot(name)` | Yes |
| `group(names, group_name)` | Yes |
| `copy_link(name) -> str` | Yes |
| `create_with_create_more(channels)` | Yes |
| `drag_to_plot(name, plot)` | Yes |

### Final Test File Structure

```python
# tests/console/channel/channel_operations.py
class ChannelOperations(ConsoleCase):
    def run(self) -> None:
        self.test_open_channel_plot()
        self.test_group_channels()
        self.test_copy_channel_link()
        self.test_create_multiple_channels()
        self.test_drag_to_plot()
```

---

## Session 2: Clusters Feature

> **IMPORTANT**: Follow the incremental workflow from Session 1.
> Write ONE method → Run test (headed) → Verify pass → Repeat.

### QA Steps Status

| QA Step | Status |
|---------|--------|
| Connect to cluster | [TODO] |
| Add new cluster | [TODO] |
| Disconnect from cluster | [TODO] |
| Rename cluster | [TODO] |
| Remove cluster | [TODO] |
| Copy link to cluster | [TODO] |
| Connection failure feedback | [TODO] |

### Step-by-Step Implementation

1. Create `console/cluster.py` with empty `ClusterClient` class
2. Add `connect()` method → write test → run headed → verify
3. Add `disconnect()` method → write test → run headed → verify
4. Add `add_cluster()` method → write test → run headed → verify
5. Add `rename()` method → write test → run headed → verify
6. Add `remove()` method → write test → run headed → verify
7. Add `copy_link()` method → write test → run headed → verify

### Methods to Implement

```python
class ClusterClient:
    def connect(self, host: str, port: int, username: str, password: str) -> bool
    def disconnect(self) -> None
    def add_cluster(self, name: str, host: str, port: int) -> None
    def rename(self, old_name: str, new_name: str) -> None
    def remove(self, name: str) -> None
    def copy_link(self, name: str) -> str
```

### Test File: `tests/console/cluster/cluster_operations.py`

---

## Session 3: Login & Authentication

### QA Steps Status

| QA Step | Status |
|---------|--------|
| Log in with valid credentials | [DONE] | `console/case.py` setup |
| Receive error with invalid credentials | [TODO] |
| Log out via dropdown | [TODO] |
| See login screen after logout | [TODO] |

### Implementation Tasks

1. **Create `console/auth.py`**
2. **Create `tests/console/auth/login_logout.py`**

---

## Session 4: Labels Feature

### QA Steps Status

| QA Step | Status |
|---------|--------|
| Add new label | [TODO] |
| Edit label name | [TODO] |
| Change label color | [TODO] |
| Label syncs with range toolbar | [TODO] |

### Implementation Tasks

1. **Create `console/labels.py`**
2. **Create `tests/console/label/label_lifecycle.py`**

---

## Session 5: Layout Feature

### QA Steps Status

| QA Step | Status | Existing Test |
|---------|--------|---------------|
| Drag mosaic leaf into new window | [TODO] | - |
| Rename tab by double-clicking | [TODO] | - |
| Close tabs via close icon | [DONE] | `pages/open_close.py` |
| Split leaf horizontally | [TODO] | - |
| Split leaf vertically | [TODO] | - |
| Focus on leaf | [TODO] | - |
| Move leaf positions | [DONE] | `pages/snap_to_grid.py` |
| Keyboard shortcuts (Cmd+W, Cmd+E, etc.) | [TODO] | - |

### Implementation Tasks

1. **Create `console/layout.py`**:
   ```python
   class LayoutClient:
       def split_horizontal(self, tab_name: str) -> None
       def split_vertical(self, tab_name: str) -> None
       def focus(self, tab_name: str) -> None
       def rename_tab(self, old_name: str, new_name: str) -> None
       def move_to_new_window(self, tab_name: str) -> None
   ```

2. **Create `tests/console/layout/mosaic_operations.py`**
3. **Create `tests/console/layout/keyboard_shortcuts.py`**

---

## Session 6: Line Plots Feature

### QA Steps Status

| QA Step | Status | Existing Test |
|---------|--------|---------------|
| Create new line plot | [DONE] | `pages/open_close.py` |
| Add channels to axes | [DONE] | `line-plot/download_csv.py` |
| Add time ranges | [DONE] | `line-plot/download_csv.py` |
| Download CSV | [DONE] | `line-plot/download_csv.py` |
| Set axis configuration | [PARTIAL] | `plot.py` has methods |
| Move channels between axes | [TODO] | - |
| Create range from selection | [TODO] | - |
| Export plot | [TODO] | - |
| Copy link to plot | [TODO] | - |
| Rename plot (context menu) | [TODO] | - |
| Delete plot | [TODO] | - |

### Implementation Tasks

1. **Enhance `console/plot.py`**:
   ```python
   def move_channel_to_axis(self, channel: str, from_axis: Axis, to_axis: Axis) -> None
   def create_range_from_selection(self, start: float, end: float, name: str) -> None
   def export(self, path: str) -> None
   def copy_link(self) -> str
   ```

2. **Create `tests/console/line-plot/plot_context_menu.py`**
3. **Create `tests/console/line-plot/plot_axes.py`**

---

## Session 7: Logs Feature

### QA Steps Status

| QA Step | Status |
|---------|--------|
| Create new log | [DONE] | `snap_to_grid.py` uses Log |
| Set channel | [DONE] | `log.py` has set_channel |
| Stream virtual channel data | [TODO] |
| Pause/resume scrolling | [TODO] |
| Copy link | [TODO] |
| Rename log | [TODO] |
| Delete log | [TODO] |

### Implementation Tasks

1. **Enhance `console/log.py`**:
   ```python
   def pause_streaming(self) -> None
   def resume_streaming(self) -> None
   def scroll_to_history(self, offset: int) -> None
   def copy_link(self) -> str
   ```

2. **Create `tests/console/log/log_streaming.py`**
3. **Create `tests/console/log/log_context_menu.py`**

---

## Session 8: Ranges Feature (Core)

### QA Steps Status

| QA Step | Status |
|---------|--------|
| Create local range | [TODO] |
| Create persisted range | [TODO] |
| Create range with parent | [TODO] |
| Add labels to range | [TODO] |
| Rename range | [TODO] |
| Navigate to parent | [TODO] |
| Copy Python/TypeScript code | [TODO] |
| Download data as CSV | [TODO] |
| Favorite/unfavorite range | [TODO] |
| Change start/end times | [TODO] |
| Set/update metadata | [TODO] |

### Implementation Tasks

1. **Create `console/ranges.py`**:
   ```python
   class RangeClient:
       def create(self, name: str, start: sy.TimeStamp, end: sy.TimeStamp,
                  persisted: bool = False, parent: str | None = None,
                  labels: list[str] | None = None) -> bool
       def rename(self, old_name: str, new_name: str) -> None
       def set_times(self, name: str, start: sy.TimeStamp, end: sy.TimeStamp) -> None
       def favorite(self, name: str) -> None
       def unfavorite(self, name: str) -> None
       def copy_code(self, name: str, language: str) -> str  # "python" or "typescript"
       def download_csv(self, name: str) -> str
       def set_metadata(self, name: str, key: str, value: str) -> None
       def add_label(self, range_name: str, label: str) -> None
   ```

2. **Create `tests/console/range/create_range.py`**
3. **Create `tests/console/range/range_details.py`**
4. **Create `tests/console/range/range_metadata.py`**

---

## Session 9: Ranges Feature (Advanced)

### QA Steps Status

| QA Step | Status |
|---------|--------|
| Create child ranges | [TODO] |
| Navigate to child | [TODO] |
| Change stage of child | [TODO] |
| Range explorer search | [TODO] |
| Filter by labels | [TODO] |
| Range toolbar operations | [TODO] |
| Switch active range | [TODO] |

### Implementation Tasks

1. **Enhance `console/ranges.py`** with child and explorer methods
2. **Create `tests/console/range/range_children.py`**
3. **Create `tests/console/range/range_explorer.py`**

---

## Session 10: Schematics Feature (Enhancements)

### QA Steps Status (Beyond Existing)

| QA Step | Status | Existing Test |
|---------|--------|---------------|
| Create schematic | [DONE] | Multiple tests |
| Add symbols | [DONE] | Multiple tests |
| Align symbols | [DONE] | `alignment.py` |
| Distribute symbols | [DONE] | `alignment.py` |
| Rotate symbols | [DONE] | `alignment.py` |
| Edit symbol properties | [DONE] | `edit_props.py` |
| Acquire/release control | [DONE] | `edit_props.py`, `setpoint_press_user.py` |
| Set control authority | [DONE] | `edit_props.py`, `setpoint_press_user.py` |
| Actuate valves | [DONE] | `simple_press_valves.py`, `setpoint_press_user.py` |
| Set setpoint values | [DONE] | `set_output.py`, `setpoint_press_user.py` |
| Connect symbols | [DONE] | `set_output.py` |
| Absolute control toggle | [DONE] | `setpoint_press_user.py` |
| Export schematic | [TODO] | - |
| Copy link | [TODO] | - |
| Snapshot to range | [TODO] | - |
| Delete schematic | [TODO] | - |

### Implementation Tasks

1. **Enhance `console/schematic/schematic.py`**:
   ```python
   def export(self, path: str) -> None
   def copy_link(self) -> str
   def snapshot_to_range(self, range_name: str) -> None
   def delete(self) -> None
   ```

2. **Create `tests/console/schematic/schematic_context_menu.py`**

---

## Session 11: Custom Symbols Feature

### QA Steps Status

| QA Step | Status |
|---------|--------|
| Create new custom symbol | [TODO] |
| Rename symbol | [TODO] |
| Add handles to symbol | [TODO] |
| Add color regions | [TODO] |
| Import symbol group | [TODO] |
| Export symbol group | [TODO] |
| Use custom symbol as actuator | [TODO] |

### Implementation Tasks

1. **Create `console/symbol_editor.py`**
2. **Create `tests/console/schematic/custom_symbols.py`**

---

## Session 12: Tables Feature

### QA Steps Status

| QA Step | Status |
|---------|--------|
| Create new table | [TODO] |
| Add rows/columns | [TODO] |
| Delete rows/columns | [TODO] |
| See live data | [TODO] |
| Add redlines | [TODO] |
| Rename table | [TODO] |
| Delete table | [TODO] |
| Export table | [TODO] |
| Import table | [TODO] |

### Implementation Tasks

1. **Create `console/table.py`**:
   ```python
   class Table(ConsolePage):
       page_type: str = "Table"
       pluto_label: str = ".pluto-table"

       def add_row(self) -> None
       def add_column(self) -> None
       def delete_row(self, index: int) -> None
       def delete_column(self, index: int) -> None
       def set_cell_value(self, row: int, col: int, value: str) -> None
       def add_redline(self, row: int, col: int, condition: str) -> None
       def export(self, path: str) -> None
   ```

2. **Create `tests/console/table/table_lifecycle.py`**

---

## Session 13: Tasks Feature (Enhancements)

### QA Steps Status

| QA Step | Status | Existing Test |
|---------|--------|---------------|
| Open task config | [DONE] | `no_device.py` |
| Configure task | [DONE] | `no_device.py` |
| Run task | [DONE] | `no_device.py` |
| Check task status | [DONE] | `no_device.py` |
| Add channels to task | [DONE] | `no_device.py` |
| Task forms validation | [DONE] | `ni_*.py` forms tests |
| Snapshot to range | [TODO] | - |
| Rename task | [TODO] | - |
| Delete task | [TODO] | - |
| Export/import task | [TODO] | - |
| Pause/play task | [TODO] | - |

### Implementation Tasks

1. **Enhance `console/task/` module** with context menu operations
2. **Create `tests/console/task/task_context_menu.py`**
3. **Create `tests/console/task/task_import_export.py`**

---

## Session 14: Users & Permissions

### QA Steps Status

| QA Step | Status |
|---------|--------|
| Register new user | [TODO] |
| Rename user | [TODO] |
| Delete user | [TODO] |
| Change username | [TODO] |
| Permission restrictions | [TODO] |

### Implementation Tasks

1. **Create `console/users.py`**
2. **Create `tests/console/user/user_management.py`**
3. **Create `tests/console/user/permissions.py`**

---

## Session 15: Workspaces Feature

### QA Steps Status

| QA Step | Status |
|---------|--------|
| Create workspace | [TODO] |
| Switch workspaces | [TODO] |
| Rename workspace | [TODO] |
| Delete workspace | [TODO] |
| Export/import workspace | [TODO] |
| Clear workspaces | [TODO] |

### Implementation Tasks

1. **Create `console/workspaces.py`**
2. **Create `tests/console/workspace/workspace_lifecycle.py`**

---

## Session 16: Devices & Racks

### QA Steps Status

| QA Step | Status |
|---------|--------|
| See device state | [TODO] |
| Rename device | [TODO] |
| Delete device | [TODO] |
| Configure device | [TODO] |
| See rack state | [TODO] |
| Rename rack | [TODO] |
| Delete rack | [TODO] |

### Implementation Tasks

1. **Create `console/devices.py`**
2. **Create `console/racks.py`**
3. **Create `tests/console/device/device_operations.py`**

---

## Session 17: Arc & Statuses

### QA Steps Status

| QA Step | Status |
|---------|--------|
| Create arc automation | [TODO] |
| Open arc editor | [TODO] |
| Start/stop arc | [TODO] |
| Deploy arc | [TODO] |
| Create status | [TODO] |
| Filter statuses by label | [TODO] |
| Delete statuses | [TODO] |

### Implementation Tasks

1. **Create `console/arc.py`**
2. **Create `console/statuses.py`**
3. **Create `tests/console/arc/arc_lifecycle.py`**
4. **Create `tests/console/status/status_lifecycle.py`**

---

## Session 18: Control Sequences

### QA Steps Status

| QA Step | Status |
|---------|--------|
| Create control sequence | [TODO] |
| See channel autocomplete | [TODO] |
| Configure read_from/write_to | [TODO] |
| Set control rate | [TODO] |
| Run control sequence | [TODO] |

### Implementation Tasks

1. **Create `console/control_sequence.py`**
2. **Create `tests/console/control/control_sequence_lifecycle.py`**

---

## Session 19: Documentation & Ontology

### QA Steps Status

| QA Step | Status |
|---------|--------|
| Open documentation | [TODO] |
| Create ontology group | [TODO] |
| Move resources to group | [TODO] |
| Rename/delete group | [TODO] |

### Implementation Tasks

1. **Create `console/documentation.py`**
2. **Create `console/ontology.py`**
3. **Create `tests/console/misc/documentation.py`**
4. **Create `tests/console/misc/ontology.py`**

---

## Recommended Implementation Order

### Phase 1: High-Value Coverage (Sessions 1, 6, 8)
Build on existing strong foundation. Adds ranges (critical missing feature).

### Phase 2: Core Resources (Sessions 2, 3, 5, 12)
Clusters, auth, layout, tables - frequently used features.

### Phase 3: Enhanced Features (Sessions 4, 7, 9, 10, 13)
Labels, logs, advanced ranges, schematic enhancements, tasks.

### Phase 4: Advanced Features (Sessions 14-19)
Users, workspaces, devices, arc, control sequences, ontology.

---

## Shared Patterns to Implement First

### 1. Context Menu Helper

```python
# console/context_menu.py
class ContextMenu:
    def __init__(self, page: Page):
        self.page = page

    def open_on(self, element: Locator) -> "ContextMenu":
        element.click(button="right")
        self.page.locator(".pluto-menu").wait_for(state="visible")
        return self

    def click_option(self, text: str) -> None:
        self.page.get_by_text(text, exact=True).first.click()

    def has_option(self, text: str) -> bool:
        return self.page.get_by_text(text, exact=True).count() > 0
```

### 2. Resource Toolbar Base

```python
# console/resource_toolbar.py
class ResourceToolbar:
    resource_button_icon: str  # e.g., "pluto-icon--channel"
    resource_pane_text: str    # e.g., "Channels"

    def show(self) -> None
    def hide(self) -> None
    def find_item(self, name: str) -> Locator
    def context_menu(self, name: str) -> ContextMenu
```

### 3. Sync Assertion Helper

```python
# console/sync.py
class SyncAsserter:
    def assert_name_visible_in(
        self,
        name: str,
        locations: list[str]  # ["resources", "tab", "toolbar"]
    ) -> None
```

---

## Test JSON Configuration

Each feature should have its own test sequence file:

```
integration/tests/
├── channel_tests.json      # Session 1
├── cluster_tests.json      # Session 2
├── auth_tests.json         # Session 3
├── label_tests.json        # Session 4
├── layout_tests.json       # Session 5
├── line_plot_tests.json    # Session 6
├── log_tests.json          # Session 7
├── range_tests.json        # Sessions 8-9
├── schematic_tests.json    # Sessions 10-11
├── table_tests.json        # Session 12
├── task_tests.json         # Session 13
├── user_tests.json         # Session 14
├── workspace_tests.json    # Session 15
├── device_tests.json       # Session 16
├── arc_tests.json          # Session 17
├── control_tests.json      # Session 18
└── misc_tests.json         # Session 19
```

---

## Summary Statistics

| Category | Total QA Steps | Covered | TODO |
|----------|---------------|---------|------|
| Channels | 11 | 3 | 8 |
| Clusters | 7 | 0 | 7 |
| Login/Auth | 4 | 1 | 3 |
| Labels | 4 | 0 | 4 |
| Layout | 8 | 2 | 6 |
| Line Plots | 11 | 4 | 7 |
| Logs | 7 | 2 | 5 |
| Ranges | 18 | 0 | 18 |
| Schematics | 16 | 12 | 4 |
| Custom Symbols | 7 | 0 | 7 |
| Tables | 9 | 0 | 9 |
| Tasks | 11 | 6 | 5 |
| Users/Perms | 5 | 0 | 5 |
| Workspaces | 6 | 0 | 6 |
| Devices/Racks | 7 | 0 | 7 |
| Arc/Statuses | 7 | 0 | 7 |
| Control Seq | 5 | 0 | 5 |
| Docs/Ontology | 4 | 0 | 4 |
| **TOTAL** | **~147** | **~30** | **~117** |

**Coverage: ~20% of RC QA steps currently automated**
