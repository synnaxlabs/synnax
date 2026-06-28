// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ReactElement } from "react";

import { Modal } from "@/layered/app/modals/Modal";
import { useStack } from "@/layered/session/modals/Provider";

export const Surface = (): ReactElement => {
  const stack = useStack();
  return (
    <>
      {stack.map((entry) => (
        <Modal key={entry.key} entry={entry} />
      ))}
    </>
  );
};
