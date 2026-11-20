# 116 Shortcuts

- **Author**: Emiliano Bonilla
- **Date**: 2026-11-20
- **Related**: [RFC 0041 - Action-based undo and redo](0041-action-based-undo-redo.md),
  [RFC 0050 - Console multi-window](0050-console-multi-window.md),
  [RFC 0066 - Lyra](0066-lyra-primitives-package.md),
  [RFC 0103 - Strongly typed task fields](0103-typed-task-fields.md),
  [RFC 0114 - Schematic symbol model](0114-schematic-symbol-model.md)

## 0 Summary

Every Console shortcut is a named command in one list, and a key is data that points at
a command. The area that owns the state provides the action, and the focused area wins.
The palette, menus, tooltips, the macOS menu bar, a shortcut sheet, and a Settings page
read keys only from that list. A schematic holds a list of control keys that drive its
symbols while the operator holds control of it.

## 1 Motivation

- **A shortcut is a hook call**: `Triggers.use` binds a key where it is called
  (`lyra/src/triggers/hooks.ts:81-143`). Each hint passes the keys again, so hints
  drift: the tab menu shows Ctrl+E and also binds it, and one press renames two tabs
  (`console/src/feature/panel/ContextMenu.tsx:33`,
  `console/src/app/triggers/use.ts:126`).
- **Commands carry no keys**: `Command.create({ trigger })` renders a hint and binds
  nothing (`console/src/platform/command/command.tsx:64-90`). 1 of 49 commands sets it.
- **The engine keeps a set of held keys**: Keys come from `e.code`, Meta becomes
  Control, and chords are unordered sets (`lyra/src/triggers/triggers.ts:235-245`).
  macOS sends no keyup while Cmd is held, and Safari drops Shift keyups, so the Provider
  patches the set by hand (`lyra/src/triggers/Provider.tsx:172-181`).
- **Priority is a number**: Tooltip 200, menu 150, code editor 120, and dialog 100 stand
  in for "the focused area wins".
- **Users cannot change or find keys**: No setting, no stored binding, no list of all
  shortcuts. Single-letter toolbar keys cannot be turned off (WCAG 2.1.4).
- **No key drives a control**: The diagram blocks Space and Enter from actuating a
  focused control (`pluto/src/vis/diagram/Diagram.tsx:507-527`, SY-4825). SY-2473 asks
  for a key to take control.

## 2 Vocabulary

- **Command**: A named action with an icon and default keys, such as `tab.rename`.
- **Chord**: One key plus modifiers. **Mod** is ⌘ on macOS and Ctrl elsewhere;
  **Control** is ⌃ and exists only on macOS.
- **Binding**: One chord, or a sequence of chords such as `C` then `C`.
- **Area**: A part of the page that provides commands, such as a window, a tab, or a
  modal. Areas nest.
- **Control key**: A key that a schematic binds to one of its symbols.

## 3 Principles

1. **One list**: A key that a person can see, run, or change is a command in the list.
   Arrow keys in a list, Enter in a field, and drag gestures are not commands.
2. **Same action, every input**: A key, the palette, a menu item, and a click run the
   same action, so they cannot do different things.
3. **A key never changes meaning across modes**: A control key never means anything
   else, anywhere (NUREG-0700 Rev 3 §7.3.1).

## 4 Design

### 4.0 Commands

The Console defines its commands as a module-level `const` list. `Command.create` gains
`keys` and a menu group; its hint-only `trigger` goes. A command is available where the
focused area, or an area around it, provides it. A schematic provides `schematic.group`,
so its key and its palette entry exist only while a schematic has focus. The window
provides the global commands. `useVisible` stays as the permission and platform gate. A
command can take a target, so the palette's per-result actions are commands with the
highlighted resource as the target.

The palette lists available commands, the ones the focused area provides first, and
shows each key. Menu items name a command and bind no key of their own.

### 4.1 Chords and matching

Lyra `Shortcut` replaces `Triggers`. It matches each `keydown` alone, with modifiers
read from the event. A key is the typed character for letters, digits, and punctuation,
with `e.code` as the fallback when the character names no ASCII key, and the key name
otherwise (`Enter`, `F3`). It skips `isComposing` events and key repeat, unless a
binding opts in to repeat. A sequence waits a short time for its next chord and shows
the pending keys. A held-key tracker remains for gestures only (Shift+drag, Alt+wheel),
reads modifiers from pointer events, and clears on blur.

### 4.2 Areas

`Shortcut.Area` grows out of `Triggers.Scope`: it is active while its condition holds
and its parent is active. A key runs the command in the deepest active area that
provides it; at equal depth, the one mounted last wins. The modal stack and the tab
mosaic mount areas as they mount scopes today. In a text field, single keys and Shift or
Alt chords go to the field, the field keeps Mod+A, Mod+C, Mod+V, Mod+X, Mod+Z, and
Mod+Shift+Z, and other Mod chords and Escape reach commands.

### 4.3 Hints, the sheet, and the menu bar

`Shortcut.Text` takes a command and renders its current keys with the platform's symbols
(`docs/claude/ux.md`, keycaps). ⌘/ opens the shortcut sheet, a floating dialog that is
also a palette command. It lists the keys of the focused area first, then the global
keys, then gestures, and in a schematic the control keys. It searches by name or by
pressed keys and links to Settings.

On macOS, the desktop Console builds its menu bar from the list with
`@tauri-apps/api/menu` in place of Tauri's default menu: File, Edit, View, Window, and
Help. An item is gray where no active area provides it. A menu key reaches the menu
before the page, so Edit items act on a focused text field first, and Mod+W, Mod+N, and
Mod+Q belong to their items. Windows keeps its custom title bar.

### 4.4 Changing shortcuts

Settings, Console, Keyboard shortcuts lists every command with its keys. A person
searches by name or by pressing keys, clicks a command and presses new keys, and resets
a command to its default. A key that another command in an overlapping area uses names
that command and asks before it takes the key. F1 to F12 are refused (§4.5). A switch
turns off every binding with no modifier. Only changes are saved, in a global-scope
session slice beside the theme (`console/src/session/store.ts:62-68`), because a
keyboard layout belongs to a computer.

### 4.5 Control keys

A schematic holds one key list, set by anyone who can edit the schematic:

```
ControlKey struct {
    function    uint8    // 1 to 12
    shifted     bool
    node        string
    interaction string?  // RFC 0114 interaction key; none means the node's action
}

Schematic struct {
    keys ControlKey[] = []
}
```

The Core refuses a duplicate key, a function outside 1 to 12, and a node the schematic
does not hold. Deleting a node deletes its entry, and copying a node copies no key. No
Console command takes F1 to F12 alone or with Shift.

A control key is armed only while the operator holds control of the schematic
(`Session.Schematic.useSelectControlIsAcquired`) and its tab has focus. Key down and key
up act as mouse down and mouse up on the symbol, so fire, momentary, pulse, and the hold
delay behave as they do with the mouse (`pluto/src/vis/button/aether/button.ts:50-60`,
`lyra/src/hooks/useHold.ts`). A press that started before arming does nothing. Blur,
release of control, or a lost tab focus releases a held key. An unarmed press writes
nothing and shows a status: message "Control key not active", description "Take control
of {schematic} to use {key}." Each symbol shows its key on the canvas. Taking and
releasing control is a schematic command.

## 5 Implementation phases

| Phase | Scope                                                     | Touches                                                                                                                                                                                             | Depends on |
| ----- | --------------------------------------------------------- | --------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- | ---------- |
| 1     | Fix the Shift+Tab block and wrong shortcut hints          | `console/src-tauri`, `pluto/src/vis/diagram`, `site/docs`                                                                                                                                           | -          |
| 2     | Add the Lyra `Shortcut` engine with areas and commands    | `lyra/src/shortcut`                                                                                                                                                                                 | -          |
| 3     | Move Console commands and window keys onto `Shortcut`     | `console/src/platform/command`, `console/src/feature/command`, `console/src/app/palette`, `console/src/app/triggers`, `console/src/feature/panel`, `console/src/platform/modals`, `pluto/src/panel` | 2          |
| 4     | Move toolbar, dialog, menu, and button keys onto commands | `console/src/platform/nav`, `console/src/feature`, `console/src/platform`, `lyra/src/button`, `lyra/src/menu`, `lyra/src/dialog`                                                                    | 3          |
| 5     | Move schematic, Arc, and code editor keys onto commands   | `pluto/src/vis/diagram`, `pluto/src/schematic`, `pluto/src/arc`, `pluto/src/code`, `console/src/feature/schematic`, `console/src/feature/arc`                                                       | 3          |
| 6     | Move plot, log, and table keys and remove `Triggers`      | `pluto/src/lineplot`, `pluto/src/log`, `pluto/src/table`, `pluto/src/viewport`, `console/src/feature/lineplot`, `lyra/src/triggers`                                                                 | 4, 5       |
| 7     | Add the shortcut sheet and the keyboard settings page     | `console/src/app/palette`, `console/src/app/settings`, `console/src/session`                                                                                                                        | 6          |
| 8     | Build the macOS menu bar from the command list            | `console/src/app`, `console/src-tauri`                                                                                                                                                              | 6          |
| 9     | Add control keys to schematics and run them in control    | `schemas/synnax`, `core/pkg/service/schematic`, `client/ts`, `client/py`, `pluto/src/schematic`, `pluto/src/vis/button`, `console/src/feature/schematic`                                            | 6          |
| 10    | Set, show, and list control keys in schematics            | `pluto/src/schematic`, `console/src/feature/schematic`, `console/src/app/palette`                                                                                                                   | 7, 9       |

Phase 2 adds `lyra/src/shortcut` beside `lyra/src/triggers`, so phases 3 to 6 each move
a set of areas and leave `main` green. Phase 3 fixes the double Ctrl+E, and phase 5
drops the Monaco re-dispatch to `window` (`pluto/src/code/Editor.tsx:76-106`), whose
synthetic events hide the focused field. Phase 6 deletes the held-key set, the numeric
priorities, and the prevent-default lists that the list now derives. Phase 8 drops the
plugin's Meta+W (`console/src-tauri/src/main.rs:37`). Phase 9 lands with no way to set a
key, so phase 10 ships the setting and the canvas badge together.

## 6 Resolved decisions

1. **One list of commands, keys as data**: VS Code, Zed, and JetBrains bind keys to
   command IDs, and every surface reads that table. Keeping hooks and adding a hint
   registry leaves two lists that drift, and a key could work where its palette entry is
   hidden. The trade is real: every shortcut moves.
2. **The focused area wins**: Zed, Windows `ScopeOwner`, and Godot `shortcut_context`
   resolve by focus depth. Numbers have no owner and collide.
3. **Each press matched alone**: tinykeys, @github/hotkey, VS Code, and TanStack Hotkeys
   do this. The held set is the cause of the Cmd, Shift, and blur patches.
4. **Mod and Control stay separate**: Apple reserves Control for the system (⌃A,
   ⌃Space). The trade is real: a Mac user who pressed ⌃P must press ⌘P.
5. **The typed character first**: Matching `e.code` fires Mod+Z on the AZERTY W key and
   labels it Z. VS Code maps through a layout map, which WKWebView does not have.
6. **A sheet beside the palette**: Linear, Figma, and VS Code keep both. The palette
   finds a key whose name you know; the sheet shows the rest.
7. **Changes saved per computer**: VS Code keeps keybindings local. The trade is real: a
   person on two computers sets keys twice.
8. **Control keys belong to the schematic**: InTouch, WinCC, FactoryTalk, LabVIEW, and
   Ableton store the key with the screen. NUREG-0700 §2.2.4 limits keys each user
   defines. The trade is real: an operator cannot pick keys that suit them.
9. **One key list, not a key per symbol**: A key points at one control by construction,
   and a copy carries no key. The trade is real: a node delete edits the list too.
10. **F1 to F12, alone or with Shift**: A letter would open the Channels toolbar until
    control is taken and a valve after. WinCC and FactoryTalk use function keys. The
    trade is real: Mac laptops need Fn, and a schematic has 24 keys.
11. **The narrow stored type**: RFC 0103 gives each field its narrowest correct type, so
    a control key stores a function number and a flag, not a chord. The trade is real:
    other keys need a new schematic version.
12. **Taking control arms the keys**: No single event starts an action (NASA-STD-8739.8
    SWE-134), and control authority already marks who may act.

## 7 Open questions

1. How an editor sets a control key: a field in the symbol form, or a key map mode in
   which each symbol shows a target, as in Ableton. Both get built in the dev Console.
2. The look of the key badge on a symbol.
3. The default key for taking and releasing control.
4. The sequence wait: 1 s (Zed, tinykeys) or 1.5 s (GitHub).
