// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { deep, TimeSpan } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor } from "@testing-library/react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { renderForm } from "@/testutil";
import { Staleness } from "@/vis/staleness";

const schema = z.object({ stalenessTimeout: z.number().optional() });
const path = deep.path<z.infer<typeof schema>>();

const field = (): HTMLInputElement =>
  screen.getByLabelText<HTMLInputElement>("Timeout");

const typeTimeout = (value: string): void => {
  fireEvent.focus(field());
  fireEvent.change(field(), { target: { value } });
  fireEvent.blur(field());
};

describe("Staleness.Fields", () => {
  it("should show and store a timeout in nanoseconds by default", async () => {
    const { value } = renderForm(
      <Staleness.Fields />,
      { stalenessTimeout: Number(TimeSpan.seconds(3).valueOf()) },
      schema,
    );
    expect(field().value).toBe("3s");
    typeTimeout("7s");
    await waitFor(() =>
      expect(value(path.stalenessTimeout)).toBe(Number(TimeSpan.seconds(7).valueOf())),
    );
  });

  it("should show and store a timeout in the given unit", async () => {
    const { value } = renderForm(
      <Staleness.Fields unit={TimeSpan.SECOND} />,
      { stalenessTimeout: 3 },
      schema,
    );
    expect(field().value).toBe("3s");
    typeTimeout("7s");
    await waitFor(() => expect(value(path.stalenessTimeout)).toBe(7));
  });
});
