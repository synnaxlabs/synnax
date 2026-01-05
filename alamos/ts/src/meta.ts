// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export class Meta {
  private readonly _noop: boolean = false;
  readonly key: string;
  readonly path: string;
  readonly serviceName: string;

  constructor(
    key: string,
    path: string,
    serviceName: string = "",
    noop: boolean = false,
  ) {
    this.key = key;
    this.path = path;
    this.serviceName = serviceName;
    this._noop = noop;
  }

  child(key: string): Meta {
    return new Meta(key, this.extendPath(key), this.serviceName, this.noop);
  }

  extendPath(key: string): string {
    return `${this.path}.${key}`;
  }

  get noop(): boolean {
    return this._noop;
  }

  static readonly NOOP = new Meta("", "", "");
}
