// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * @returns a deep copy of the value, made with `structuredClone`.
 * @throws {DOMException} if the value holds something that cannot be cloned, such as a
 * function or a proxy. Copy an Immer draft through Immer's `current`.
 */
export const copy = <T>(obj: T): T => structuredClone(obj);
