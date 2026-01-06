// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Action, type UnknownAction } from "@reduxjs/toolkit";

import { type Event } from "@/runtime";
import { type StoreState } from "@/state";

export const encode = <S extends StoreState, A extends Action = UnknownAction>(
  event: Event<S, A>,
): string => JSON.stringify(event);
export const decode = <S extends StoreState, A extends Action = UnknownAction>(
  event: string,
): Event<S, A> => JSON.parse(event) as Event<S, A>;
