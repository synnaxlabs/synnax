// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TelemFactory } from "../factory";

import {
  BooleanTelemSink,
  BooleanTelemSinkSpec,
  BooleanTelemSource,
  BooleanTelemSourceSpec,
  NumericTelemSink,
  NumericTelemSinkSpec,
  NumericTelemSource,
  NumericTelemSourceSpec,
  Telem,
  TelemSpec,
} from "@/vis/telem";

export namespace AetherNoopTelem {
  class Noop implements Telem {
    setProps(): void {}
    cleanup(): void {}
    invalidate(): void {}
  }

  class BooleanSink extends Noop implements BooleanTelemSink {
    static readonly TYPE = "noop-boolean-sink";

    async set(): Promise<void> {
      return await Promise.resolve();
    }
  }

  export const booleanSinkSpec: BooleanTelemSinkSpec = {
    type: BooleanSink.TYPE,
    props: {},
    variant: "boolean-sink",
  };

  class NumericSink extends Noop implements NumericTelemSink {
    static readonly TYPE = "noop-numeric-sink";

    async set(): Promise<void> {
      return await Promise.resolve();
    }
  }

  export const numericSinkSpec: NumericTelemSinkSpec = {
    type: NumericSink.TYPE,
    props: {},
    variant: "numeric-sink",
  };

  class BooleanSource extends Noop implements BooleanTelemSource {
    static readonly TYPE = "noop-boolean-source";

    async value(): Promise<boolean> {
      return await Promise.resolve(false);
    }

    onChange(): void {}
  }

  export const booleanSourceSpec: BooleanTelemSourceSpec = {
    type: BooleanSource.TYPE,
    props: {},
    variant: "boolean-source",
  };

  class NumericSource extends Noop implements NumericTelemSource {
    static readonly TYPE = "noop-numeric-source";

    async value(): Promise<number> {
      return 0;
    }

    onChange(): void {}
  }

  export const numericSourceSpec: NumericTelemSourceSpec = {
    type: NumericSource.TYPE,
    props: {},
    variant: "numeric-source",
  };

  export class Factory implements TelemFactory {
    static readonly REGISTRY: Record<string, Telem> = {
      [BooleanSink.TYPE]: new BooleanSink(),
      [NumericSink.TYPE]: new NumericSink(),
      [BooleanSource.TYPE]: new BooleanSource(),
      [NumericSource.TYPE]: new NumericSource(),
    };

    create(key: string, spec: TelemSpec, _: TelemFactory): Telem | null {
      const t = Factory.REGISTRY[spec.type];
      if (t == null) return null;
      t.setProps(spec.props);
      return t;
    }
  }
}
