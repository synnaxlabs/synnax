// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep, type destructor } from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { UnexpectedError } from "@/errors";
import { Latest, type LatestProps } from "@/framer/cache/latest";
import { Points, type PointsProps } from "@/framer/cache/points";
import {
  type Snapshot,
  type SubscribeRequest,
  type Subscription,
} from "@/framer/subscription";

export interface HandleProps extends PointsProps, LatestProps {}

/** A subscription that serves each channel through a points or latest consumer. */
export class Handle implements Subscription {
  private request: SubscribeRequest | null = null;
  private readonly consumers = new Map<channel.Key, Points | Latest>();
  private readonly handlers = new Set<() => void>();
  private closed = false;

  constructor(
    private readonly props: HandleProps,
    request: SubscribeRequest,
  ) {
    this.update(request);
  }

  value(key: channel.Key): Snapshot {
    const consumer = this.consumers.get(key);
    if (consumer == null)
      throw new UnexpectedError(`channel ${key} is not in the subscription`);
    return consumer.value();
  }

  update(request: SubscribeRequest): void {
    if (this.closed) throw new UnexpectedError("subscription is closed");
    const prev = this.request;
    if (prev != null && deep.equal(prev, request)) return;
    this.request = request;
    const keys = new Set(request.keys);
    this.consumers.forEach((c, key) => {
      if (keys.has(key) && prev?.limit.variant === request.limit.variant) return;
      c.close();
      this.consumers.delete(key);
    });
    keys.forEach((key) => {
      const c = this.consumers.get(key);
      if (request.limit.variant === "points") {
        const { window, limit } = request as Extract<
          SubscribeRequest,
          { limit: { variant: "points" } }
        >;
        if (c instanceof Points) c.update(window, limit);
        else
          this.consumers.set(
            key,
            new Points(this.props, key, window, limit, this.notify),
          );
      } else if (c instanceof Latest) c.update(request.window, request.limit.count);
      else
        this.consumers.set(
          key,
          new Latest(this.props, key, request.window, request.limit.count, this.notify),
        );
    });
  }

  onChange(handler: () => void): destructor.Destructor {
    this.handlers.add(handler);
    return () => this.handlers.delete(handler);
  }

  close(): void {
    this.closed = true;
    this.consumers.forEach((c) => c.close());
    this.consumers.clear();
    this.handlers.clear();
  }

  private readonly notify = (): void => {
    if (!this.closed) this.handlers.forEach((h) => h());
  };
}
