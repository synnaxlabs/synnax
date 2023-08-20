// Copyright 2023 synnax labs, inc.
//
// use of this software is governed by the business source license included in the file
// licenses/bsl.txt.
//
// as of the change date specified in that file, in accordance with the business source
// license, use of this software will be governed by the apache license, version 2.0,
// included in the file licenses/apl.txt.

import { Bounds } from "@synnaxlabs/x";
import { z } from "zod";

import {
  BooleanTelemSink,
  BooleanTelemSource,
  NumericTelemSink,
  NumericTelemSource,
  Telem,
  TelemSpec,
  numericTelemSinkSpec,
  numericTelemSourceSpec,
} from "@/core/vis/telem";
import { TelemMeta } from "@/telem/base";
import { TelemFactory } from "@/telem/factory";

export namespace AetherBooleanTelem {
  export class Factory implements TelemFactory {
    create(key: string, spec: TelemSpec, root: TelemFactory): Telem | null {
      switch (spec.type) {
        case NumericConverterSink.TYPE: {
          const props_ = NumericConverterSink.propsZ.parse(spec.props);
          const wrap = root.create(`${key}.wrap`, props_.wrap, root);
          if (wrap == null) return null;
          const t = new NumericConverterSink(key, wrap as NumericTelemSink);
          t.setProps(props_);
          return t;
        }
        case NumericConverterSource.TYPE: {
          const props_ = NumericConverterSource.propsZ.parse(spec.props);
          const wrap = root.create(`${key}.wrap`, props_.wrap, root);
          if (wrap == null) return null;
          const t = new NumericConverterSource(key, wrap as NumericTelemSource);
          t.setProps(props_);
          return t;
        }
      }
      return null;
    }
  }

  const numericConverterSinkProps = z.object({
    wrap: numericTelemSinkSpec,
    truthy: z.number().optional().default(1),
    falsy: z.number().optional().default(0),
  });

  export type NumericConverterSinkProps = z.infer<typeof numericConverterSinkProps>;

  export class NumericConverterSink
    extends TelemMeta<typeof numericConverterSinkProps>
    implements BooleanTelemSink
  {
    static readonly propsZ = numericConverterSinkProps;
    schema = NumericConverterSink.propsZ;
    wrap: NumericTelemSink;

    static readonly TYPE = "boolean-numeric-converter-sink";

    constructor(key: string, wrap: NumericTelemSink) {
      super(key);
      this.wrap = wrap;
    }

    invalidate(): void {
      this.wrap.invalidate();
    }

    async set(value: boolean): Promise<void> {
      await this.wrap.set(value ? this.props.truthy : this.props.falsy);
    }

    setProps(props: any): void {
      super.setProps(props);
      this.wrap.setProps(props.wrap.props);
    }
  }

  const numericConverterSourceProps = z.object({
    wrap: numericTelemSourceSpec,
    trueBound: Bounds.strictZ,
  });

  export type NumericConverterSourceProps = z.infer<typeof numericConverterSourceProps>;

  export class NumericConverterSource
    extends TelemMeta<typeof numericConverterSourceProps>
    implements BooleanTelemSource
  {
    wrapped: NumericTelemSource;
    curr: boolean | null = null;

    static readonly propsZ = numericConverterSourceProps;
    schema = NumericConverterSource.propsZ;

    static readonly TYPE = "boolean-source";

    constructor(key: string, wraps: NumericTelemSource) {
      super(key);
      this.wrapped = wraps;

      this.wrapped.onChange(() => {
        void this.update();
      });
    }

    invalidate(): void {
      this.wrapped.invalidate();
    }

    private async update(): Promise<void> {
      const raw = await this.wrapped.value();
      const value = this.props.trueBound.contains(raw);
      if (this.curr !== value) {
        this.curr = value;
        this.notify?.();
      }
    }

    async value(): Promise<boolean> {
      if (this.curr == null) await this.update();
      return this.curr ?? false;
    }

    setProps(props: any): void {
      super.setProps(props);
      this.wrapped.setProps(props.wrap.props);
      void this.update();
    }
  }
}
