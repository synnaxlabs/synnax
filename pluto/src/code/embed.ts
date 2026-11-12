// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Theming } from "@synnaxlabs/lyra/theming";

import { BASE_THEMES } from "@/code/language";
import { type Monaco } from "@/code/Provider";

export const disableMonacoCommandPalette = (
  mon: Pick<typeof Monaco, "editor" | "KeyMod" | "KeyCode">,
): void => {
  const NOOP_COMMAND = "noop";
  mon.editor.addKeybindingRule({
    keybinding: mon.KeyMod.CtrlCmd | mon.KeyCode.KeyP,
    command: NOOP_COMMAND,
  });
  mon.editor.addKeybindingRule({
    keybinding: mon.KeyMod.CtrlCmd | mon.KeyCode.KeyP | mon.KeyMod.Shift,
    command: NOOP_COMMAND,
  });
};

const hasGlobalModifier = (e: KeyboardEvent): boolean =>
  e.ctrlKey || e.metaKey || e.altKey;

const redispatchToWindow = (e: KeyboardEvent): void => {
  const synthetic = new KeyboardEvent(e.type, {
    key: e.key,
    code: e.code,
    ctrlKey: e.ctrlKey,
    shiftKey: e.shiftKey,
    altKey: e.altKey,
    metaKey: e.metaKey,
    bubbles: true,
  });
  window.dispatchEvent(synthetic);
};

export const forwardGlobalTriggers = (
  editor: Monaco.editor.ICodeEditor,
): Monaco.IDisposable => {
  const downDispose = editor.onKeyDown((e) => {
    if (hasGlobalModifier(e.browserEvent)) redispatchToWindow(e.browserEvent);
  });
  const upDispose = editor.onKeyUp((e) => {
    if (hasGlobalModifier(e.browserEvent)) redispatchToWindow(e.browserEvent);
  });
  return {
    dispose: () => {
      downDispose.dispose();
      upDispose.dispose();
    },
  };
};

export const useTheme = (hasCustomTheme: boolean): string => {
  const theme = Theming.use();
  const prefersDark = theme.key.includes("Dark");
  if (hasCustomTheme) return prefersDark ? BASE_THEMES.dark : BASE_THEMES.light;
  return prefersDark ? "vs-dark" : "vs";
};
