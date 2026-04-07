// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

<<<<<<<< HEAD:core/pkg/service/status/migrations/v0/status.go
package v0

import status "github.com/synnaxlabs/x/status/migrations/v0"

type Status[Details any] = status.Status[Details]
========
export * as Task from "@/hardware/pagerduty/task/external";
>>>>>>>> 34b0954637458d9753d6f627309f10bb2db32291:console/src/hardware/pagerduty/task/index.ts
