// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Destructor } from "@/destructor";

export type Handler<T> = (value: T) => void;

export interface Observable<T> {
  onChange: (handler: Handler<T>) => Destructor;
}

export class Observer<T> implements Observable<T> {
  private readonly handlers: Map<Handler<T>, null>;

  constructor(handlers?: Map<Handler<T>, null>) {
    this.handlers = handlers ?? new Map();
  }

  onChange(handler: Handler<T>): Destructor {
    this.handlers.set(handler, null);
    return () => this.handlers.delete(handler);
  }

  notify(value: T): void {
    this.handlers.forEach((_, handler) => handler(value));
  }
}
