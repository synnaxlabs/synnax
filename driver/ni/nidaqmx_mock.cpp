// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "daqmx.h"
#include "nidaqmx_api.h"

int32 ni::NiDAQmxInterface::CreateDIChan(
    TaskHandle task, const char lines[],
    const char nameToAssignToLines[],
    int32 lineGrouping
) { return 0; }

int32 ni::NiDAQmxInterface::CreateDOChan(
    TaskHandle task, const char lines[],
    const char nameToAssignToLines[],
    int32 lineGrouping
) { return 0; }

int32 ni::NiDAQmxInterface::CfgSampClkTiming(
    TaskHandle task,
    const char source[],
    float64 rate,
    int32 activeEdge,
    int32 sampleMode,
    uInt64 sampsPerChan
) { return 0; }

int32 ni::NiDAQmxInterface::StartTask(TaskHandle task) { return 0; }

int32 ni::NiDAQmxInterface::StopTask(TaskHandle task) { return 0; }

int32 ni::NiDAQmxInterface::ClearTask(TaskHandle task) { return 0; }

int32 ni::NiDAQmxInterface::ReadAnalogF64(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    float64 readArray[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved) {
    return 0;
}

int32 ni::NiDAQmxInterface::ReadDigitalLines(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    uInt8 readArray[],
    uInt32 arraySizeInBytes,
    int32 *sampsPerChanRead,
    int32 *numBytesPerSamp,
    bool32 *reserved) {
    return 0;
}

int32 ni::NiDAQmxInterface::WriteDigitalLines(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const uInt8 writeArray[],
    int32 *sampsPerChanWritten,
    bool32 *reserved
) { return 0; }

int32 ni::NiDAQmxInterface::GetExtendedErrorInfo(
    char errorString[],
    uInt32 bufferSize
) { return 0; }

int32 ni::NiDAQmxInterface::CreateLinScale(
    const char name[],
    float64 slope,
    float64 yIntercept,
    int32 preScaledUnits,
    const char scaledUnits[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateMapScale(
    const char name[],
    float64 prescaledMin,
    float64 prescaledMax,
    float64 scaledMin,
    float64 scaledMax,
    int32 preScaledUnits,
    const char scaledUnits[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreatePolynomialScale(
    const char name[],
    const float64 forwardCoeffs[],
    uInt32 numForwardCoeffsIn,
    const float64 reverseCoeffs[],
    uInt32 numReverseCoeffsIn,
    int32 preScaledUnits,
    const char scaledUnits[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateTableScale(
    const char name[],
    const float64 prescaledVals[],
    uInt32 numPrescaledValsIn,
    const float64 scaledVals[],
    uInt32 numScaledValsIn,
    int32 preScaledUnits,
    const char scaledUnits[]
) { return 0; }

int32 ni::NiDAQmxInterface::CalculateReversePolyCoeff(
    const float64 forwardCoeffs[], uInt32 numForwardCoeffsIn,
    float64 minValX, float64 maxValX, int32 numPointsToCompute,
    int32 reversePolyOrder, float64 reverseCoeffs[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateTask(const char sessionName[], TaskHandle *task) {
    return 0;
}

int32 ni::NiDAQmxInterface::CreateAIVoltageChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 terminalConfig,
    float64 minVal,
    float64 maxVal,
    int32 units,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIVoltageRMSChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 terminalConfig, float64 minVal,
    float64 maxVal,
    int32 units,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIVoltageChanWithExcit(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 terminalConfig,
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 bridgeConfig,
    int32 voltageExcitSource,
    float64 voltageExcitVal,
    bool32 useExcitForScaling,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIAccel4WireDCVoltageChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 terminalConfig,
    float64 minVal,
    float64 maxVal,
    int32 units,
    float64 sensitivity,
    int32 sensitivityUnits,
    int32 voltageExcitSource,
    float64 voltageExcitVal,
    bool32 useExcitForScaling,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIAccelChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 terminalConfig,
    float64 minVal,
    float64 maxVal,
    int32 units,
    float64 sensitivity,
    int32 sensitivityUnits,
    int32 currentExcitSource,
    float64 currentExcitVal,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIAccelChargeChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 terminalConfig,
    float64 minVal,
    float64 maxVal,
    int32 units,
    float64 sensitivity,
    int32 sensitivityUnits,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIBridgeChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal, int32 units,
    int32 bridgeConfig,
    int32 voltageExcitSource,
    float64 voltageExcitVal,
    float64 nominalBridgeResistance,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIChargeChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 terminalConfig,
    float64 minVal,
    float64 maxVal,
    int32 units,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAICurrentChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 terminalConfig,
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 shuntResistorLoc,
    float64 extShuntResistorVal,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAICurrentRMSChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 terminalConfig,
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 shuntResistorLoc,
    float64 extShuntResistorVal,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIForceBridgePolynomialChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal, float64 maxVal,
    int32 units,
    int32 bridgeConfig,
    int32 voltageExcitSource,
    float64 voltageExcitVal,
    float64 nominalBridgeResistance,
    const float64 forwardCoeffs[],
    uInt32 numForwardCoeffs,
    const float64 reverseCoeffs[],
    uInt32 numReverseCoeffs,
    int32 electricalUnits,
    int32 physicalUnits,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIForceBridgeTableChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units, int32 bridgeConfig,
    int32 voltageExcitSource,
    float64 voltageExcitVal,
    float64 nominalBridgeResistance,
    const float64 electricalVals[],
    uInt32 numElectricalVals,
    int32 electricalUnits,
    const float64 physicalVals[],
    uInt32 numPhysicalVals,
    int32 physicalUnits,
    const char customScaleName[]) {
    return 0;
}

int32 ni::NiDAQmxInterface::CreateAIForceBridgeTwoPointLinChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 bridgeConfig,
    int32 voltageExcitSource,
    float64 voltageExcitVal,
    float64 nominalBridgeResistance,
    float64 firstElectricalVal,
    float64 secondElectricalVal,
    int32 electricalUnits,
    float64 firstPhysicalVal,
    float64 secondPhysicalVal,
    int32 physicalUnits,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIForceIEPEChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 terminalConfig,
    float64 minVal,
    float64 maxVal,
    int32 units,
    float64 sensitivity,
    int32 sensitivityUnits,
    int32 currentExcitSource,
    float64 currentExcitVal,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIFreqVoltageChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    float64 thresholdLevel,
    float64 hysteresis,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIMicrophoneChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 terminalConfig,
    int32 units,
    float64 micSensitivity,
    float64 maxSndPressLevel,
    int32 currentExcitSource,
    float64 currentExcitVal,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIPosEddyCurrProxProbeChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    float64 sensitivity,
    int32 sensitivityUnits,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIPosLVDTChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal, int32 units,
    float64 sensitivity,
    int32 sensitivityUnits,
    int32 voltageExcitSource,
    float64 voltageExcitVal,
    float64 voltageExcitFreq,
    int32 acExcitWireMode,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIPosRVDTChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    float64 sensitivity,
    int32 sensitivityUnits,
    int32 voltageExcitSource,
    float64 voltageExcitVal,
    float64 voltageExcitFreq,
    int32 acExcitWireMode,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIPowerChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 voltageSetpoint,
    float64 currentSetpoint,
    bool32 outputEnable
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIPressureBridgePolynomialChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 bridgeConfig,
    int32 voltageExcitSource,
    float64 voltageExcitVal,
    float64 nominalBridgeResistance,
    const float64 forwardCoeffs[],
    uInt32 numForwardCoeffs,
    const float64 reverseCoeffs[],
    uInt32 numReverseCoeffs,
    int32 electricalUnits,
    int32 physicalUnits,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIPressureBridgeTableChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 bridgeConfig,
    int32 voltageExcitSource,
    float64 voltageExcitVal,
    float64 nominalBridgeResistance,
    const float64 electricalVals[],
    uInt32 numElectricalVals,
    int32 electricalUnits,
    const float64 physicalVals[],
    uInt32 numPhysicalVals,
    int32 physicalUnits,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIPressureBridgeTwoPointLinChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 bridgeConfig,
    int32 voltageExcitSource,
    float64 voltageExcitVal,
    float64 nominalBridgeResistance,
    float64 firstElectricalVal,
    float64 secondElectricalVal,
    int32 electricalUnits,
    float64 firstPhysicalVal,
    float64 secondPhysicalVal,
    int32 physicalUnits,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIRTDChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 rtdType,
    int32 resistanceConfig,
    int32 currentExcitSource,
    float64 currentExcitVal,
    float64 r0
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIResistanceChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 resistanceConfig,
    int32 currentExcitSource,
    float64 currentExcitVal,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIRosetteStrainGageChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 rosetteType,
    float64 gageOrientation,
    const int32 rosetteMeasTypes[],
    uInt32 numRosetteMeasTypes,
    int32 strainConfig,
    int32 voltageExcitSource,
    float64 voltageExcitVal,
    float64 gageFactor,
    float64 nominalGageResistance,
    float64 poissonRatio,
    float64 leadWireResistance
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIStrainGageChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 strainConfig,
    int32 voltageExcitSource,
    float64 voltageExcitVal,
    float64 gageFactor,
    float64 initialBridgeVoltage,
    float64 nominalGageResistance,
    float64 poissonRatio,
    float64 leadWireResistance,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAITempBuiltInSensorChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 units
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIThrmcplChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 thermocoupleType,
    int32 cjcSource,
    float64 cjcVal,
    const char cjcChannel[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIThrmstrChanIex(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 resistanceConfig,
    int32 currentExcitSource,
    float64 currentExcitVal,
    float64 a,
    float64 b,
    float64 c
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIThrmstrChanVex(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 resistanceConfig,
    int32 voltageExcitSource,
    float64 voltageExcitVal,
    float64 a,
    float64 b,
    float64 c,
    float64 r1
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAITorqueBridgePolynomialChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 bridgeConfig,
    int32 voltageExcitSource,
    float64 voltageExcitVal,
    float64 nominalBridgeResistance,
    const float64 forwardCoeffs[],
    uInt32 numForwardCoeffs,
    const float64 reverseCoeffs[],
    uInt32 numReverseCoeffs,
    int32 electricalUnits,
    int32 physicalUnits,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAITorqueBridgeTableChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 bridgeConfig,
    int32 voltageExcitSource,
    float64 voltageExcitVal,
    float64 nominalBridgeResistance,
    const float64 electricalVals[],
    uInt32 numElectricalVals,
    int32 electricalUnits,
    const float64 physicalVals[],
    uInt32 numPhysicalVals,
    int32 physicalUnits,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAITorqueBridgeTwoPointLinChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 bridgeConfig,
    int32 voltageExcitSource,
    float64 voltageExcitVal,
    float64 nominalBridgeResistance,
    float64 firstElectricalVal,
    float64 secondElectricalVal,
    int32 electricalUnits,
    float64 firstPhysicalVal,
    float64 secondPhysicalVal,
    int32 physicalUnits,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAIVelocityIEPEChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 terminalConfig,
    float64 minVal,
    float64 maxVal,
    int32 units,
    float64 sensitivity,
    int32 sensitivityUnits,
    int32 currentExcitSource,
    float64 currentExcitVal,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAOCurrentChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAOFuncGenChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 type,
    float64 freq,
    float64 amplitude,
    float64 offset
) { return 0; }

int32 ni::NiDAQmxInterface::CreateAOVoltageChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    const char customScaleName[]
) { return 0; }

int32 ni::NiDAQmxInterface::WriteAnalogF64(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const float64 writeArray[],
    int32 *sampsPerChanWritten, bool32 *reserved
) { return 0; }

int32 ni::NiDAQmxInterface::WriteAnalogScalarF64(
    TaskHandle task,
    bool32 autoStart,
    float64 timeout,
    float64 value,
    bool32 *reserved
) { return 0; }
