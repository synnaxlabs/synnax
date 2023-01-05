// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export { default as Synnax, synnaxPropsSchema } from "@/client";
export type { SynnaxProps } from "@/client";
export {
  TimeRange,
  TimeSpan,
  TimeStamp,
  Density,
  DataType,
  Rate,
  TypedArray,
} from "@/telem";
export type {
  NativeTypedArray,
  UnparsedDataType,
  UnparsedDensity,
  UnparsedRate,
  UnparsedSize,
  UnparsedTimeSpan,
  UnparsedTimeStamp,
} from "@/telem";
export {
  AuthError,
  ContiguityError,
  GeneralError,
  ParseError,
  QueryError,
  RouteError,
  UnexpectedError,
  ValidationError,
} from "@/errors";
export * from "@/channel";
export * from "@/ontology";
export * from "@/connectivity";
export * from "@/ontology";
export type { Frame } from "@/framer";
