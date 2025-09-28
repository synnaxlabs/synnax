// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  bounds,
  color,
  MultiSeries,
  observe,
  type status,
  TimeStamp,
} from "@synnaxlabs/x";

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
  cleanup(): void {}
}

class NoopBooleanSink extends Noop implements BooleanSink {
  static readonly TYPE = "noop-boolean-sink";

  set(): void {}
}

export const noopBooleanSinkSpec: BooleanSinkSpec = {
  type: NoopBooleanSink.TYPE,
  props: {},
  variant: "sink",
  valueType: "boolean",
};

class NumericSink extends Noop implements NumberSink {
  static readonly TYPE = "noop-numeric-sink";

  set(): void {}
}

export const noopNumericSinkSpec: NumberSinkSpec = {
  type: NumericSink.TYPE,
  props: {},
  variant: "sink",
  valueType: "number",
};

class NoopBooleanSource extends Noop implements BooleanSource {
  static readonly TYPE = "noop-boolean-source";

  value(): boolean {
    return false;
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

  value(): number {
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

  value(): string {
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

  value(): status.Status {
    return {
      key: "noop",
      name: "noop",
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

  value(): color.Color {
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

  value(): [bounds.Bounds, MultiSeries] {
    return [bounds.ZERO, new MultiSeries([])];
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
