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
import { Note } from "@synnaxlabs/lyra/note";
import { Text } from "@synnaxlabs/lyra/text";
import { Rack, Synnax } from "@synnaxlabs/pluto";
import { type ReactElement, useEffect, useRef, useState } from "react";

import { Code } from "@/platform/code";
import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

interface NameProps {
  onCreated: (created: Rack.CreateResult) => void;
}

const Name = ({ onCreated }: NameProps): ReactElement => {
  const client = Synnax.use();
  const [name, setName] = useState("");
  const mounted = useRef(true);
  useEffect(
    () => () => {
      mounted.current = false;
    },
    [],
  );
  const create = Rack.useCreate({
    afterSuccess: async ({ data }) => {
      if (mounted.current) return onCreated(data);
      // unmounted before the key could be shown, so nobody holds its secret
      await client?.auth.deleteKeys(data.key.key);
    },
  });
  // a user who cannot delete keys could not clean up a key created after a close
  Modals.useBusy(create.variant === "loading");
  return (
    <>
      <Modal.Body>
        <Input.Text
          level="h3"
          variant="text"
          value={name}
          onChange={setName}
          placeholder="Rack name"
          aria-label="Rack name"
          autoFocus
          full="x"
        />
      </Modal.Body>
      <Modal.Footer>
        <Nav.Bar.End>
          <Button.Button
            variant="filled"
            status={status.keepVariants(create.variant, "loading")}
            disabled={client == null || name.trim() === ""}
            onClick={() => create.update({ name: name.trim() })}
            trigger={Triggers.SAVE}
            triggerIndicator
          >
            Create
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </>
  );
};

interface KeyProps {
  created: Rack.CreateResult;
  close: () => void;
}

const Key = ({ created, close }: KeyProps): ReactElement => (
  <>
    <Modal.Body>
      <Flex.Box y>
        <Text.Text level="h4">{`Copy the key for ${created.rack.name}`}</Text.Text>
        <Note.Note variant="warning" y gap="small">
          <Text.Text weight={500} status="warning">
            <Icon.Warning />
            You can only see this key once
          </Text.Text>
          <Text.Text level="small" color={9}>
            Give it to the Driver on the rack. If you lose it, create a new key.
          </Text.Text>
        </Note.Note>
        <Code.Snippet label="API key" code={[[created.token]]} primary />
      </Flex.Box>
    </Modal.Body>
    <Modal.Footer>
      <Nav.Bar.End>
        <Button.Button variant="outlined" onClick={close}>
          Done
        </Button.Button>
      </Nav.Bar.End>
    </Modal.Footer>
  </>
);

const Create = ({
  close,
}: Modals.ContentProps<Record<never, never>, void>): ReactElement => {
  const [created, setCreated] = useState<Rack.CreateResult | null>(null);
  return (
    <Modal.Frame>
      <Modal.Header icon={<Icon.Rack />}>Rack.Create</Modal.Header>
      {created == null ? (
        <Name onCreated={setCreated} />
      ) : (
        <Key created={created} close={() => close()} />
      )}
    </Modal.Frame>
  );
};

/** Opens a modal that creates a rack together with the API key its Driver uses. */
export const useCreateModal = Modals.create(Create);
