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
  AiOutlineExpand,
} from "react-icons/ai";
import { BsStack, BsLightbulbFill } from "react-icons/bs";
import {
  FaApple,
  FaBezierCurve,
  FaDiceD20,
  FaDocker,
  FaLinux,
  FaStream,
  FaWindows,
} from "react-icons/fa";
import { FiMenu } from "react-icons/fi";
import { HiDownload, HiLightningBolt, HiOutlinePlus } from "react-icons/hi";
import { IoMdRefresh } from "react-icons/io";
import { IoBookSharp, IoCopySharp, IoTime } from "react-icons/io5";
import {
  MdEdit,
  MdOutlineDeviceHub,
  MdQuestionMark,
  MdSensors,
  MdWorkspacesFilled,
  MdAreaChart,
  MdEditOff,
} from "react-icons/md";
import { SiNpm, SiPnpm, SiPython, SiTypescript, SiYarn } from "react-icons/si";

const IconOS: Record<OS, React.ComponentType> = {
  Linux: FaLinux,
  MacOS: FaApple,
  Windows: FaWindows,
  Docker: FaDocker,
};

export const Icon: IconType = {
  Edit: MdEdit,
  EditOff: MdEditOff,
  Add: HiOutlinePlus,
  Subtract: AiOutlineMinus,
  Copy: IoCopySharp,
  Close: AiOutlineClose,
  Info: AiFillInfoCircle,
  Warning: AiFillWarning,
  Check: AiOutlineCheck,
  Refresh: IoMdRefresh,
  Delete: AiFillDelete,
  Time: IoTime,
  Acquire: FaStream,
  Analyze: FaBezierCurve,
  Concepts: BsLightbulbFill,
  Visualize: MdAreaChart,
  Expand: AiOutlineExpand,
  Cluster: BsStack,
  Control: FaDiceD20,
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
  Python: SiPython,
  Typescript: SiTypescript,
  NPM: SiNpm,
  PNPM: SiPnpm,
  Yarn: SiYarn,
  QuestionMark: MdQuestionMark,
  Menu: FiMenu,
};

export interface IconType {
  Edit: React.ComponentType;
  EditOff: React.ComponentType;
  Add: React.ComponentType;
  Subtract: React.ComponentType;
  Copy: React.ComponentType;
  Close: React.ComponentType;
  Info: React.ComponentType;
  Warning: React.ComponentType;
  Check: React.ComponentType;
  Refresh: React.ComponentType;
  Delete: React.ComponentType;
  Time: React.ComponentType;
  Acquire: React.ComponentType;
  Analyze: React.ComponentType;
  Concepts: React.ComponentType;
  Expand: React.ComponentType;
  Visualize: React.ComponentType;
  Cluster: React.ComponentType;
  Control: React.ComponentType;
  Caret: {
    Right: React.ComponentType;
    Left: React.ComponentType;
    Up: React.ComponentType;
    Down: React.ComponentType;
  };
  Reference: React.ComponentType;
  Bolt: React.ComponentType;
  Download: React.ComponentType;
  Range: React.ComponentType;
  Node: React.ComponentType;
  Channel: React.ComponentType;
  Resources: React.ComponentType;
  Workspace: React.ComponentType;
  OS: Record<OS, React.ComponentType>;
  Box: React.ComponentType;
  Python: React.ComponentType;
  Typescript: React.ComponentType;
  NPM: React.ComponentType;
  PNPM: React.ComponentType;
  Yarn: React.ComponentType;
  QuestionMark: React.ComponentType;
  Menu: React.ComponentType;
}
