// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/credential/useModal.css";

import { type resource } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, useState } from "react";

import { Keys } from "@/platform/credential/Keys";
import { Password } from "@/platform/credential/Password";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";

export interface ModalParams {
  /** The user or rack whose credentials to manage. */
  subject: resource.ID;
  title?: string | string[];
}

const Credentials = ({
  subject,
  title,
}: Modals.ContentProps<ModalParams, void>): ReactElement => {
  const [passwordChanged, setPasswordChanged] = useState(false);
  return (
    <Modal.Frame className={CSS.B("credential")}>
      <Modal.Header icon={<Icon.Lock />}>{title ?? "Sign-in"}</Modal.Header>
      <Modal.Body>
        <Flex.Box y gap="large">
          <Flex.Box y>
            <Text.Text level="h5">API keys</Text.Text>
            <Keys subject={subject} />
          </Flex.Box>
          {subject.type === "user" && (
            <Flex.Box y>
              <Text.Text level="h5">Password</Text.Text>
              <Password subject={subject} onChanged={setPasswordChanged} />
            </Flex.Box>
          )}
        </Flex.Box>
      </Modal.Body>
      {passwordChanged && (
        <Modal.Footer>
          <Status.Summary variant="success" message="Successfully changed password" />
        </Modal.Footer>
      )}
    </Modal.Frame>
  );
};

export const useModal = Modals.create<ModalParams>(Credentials);
