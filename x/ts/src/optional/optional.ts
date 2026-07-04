// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export type Optional<T, K extends keyof T> = Pick<Partial<T>, K> & Omit<T, K>;

/**
 * Arg is a unary function type whose argument is optional when A has no required fields
 * (so it can be called with no argument) and required otherwise. A return type of R.
 */
export type Arg<A, R> = {} extends A ? (arg?: A) => R : (arg: A) => R;
