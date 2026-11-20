// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { connectServerScript, INTEGRATIONS, MQTT } from "./common/server";

export const driver = true;
export const integrations = INTEGRATIONS;

/**
 * Docs `reference/hardware/mqtt/connect-broker` (Connect an MQTT broker): connect a
 * simulated broker by its host, with the default port, from the rack of the Core.
 */
export default connectServerScript(MQTT);
