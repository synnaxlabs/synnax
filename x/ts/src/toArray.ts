// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export const toArray = <T>(value: T | T[]): T[] =>
  Array.isArray(value) ? value : [value];

export const nullToArr = <T>(value: T | T[] | null): T[] =>
  Array.isArray(value) ? value : value === null ? [] : [value];
