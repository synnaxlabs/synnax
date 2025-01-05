// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch, type Store } from "@reduxjs/toolkit";
import { type Synnax } from "@synnaxlabs/client";
import { type location, type UnknownRecord } from "@synnaxlabs/x";

import { type CreateConfirmModal } from "@/confirm/Confirm";
import { type Placer } from "@/layout/hooks";

export interface FileHandlerProps {
  file: UnknownRecord;
  name: string;
  place: Placer;
  store: Store;
  confirm: CreateConfirmModal;
  client: Synnax | null;
  workspaceKey?: string;
  dispatch: Dispatch;
  tab?: { mosaicKey: number; location: location.Location };
}

export type FileHandler = (props: FileHandlerProps) => Promise<boolean>;
