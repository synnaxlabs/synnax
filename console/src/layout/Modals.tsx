// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layout/Modals.css";

import { type ReactElement } from "react";

import { Modal } from "@/layout/Modal";
import { useSelectWindowModals } from "@/layout/selectors";
import { useRemover } from "@/layout/useRemover";

export const Modals = (): ReactElement => {
  const layouts = useSelectWindowModals();
  const remove = useRemover();
  return (
    <>
      {layouts.map((l) => (
        <Modal key={l.key} state={l} remove={remove} />
      ))}
    </>
  );
};
