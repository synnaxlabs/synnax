// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export const ERROR_NOT_COMPILED = new Error("webgl program executed without compiling");

export const errorUnsupported = (msg: string): Error =>
  new Error(`unsupported webgl feature: ${msg}`);

export const errorCompile = (msg: string): Error =>
  new Error(`failed to compile webgl program: ${msg}`);

export const ERROR_BAD_SHADER = new Error("null shader encountered");
