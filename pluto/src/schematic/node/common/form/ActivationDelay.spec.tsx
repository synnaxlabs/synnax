// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { render } from "@testing-library/react";
import { type ReactElement } from "react";
import { describe, expect, it } from "vitest";
import { z } from "zod";

import { Form as Base } from "@/form";
import { Form } from "@/schematic/node/common/form";

const schema = z.object({ onClickDelay: z.number().optional() });

const Host = ({ values }: { values: z.infer<typeof schema> }): ReactElement => {
  const methods = Base.use({ values, schema });
  return (
    <Base.Form<typeof schema> {...methods}>
      <Form.ActivationDelayField />
    </Base.Form>
  );
};

describe("ActivationDelayField", () => {
  it("should show the saved delay", () => {
    const c = render(<Host values={{ onClickDelay: 250 }} />);
    expect(c.getByLabelText(/Activation delay/)).toHaveProperty("value", "250");
  });

  // A symbol saved before it had a delay has no key at all; the field must still show.
  it("should show a zero delay when the config has no key", () => {
    const c = render(<Host values={{}} />);
    expect(c.getByLabelText(/Activation delay/)).toHaveProperty("value", "0");
  });
});
