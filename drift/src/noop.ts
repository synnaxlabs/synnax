// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Action, type UnknownAction } from "@reduxjs/toolkit";

import { type Runtime } from "@/runtime";
import { type StoreState } from "@/state";
import { MAIN_WINDOW, type WindowProps } from "@/window";

/**
 * In certain environments (such as the web browser), it is not really possible to spawn
 * new windows. NoopRuntime is intended to stand in for drift in these environments.
 */
export class NoopRuntime<S extends StoreState, A extends Action = UnknownAction>
  implements Runtime<S, A>
{
  async emit(): Promise<void> {}

  async subscribe(): Promise<void> {}

  isMain(): boolean {
    return true;
  }

  label(): string {
    return MAIN_WINDOW;
  }

  onCloseRequested(): void {}

  async listLabels(): Promise<string[]> {
    return [];
  }

  async getProps(): Promise<Omit<WindowProps, "key">> {
    return {};
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

  async setDecorations(): Promise<void> {}

  async setTitle(): Promise<void> {}

  async configure(): Promise<void> {}
}
