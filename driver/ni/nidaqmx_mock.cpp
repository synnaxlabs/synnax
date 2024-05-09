// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once
#include "daqmx.h"
#include "nidaqmx_api.h"

int32 ni::NiDAQmxInterface::CreateAIVoltageChan(TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[], int32 terminalConfig, float64 minVal, float64 maxVal, int32 units, const char customScaleName[]){
    return 0;
}

int32 ni::NiDAQmxInterface::CreateDIChan(TaskHandle task, const char lines[], const char nameToAssignToLines[], int32 lineGrouping){
    return 0;
}

int32 ni::NiDAQmxInterface::CreateDOChan(TaskHandle task, const char lines[], const char nameToAssignToLines[], int32 lineGrouping){
    return 0;
}

int32 ni::NiDAQmxInterface::CfgSampClkTiming(TaskHandle task, const char source[], float64 rate, int32 activeEdge, int32 sampleMode, uInt64 sampsPerChan){
    return 0;
}

int32 ni::NiDAQmxInterface::StartTask(TaskHandle task){
    return 0;
}

int32 ni::NiDAQmxInterface::StopTask(TaskHandle task){
    return 0;
}

int32 ni::NiDAQmxInterface::ClearTask(TaskHandle task){
    return 0;
}  

int32 ni::NiDAQmxInterface::ReadAnalogF64(TaskHandle task, int32 numSampsPerChan, float64 timeout, int32 fillMode, float64 readArray[], uInt32 arraySizeInSamps, int32* sampsPerChanRead, bool32* reserved){
    return 0;
}

int32 ni::NiDAQmxInterface::ReadDigitalLines(TaskHandle task, int32 numSampsPerChan, float64 timeout, int32 fillMode, uInt8 readArray[], uInt32 arraySizeInBytes, int32* sampsPerChanRead, int32* numBytesPerSamp, bool32* reserved){
    return 0;
}

int32 ni::NiDAQmxInterface::WriteDigitalLines(TaskHandle task, int32 numSampsPerChan, bool32 autoStart, float64 timeout, int32 dataLayout, const uInt8 writeArray[], int32* sampsPerChanWritten, bool32* reserved){
    return 0;
}

int32 ni::NiDAQmxInterface::GetExtendedErrorInfo(char errorString[], uInt32 bufferSize){
    return 0;
}    