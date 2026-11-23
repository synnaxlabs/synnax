// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeStamp } from "@synnaxlabs/x";

import { context } from "@/context";

/** A source of the current time. */
export type Clock = () => TimeStamp;

const [Context, useContext] = context.create<Clock>({
  defaultValue: () => TimeStamp.now(),
  displayName: "Clock.Context",
});

/** Supplies the clock that {@link useClock} and `useNow` read. */
export const ClockProvider = Context;

/** @returns the clock of the nearest {@link ClockProvider}, or the real clock. */
export const useClock = (): Clock => useContext();
