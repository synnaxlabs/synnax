/*
 * Copyright 2024 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

import { Button, Divider, Status } from "@synnaxlabs/pluto";
import { caseconv } from "@synnaxlabs/x";

import { controlsLayout, STATUS_MAP } from "@/cluster/embedded/types";
import { useSelectEmbeddedState } from "@/cluster/selectors";
import { Layout } from "@/layout";

export const Toolbar = () => {
  const { status } = useSelectEmbeddedState();
  const p = Layout.usePlacer();
  return (
    <>
      <Divider.Divider />
      <Button.Button variant="text" onClick={() => p(controlsLayout)}>
        <Status.Text variant={STATUS_MAP[status]}>
          {caseconv.capitalize(status)}
        </Status.Text>
      </Button.Button>
    </>
  );
};
