// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp } from "@synnaxlabs/x";

import { color } from "@/color/core";
import { type status } from "@/status/aether";
import { type telem } from "@/telem/core";

class Noop implements telem.Telem {
  key: string;
  type: string;
  setProps(): void {}
  cleanup(): void {}
  invalidate(): void {}
}

class BooleanSink extends Noop implements telem.BooleanSink {
  key: string;
  type: string;
  static readonly TYPE = "noop-boolean-sink";

  async setBoolean(): Promise<void> {
    return await Promise.resolve();
  }
}

export const booleanSinkSpec: telem.BooleanSinkSpec = {
  type: BooleanSink.TYPE,
  props: {},
  variant: "boolean-sink",
};

class NumericSink extends Noop implements telem.NumericSink {
  key: string;
  type: string;
  static readonly TYPE = "noop-numeric-sink";

  async setNumber(): Promise<void> {
    return await Promise.resolve();
  }
}

export const numericSinkSpec: telem.NumericSinkSpec = {
  type: NumericSink.TYPE,
  props: {},
  variant: "numeric-sink",
};

class BooleanSource extends Noop implements telem.BooleanSource {
  key: string;
  type: string;
  static readonly TYPE = "noop-boolean-source";

  async boolean(): Promise<boolean> {
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
  key: string;
  type: string;
  static readonly TYPE = "noop-numeric-source";

  async number(): Promise<number> {
    return 0;
  }

  onChange(): void {}
}

export const numericSourceSpec: telem.NumericSourceSpec = {
  type: NumericSource.TYPE,
  props: {},
  variant: "numeric-source",
};

class StringSource extends Noop implements telem.StringSource {
  key: string;
  type: string;
  static readonly TYPE = "noop-string-source";

  async string(): Promise<string> {
    return "";
  }

  onChange(): void {}
}

export const stringSourceSpec: telem.StringSourceSpec = {
  type: StringSource.TYPE,
  props: {},
  variant: "string-source",
};

class StatusSource extends Noop implements telem.StatusSource {
  key: string;
  type: string;
  static readonly TYPE = "noop-status-source";

  async status(): Promise<status.Spec> {
    return {
      key: "noop",
      variant: "disabled",
      message: "unknown",
      time: TimeStamp.now(),
    };
  }

  onChange(): void {}
}

export const statusSourceSpec: telem.StatusSourceSpec = {
  type: StatusSource.TYPE,
  props: {},
  variant: "status-source",
};

class ColorSource extends Noop implements telem.ColorSource {
  key: string;
  type: string;
  static readonly TYPE = "noop-color-source";

  async color(): Promise<color.Color> {
    return color.ZERO;
  }

  onChange(): void {}
}

export const colorSourceSpec: telem.ColorSourceSpec = {
  type: ColorSource.TYPE,
  props: {},
  variant: "color-source",
};

const REGISTRY: Record<string, telem.Telem> = {
  [BooleanSink.TYPE]: new BooleanSink(),
  [NumericSink.TYPE]: new NumericSink(),
  [BooleanSource.TYPE]: new BooleanSource(),
  [NumericSource.TYPE]: new NumericSource(),
  [StatusSource.TYPE]: new StatusSource(),
  [ColorSource.TYPE]: new ColorSource(),
  [StringSource.TYPE]: new StringSource(),
};

export class Factory implements telem.Factory {
  create(_: string, spec: telem.Spec): telem.Telem | null {
    const t = REGISTRY[spec.type];
    if (t == null) return null;
    t.setProps(spec.props);
    return t;
  }
}
