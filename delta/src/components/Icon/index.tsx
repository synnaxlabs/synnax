// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { AiFillDelete, AiOutlineMinus, AiOutlinePlus } from "react-icons/ai";
import { IoMdRefresh } from "react-icons/io";
import { IoCopySharp, IoTime } from "react-icons/io5";
import { MdEdit } from "react-icons/md";
import { SiPython, SiTypescript } from "react-icons/si";

export const Icon = {
  Edit: MdEdit,
  Add: AiOutlinePlus,
  Subtract: AiOutlineMinus,
  Copy: IoCopySharp,
  Refresh: IoMdRefresh,
  Delete: AiFillDelete,
  Python: SiPython,
  Typescript: SiTypescript,
  Time: IoTime,
};
