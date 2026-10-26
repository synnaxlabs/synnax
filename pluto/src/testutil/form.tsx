// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { render, type RenderResult } from "@testing-library/react";
import { type FC, type PropsWithChildren, type ReactElement, useEffect } from "react";
import { type z } from "zod";

export interface RenderFormResult extends RenderResult {
  /** Reads the form's current value at the path. */
  value: (path: string) => unknown;
}

/**
 * Renders a form body inside a form that holds the given values, so a spec can drive
 * the form's inputs and read back what they stored.
 */
export const renderForm = <Z extends z.ZodType>(
  ui: ReactElement,
  values: NoInfer<z.infer<Z>>,
  schema: Z,
  wrapper?: FC<PropsWithChildren>,
): RenderFormResult => {
  let read: (path: string) => unknown = () => undefined;
  const setRead = (next: typeof read): void => {
    read = next;
  };
  const Harness = (): ReactElement => {
    const methods = Form.use({ values, schema });
    useEffect(() => setRead((path) => methods.get(path, { optional: true })?.value));
    return <Form.Form<Z> {...methods}>{ui}</Form.Form>;
  };
  const result = render(<Harness />, { wrapper });
  return { ...result, value: (path) => read(path) };
};
