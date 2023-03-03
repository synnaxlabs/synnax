// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { OS } from "@synnaxlabs/x";
import {
  AiFillBoxPlot,
  AiFillCaretDown,
  AiFillCaretLeft,
  AiFillCaretRight,
  AiFillCaretUp,
  AiFillDelete,
  AiFillFolder,
  AiFillInfoCircle,
  AiFillWarning,
  AiOutlineBorder,
  AiOutlineCheck,
  AiOutlineClose,
  AiOutlineMinus,
} from "react-icons/ai";
import { BsStack, BsLightbulbFill } from "react-icons/bs";
import {
  FaApple,
  FaBezierCurve,
  FaDocker,
  FaLinux,
  FaStream,
  FaWindows,
} from "react-icons/fa";
import { HiChartBar, HiDownload, HiLightningBolt, HiOutlinePlus } from "react-icons/hi";
import { IoMdRefresh } from "react-icons/io";
import { IoBookSharp, IoCopySharp, IoTime } from "react-icons/io5";
import {
  MdEdit,
  MdOutlineDeviceHub,
  MdSensors,
  MdWorkspacesFilled,
} from "react-icons/md";
import { SiPython, SiTypescript } from "react-icons/si";

const IconOS: Record<OS, React.ComponentType> = {
  Linux: FaLinux,
  MacOS: FaApple,
  Windows: FaWindows,
  Docker: FaDocker,
};

export const Icon = {
  Edit: MdEdit,
  Add: HiOutlinePlus,
  Subtract: AiOutlineMinus,
  Copy: IoCopySharp,
  Close: AiOutlineClose,
  Info: AiFillInfoCircle,
  Warning: AiFillWarning,
  Check: AiOutlineCheck,
  Refresh: IoMdRefresh,
  Delete: AiFillDelete,
  Python: SiPython,
  Typescript: SiTypescript,
  Time: IoTime,
  Acquire: FaStream,
  Analyze: FaBezierCurve,
  Concepts: BsLightbulbFill,
  Visualize: HiChartBar,
  Cluster: BsStack,
  Caret: {
    Right: AiFillCaretRight,
    Left: AiFillCaretLeft,
    Up: AiFillCaretUp,
    Down: AiFillCaretDown,
  },
  Reference: IoBookSharp,
  Bolt: HiLightningBolt,
  Download: HiDownload,
  Range: AiFillBoxPlot,
  Node: MdOutlineDeviceHub,
  Channel: MdSensors,
  Resources: AiFillFolder,
  Workspace: MdWorkspacesFilled,
  OS: IconOS,
  Box: AiOutlineBorder,
};
