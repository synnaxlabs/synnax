// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import * as v0 from "@/hardware/opc/device/types/v0";
import * as v1 from "@/hardware/opc/device/types/v1";

export const securityModeZ = v0.securityModeZ;
export type SecurityMode = v0.SecurityMode;
export const securityPolicyZ = v0.securityPolicyZ;
export type SecurityPolicy = v0.SecurityPolicy;
export const connectionConfigZ = v0.connectionConfigZ;
export type ConnectionConfig = v0.ConnectionConfig;
export const scannedNodeZ = v0.scannedNodeZ;
export type ScannedNode = v0.ScannedNode;
export const propertiesZ = v1.propertiesZ;
export type Properties = v1.Properties;
export type Device = v1.Device;
export interface TestConnCommandResponse extends v0.TestConnCommandResponse {}
export type TestConnCommandState = v0.TestConnCommandState;
export const scannerScanCommandResult = v0.scannerScanCommandResult;
export type ScannerScanCommandResult = v0.ScannerScanCommandResult;
