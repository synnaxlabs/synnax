// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { alamos } from "@synnaxlabs/alamos";
import { deep, type destructor, type TimeStamp } from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { UnexpectedError, ValidationError } from "@/errors";
import { Frames, type FramesProps } from "@/framer/cache/frames";
import { Latest, type LatestProps } from "@/framer/cache/latest";
import { Points, type PointsProps } from "@/framer/cache/points";
import {
  type FramesRequest,
  type LatestRequest,
  type PointsRequest,
  type Snapshot,
  type SubscribeRequest,
  type Subscription,
} from "@/framer/subscription";

export interface HandleProps extends PointsProps, LatestProps, FramesProps {
  /** Called once when the subscription closes. */
  onClose: () => void;
  /** Logs the errors that change handlers throw. */
  instrumentation?: alamos.Instrumentation;
}

const isPoints = (request: SubscribeRequest): request is PointsRequest =>
  request.limit.variant === "points";

const isLatest = (request: SubscribeRequest): request is LatestRequest =>
  request.limit.variant === "latest";

const isFrames = (request: SubscribeRequest): request is FramesRequest =>
  request.limit.variant === "frames";

type Consumer = Points | Latest | Frames;

/**
 * A subscription that serves each channel through a points, latest, or frames
 * consumer. Its change handlers run once per microtask, however many changes it holds.
 */
export class Handle implements Subscription {
  private request: SubscribeRequest | null = null;
  private readonly consumers = new Map<channel.Key, Consumer>();
  private readonly handlers = new Set<() => void>();
  private notifying = false;
  private closed = false;

  constructor(
    private readonly props: HandleProps,
    request: SubscribeRequest,
  ) {
    this.update(request);
  }

  value(key: channel.Key): Snapshot {
    return this.consumer(key).value();
  }

  timeAt(key: channel.Key, alignment: bigint): TimeStamp | null {
    const consumer = this.consumer(key);
    return consumer instanceof Latest ? consumer.timeAt(alignment) : null;
  }

  update(request: SubscribeRequest): void {
    if (this.closed) throw new UnexpectedError("subscription is closed");
    if (isPoints(request)) {
      const { count } = request.limit;
      if (!Number.isFinite(count) || count < 0)
        throw new ValidationError(
          `points limit count ${count} must be finite and not negative`,
        );
    }
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
      if (c != null) this.updateConsumer(c, request);
      else this.consumers.set(key, this.createConsumer(key, request));
    });
  }

  private createConsumer(key: channel.Key, request: SubscribeRequest): Consumer {
    if (isPoints(request))
      return new Points(this.props, key, request.window, request.limit, this.notify);
    if (isLatest(request))
      return new Latest(
        this.props,
        key,
        request.window,
        request.limit.count,
        this.notify,
      );
    if (isFrames(request))
      return new Frames(this.props, key, request.window, this.notify);
    throw new UnexpectedError("subscription request has an unknown limit");
  }

  // update closes every consumer of another variant first.
  private updateConsumer(c: Consumer, request: SubscribeRequest): void {
    if (c instanceof Points && isPoints(request))
      c.update(request.window, request.limit);
    else if (c instanceof Latest && isLatest(request))
      c.update(request.window, request.limit.count);
    else if (c instanceof Frames && isFrames(request)) c.update(request.window);
    else
      throw new UnexpectedError(`consumer of channel ${c.key} is of another variant`);
  }

  onChange(handler: () => void): destructor.Destructor {
    this.handlers.add(handler);
    return () => this.handlers.delete(handler);
  }

  close(): void {
    if (this.closed) return;
    this.closed = true;
    this.props.onClose();
    this.consumers.forEach((c) => c.close());
    this.consumers.clear();
    this.handlers.clear();
  }

  private consumer(key: channel.Key): Consumer {
    const consumer = this.consumers.get(key);
    if (consumer == null)
      throw new UnexpectedError(`channel ${key} is not in the subscription`);
    return consumer;
  }

  private readonly notify = (): void => {
    if (this.notifying || this.closed) return;
    this.notifying = true;
    queueMicrotask(() => {
      this.notifying = false;
      if (this.closed) return;
      const { instrumentation = alamos.NOOP } = this.props;
      this.handlers.forEach((h) => {
        try {
          h();
        } catch (e) {
          instrumentation.L.error("subscription change handler failed", { error: e });
        }
      });
    });
  };
}
