// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { observe } from "@synnaxlabs/x";
import { type Mock, vi } from "vitest";

import { type telem } from "@/telem/aether";

class Base {
  cleanupStub: Mock = vi.fn();

  async cleanup(): Promise<void> {
    this.cleanupStub();
  }
}

export class BooleanSink extends Base implements telem.BooleanSink {
  value: boolean | null = null;
  fn: Mock = vi.fn();

  async set(value: boolean): Promise<void> {
    this.value = value;
    this.fn(value);
  }
}

export const booleanSinkSpec: telem.BooleanSinkSpec = {
  type: "boolean-sink",
  props: {},
  variant: "sink",
  valueType: "boolean",
};

export class BooleanSource
  extends observe.Observer<void>
  implements telem.BooleanSource
{
  private iValue: boolean;
  cleanupStub: Mock = vi.fn();

  constructor(value: boolean = false) {
    super();
    this.iValue = value;
  }

  setValue(value: boolean): void {
    this.iValue = value;
    this.notify();
  }

  async value(): Promise<boolean> {
    return this.iValue;
  }

  async cleanup(): Promise<void> {
    this.cleanupStub();
  }
}

export const booleanSourceSpec: telem.BooleanSourceSpec = {
  type: "boolean-source",
  props: {},
  variant: "source",
  valueType: "boolean",
};

const REGISTRY: Record<string, new () => telem.Telem> = {
  "boolean-sink": BooleanSink,
  "boolean-source": BooleanSource,
};

export const FACTORY: telem.Factory = {
  type: "mock",
  create: (spec: telem.Spec): telem.Telem | null => {
    const Constructor = REGISTRY[spec.type];
    return new Constructor();
  },
};

export class Provider implements telem.Provider {
  clusterKey: string = "test";
  key: string = "test";

  equals(): boolean {
    return true;
  }

  created: Record<string, telem.Telem> = {};

  registerFactory(): void {}

  create<T>(spec: telem.Spec): T {
    const t = FACTORY.create(spec) as T;
    this.created[spec.type] = t as unknown as telem.Telem;
    return t;
  }
}
