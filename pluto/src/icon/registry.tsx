// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/icon/registry.css";

import { deep } from "@synnaxlabs/x";
import { cloneElement, type FC } from "react";
import {
  AiFillDelete,
  AiFillFolder,
  AiFillGithub,
  AiFillLinkedin,
  AiFillWarning,
  AiOutlineBorder,
  AiOutlineCheck,
  AiOutlineClose,
  AiOutlineExpand,
  AiOutlineLoading,
  AiOutlineMinus,
  AiOutlineSync,
} from "react-icons/ai";
import { BiLinkExternal, BiMath, BiRename } from "react-icons/bi";
import {
  BsBorderWidth,
  BsFillInfoSquareFill,
  BsLightbulbFill,
  BsShiftFill,
} from "react-icons/bs";
import {
  FaAlignCenter,
  FaAlignLeft,
  FaAlignRight,
  FaApple,
  FaBezierCurve,
  FaCarSide,
  FaCreativeCommonsZero,
  FaDocker,
  FaLinux,
  FaMicrophone,
  FaRegStar,
  FaStar,
  FaStream,
  FaWind,
  FaWindows,
} from "react-icons/fa";
import {
  FaBridge,
  FaCheck,
  FaGaugeHigh,
  FaGear,
  FaHelmetSafety,
  FaXmark,
} from "react-icons/fa6";
import { FiTable } from "react-icons/fi";
import { GiHamburgerMenu } from "react-icons/gi";
import { GoNumber } from "react-icons/go";
import { GrAttachment, GrDrag, GrPan, GrRotateRight } from "react-icons/gr";
import { HiCursorClick, HiLightningBolt, HiOutlinePlus } from "react-icons/hi";
import { HiSquare3Stack3D } from "react-icons/hi2";
import { IoMdColorFill, IoMdHeart, IoMdRefresh } from "react-icons/io";
import {
  IoBookSharp,
  IoCopy,
  IoNotifications,
  IoNotificationsOff,
  IoShapes,
  IoTerminal,
  IoTime,
} from "react-icons/io5";
import {
  MdAlignHorizontalCenter,
  MdAlignHorizontalLeft,
  MdAlignHorizontalRight,
  MdAlignVerticalBottom,
  MdAlignVerticalCenter,
  MdAlignVerticalTop,
  MdAreaChart,
  MdArrowOutward,
  MdAutoAwesome,
  MdBlurLinear,
  MdBook,
  MdBorderColor,
  MdCalendarToday,
  MdComment,
  MdCommit,
  MdDarkMode,
  MdDataArray,
  MdDataObject,
  MdEdit,
  MdEditOff,
  MdFeedback,
  MdFiberManualRecord,
  MdFilterCenterFocus,
  MdHardware,
  MdHive,
  MdInsights,
  MdKeyboardAlt,
  MdKeyboardArrowDown,
  MdKeyboardArrowLeft,
  MdKeyboardArrowRight,
  MdKeyboardArrowUp,
  MdKeyboardBackspace,
  MdKeyboardCapslock,
  MdKeyboardCommandKey,
  MdKeyboardControlKey,
  MdKeyboardHide,
  MdKeyboardOptionKey,
  MdKeyboardReturn,
  MdKeyboardTab,
  MdLabel,
  MdLightMode,
  MdLink,
  MdLogout,
  MdMoreVert,
  MdNewReleases,
  MdOutlineControlCamera,
  MdOutlineDeviceHub,
  MdOutlineExplore,
  MdOutlineFilterList,
  MdOutlineLinearScale,
  MdOutlineMap,
  MdOutlineMotionPhotosOff,
  MdOutlineMotionPhotosOn,
  MdOutlineOpenInNew,
  MdOutlineTableRows,
  MdOutlineTimelapse,
  MdOutlineVisibility,
  MdOutlineVisibilityOff,
  MdOutlineWebAsset,
  MdPause,
  MdPerson,
  MdPictureInPicture,
  MdPlayArrow,
  MdQuestionMark,
  MdSaveAlt,
  MdSensors,
  MdShield,
  MdSquareFoot,
  MdTextFields,
  MdTypeSpecimen,
  MdWorkspacesFilled,
} from "react-icons/md";
import {
  PiBinary,
  PiCaretDown,
  PiCaretLeft,
  PiCaretRight,
  PiCaretUpBold,
  PiDownloadSimple,
  PiFileCsv,
  PiMagnifyingGlassBold,
  PiSelectionPlusBold,
  PiThermometerSimpleFill,
  PiUploadSimple,
  PiWaveSawtoothBold,
  PiWaveSineBold,
  PiWaveSquareBold,
  PiWaveTriangleBold,
} from "react-icons/pi";
import {
  RiProgress1Line,
  RiProgress4Line,
  RiProgress8Line,
  RiSettings3Fill,
  RiWeightFill,
} from "react-icons/ri";
import { RxComponentBoolean, RxReset } from "react-icons/rx";
import {
  SiGooglenearby,
  SiNpm,
  SiPnpm,
  SiPython,
  SiSpringCreators,
  SiTypescript,
  SiYarn,
} from "react-icons/si";
import {
  TbArrowAutofitWidth,
  TbArrowDown,
  TbArrowLeft,
  TbArrowRight,
  TbArrowsSplit,
  TbArrowUp,
  TbChartArcs,
  TbCircleDashed,
  TbCircleLetterAFilled,
  TbCircleLetterVFilled,
  TbCircuitResistor,
  TbDecimal,
  TbLivePhoto,
  TbMathFunction,
  TbPlugConnected,
  TbPlugConnectedX,
  TbRadarFilled,
  TbVariable,
} from "react-icons/tb";
import {
  VscSplitHorizontal,
  VscSplitVertical,
  VscSymbolConstant,
  VscSymbolString,
} from "react-icons/vsc";

import { CSS } from "@/css";
import { Fitting } from "@/icon/Fitting";
import { type IconProps, type ReactElement, wrapSVGIcon } from "@/icon/Icon";
import { LabJack } from "@/icon/LabJack";
import { Modbus } from "@/icon/Modbus";
import { NI } from "@/icon/NI";
import { OPC } from "@/icon/OPC";
import { Process } from "@/icon/Process";
import { Pump } from "@/icon/Pump";
import { Valve } from "@/icon/Valve";

export const Pause = wrapSVGIcon(MdPause, "pause");
export const Play = wrapSVGIcon(MdPlayArrow, "play");
export const Circle = wrapSVGIcon(MdFiberManualRecord, "circle");
export const Edit = wrapSVGIcon(MdEdit, "edit");
export const EditOff = wrapSVGIcon(MdEditOff, "edit-off");
export const Add = wrapSVGIcon(HiOutlinePlus, "add");
export const Subtract = wrapSVGIcon(AiOutlineMinus, "subtract");
export const Copy = wrapSVGIcon(IoCopy, "copy");
export const Close = wrapSVGIcon(AiOutlineClose, "close");
export const Info = wrapSVGIcon(BsFillInfoSquareFill, "info");
export const Warning = wrapSVGIcon(AiFillWarning, "warning");
export const Check = wrapSVGIcon(AiOutlineCheck, "check");
export const Refresh = wrapSVGIcon(IoMdRefresh, "refresh");
export const Delete = wrapSVGIcon(AiFillDelete, "delete");
export const Time = wrapSVGIcon(IoTime, "time");
export const Acquire = wrapSVGIcon(FaStream, "acquire");
export const Analyze = wrapSVGIcon(FaBezierCurve, "analyze");
export const Concepts = wrapSVGIcon(BsLightbulbFill, "concepts");
export const Visualize = wrapSVGIcon(MdAreaChart, "visualize");
export const LinePlot = wrapSVGIcon(MdAreaChart, "line-plot");
export const Expand = wrapSVGIcon(AiOutlineExpand, "expand");
export const Cluster = wrapSVGIcon(HiSquare3Stack3D, "cluster");
export const Loading = wrapSVGIcon(AiOutlineLoading, "loading", {
  className: CSS.M("spin"),
});
export const Schematic = wrapSVGIcon(IoShapes, "schematic");
export const Caret = {
  Right: wrapSVGIcon(PiCaretRight, "caret-right"),
  Bottom: wrapSVGIcon(PiCaretDown, "caret-bottom"),
  Left: wrapSVGIcon(PiCaretLeft, "caret-left"),
  Up: wrapSVGIcon(PiCaretUpBold, "caret-up"),
  Top: wrapSVGIcon(PiCaretUpBold, "caret-top"),
  Down: wrapSVGIcon(PiCaretDown, "caret-down"),
};
export const Settings = wrapSVGIcon(RiSettings3Fill, "settings");
export const Reference = wrapSVGIcon(IoBookSharp, "reference");
export const Bolt = wrapSVGIcon(HiLightningBolt, "bolt");
export const Import = wrapSVGIcon(PiUploadSimple, "import");
export const Export = wrapSVGIcon(PiDownloadSimple, "export");
export const Download = Export;
export const Range = wrapSVGIcon(MdOutlineTimelapse, "range");
export const Node = wrapSVGIcon(MdOutlineDeviceHub, "node");
export const Channel = wrapSVGIcon(MdSensors, "channel");
export const Resources = wrapSVGIcon(AiFillFolder, "resources");
export const Group = wrapSVGIcon(AiFillFolder, "group");
export const Workspace = wrapSVGIcon(MdWorkspacesFilled, "workspace");
export const Box = wrapSVGIcon(AiOutlineBorder, "box");
export const Python = wrapSVGIcon(SiPython, "python");
export const TypeScript = wrapSVGIcon(SiTypescript, "typescript");
export const NPM = wrapSVGIcon(SiNpm, "npm");
export const PNPM = wrapSVGIcon(SiPnpm, "pnpm");
export const Yarn = wrapSVGIcon(SiYarn, "yarn");
export const QuestionMark = wrapSVGIcon(MdQuestionMark, "question-mark");
export const Menu = wrapSVGIcon(GiHamburgerMenu, "menu");
export const Logo = {
  Apple: wrapSVGIcon(FaApple, "logo-apple"),
  Docker: wrapSVGIcon(FaDocker, "logo-docker"),
  Github: wrapSVGIcon(AiFillGithub, "logo-github"),
  LabJack,
  LinkedIn: wrapSVGIcon(AiFillLinkedin, "logo-linkedin"),
  Linux: wrapSVGIcon(FaLinux, "logo-linux"),
  Modbus,
  NI,
  OPC,
  Windows: wrapSVGIcon(FaWindows, "logo-windows"),
};
export const Arrow = {
  Right: wrapSVGIcon(TbArrowRight, "arrow-right"),
  Down: wrapSVGIcon(TbArrowDown, "arrow-down"),
  Bottom: wrapSVGIcon(TbArrowDown, "arrow-bottom"),
  Up: wrapSVGIcon(TbArrowUp, "arrow-up"),
  Left: wrapSVGIcon(TbArrowLeft, "arrow-left"),
  Top: wrapSVGIcon(TbArrowUp, "arrow-top"),
};
export const Keyboard = {
  Command: wrapSVGIcon(MdKeyboardCommandKey, "keyboard-command"),
  Windows: wrapSVGIcon(FaWindows, "keyboard-windows"),
  Tab: wrapSVGIcon(MdKeyboardTab, "keyboard-tab"),
  Return: wrapSVGIcon(MdKeyboardReturn, "keyboard-return"),
  Backspace: wrapSVGIcon(MdKeyboardBackspace, "keyboard-backspace"),
  Capslock: wrapSVGIcon(MdKeyboardCapslock, "keyboard-capslock"),
  Hide: wrapSVGIcon(MdKeyboardHide, "keyboard-hide"),
  Control: wrapSVGIcon(MdKeyboardControlKey, "keyboard-control"),
  Arrow: {
    Up: wrapSVGIcon(MdKeyboardArrowUp, "keyboard-arrow-up"),
    Down: wrapSVGIcon(MdKeyboardArrowDown, "keyboard-arrow-down"),
    Left: wrapSVGIcon(MdKeyboardArrowLeft, "keyboard-arrow-left"),
    Right: wrapSVGIcon(MdKeyboardArrowRight, "keyboard-arrow-right"),
  },
  Alt: wrapSVGIcon(MdKeyboardAlt, "keyboard-alt"),
  Option: wrapSVGIcon(MdKeyboardOptionKey, "keyboard-option"),
  Shift: wrapSVGIcon(BsShiftFill, "keyboard-shift"),
};
export const Tooltip = wrapSVGIcon(MdInsights, "tooltip");
export const Annotate = wrapSVGIcon(MdLabel, "annotate");
export const Zoom = wrapSVGIcon(PiMagnifyingGlassBold, "zoom");
export const Selection = wrapSVGIcon(PiSelectionPlusBold, "selection");
export const Pan = wrapSVGIcon(GrPan, "pan");
export const Rule = wrapSVGIcon(MdSquareFoot, "rule");
export const User = wrapSVGIcon(MdPerson, "user");
export const Rename = wrapSVGIcon(BiRename, "rename");
export const Snapshot = wrapSVGIcon(MdPictureInPicture, "snapshot");
export const Sync = wrapSVGIcon(AiOutlineSync, "sync");
export const Search = wrapSVGIcon(PiMagnifyingGlassBold, "search");
export const Auto = wrapSVGIcon(MdAutoAwesome, "auto");
export const Table = wrapSVGIcon(FiTable, "table");
export const Wave = {
  Sawtooth: wrapSVGIcon(PiWaveSawtoothBold, "wave-sawtooth"),
  Sine: wrapSVGIcon(PiWaveSineBold, "wave-sine"),
  Triangle: wrapSVGIcon(PiWaveTriangleBold, "wave-triangle"),
  Square: wrapSVGIcon(PiWaveSquareBold, "wave-square"),
};
export const Align = {
  Right: wrapSVGIcon(MdAlignHorizontalRight, "align-right"),
  Left: wrapSVGIcon(MdAlignHorizontalLeft, "align-left"),
  XCenter: wrapSVGIcon(MdAlignHorizontalCenter, "align-x-center"),
  YCenter: wrapSVGIcon(MdAlignVerticalCenter, "align-y-center"),
  Top: wrapSVGIcon(MdAlignVerticalTop, "align-top"),
  Bottom: wrapSVGIcon(MdAlignVerticalBottom, "align-bottom"),
};
export const TextAlign = {
  Center: wrapSVGIcon(FaAlignCenter, "text-align-center"),
  Left: wrapSVGIcon(FaAlignLeft, "text-align-left"),
  Right: wrapSVGIcon(FaAlignRight, "text-align-right"),
};
export const Connect = wrapSVGIcon(TbPlugConnected, "connect");
export const Disconnect = wrapSVGIcon(TbPlugConnectedX, "disconnect");
export const Hardware = wrapSVGIcon(MdHardware, "hardware");
export const Save = wrapSVGIcon(MdSaveAlt, "save");
export const Task = wrapSVGIcon(TbRadarFilled, "task");
export const Device = wrapSVGIcon(SiGooglenearby, "device");
export const Link = wrapSVGIcon(MdLink, "link");
export const Attachment = wrapSVGIcon(GrAttachment, "attachment");
export const Drag = wrapSVGIcon(GrDrag, "drag");
export const Dynamic = wrapSVGIcon(TbLivePhoto, "dynamic");
export const Enable = wrapSVGIcon(MdOutlineMotionPhotosOn, "enable");
export const Disable = wrapSVGIcon(MdOutlineMotionPhotosOff, "disable");
export const Variable = wrapSVGIcon(TbVariable, "variable");
export const Type = wrapSVGIcon(MdTypeSpecimen, "type");
export const Array = wrapSVGIcon(MdDataArray, "array");
export const Label = wrapSVGIcon(MdLabel, "label");
export const Details = wrapSVGIcon(MdOutlineTableRows, "details");
export const LinkExternal = wrapSVGIcon(BiLinkExternal, "link-external");
export const Access = wrapSVGIcon(MdShield, "access");
export const JSON = wrapSVGIcon(MdDataObject, "json");
export const Guide = wrapSVGIcon(MdBook, "guide");
export const Focus = wrapSVGIcon(MdFilterCenterFocus, "focus");
export const OpenInNewWindow = wrapSVGIcon(MdOutlineOpenInNew, "open-in-new-window");
export const MoveToMainWindow = wrapSVGIcon(MdOutlineWebAsset, "move-to-main-window");
export const SplitX = wrapSVGIcon(VscSplitHorizontal, "split-x");
export const SplitY = wrapSVGIcon(VscSplitVertical, "split-y");
export const AutoFitWidth = wrapSVGIcon(TbArrowAutofitWidth, "auto-fit-width");
export const Commit = wrapSVGIcon(MdCommit, "commit");
export const Snooze = wrapSVGIcon(IoNotificationsOff, "snooze");
export const Log = wrapSVGIcon(FaStream, "log");
export const Tare = wrapSVGIcon(FaCreativeCommonsZero, "tare");
export const Rotate = wrapSVGIcon(GrRotateRight, "rotate");
export const Text = wrapSVGIcon(MdTextFields, "text");
export const Value = wrapSVGIcon(GoNumber, "value");
export const Calendar = wrapSVGIcon(MdCalendarToday, "calendar");
export const Release = wrapSVGIcon(MdNewReleases, "release");
export const OpenExternal = wrapSVGIcon(MdArrowOutward, "open-external");
export const Feedback = wrapSVGIcon(MdFeedback, "feedback");
export const Calculation = wrapSVGIcon(BiMath, "calculation");
export const Binary = wrapSVGIcon(PiBinary, "binary");
export const Index = wrapSVGIcon(IoTime, "index");
export const Decimal = wrapSVGIcon(TbDecimal, "decimal");
export const String = wrapSVGIcon(VscSymbolString, "string");
export const Control = wrapSVGIcon(MdOutlineControlCamera, "control");
export const Rack = wrapSVGIcon(MdHive, "rack");
export const Units = {
  Acceleration: wrapSVGIcon(FaCarSide, "units-acceleration"),
  Current: wrapSVGIcon(TbCircleLetterAFilled, "units-current"),
  Force: wrapSVGIcon(RiWeightFill, "units-force"),
  Pressure: wrapSVGIcon(FaGaugeHigh, "units-pressure"),
  Resistance: wrapSVGIcon(TbCircuitResistor, "units-resistance"),
  Strain: wrapSVGIcon(SiSpringCreators, "units-strain"),
  Temperature: wrapSVGIcon(PiThermometerSimpleFill, "units-temperature"),
  Torque: wrapSVGIcon(FaGear, "units-torque"),
  Velocity: wrapSVGIcon(FaWind, "units-velocity"),
  Voltage: wrapSVGIcon(TbCircleLetterVFilled, "units-voltage"),
};
export const Bridge = wrapSVGIcon(FaBridge, "bridge");
export const Sound = wrapSVGIcon(FaMicrophone, "sound");
export const Function = wrapSVGIcon(TbMathFunction, "function");
export const Visible = wrapSVGIcon(MdOutlineVisibility, "visible");
export const Hidden = wrapSVGIcon(MdOutlineVisibilityOff, "invisible");
export const Virtual = wrapSVGIcon(TbCircleLetterVFilled, "virtual");
export const Explore = wrapSVGIcon(MdOutlineExplore, "explore");
export const Filter = wrapSVGIcon(MdOutlineFilterList, "filter");
export const StarFilled = wrapSVGIcon(FaStar, "star-filled");
export const StarOutlined = wrapSVGIcon(FaRegStar, "star-outlined");
export const Heart = wrapSVGIcon(IoMdHeart, "heart");
export const StrokeWidth = wrapSVGIcon(BsBorderWidth, "stroke-width");
export const Downsample = wrapSVGIcon(MdBlurLinear, "downsample");
export const Terminal = wrapSVGIcon(IoTerminal, "terminal");
export const Map = wrapSVGIcon(MdOutlineMap, "map");
export const Linear = wrapSVGIcon(MdOutlineLinearScale, "linear");
export const None = wrapSVGIcon(TbCircleDashed, "none");
export const Arc = wrapSVGIcon(TbChartArcs, "arc");
export const Select = wrapSVGIcon(TbArrowsSplit, "select");
export const Notification = wrapSVGIcon(IoNotifications, "notification");
export const Status = Notification;
export const InProgress = wrapSVGIcon(RiProgress4Line, "in-progress");
export const Completed = wrapSVGIcon(RiProgress8Line, "completed");
export const ToDo = wrapSVGIcon(RiProgress1Line, "to-do");
export const Constant = wrapSVGIcon(VscSymbolConstant, "constant");
export const Boolean = wrapSVGIcon(RxComponentBoolean, "boolean");
export const True = wrapSVGIcon(FaCheck, "true");
export const False = wrapSVGIcon(FaXmark, "false");
export const KebabMenu = wrapSVGIcon(MdMoreVert, "kebab-menu");
export const Annotation = wrapSVGIcon(MdComment, "annotation");
export const Click = wrapSVGIcon(HiCursorClick, "click");
export const DarkMode = wrapSVGIcon(MdDarkMode, "dark-mode");
export const LightMode = wrapSVGIcon(MdLightMode, "light-mode");
export const Safety = wrapSVGIcon(FaHelmetSafety, "safety");
export const CSV = wrapSVGIcon(PiFileCsv, "csv");
export const Reset = wrapSVGIcon(RxReset, "reset");
export const FillColor = wrapSVGIcon(IoMdColorFill, "fill-color");
export const StrokeColor = wrapSVGIcon(MdBorderColor, "stroke-color");
export const Logout = wrapSVGIcon(MdLogout, "logout");

export interface CreateProps extends Omit<IconProps, "topRight"> {}

interface Resolve {
  (icon?: ReactElement | string, overrides?: IconProps): ReactElement | undefined;

  (icon: ReactElement | string, overrides?: IconProps): ReactElement;
}

const icons = {
  Pause,
  Play,
  Circle,
  Edit,
  EditOff,
  Add,
  Subtract,
  Copy,
  Close,
  Info,
  Warning,
  Check,
  Refresh,
  Delete,
  Time,
  Acquire,
  Analyze,
  Concepts,
  Visualize,
  LinePlot,
  Expand,
  Cluster,
  Loading,
  Schematic,
  Caret,
  Settings,
  Reference,
  Bolt,
  Download,
  Import,
  Export,
  Range,
  Node,
  Channel,
  Resources,
  Group,
  Workspace,
  Box,
  Python,
  TypeScript,
  NPM,
  PNPM,
  Yarn,
  QuestionMark,
  Menu,
  Logo,
  Arrow,
  Keyboard,
  Tooltip,
  Annotate,
  Zoom,
  Selection,
  Pan,
  Rule,
  User,
  Rename,
  Snapshot,
  Sync,
  Search,
  Auto,
  Table,
  Wave,
  Align,
  TextAlign,
  Connect,
  Disconnect,
  Hardware,
  Save,
  Task,
  Device,
  Link,
  Attachment,
  Drag,
  Dynamic,
  Enable,
  Disable,
  Variable,
  Type,
  Array,
  Label,
  Details,
  LinkExternal,
  Access,
  JSON,
  Guide,
  Focus,
  OpenInNewWindow,
  MoveToMainWindow,
  SplitX,
  SplitY,
  AutoFitWidth,
  Commit,
  Snooze,
  Log,
  Tare,
  Rotate,
  Text,
  Value,
  Calendar,
  Release,
  OpenExternal,
  Feedback,
  Calculation,
  Binary,
  Index,
  Decimal,
  String,
  Control,
  Rack,
  Units,
  Bridge,
  Sound,
  Function,
  Visible,
  Hidden,
  Virtual,
  Explore,
  Filter,
  StarFilled,
  StarOutlined,
  Heart,
  Map,
  Linear,
  None,
  Constant,
  Terminal,
  Arc,
  Select,
  Notification,
  InProgress,
  Completed,
  ToDo,
  Boolean,
  Annotation,
  CSV,
  Valve,
  Safety,
  Process,
  Fitting,
  Pump,
  Reset,
  Status,
  FillColor,
  StrokeColor,
  Logout,
};

export const resolve = ((
  icon?: ReactElement | string | undefined,
  overrides?: IconProps,
): ReactElement | undefined => {
  if (icon == null) return;
  if (typeof icon === "string")
    try {
      const C = deep.get<FC<IconProps>>(icons, icon);
      return <C {...overrides} />;
    } catch {
      throw new Error(`Unable to find icon with path ${icon} in registry`);
    }

  return cloneElement(icon, overrides);
}) as Resolve;
