// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type access } from "@/access";
import { type arc } from "@/arc";
import { channel } from "@/channel";
import { framer } from "@/framer";

export { access } from "@/access";
export { actions } from "@/actions";
export { arc } from "@/arc";
export { auth } from "@/auth";
export { channel } from "@/channel";
export {
  type ConnectOptions,
  default as Synnax,
  type SynnaxParams,
  synnaxParamsZ,
} from "@/client";
export * from "@/connection";
export { control } from "@/control";
export { device } from "@/device";
export {
  AccessDeniedError,
  AuthError,
  ContiguityError,
  ControlError,
  DisconnectedError,
  ExpiredLicenseError,
  InvalidLicenseError,
  isConnectionError,
  LicenseError,
  LicenseFingerprintError,
  LicenseLimitError,
  MissingLicenseError,
  MultipleFoundError,
  NotFoundError,
  PathError,
  QueryError,
  RouteError,
  SessionEndedError,
  UnauthorizedError,
  UnexpectedError,
  UnsupportedError,
  ValidationError,
} from "@/errors";
export { ethercat } from "@/ethercat";
export { framer } from "@/framer";
export { group } from "@/group";
export { http } from "@/http";
export { imex } from "@/imex";
export { label } from "@/label";
export { labjack } from "@/labjack";
export { license } from "@/license";
export { lineplot } from "@/lineplot";
export { log } from "@/log";
export { modbus } from "@/modbus";
export { ni } from "@/ni";
export { node } from "@/node";
export { ontology } from "@/ontology";
export { opcua } from "@/opcua";
export { pagerduty } from "@/pagerduty";
export { panel } from "@/panel";
export { project } from "@/project";
export { query } from "@/query";
export { rack } from "@/rack";
export { ranger } from "@/ranger";
export { schematic } from "@/schematic";
export { status } from "@/status";
export type { StatusZodObject } from "@/status/types.gen";
export { table } from "@/table";
export { task } from "@/task";
export { user } from "@/user";
export { view } from "@/view";
export {
  type CrudeDataType,
  type CrudeDensity,
  type CrudeRate,
  type CrudeSize,
  type CrudeTimeSpan,
  type CrudeTimeStamp,
  DataType,
  Density,
  MultiSeries,
  Rate,
  Series,
  type TelemValue,
  TimeRange,
  TimeSpan,
  TimeStamp,
  type TimestampFormat,
  type TimeZone,
  type TypedArray,
} from "@synnaxlabs/x";

export type Policy = access.policy.Policy;
export type Role = access.role.Role;
export type Param = arc.types.Param;
export const Channel = channel.Channel;
export type Channel = channel.Channel;
export const isCalculated = channel.isCalculated;
export const Frame = framer.Frame;
export type Frame = framer.Frame;
