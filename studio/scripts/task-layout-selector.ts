// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createComponentScript } from "./common/create-component";

/**
 * Docs `reference/driver/task-basics` (Layout Selector tab): from an empty panel,
 * click the mosaic "+" to open the component selector and pick a task type; the task
 * configuration form opens in its place as a tab.
 */
export default createComponentScript("NI analog read task", ".console-task-configure");
