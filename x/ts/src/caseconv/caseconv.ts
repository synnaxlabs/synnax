// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { camelKeys as _camelKeys, snakeKeys as _snakeKeys } from "js-convert-case";

const options = {
  recursive: true,
  recursiveInArray: true,
  keepTypesOnRecursion: [Number, String, Uint8Array],
};

export const toSnake = <T>(entity: T): T => _snakeKeys(entity, options) as T;

export const toCamel = <T>(entity: T): T => _camelKeys(entity, options) as T;

export const capitalize = (str: string): string => str[0].toUpperCase() + str.slice(1);
