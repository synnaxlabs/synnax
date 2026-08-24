// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

authority 200

start_sim_cmd => main

sequence main {
    stage open_ox_mpv {
        1 -> ox_mpv_cmd
        wait{duration=500ms} => next
    }
    stage open_fuel_mpv {
        1 -> fuel_mpv_cmd
    }
}
