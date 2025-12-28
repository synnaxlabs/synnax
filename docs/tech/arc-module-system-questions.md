# Arc Module System - Decision Questions

This document contains the questions we need to answer to finalize Arc's module system design.

---

## Category 1: Module Identity

**Core Question**: How is a module identified and named?

### Context Questions
1. Where do Arc programs live today? (Database only? Files too? Both?)
2. Do users ever work with Arc files on disk, or is everything through Console UI?
3. Should the same Arc program be usable across multiple Synnax clusters?
4. Do we anticipate users sharing Arc code outside of Synnax (GitHub, etc.)?

### Design Questions
5. Should `std` modules feel different from user modules syntactically?
6. Is there a concept of "projects" or "workspaces" grouping related Arc programs?
7. Should module names be globally unique, or scoped to a project/cluster?
8. How do we handle name collisions (two modules named "helpers")?

### User Experience Questions
9. What's the mental model for a hardware engineer? "I'm importing a library" vs "I'm referencing another script"?
10. Should imports be human-readable names or technical identifiers (UUIDs)?

---

## Category 2: Import Syntax

**Core Question**: What does the import statement look like?

### Syntax Questions
1. Full path or short qualifier? `std.math.sqrt()` vs `math.sqrt()`?
2. Should we allow importing individual items? `import std.math.sqrt`?
3. Should aliasing be supported from day one? `import std.math as m`?
4. What separator? Dots (`std.math`), slashes (`std/math`), or colons (`std::math`)?

### Usability Questions
5. How verbose is too verbose for your target users?
6. Should common things require less typing than rare things?
7. Do users need to see where a function comes from at the call site?

### Consistency Questions
8. Should import syntax mirror how you access items? (`import std.math` → `math.foo()`)
9. Are there existing patterns in Arc syntax we should align with?

---

## Category 3: Visibility & Exports

**Core Question**: How do modules control what's public?

### Philosophy Questions
1. Are Arc modules typically small (10-20 functions) or large (100+ functions)?
2. Do users need to hide "internal helper" functions, or is everything fair game?
3. Is encapsulation a priority, or is simplicity more important?

### Syntax Questions
4. If we add visibility control, which default? Public or private?
5. Is a keyword (`pub`) acceptable, or too much ceremony for Arc users?
6. Should visibility apply to functions only, or also variables/constants?

### Evolution Questions
7. Can we start with "everything public" and add visibility later without breaking?
8. What would trigger the need for visibility control? Scale? Collaboration?

---

## Category 4: Module Resolution

**Core Question**: How does the compiler find modules?

### Storage Questions
1. Where does the standard library "live"? Compiled into runtime? Fetched from server?
2. Can users create reusable modules, or is all user code single-file?
3. If user modules exist, where are they stored? Same database as Arc programs?

### Compilation Questions
4. Is module resolution at compile-time only, or could it happen at deploy-time?
5. Should imported modules be "frozen" at import time, or always use latest?
6. How do we handle a module being deleted/renamed after it's imported?

### Network Questions
7. Is network access during compilation acceptable? (Database fetch)
8. Should there be an offline mode where stdlib is sufficient?

---

## Category 5: Compilation Model

**Core Question**: How do modules compile together?

### Performance Questions
1. How large are typical Arc programs? (Lines of code, number of functions)
2. How often do users recompile? (Every edit? Only on deploy?)
3. Is compilation speed a concern, or are programs small enough it doesn't matter?

### Output Questions
4. Single WASM module per program, or multiple linked modules?
5. Should the same stdlib module compile once and be reused, or inline every time?
6. How does the C++ driver runtime affect compilation decisions?

### Caching Questions
7. Should we cache compiled modules? Where? (Server? Client? Both?)
8. How do we invalidate cache when a module changes?

---

## Category 6: Standard Library Organization

**Core Question**: What's in the stdlib and how is it structured?

### Scope Questions
1. What functions do hardware engineers use most often?
2. What's currently missing that users have asked for?
3. Should stdlib include domain-specific things (PID controllers, unit conversions)?

### Organization Questions
4. Flat namespace (`std.sqrt`) or hierarchical (`std.math.sqrt`)?
5. How many modules is too many? How few is too limiting?
6. Should related functions be grouped (all trig in `std.math`) or split (separate `std.trig`)?

### Growth Questions
7. How do we add new stdlib modules without breaking existing code?
8. Should stdlib be versioned separately from Arc language version?

---

## Category 7: Standard Library Implementation

**Core Question**: How is the stdlib implemented?

### Implementation Questions
1. Are all stdlib functions intrinsics (Go/C++ host functions), or can some be pure Arc?
2. Which functions MUST be intrinsics? (I/O, time, channels)
3. Which functions COULD be pure Arc? (Math utilities, data transformations)

### Maintenance Questions
4. Who maintains stdlib? Core team only? Community contributions?
5. Should stdlib source be visible to users (for learning/debugging)?
6. How do we document stdlib functions?

### Bootstrap Questions
7. Can Arc compile itself (stdlib in Arc), or is that overkill?
8. If we add pure-Arc stdlib functions, how do we test them?

---

## Category 8: Circular Dependencies

**Core Question**: How are circular imports handled?

### Use Case Questions
1. Can you think of a legitimate reason for circular imports in Arc?
2. Do hardware automation patterns ever naturally create cycles?

### Error Handling Questions
3. If forbidden, what error message helps users understand and fix it?
4. Should we suggest how to restructure to break the cycle?

### Implementation Questions
5. How hard is cycle detection to implement?
6. Does the reactive/flow nature of Arc make cycles more or less likely?

---

## Meta Questions (Across All Categories)

### Prioritization
1. Which decisions are foundational (must decide first)?
2. Which can be deferred to Phase 2/3?
3. Which decisions are reversible vs. permanent?

### User Research
4. Have users asked for any of this? What pain points exist today?
5. Can we get feedback on proposals before implementing?

### Implementation Order
6. What's the minimum viable module system?
7. What can we ship in weeks vs. months?

---

## Decision Dependencies

```
Category 1 (Identity)
    ↓
Category 2 (Syntax) ←→ Category 4 (Resolution)
    ↓                      ↓
Category 3 (Visibility)   Category 5 (Compilation)
    ↓                      ↓
Category 6 (Stdlib Org) ←→ Category 7 (Stdlib Impl)
    ↓
Category 8 (Cycles)
```

**Start with Category 1** - all other decisions flow from how we identify modules.
