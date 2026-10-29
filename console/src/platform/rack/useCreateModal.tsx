// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { status } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Rack, Synnax } from "@synnaxlabs/pluto";
import { type ReactElement, useEffect, useRef, useState } from "react";

import { Credential } from "@/platform/credential";
import { Modals } from "@/platform/modals";

const Create = (_: Modals.ContentProps<Record<never, never>, void>): ReactElement => {
  const client = Synnax.use();
  const [name, setName] = useState("");
  const [token, setToken] = useState<string | null>(null);
  const mounted = useRef(true);
  useEffect(
    () => () => {
      mounted.current = false;
    },
    [],
  );
  const create = Rack.useCreate({
    afterSuccess: async ({ data }) => {
      if (mounted.current) return setToken(data.token);
      // unmounted before the key could be shown, so nobody holds its secret
      await client?.auth.apiKeys.delete(data.key.key);
    },
  });
  // a user who cannot delete keys could not clean up a key created after a close
  const creating = create.variant === "loading";
  Modals.useLock(creating);
  return (
    <Modal.Frame>
      <Modal.Header icon={<Icon.Rack />} closeDisabled={creating}>
        Rack.Create
      </Modal.Header>
      <Modal.Body>
        <Flex.Box y>
          <Input.Text
            value={name}
            onChange={setName}
            placeholder="Rack name"
            disabled={token != null}
            autoFocus
            full="x"
          />
          {token != null && <Credential.APIKey.Token token={token} />}
        </Flex.Box>
      </Modal.Body>
      <Modal.Footer>
        <Nav.Bar.End>
          <Button.Button
            variant="filled"
            status={status.keepVariants(create.variant, "loading")}
            disabled={client == null || name === "" || token != null}
            onClick={() => create.update({ name })}
          >
            Create
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

/** Opens a modal that creates a rack together with the API key its Driver uses. */
export const useCreateModal = Modals.create(Create);
