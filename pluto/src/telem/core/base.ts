// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError, ValidationError } from "@synnaxlabs/client";
import { deep } from "@synnaxlabs/x";
import { type z } from "zod";

import { prettyParse } from "@/util/zod";

export class TelemMeta<P extends z.ZodTypeAny> {
  key: string;
  _props: z.output<P> | null = null;
  _prevProps: z.output<P> | null = null;
  schema: P | undefined = undefined;
  notify: (() => void) | null = null;

  constructor(key: string) {
    this.key = key;
  }

  get props(): z.output<P> {
    if (this._props == null)
      throw new UnexpectedError(
        "[TelemMeta] - props is not defined. Please call setProps",
      );
    return this._props;
  }

  get prevProps(): z.output<P> {
    if (this._prevProps == null)
      throw new UnexpectedError(
        "[TelemMeta] - prevProps is not defined. Please call setProps",
      );
    return this._prevProps;
  }

  private get _schema(): P {
    if (this.schema == null)
      throw new ValidationError(
        `[BaseTelem] - expected subclass to define props schema, but none was found.
    Make sure to define a property 'schema' on the class.`,
      );
    return this.schema;
  }

  setProps(nextProps: any): void {
    const p = this._props;
    this._props = prettyParse(this._schema, nextProps);
    this._prevProps = p ?? this._props;
  }

  onChange(f: () => void): void {
    this.notify = f;
  }

  get propsDeepEqual(): boolean {
    return deep.equal(this.prevProps, this.props);
  }

  cleanup(): void {
    this.notify = null;
    this._prevProps = null;
    this._props = null;
  }
}
