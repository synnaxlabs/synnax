// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { connectServerScript, INTEGRATIONS, OPCUA } from "./common/server";

export const driver = true;
export const integrations = INTEGRATIONS;

/**
 * Docs `reference/hardware/opc-ua/connect-server` (Connect an OPC UA server): connect a
 * simulated OPC UA server by its endpoint, with no login and no security.
 */
export default connectServerScript(OPCUA);
