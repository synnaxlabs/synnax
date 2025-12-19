// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id, observe } from "@synnaxlabs/x";

import {
  type BooleanSourceSpec,
  type NumberSourceSpec,
  type Source,
  type StringSourceSpec,
  type Telem,
} from "@/telem/aether/telem";
import {
  registerTestInstance,
  TEST_SOURCE_TYPE,
  unregisterTestInstance,
} from "@/telem/aether/test/factory";

export class TestSource<V> extends observe.Observer<void> implements Source<V>, Telem {
  static readonly TYPE = TEST_SOURCE_TYPE;
  readonly id: string;
  private _value: V;

  constructor(initialValue: V) {
    super();
    this.id = id.create();
    this._value = initialValue;
    registerTestInstance(this.id, this);
  }

  value(): V {
    return this._value;
  }

  setValue(v: V): void {
    this._value = v;
    this.notify();
  }

  cleanup(): void {
    unregisterTestInstance(this.id);
  }
}

export const source = <V>(initialValue: V): TestSource<V> =>
  new TestSource(initialValue);

export const booleanSourceSpec = (source: TestSource<boolean>): BooleanSourceSpec => ({
  type: TestSource.TYPE,
  props: { testId: source.id },
  variant: "source",
  valueType: "boolean",
});

export const numberSourceSpec = (source: TestSource<number>): NumberSourceSpec => ({
  type: TestSource.TYPE,
  props: { testId: source.id },
  variant: "source",
  valueType: "number",
});

export const stringSourceSpec = (source: TestSource<string>): StringSourceSpec => ({
  type: TestSource.TYPE,
  props: { testId: source.id },
  variant: "source",
  valueType: "string",
});
