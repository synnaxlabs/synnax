// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { bounds, observe, type Series, TimeStamp } from "@synnaxlabs/x";

import { color } from "@/color/core";
import { type status } from "@/status/aether";
import { type Factory } from "@/telem/aether/factory";
import {
  type BooleanSink,
  type BooleanSinkSpec,
  type BooleanSource,
  type BooleanSourceSpec,
  type ColorSource,
  type ColorSourceSpec,
  type NumberSink,
  type NumberSinkSpec,
  type NumberSource,
  type NumberSourceSpec,
  type SeriesSource,
  type SeriesSourceSpec,
  type Spec,
  type StatusSourceSpec,
  type StringSourceSpec,
  type Telem,
} from "@/telem/aether/telem";

class Noop extends observe.Observer<void> implements Telem {
  async cleanup(): Promise<void> {}
}

class NoopBooleanSink extends Noop implements BooleanSink {
  static readonly TYPE = "noop-boolean-sink";

  async set(): Promise<void> {
    return await Promise.resolve();
  }
}

export const noopBooleanSinkSpec: BooleanSinkSpec = {
  type: NoopBooleanSink.TYPE,
  props: {},
  variant: "sink",
  valueType: "boolean",
};

class NumericSink extends Noop implements NumberSink {
  static readonly TYPE = "noop-numeric-sink";

  async set(): Promise<void> {
    return await Promise.resolve();
  }
}

export const noopNumericSinkSpec: NumberSinkSpec = {
  type: NumericSink.TYPE,
  props: {},
  variant: "sink",
  valueType: "number",
};

class NoopBooleanSource extends Noop implements BooleanSource {
  static readonly TYPE = "noop-boolean-source";

  async value(): Promise<boolean> {
    return await Promise.resolve(false);
  }
}

export const noopBooleanSourceSpec: BooleanSourceSpec = {
  type: NoopBooleanSource.TYPE,
  props: {},
  variant: "source",
  valueType: "boolean",
};

class NumericSource extends Noop implements NumberSource {
  static readonly TYPE = "noop-numeric-source";

  async value(): Promise<number> {
    return 0;
  }
}

export const noopNumericSourceSpec: NumberSourceSpec = {
  type: NumericSource.TYPE,
  props: {},
  variant: "source",
  valueType: "number",
};

class StringSource extends Noop implements StringSource {
  static readonly TYPE = "noop-string-source";

  async value(): Promise<string> {
    return "";
  }
}

export const noopStringSourceSpec: StringSourceSpec = {
  type: StringSource.TYPE,
  props: {},
  variant: "source",
  valueType: "string",
};

class StatusSource extends Noop implements StatusSource {
  static readonly TYPE = "noop-status-source";

  async value(): Promise<status.Spec> {
    return {
      key: "noop",
      variant: "disabled",
      message: "unknown",
      time: TimeStamp.now(),
    };
  }
}

export const noopStatusSourceSpec: StatusSourceSpec = {
  type: StatusSource.TYPE,
  props: {},
  variant: "source",
  valueType: "status",
};

class NoopColorSource extends Noop implements ColorSource {
  static readonly TYPE = "noop-color-source";

  async value(): Promise<color.Color> {
    return color.ZERO;
  }
}

export const noopColorSourceSpec: ColorSourceSpec = {
  type: NoopColorSource.TYPE,
  props: {},
  variant: "source",
  valueType: "color",
};

class NoopSeries extends Noop implements SeriesSource {
  static readonly TYPE = "noop-series";

  async value(): Promise<[bounds.Bounds, Series[]]> {
    return [bounds.ZERO, []];
  }
}

export const noopSeriesSourceSpec: SeriesSourceSpec = {
  type: NoopSeries.TYPE,
  props: {},
  variant: "source",
  valueType: "series",
};

const REGISTRY: Record<string, new () => Telem> = {
  [NoopBooleanSink.TYPE]: NoopBooleanSink,
  [NumericSink.TYPE]: NumericSink,
  [NoopBooleanSource.TYPE]: NoopBooleanSource,
  [NumericSource.TYPE]: NumericSource,
  [StatusSource.TYPE]: StatusSource,
  [NoopColorSource.TYPE]: NoopColorSource,
  [StringSource.TYPE]: StringSource,
  [NoopSeries.TYPE]: NoopSeries,
};

export class NoopFactory implements Factory {
  type = "noop";
  create(spec: Spec): Telem | null {
    const F = REGISTRY[spec.type];
    if (F == null) return null;
    return new F();
  }
}
