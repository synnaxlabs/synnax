// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp, observe } from "@synnaxlabs/x";

import { color } from "@/color/core";
import { type status } from "@/status/aether";
import { type telem } from "@/telem/aether";

class Noop extends observe.Observer<void> implements telem.Telem {
  async cleanup(): Promise<void> {}
}

class NoopBooleanSink extends Noop implements telem.BooleanSink {
  static readonly TYPE = "noop-boolean-sink";

  async set(): Promise<void> {
    return await Promise.resolve();
  }
}

export const noopBooleanSinkSpec: telem.BooleanSinkSpec = {
  type: NoopBooleanSink.TYPE,
  props: {},
  variant: "sink",
  valueType: "boolean",
};

class NumericSink extends Noop implements telem.NumberSink {
  static readonly TYPE = "noop-numeric-sink";

  async set(): Promise<void> {
    return await Promise.resolve();
  }
}

export const noopNumericSinkSpec: telem.NumberSinkSPec = {
  type: NumericSink.TYPE,
  props: {},
  variant: "sink",
  valueType: "number",
};

class NoopBooleanSource extends Noop implements telem.BooleanSource {
  static readonly TYPE = "noop-boolean-source";

  async value(): Promise<boolean> {
    return await Promise.resolve(false);
  }
}

export const noopBooleanSourceSpec: telem.BooleanSourceSpec = {
  type: NoopBooleanSource.TYPE,
  props: {},
  variant: "source",
  valueType: "boolean",
};

class NumericSource extends Noop implements telem.NumberSource {
  static readonly TYPE = "noop-numeric-source";

  async value(): Promise<number> {
    return 0;
  }
}

export const noopNumericSourceSpec: telem.NumberSourceSpec = {
  type: NumericSource.TYPE,
  props: {},
  variant: "source",
  valueType: "number",
};

class StringSource extends Noop implements telem.StringSource {
  static readonly TYPE = "noop-string-source";

  async value(): Promise<string> {
    return "";
  }
}

export const noopStringSourceSpec: telem.StringSourceSpec = {
  type: StringSource.TYPE,
  props: {},
  variant: "source",
  valueType: "string",
};

class StatusSource extends Noop implements telem.StatusSource {
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

export const noopStatusSourceSpec: telem.StatusSourceSpec = {
  type: StatusSource.TYPE,
  props: {},
  variant: "source",
  valueType: "status",
};

class NoopColorSource extends Noop implements telem.ColorSource {
  static readonly TYPE = "noop-color-source";

  async value(): Promise<color.Color> {
    return color.ZERO;
  }
}

export const noopColorSourceSpec: telem.ColorSourceSpec = {
  type: NoopColorSource.TYPE,
  props: {},
  variant: "source",
  valueType: "color",
};

const REGISTRY: Record<string, new () => telem.Telem> = {
  [NoopBooleanSink.TYPE]: NoopBooleanSink,
  [NumericSink.TYPE]: NumericSink,
  [NoopBooleanSource.TYPE]: NoopBooleanSource,
  [NumericSource.TYPE]: NumericSource,
  [StatusSource.TYPE]: StatusSource,
  [NoopColorSource.TYPE]: NoopColorSource,
  [StringSource.TYPE]: StringSource,
};

export class NoopFactory implements telem.Factory {
  type = "noop";
  create(spec: telem.Spec): telem.Telem | null {
    const F = REGISTRY[spec.type];
    if (F == null) return null;
    return new F();
  }
}
