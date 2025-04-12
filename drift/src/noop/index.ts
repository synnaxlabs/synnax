// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { Action, UnknownAction } from "@reduxjs/toolkit";

import { type Runtime } from "@/runtime";
import { type StoreState } from "@/state";
import { MAIN_WINDOW, type WindowProps } from "@/window";

export class NoopRuntime<S extends StoreState, A extends Action = UnknownAction>
  implements Runtime<S, A>
{
  async configure(): Promise<void> {}

  isMain(): boolean {
    return true;
  }

  label(): string {
    return MAIN_WINDOW;
  }

  async emit(): Promise<void> {}

  async subscribe(): Promise<void> {}

  onCloseRequested(): void {}

  async listLabels(): Promise<string[]> {
    return [MAIN_WINDOW];
  }

  async create(): Promise<void> {}

  async close(): Promise<void> {}

  async focus(): Promise<void> {}

  async setMinimized(): Promise<void> {}

  async setMaximized(): Promise<void> {}

  async setVisible(): Promise<void> {}

  async setFullscreen(): Promise<void> {}

  async center(): Promise<void> {}

  async setPosition(): Promise<void> {}

  async setSize(): Promise<void> {}

  async setMinSize(): Promise<void> {}

  async setMaxSize(): Promise<void> {}

  async setResizable(): Promise<void> {}

  async setSkipTaskbar(): Promise<void> {}

  async setAlwaysOnTop(): Promise<void> {}

  async setTitle(): Promise<void> {}

  async setDecorations(): Promise<void> {}

  async getProps(): Promise<Omit<WindowProps, "key">> {
    return {
      visible: true,
      focus: true,
      maximized: false,
      minimized: false,
      fullscreen: false,
      position: { x: 0, y: 0 },
      size: { width: 800, height: 600 },
    };
  }
}
