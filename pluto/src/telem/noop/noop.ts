// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { telem } from "@/telem/core";

class Noop implements telem.Telem {
  setProps(): void {}
  cleanup(): void {}
  invalidate(): void {}
}

class BooleanSink extends Noop implements telem.BooleanSink {
  static readonly TYPE = "noop-boolean-sink";

  async set(): Promise<void> {
    return await Promise.resolve();
  }
}

export const booleanSinkSpec: telem.BooleanSinkSpec = {
  type: BooleanSink.TYPE,
  props: {},
  variant: "boolean-sink",
};

class NumericSink extends Noop implements telem.NumericSink {
  static readonly TYPE = "noop-numeric-sink";

  async set(): Promise<void> {
    return await Promise.resolve();
  }
}

export const numericSinkSpec: telem.NumericSinkSpec = {
  type: NumericSink.TYPE,
  props: {},
  variant: "numeric-sink",
};

class BooleanSource extends Noop implements telem.BooleanSource {
  static readonly TYPE = "noop-boolean-source";

  async value(): Promise<boolean> {
    return await Promise.resolve(false);
  }

  onChange(): void {}
}

export const booleanSourceSpec: telem.BooleanSourceSpec = {
  type: BooleanSource.TYPE,
  props: {},
  variant: "boolean-source",
};

class NumericSource extends Noop implements telem.NumericSource {
  static readonly TYPE = "noop-numeric-source";

  async value(): Promise<number> {
    return 0;
  }

  onChange(): void {}
}

export const numericSourceSpec: telem.NumericSourceSpec = {
  type: NumericSource.TYPE,
  props: {},
  variant: "numeric-source",
};
const REGISTRY: Record<string, telem.Telem> = {
  [BooleanSink.TYPE]: new BooleanSink(),
  [NumericSink.TYPE]: new NumericSink(),
  [BooleanSource.TYPE]: new BooleanSource(),
  [NumericSource.TYPE]: new NumericSource(),
};

export class Factory implements telem.Factory {
  create(_: string, spec: telem.Spec): telem.Telem | null {
    const t = REGISTRY[spec.type];
    if (t == null) return null;
    t.setProps(spec.props);
    return t;
  }
}
