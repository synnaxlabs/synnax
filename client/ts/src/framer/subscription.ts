// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type Aggregation,
  type destructor,
  type MultiSeries,
  type TimeRange,
  type TimeStamp,
  type Window,
} from "@synnaxlabs/x";

import { type channel } from "@/channel";
import { type status } from "@/status";

/** Draws a channel across a window with about count points. */
export interface PointsLimit {
  variant: "points";
  /** About how many points to draw across the view. */
  count: number;
  aggregation: Aggregation;
  /** Reduces with min_max, so the bounds of the samples keep their extremes. */
  extremesKept?: boolean;
  /**
   * The range to draw, as when zoomed or panned. It may reach past a static window.
   * Defaults to the window.
   */
  view?: TimeRange;
  /** Draws the view from the samples held and reads none, as while a zoom moves. */
  settling?: boolean;
}

/** Keeps the newest count samples of each channel. */
export interface LatestLimit {
  variant: "latest";
  count: number;
}

/** A request that draws its channels under a points limit. */
export interface PointsRequest {
  keys: channel.Key[];
  window: Window;
  limit: PointsLimit;
}

/** A request that keeps the newest samples of its channels. */
export interface LatestRequest {
  keys: channel.Key[];
  /** With no window, the request keeps samples of any age. */
  window?: Window;
  limit: LatestLimit;
}

/** The channels, window, and limit of a subscription. */
export type SubscribeRequest = PointsRequest | LatestRequest;

/** The samples a subscription serves for one channel. */
export interface Snapshot {
  /** The channel's samples in time order. */
  series: MultiSeries;
  /** The worst of the channel's stream and read statuses. */
  status: status.Status;
  /** The end of the newest sample's span, once the channel streams. */
  time?: TimeStamp;
  /**
   * For a latest request, the count of samples appended to the channel. Samples that a
   * history change reads again do not count.
   */
  appended?: number;
}

/**
 * A pull handle on channel telemetry. Call value for a channel's samples whenever they
 * are needed. The series of a snapshot stay valid until the next call to value or
 * close.
 */
export interface Subscription {
  /** @returns the samples of the channel. The channel must be in the request. */
  value: (key: channel.Key) => Snapshot;
  /** Replaces the request. An equal request does nothing. */
  update: (request: SubscribeRequest) => void;
  /** Registers a handler called when new samples or a new status are available. */
  onChange: (handler: () => void) => destructor.Destructor;
  /** Releases every sample and stream the subscription holds. */
  close: () => void;
}
