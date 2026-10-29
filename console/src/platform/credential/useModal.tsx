// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ontology } from "@synnaxlabs/client";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { type ReactElement } from "react";

import { APIKey } from "@/platform/credential/apikey";
import { Password } from "@/platform/credential/password";
import { Modals } from "@/platform/modals";

export interface ModalParams {
  /** The user or rack whose credentials to manage. */
  subject: ontology.ID;
  title?: string | string[];
}

const Credentials = ({
  subject,
  title,
}: Modals.ContentProps<ModalParams, void>): ReactElement => (
  <Modal.Frame>
    <Modal.Header icon={<Icon.Lock />}>{title ?? "Credentials"}</Modal.Header>
    <Modal.Body>
      <Flex.Box y gap="large">
        <APIKey.Section subject={subject} />
        {subject.type === "user" && <Password.Section subject={subject} />}
      </Flex.Box>
    </Modal.Body>
  </Modal.Frame>
);

export const useModal = Modals.create<ModalParams>(Credentials);
