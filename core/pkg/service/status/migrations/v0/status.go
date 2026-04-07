// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

package v0

import status "github.com/synnaxlabs/x/status/migrations/v0"

<<<<<<<< HEAD:core/pkg/service/log/migrate.gen.go
import (
	"github.com/synnaxlabs/x/gorp"
)

func LogMigrations() []gorp.Migration {
	return []gorp.Migration{
		gorp.NewCodecTransition[Key, Log]("msgpack_to_binary", LogCodec),
	}
}
========
type Status[Details any] = status.Status[Details]
>>>>>>>> 34b0954637458d9753d6f627309f10bb2db32291:core/pkg/service/status/migrations/v0/status.go
