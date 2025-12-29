// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, id, observe } from "@synnaxlabs/x";

import {
  type BooleanSinkSpec,
  type NumberSinkSpec,
  type Sink,
  type StringSinkSpec,
  type Telem,
} from "@/telem/aether/telem";
import { registerInstance, TEST_SINK_TYPE } from "@/telem/aether/test/factory";

export class TestSink<V> extends observe.Observer<void> implements Sink<V>, Telem {
  static readonly TYPE = TEST_SINK_TYPE;
  readonly id: string;
  values: V[] = [];
  private _destructor: destructor.Destructor;

  constructor() {
    super();
    this.id = id.create();
    this._destructor = registerInstance(this.id, this);
  }

  get lastValue(): V | undefined {
    return this.values[this.values.length - 1];
  }

  set(...values: V[]): void {
    this.values.push(...values);
    this.notify();
  }

  clear(): void {
    this.values = [];
  }

  cleanup(): void {
    this._destructor();
  }
}

export const sink = <V>(): TestSink<V> => new TestSink<V>();

export const booleanSinkSpec = (sink: TestSink<boolean>): BooleanSinkSpec => ({
  type: TestSink.TYPE,
  props: { testId: sink.id },
  variant: "sink",
  valueType: "boolean",
});

export const numberSinkSpec = (sink: TestSink<number>): NumberSinkSpec => ({
  type: TestSink.TYPE,
  props: { testId: sink.id },
  variant: "sink",
  valueType: "number",
});

export const stringSinkSpec = (sink: TestSink<string>): StringSinkSpec => ({
  type: TestSink.TYPE,
  props: { testId: sink.id },
  variant: "sink",
  valueType: "string",
});
