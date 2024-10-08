// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export { access } from "@/access";
export { policy } from "@/access/policy";
export * from "@/channel";
export { Channel } from "@/channel/client";
export { default as Synnax, type SynnaxProps, synnaxPropsZ } from "@/client";
export * from "@/connection";
export { control } from "@/control";
export {
  AuthError,
  ContiguityError,
  MultipleFoundError,
  NotFoundError,
  QueryError,
  RouteError,
  UnexpectedError,
  ValidationError,
} from "@/errors";
export { framer } from "@/framer";
export { Frame } from "@/framer/frame";
export { hardware } from "@/hardware";
export { device } from "@/hardware/device";
export { rack } from "@/hardware/rack";
export { task } from "@/hardware/task";
export { label } from "@/label";
export { ontology } from "@/ontology";
export { ranger } from "@/ranger";
export { signals } from "@/signals";
export { user } from "@/user";
export { workspace } from "@/workspace";
export { linePlot } from "@/workspace/lineplot";
export { schematic } from "@/workspace/schematic";
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
  type NumericTelemValue,
  Rate,
  Series,
  type TelemValue,
  TimeRange,
  TimeSpan,
  TimeStamp,
  type TimeStampStringFormat,
  type TypedArray,
  type TZInfo,
} from "@synnaxlabs/x/telem";
import { control } from "@synnaxlabs/x";
export const Authority = control.Authority;
