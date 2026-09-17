// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Show, UserButton } from "@clerk/astro/react";
import { Button } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

/** Account is the header control: a sign-in link when signed out, the user menu when
 * signed in. */
export const Account = (): ReactElement => (
  <>
    <Show when="signed-out">
      <Button.Button className="account-button" variant="outlined" href="/sign-in">
        Sign in
      </Button.Button>
    </Show>
    <Show when="signed-in">
      <Button.Button className="account-button" variant="text" href="/licenses">
        Licenses
      </Button.Button>
      <UserButton />
    </Show>
  </>
);
