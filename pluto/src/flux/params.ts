// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type primitive } from "@synnaxlabs/x";

/**
 * Parameters used to retrieve and or/update a resource from within a query. The query
 * re-executes whenever the parameters change.
 */
export type Params = Record<string, primitive.Value>;
