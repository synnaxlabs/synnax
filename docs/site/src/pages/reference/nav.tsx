// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { IoBookSharp } from "react-icons/io5";

import { clientCLINav } from "./client-cli/nav";

import { PageNavLeaf } from "@/components/PageNav";

export const referenceNav: PageNavLeaf = {
  key: "reference",
  name: "Reference",
  icon: <IoBookSharp />,
  children: [clientCLINav],
};
