// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/credential/useModal.css";

import { type ontology } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Nav } from "@synnaxlabs/lyra/nav";
import { Status } from "@synnaxlabs/lyra/status";
import { Synnax } from "@synnaxlabs/pluto";
import { type ReactElement, useState } from "react";

import { Core } from "@/platform/core";
import { APIKey } from "@/platform/credential/apikey";
import { OIDC } from "@/platform/credential/oidc";
import { Password } from "@/platform/credential/password";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";

export interface ModalParams {
  /** The user or rack whose credentials to manage. */
  subject: ontology.ID;
  title?: string | string[];
}

const Credentials = ({
  subject,
  title,
}: Modals.ContentProps<ModalParams, void>): ReactElement => {
  const client = Synnax.use();
  const { password, oidc } = Core.useAcceptedMethods(client?.params);
  const [message, setMessage] = useState<string | null>(null);
  return (
    <Modal.Frame className={CSS.B("credential")}>
      <Modal.Header icon={<Icon.Lock />}>{title ?? "Credentials"}</Modal.Header>
      <Modal.Body>
        <Flex.Box y gap="large">
          <APIKey.Section subject={subject} />
          {subject.type === "user" && password && (
            <Password.Section subject={subject} onMessage={setMessage} />
          )}
          {subject.type === "user" && oidc && (
            <OIDC.Section subject={subject} onMessage={setMessage} />
          )}
        </Flex.Box>
      </Modal.Body>
      {message != null && (
        <Modal.Footer>
          <Nav.Bar.Start>
            <Status.Summary variant="success" message={message} />
          </Nav.Bar.Start>
        </Modal.Footer>
      )}
    </Modal.Frame>
  );
};

export const useModal = Modals.create<ModalParams>(Credentials);
