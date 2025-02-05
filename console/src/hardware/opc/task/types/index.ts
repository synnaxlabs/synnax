// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import * as v0 from "@/hardware/opc/task/types/v0";

// General
export const PREFIX = v0.PREFIX;

// Reads
export const READ_TYPE = v0.READ_TYPE;
export type ReadType = v0.ReadType;
export type ReadChannelConfig = v0.ReadChannelConfig;
export const readStateDetails = v0.readStateDetails;
export type ReadStateDetails = v0.ReadStateDetails;
export type ReadState = v0.ReadState;
export const readChanZ = v0.readChanZ;
export const readConfigZ = v0.readConfigZ;
export type ReadConfig = v0.ReadConfig;
export type Read = v0.Read;
export type ReadPayload = v0.ReadPayload;
export const ZERO_READ_PAYLOAD = v0.ZERO_READ_PAYLOAD;

// Writes
export const WRITE_TYPE = v0.WRITE_TYPE;
export type WriteType = v0.WriteType;
export type WriteChannelConfig = v0.WriteChannelConfig;
export const writeStateDetails = v0.writeStateDetails;
export type WriteStateDetails = v0.WriteStateDetails;
export type WriteState = v0.WriteState;
export const writeChanZ = v0.writeChanZ;
export const writeConfigZ = v0.writeConfigZ;
export type WriteConfig = v0.WriteConfig;
export type Write = v0.Write;
export type WritePayload = v0.WritePayload;
export const ZERO_WRITE_PAYLOAD = v0.ZERO_WRITE_PAYLOAD;

// Scan
export const TEST_CONNECTION_COMMAND = v0.TEST_CONNECTION_COMMAND;
export const SCAN_NAME = v0.SCAN_NAME;
export interface TestConnectionCommandResponse
  extends v0.TestConnectionCommandResponse {}
export interface TestConnectionCommandState extends v0.TestConnectionCommandState {}
export type ScanCommandResult = v0.ScanCommandResult;

export interface NodeId extends v0.NodeId {}
export const parseNodeId = v0.parseNodeId;
export const nodeIdToString = v0.nodeIdToString;
