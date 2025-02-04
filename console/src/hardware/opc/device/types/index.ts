// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { migrate } from "@synnaxlabs/x";

import * as v0 from "@/hardware/opc/device/types/v0";
import * as v1 from "@/hardware/opc/device/types/v1";

export const securityModeZ = v0.securityModeZ;
export type SecurityMode = v0.SecurityMode;
export const securityPolicyZ = v0.securityPolicyZ;
export type SecurityPolicy = v0.SecurityPolicy;
export const connectionConfigZ = v0.connectionConfigZ;
export type ConnectionConfig = v0.ConnectionConfig;
export const ZERO_CONNECTION_CONFIG = v0.ZERO_CONNECTION_CONFIG;
export const scannedNodeZ = v0.scannedNodeZ;
export type ScannedNode = v0.ScannedNode;
export const propertiesZ = v1.propertiesZ;
export type Properties = v1.Properties;
export const ZERO_PROPERTIES = v1.ZERO_PROPERTIES;
export type Device = v1.Device;
export interface TestConnCommandResponse extends v0.TestConnCommandResponse {}
export type TestConnCommandState = v0.TestConnCommandState;
export const scannerScanCommandResult = v0.scannerScanCommandResult;
export type ScannerScanCommandResult = v0.ScannerScanCommandResult;

const PROPERTIES_MIGRATIONS: migrate.Migrations = {
  "0.0.0": v1.propertiesMigration,
};
export type AnyProperties = v0.Properties | v1.Properties;

export const migrateProperties = migrate.migrator<AnyProperties, v1.Properties>({
  name: "hardware.opc.device.properties",
  migrations: PROPERTIES_MIGRATIONS,
  defaultVersion: "0.0.0",
  def: v1.ZERO_PROPERTIES,
});
