// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export { access } from "@/access";
export { arc } from "@/arc";
export { channel } from "@/channel";
export { Channel, isCalculated } from "@/channel/client";
export {
  checkConnection,
  type CheckConnectionParams,
  newConnectionChecker,
  default as Synnax,
  type SynnaxParams,
  synnaxParamsZ,
} from "@/client";
export * from "@/connection";
export { control } from "@/control";
export { device } from "@/device";
export {
  AuthError,
  ContiguityError,
  DisconnectedError,
  MultipleFoundError,
  NotFoundError,
  QueryError,
  RouteError,
  UnexpectedError,
  ValidationError,
} from "@/errors";
export { framer } from "@/framer";
export { Frame } from "@/framer/frame";
export { group } from "@/group";
export { label } from "@/label";
export { lineplot } from "@/lineplot";
export { log } from "@/log";
export { ontology } from "@/ontology";
export { rack } from "@/rack";
export { ranger } from "@/ranger";
export { schematic } from "@/schematic";
export { status } from "@/status";
export { table } from "@/table";
export { task } from "@/task";
export { createTestClientWithPolicy } from "@/testutil/access";
export { createTestClient, TEST_CLIENT_PARAMS } from "@/testutil/client";
export { user } from "@/user";
export { view } from "@/view";
export { workspace } from "@/workspace";
import { telem } from "@synnaxlabs/x/telem";

export const DataType = telem.DataType;
export type DataType = telem.DataType;
export const Density = telem.Density;
export type Density = telem.Density;
export const MultiSeries = telem.MultiSeries;
export type MultiSeries = telem.MultiSeries;
export const Rate = telem.Rate;
export type Rate = telem.Rate;
export const Series = telem.Series;
export type Series = telem.Series;
export const TimeRange = telem.TimeRange;
export type TimeRange = telem.TimeRange;
export const TimeSpan = telem.TimeSpan;
export type TimeSpan = telem.TimeSpan;
export const TimeStamp = telem.TimeStamp;
export type TimeStamp = telem.TimeStamp;
export type CrudeDataType = telem.CrudeDataType;
export type CrudeDensity = telem.CrudeDensity;
export type CrudeRate = telem.CrudeRate;
export type CrudeSize = telem.CrudeSize;
export type CrudeTimeSpan = telem.CrudeTimeSpan;
export type CrudeTimeStamp = telem.CrudeTimeStamp;
export type TelemValue = telem.TelemValue;
export type TimeStampStringFormat = telem.TimeStampStringFormat;
export type TypedArray = telem.TypedArray;
export type TZInfo = telem.TZInfo;
