// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type enumeration } from "@synnaxlabs/client";
import { Button } from "@synnaxlabs/lyra/button";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Input } from "@synnaxlabs/lyra/input";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Enumeration } from "@synnaxlabs/pluto";
import { uuid } from "@synnaxlabs/x";
import { type PropsWithChildren, type ReactElement, useState } from "react";

import { Modals } from "@/platform/modals";
import { Triggers } from "@/platform/triggers";

const Create = ({
  close,
  name: initialName,
  entries,
}: Modals.ContentProps<
  Enumeration.CreatePromptParams,
  enumeration.Key
>): ReactElement => {
  const [name, setName] = useState(initialName);
  const [error, setError] = useState<string>();
  const { updateAsync, variant } = Enumeration.useCreate();
  const loading = variant === "loading";
  Modals.useBusy(loading);
  const handleChange = (next: string): void => {
    setName(next);
    setError(undefined);
  };
  const handleCreate = (): void => {
    const trimmed = name.trim();
    if (trimmed === "") return setError("Name is required");
    const key = uuid.create();
    void (async () => {
      if ((await updateAsync({ key, name: trimmed, entries })) !== false) close(key);
    })();
  };
  return (
    <Modal.Frame>
      <Modal.Header icon={<Icon.Enumeration />}>Enumeration.Create</Modal.Header>
      <Modal.Body>
        <Input.Item
          showLabel={false}
          helpText={error}
          status={error == null ? undefined : "error"}
          padHelpText
        >
          <Input.Text
            value={name}
            onChange={handleChange}
            placeholder="Enumeration name"
            aria-label="Enumeration name"
            variant="text"
            autoFocus
            selectOnFocus
            level="h3"
            full="x"
          />
        </Input.Item>
      </Modal.Body>
      <Modal.Footer>
        <Nav.Bar.End>
          <Button.Button
            variant="filled"
            status={loading ? "loading" : undefined}
            onClick={handleCreate}
            trigger={Triggers.SAVE}
            triggerIndicator
          >
            Create
          </Button.Button>
        </Nav.Bar.End>
      </Modal.Footer>
    </Modal.Frame>
  );
};

const useCreateModal = Modals.createPrompt<
  enumeration.Key,
  Enumeration.CreatePromptParams
>(Create);

/** Gives the symbol forms below it the modal that names a new enumeration. */
export const CreatePromptProvider = ({ children }: PropsWithChildren): ReactElement => {
  const prompt = useCreateModal();
  return (
    <Enumeration.CreatePromptProvider value={prompt}>
      {children}
    </Enumeration.CreatePromptProvider>
  );
};
