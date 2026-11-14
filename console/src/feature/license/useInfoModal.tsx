// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/license/useInfoModal.css";

import { Icon } from "@synnaxlabs/lyra/icon";
import { Modal } from "@synnaxlabs/lyra/modal";
import { type ReactElement } from "react";

import { Info } from "@/feature/license/Info";
import { CSS } from "@/platform/css";
import { Modals } from "@/platform/modals";

const Content = (): ReactElement => (
  <Modal.Frame className={CSS.B("license-info")}>
    <Modal.Header icon={<Icon.License />}>License</Modal.Header>
    <Modal.Body className={CSS.BE("license-info", "body")} gap="huge">
      <Info />
    </Modal.Body>
  </Modal.Frame>
);

/**
 * Opens a dialog with the active Core's license and the facts support asks for: the
 * license's claims, the host fingerprint, and the Core and Console versions.
 */
export const useInfoModal = Modals.create(Content);
