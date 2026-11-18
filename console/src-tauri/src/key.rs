// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

//! Prints the API key that Synnax Desktop saved in the OS keychain to stdout. The
//! discovery file names this program, so scripts on this computer can read the key.
//! Exits non-zero with one line on stderr when there is no key or no keychain.

use std::io;
use std::process::ExitCode;

use desktop::discovery;

fn main() -> ExitCode {
    discovery::print_key(
        discovery::key_entry(),
        &mut io::stdout().lock(),
        &mut io::stderr().lock(),
    )
}
