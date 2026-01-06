// Copyright 2026 Synnax Labs, Inc.
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

export const MAKE = v0.MAKE;
export type Make = v0.Make;

export const NO_SECURITY_MODE = v0.NO_SECURITY_MODE;
export const SIGN_SECURITY_MODE = v0.SIGN_SECURITY_MODE;
export const SIGN_AND_ENCRYPT_SECURITY_MODE = v0.SIGN_AND_ENCRYPT_SECURITY_MODE;
export type SecurityMode = v0.SecurityMode;

export const NO_SECURITY_POLICY = v0.NO_SECURITY_POLICY;
export const BASIC128_RSA15_SECURITY_POLICY = v0.BASIC128_RSA15_SECURITY_POLICY;
export const BASIC256_SECURITY_POLICY = v0.BASIC256_SECURITY_POLICY;
export const BASIC256_SHA256_SECURITY_POLICY = v0.BASIC256_SHA256_SECURITY_POLICY;
export const AES128_SHA256_RSAOAEP_SECURITY_POLICY =
  v0.AES128_SHA256_RSAOAEP_SECURITY_POLICY;
export const AES256_SHA256_RSAPSS_SECURITY_POLICY =
  v0.AES256_SHA256_RSAPSS_SECURITY_POLICY;
export type SecurityPolicy = v0.SecurityPolicy;

export const connectionConfigZ = v0.connectionConfigZ;
export interface ConnectionConfig extends v0.ConnectionConfig {}
export const ZERO_CONNECTION_CONFIG = v0.ZERO_CONNECTION_CONFIG;

export interface Properties extends v1.Properties {}
export const ZERO_PROPERTIES = v1.ZERO_PROPERTIES;

export interface Device extends v1.Device {}
export interface New extends v1.New {}

const PROPERTIES_MIGRATIONS: migrate.Migrations = {
  [v0.VERSION]: v1.propertiesMigration,
};
export type AnyProperties = v0.Properties | v1.Properties;

export const migrateProperties = migrate.migrator<AnyProperties, v1.Properties>({
  name: v1.PROPERTIES_MIGRATION_NAME,
  migrations: PROPERTIES_MIGRATIONS,
  defaultVersion: v0.VERSION,
  def: v1.ZERO_PROPERTIES,
});
