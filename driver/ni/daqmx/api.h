// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "driver/errors/errors.h"
#include "driver/ni/daqmx/nidaqmx.h"

namespace driver::ni::daqmx {
const LibraryInfo LIBRARY_INFO = {
    "National Instruments NI-DAQmx shared",
    "https://www.ni.com/en/support/downloads/drivers/download.ni-daq-mx.html"
};

class API {
public:
    virtual ~API() = default;

    virtual int32 AddCDAQSyncConnection(const char portList[]) = 0;
    virtual int32 AddGlobalChansToTask(TaskHandle task, const char channelNames[]) = 0;
    virtual int32 AddNetworkDevice(
        const char ipAddress[],
        const char deviceName[],
        bool32 attemptReservation,
        float64 timeout,
        char deviceNameOut[],
        uInt32 deviceNameOutBufferSize
    ) = 0;
    virtual int32 AreConfiguredCDAQSyncPortsDisconnected(
        const char chassisDevicesPorts[],
        float64 timeout,
        bool32 *disconnectedPortsExist
    ) = 0;
    virtual int32 AutoConfigureCDAQSyncConnections(
        const char chassisDevicesPorts[],
        float64 timeout
    ) = 0;
    virtual int32 CalculateReversePolyCoeff(
        const float64 forwardCoeffs[],
        uInt32 numForwardCoeffsIn,
        float64 minValX,
        float64 maxValX,
        int32 numPointsToCompute,
        int32 reversePolyOrder,
        float64 reverseCoeffs[]
    ) = 0;
    virtual int32 CfgAnlgEdgeRefTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerSlope,
        float64 triggerLevel,
        uInt32 pretriggerSamples
    ) = 0;
    virtual int32 CfgAnlgEdgeStartTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerSlope,
        float64 triggerLevel
    ) = 0;
    virtual int32 CfgAnlgMultiEdgeRefTrig(
        TaskHandle task,
        const char triggerSources[],
        const int32 triggerSlopeArray[],
        const float64 triggerLevelArray[],
        uInt32 pretriggerSamples,
        uInt32 arraySize
    ) = 0;
    virtual int32 CfgAnlgMultiEdgeStartTrig(
        TaskHandle task,
        const char triggerSources[],
        const int32 triggerSlopeArray[],
        const float64 triggerLevelArray[],
        uInt32 arraySize
    ) = 0;
    virtual int32 CfgAnlgWindowRefTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerWhen,
        float64 windowTop,
        float64 windowBottom,
        uInt32 pretriggerSamples
    ) = 0;
    virtual int32 CfgAnlgWindowStartTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerWhen,
        float64 windowTop,
        float64 windowBottom
    ) = 0;
    virtual int32 CfgBurstHandshakingTimingExportClock(
        TaskHandle task,
        int32 sampleMode,
        uInt64 sampsPerChan,
        float64 sampleClkRate,
        const char sampleClkOutpTerm[],
        int32 sampleClkPulsePolarity,
        int32 pauseWhen,
        int32 readyEventActiveLevel
    ) = 0;
    virtual int32 CfgBurstHandshakingTimingImportClock(
        TaskHandle task,
        int32 sampleMode,
        uInt64 sampsPerChan,
        float64 sampleClkRate,
        const char sampleClkSrc[],
        int32 sampleClkActiveEdge,
        int32 pauseWhen,
        int32 readyEventActiveLevel
    ) = 0;
    virtual int32 CfgChangeDetectionTiming(
        TaskHandle task,
        const char risingEdgeChan[],
        const char fallingEdgeChan[],
        int32 sampleMode,
        uInt64 sampsPerChan
    ) = 0;
    virtual int32 CfgDigEdgeRefTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerEdge,
        uInt32 pretriggerSamples
    ) = 0;
    virtual int32 CfgDigEdgeStartTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerEdge
    ) = 0;
    virtual int32 CfgDigPatternRefTrig(
        TaskHandle task,
        const char triggerSource[],
        const char triggerPattern[],
        int32 triggerWhen,
        uInt32 pretriggerSamples
    ) = 0;
    virtual int32 CfgDigPatternStartTrig(
        TaskHandle task,
        const char triggerSource[],
        const char triggerPattern[],
        int32 triggerWhen
    ) = 0;
    virtual int32
    CfgHandshakingTiming(TaskHandle task, int32 sampleMode, uInt64 sampsPerChan) = 0;
    virtual int32
    CfgImplicitTiming(TaskHandle task, int32 sampleMode, uInt64 sampsPerChan) = 0;
    virtual int32 CfgInputBuffer(TaskHandle task, uInt32 numSampsPerChan) = 0;
    virtual int32 CfgOutputBuffer(TaskHandle task, uInt32 numSampsPerChan) = 0;
    virtual int32 CfgPipelinedSampClkTiming(
        TaskHandle task,
        const char source[],
        float64 rate,
        int32 activeEdge,
        int32 sampleMode,
        uInt64 sampsPerChan
    ) = 0;
    virtual int32 CfgSampClkTiming(
        TaskHandle task,
        const char source[],
        float64 rate,
        int32 activeEdge,
        int32 sampleMode,
        uInt64 sampsPerChan
    ) = 0;
    virtual int32
    CfgTimeStartTrig(TaskHandle task, CVIAbsoluteTime when, int32 timescale) = 0;
    virtual int32 CfgWatchdogAOExpirStates(
        TaskHandle task,
        const char channelNames[],
        const float64 expirStateArray[],
        const int32 outputTypeArray[],
        uInt32 arraySize
    ) = 0;
    virtual int32 CfgWatchdogCOExpirStates(
        TaskHandle task,
        const char channelNames[],
        const int32 expirStateArray[],
        uInt32 arraySize
    ) = 0;
    virtual int32 CfgWatchdogDOExpirStates(
        TaskHandle task,
        const char channelNames[],
        const int32 expirStateArray[],
        uInt32 arraySize
    ) = 0;
    virtual int32 ClearTEDS(const char physicalChannel[]) = 0;
    virtual int32 ClearTask(TaskHandle task) = 0;
    virtual int32 ConfigureLogging(
        TaskHandle task,
        const char filePath[],
        int32 loggingMode,
        const char groupName[],
        int32 operation
    ) = 0;
    virtual int32
    ConfigureTEDS(const char physicalChannel[], const char filePath[]) = 0;
    virtual int32 ConnectTerms(
        const char sourceTerminal[],
        const char destinationTerminal[],
        int32 signalModifiers
    ) = 0;
    virtual int32 ControlWatchdogTask(TaskHandle task, int32 action) = 0;
    virtual int32 CreateAIAccel4WireDCVoltageChan(
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
    ) = 0;
    virtual int32 CreateAIAccelChan(
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
    ) = 0;
    virtual int32 CreateAIAccelChargeChan(
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
    ) = 0;
    virtual int32 CreateAIBridgeChan(
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
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateAIChargeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateAICurrentChan(
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
    ) = 0;
    virtual int32 CreateAICurrentRMSChan(
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
    ) = 0;
    virtual int32 CreateAIForceBridgePolynomialChan(
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
    ) = 0;
    virtual int32 CreateAIForceBridgeTableChan(
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
    ) = 0;
    virtual int32 CreateAIForceBridgeTwoPointLinChan(
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
    ) = 0;
    virtual int32 CreateAIForceIEPEChan(
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
    ) = 0;
    virtual int32 CreateAIFreqVoltageChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        float64 thresholdLevel,
        float64 hysteresis,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateAIMicrophoneChan(
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
    ) = 0;
    virtual int32 CreateAIPosEddyCurrProxProbeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        float64 sensitivity,
        int32 sensitivityUnits,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateAIPosLVDTChan(
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
    ) = 0;
    virtual int32 CreateAIPosRVDTChan(
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
    ) = 0;
    virtual int32 CreateAIPowerChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 voltageSetpoint,
        float64 currentSetpoint,
        bool32 outputEnable
    ) = 0;
    virtual int32 CreateAIPressureBridgePolynomialChan(
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
    ) = 0;
    virtual int32 CreateAIPressureBridgeTableChan(
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
    ) = 0;
    virtual int32 CreateAIPressureBridgeTwoPointLinChan(
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
    ) = 0;
    virtual int32 CreateAIRTDChan(
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
    ) = 0;
    virtual int32 CreateAIResistanceChan(
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
    ) = 0;
    virtual int32 CreateAIRosetteStrainGageChan(
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
    ) = 0;
    virtual int32 CreateAIStrainGageChan(
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
    ) = 0;
    virtual int32 CreateAITempBuiltInSensorChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 units
    ) = 0;
    virtual int32 CreateAIThrmcplChan(
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
    ) = 0;
    virtual int32 CreateAIThrmstrChanIex(
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
    ) = 0;
    virtual int32 CreateAIThrmstrChanVex(
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
    ) = 0;
    virtual int32 CreateAITorqueBridgePolynomialChan(
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
    ) = 0;
    virtual int32 CreateAITorqueBridgeTableChan(
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
    ) = 0;
    virtual int32 CreateAITorqueBridgeTwoPointLinChan(
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
    ) = 0;
    virtual int32 CreateAIVelocityIEPEChan(
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
    ) = 0;
    virtual int32 CreateAIVoltageChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateAIVoltageChanWithExcit(
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
    ) = 0;
    virtual int32 CreateAIVoltageRMSChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateAOCurrentChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateAOFuncGenChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 type,
        float64 freq,
        float64 amplitude,
        float64 offset
    ) = 0;
    virtual int32 CreateAOVoltageChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateCIAngEncoderChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        int32 decodingType,
        bool32 zidxEnable,
        float64 zidxVal,
        int32 zidxPhase,
        int32 units,
        uInt32 pulsesPerRev,
        float64 initialAngle,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateCIAngVelocityChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 decodingType,
        int32 units,
        uInt32 pulsesPerRev,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateCICountEdgesChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        int32 edge,
        uInt32 initialCount,
        int32 countDirection
    ) = 0;
    virtual int32 CreateCIDutyCycleChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minFreq,
        float64 maxFreq,
        int32 edge,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateCIFreqChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 edge,
        int32 measMethod,
        float64 measTime,
        uInt32 divisor,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateCIGPSTimestampChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        int32 units,
        int32 syncMethod,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateCILinEncoderChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        int32 decodingType,
        bool32 zidxEnable,
        float64 zidxVal,
        int32 zidxPhase,
        int32 units,
        float64 distPerPulse,
        float64 initialPos,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateCILinVelocityChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 decodingType,
        int32 units,
        float64 distPerPulse,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateCIPeriodChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 edge,
        int32 measMethod,
        float64 measTime,
        uInt32 divisor,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateCIPulseChanFreq(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units
    ) = 0;
    virtual int32 CreateCIPulseChanTicks(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        const char sourceTerminal[],
        float64 minVal,
        float64 maxVal
    ) = 0;
    virtual int32 CreateCIPulseChanTime(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units
    ) = 0;
    virtual int32 CreateCIPulseWidthChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 startingEdge,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateCISemiPeriodChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateCITwoEdgeSepChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 firstEdge,
        int32 secondEdge,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateCOPulseChanFreq(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        int32 units,
        int32 idleState,
        float64 initialDelay,
        float64 freq,
        float64 dutyCycle
    ) = 0;
    virtual int32 CreateCOPulseChanTicks(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        const char sourceTerminal[],
        int32 idleState,
        int32 initialDelay,
        int32 lowTicks,
        int32 highTicks
    ) = 0;
    virtual int32 CreateCOPulseChanTime(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        int32 units,
        int32 idleState,
        float64 initialDelay,
        float64 lowTime,
        float64 highTime
    ) = 0;
    virtual int32 CreateDIChan(
        TaskHandle task,
        const char lines[],
        const char nameToAssignToLines[],
        int32 lineGrouping
    ) = 0;
    virtual int32 CreateDOChan(
        TaskHandle task,
        const char lines[],
        const char nameToAssignToLines[],
        int32 lineGrouping
    ) = 0;
    virtual int32 CreateLinScale(
        const char name[],
        float64 slope,
        float64 yIntercept,
        int32 preScaledUnits,
        const char scaledUnits[]
    ) = 0;
    virtual int32 CreateMapScale(
        const char name[],
        float64 prescaledMin,
        float64 prescaledMax,
        float64 scaledMin,
        float64 scaledMax,
        int32 preScaledUnits,
        const char scaledUnits[]
    ) = 0;
    virtual int32 CreatePolynomialScale(
        const char name[],
        const float64 forwardCoeffs[],
        uInt32 numForwardCoeffsIn,
        const float64 reverseCoeffs[],
        uInt32 numReverseCoeffsIn,
        int32 preScaledUnits,
        const char scaledUnits[]
    ) = 0;
    virtual int32 CreateTEDSAIAccelChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 currentExcitSource,
        float64 currentExcitVal,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateTEDSAIBridgeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateTEDSAICurrentChan(
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
    ) = 0;
    virtual int32 CreateTEDSAIForceBridgeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateTEDSAIForceIEPEChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 currentExcitSource,
        float64 currentExcitVal,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateTEDSAIMicrophoneChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        int32 units,
        float64 maxSndPressLevel,
        int32 currentExcitSource,
        float64 currentExcitVal,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateTEDSAIPosLVDTChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        float64 voltageExcitFreq,
        int32 acExcitWireMode,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateTEDSAIPosRVDTChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        float64 voltageExcitFreq,
        int32 acExcitWireMode,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateTEDSAIPressureBridgeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateTEDSAIRTDChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 resistanceConfig,
        int32 currentExcitSource,
        float64 currentExcitVal
    ) = 0;
    virtual int32 CreateTEDSAIResistanceChan(
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
    ) = 0;
    virtual int32 CreateTEDSAIStrainGageChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        float64 initialBridgeVoltage,
        float64 leadWireResistance,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateTEDSAIThrmcplChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 cjcSource,
        float64 cjcVal,
        const char cjcChannel[]
    ) = 0;
    virtual int32 CreateTEDSAIThrmstrChanIex(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 resistanceConfig,
        int32 currentExcitSource,
        float64 currentExcitVal
    ) = 0;
    virtual int32 CreateTEDSAIThrmstrChanVex(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 resistanceConfig,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        float64 r1
    ) = 0;
    virtual int32 CreateTEDSAITorqueBridgeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateTEDSAIVoltageChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateTEDSAIVoltageChanWithExcit(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        const char customScaleName[]
    ) = 0;
    virtual int32 CreateTableScale(
        const char name[],
        const float64 prescaledVals[],
        uInt32 numPrescaledValsIn,
        const float64 scaledVals[],
        uInt32 numScaledValsIn,
        int32 preScaledUnits,
        const char scaledUnits[]
    ) = 0;
    virtual int32 CreateTask(const char sessionName[], TaskHandle *task) = 0;
    virtual int32 CreateWatchdogTimerTaskEx(
        const char deviceName[],
        const char sessionName[],
        TaskHandle *task,
        float64 timeout
    ) = 0;
    virtual int32 DeleteNetworkDevice(const char deviceName[]) = 0;
    virtual int32 DeleteSavedGlobalChan(const char channelName[]) = 0;
    virtual int32 DeleteSavedScale(const char scaleName[]) = 0;
    virtual int32 DeleteSavedTask(const char taskName[]) = 0;
    virtual int32 DeviceSupportsCal(const char deviceName[], bool32 *calSupported) = 0;
    virtual int32 DisableRefTrig(TaskHandle task) = 0;
    virtual int32 DisableStartTrig(TaskHandle task) = 0;
    virtual int32
    DisconnectTerms(const char sourceTerminal[], const char destinationTerminal[]) = 0;
    virtual int32
    ExportSignal(TaskHandle task, int32 signalID, const char outputTerminal[]) = 0;
    virtual int32 GetAIChanCalCalDate(
        TaskHandle task,
        const char channelName[],
        uInt32 *year,
        uInt32 *month,
        uInt32 *day,
        uInt32 *hour,
        uInt32 *minute
    ) = 0;
    virtual int32 GetAIChanCalExpDate(
        TaskHandle task,
        const char channelName[],
        uInt32 *year,
        uInt32 *month,
        uInt32 *day,
        uInt32 *hour,
        uInt32 *minute
    ) = 0;
    virtual int32 GetAnalogPowerUpStatesWithOutputType(
        const char channelNames[],
        float64 stateArray[],
        int32 channelTypeArray[],
        uInt32 *arraySize
    ) = 0;
    virtual int32
    GetArmStartTrigTimestampVal(TaskHandle task, CVIAbsoluteTime *data) = 0;
    virtual int32 GetArmStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime *data) = 0;
    virtual int32
    GetAutoConfiguredCDAQSyncConnections(char portList[], uInt32 portListSize) = 0;
    virtual int32
    GetBufferAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) = 0;
    virtual int32 GetCalInfoAttributeBool(
        const char deviceName[],
        int32 attribute,
        bool32 *value
    ) = 0;
    virtual int32 GetCalInfoAttributeDouble(
        const char deviceName[],
        int32 attribute,
        float64 *value
    ) = 0;
    virtual int32 GetCalInfoAttributeString(
        const char deviceName[],
        int32 attribute,
        char value[],
        uInt32 size
    ) = 0;
    virtual int32 GetCalInfoAttributeUInt32(
        const char deviceName[],
        int32 attribute,
        uInt32 *value
    ) = 0;
    virtual int32 GetChanAttributeBool(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        bool32 *value
    ) = 0;
    virtual int32 GetChanAttributeDouble(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        float64 *value
    ) = 0;
    virtual int32 GetChanAttributeDoubleArray(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        float64 value[],
        uInt32 size
    ) = 0;
    virtual int32 GetChanAttributeInt32(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        int32 *value
    ) = 0;
    virtual int32 GetChanAttributeString(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        char value[],
        uInt32 size
    ) = 0;
    virtual int32 GetChanAttributeUInt32(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        uInt32 *value
    ) = 0;
    virtual int32
    GetDeviceAttributeBool(const char deviceName[], int32 attribute, bool32 *value) = 0;
    virtual int32 GetDeviceAttributeDouble(
        const char deviceName[],
        int32 attribute,
        float64 *value
    ) = 0;
    virtual int32 GetDeviceAttributeDoubleArray(
        const char deviceName[],
        int32 attribute,
        float64 value[],
        uInt32 size
    ) = 0;
    virtual int32
    GetDeviceAttributeInt32(const char deviceName[], int32 attribute, int32 *value) = 0;
    virtual int32 GetDeviceAttributeInt32Array(
        const char deviceName[],
        int32 attribute,
        int32 value[],
        uInt32 size
    ) = 0;
    virtual int32 GetDeviceAttributeString(
        const char deviceName[],
        int32 attribute,
        char value[],
        uInt32 size
    ) = 0;
    virtual int32 GetDeviceAttributeUInt32(
        const char deviceName[],
        int32 attribute,
        uInt32 *value
    ) = 0;
    virtual int32 GetDeviceAttributeUInt32Array(
        const char deviceName[],
        int32 attribute,
        uInt32 value[],
        uInt32 size
    ) = 0;
    virtual int32
    GetDigitalLogicFamilyPowerUpState(const char deviceName[], int32 *logicFamily) = 0;
    virtual int32
    GetDisconnectedCDAQSyncPorts(char portList[], uInt32 portListSize) = 0;
    virtual int32
    GetErrorString(int32 errorCode, char errorString[], uInt32 bufferSize) = 0;
    virtual int32
    GetExportedSignalAttributeBool(TaskHandle task, int32 attribute, bool32 *value) = 0;
    virtual int32 GetExportedSignalAttributeDouble(
        TaskHandle task,
        int32 attribute,
        float64 *value
    ) = 0;
    virtual int32
    GetExportedSignalAttributeInt32(TaskHandle task, int32 attribute, int32 *value) = 0;
    virtual int32 GetExportedSignalAttributeString(
        TaskHandle task,
        int32 attribute,
        char value[],
        uInt32 size
    ) = 0;
    virtual int32 GetExportedSignalAttributeUInt32(
        TaskHandle task,
        int32 attribute,
        uInt32 *value
    ) = 0;
    virtual int32 GetExtCalLastDateAndTime(
        const char deviceName[],
        uInt32 *year,
        uInt32 *month,
        uInt32 *day,
        uInt32 *hour,
        uInt32 *minute
    ) = 0;
    virtual int32 GetExtendedErrorInfo(char errorString[], uInt32 bufferSize) = 0;
    virtual int32 GetFirstSampClkWhen(TaskHandle task, CVIAbsoluteTime *data) = 0;
    virtual int32 GetFirstSampTimestampVal(TaskHandle task, CVIAbsoluteTime *data) = 0;
    virtual int32 GetNthTaskChannel(
        TaskHandle task,
        uInt32 index,
        char buffer[],
        int32 bufferSize
    ) = 0;
    virtual int32 GetNthTaskDevice(
        TaskHandle task,
        uInt32 index,
        char buffer[],
        int32 bufferSize
    ) = 0;
    virtual int32 GetNthTaskReadChannel(
        TaskHandle task,
        uInt32 index,
        char buffer[],
        int32 bufferSize
    ) = 0;
    virtual int32 GetPersistedChanAttributeBool(
        const char channel[],
        int32 attribute,
        bool32 *value
    ) = 0;
    virtual int32 GetPersistedChanAttributeString(
        const char channel[],
        int32 attribute,
        char value[],
        uInt32 size
    ) = 0;
    virtual int32 GetPersistedScaleAttributeBool(
        const char scaleName[],
        int32 attribute,
        bool32 *value
    ) = 0;
    virtual int32 GetPersistedScaleAttributeString(
        const char scaleName[],
        int32 attribute,
        char value[],
        uInt32 size
    ) = 0;
    virtual int32 GetPersistedTaskAttributeBool(
        const char taskName[],
        int32 attribute,
        bool32 *value
    ) = 0;
    virtual int32 GetPersistedTaskAttributeString(
        const char taskName[],
        int32 attribute,
        char value[],
        uInt32 size
    ) = 0;
    virtual int32 GetPhysicalChanAttributeBool(
        const char physicalChannel[],
        int32 attribute,
        bool32 *value
    ) = 0;
    virtual int32 GetPhysicalChanAttributeBytes(
        const char physicalChannel[],
        int32 attribute,
        uInt8 value[],
        uInt32 size
    ) = 0;
    virtual int32 GetPhysicalChanAttributeDouble(
        const char physicalChannel[],
        int32 attribute,
        float64 *value
    ) = 0;
    virtual int32 GetPhysicalChanAttributeDoubleArray(
        const char physicalChannel[],
        int32 attribute,
        float64 value[],
        uInt32 size
    ) = 0;
    virtual int32 GetPhysicalChanAttributeInt32(
        const char physicalChannel[],
        int32 attribute,
        int32 *value
    ) = 0;
    virtual int32 GetPhysicalChanAttributeInt32Array(
        const char physicalChannel[],
        int32 attribute,
        int32 value[],
        uInt32 size
    ) = 0;
    virtual int32 GetPhysicalChanAttributeString(
        const char physicalChannel[],
        int32 attribute,
        char value[],
        uInt32 size
    ) = 0;
    virtual int32 GetPhysicalChanAttributeUInt32(
        const char physicalChannel[],
        int32 attribute,
        uInt32 *value
    ) = 0;
    virtual int32 GetPhysicalChanAttributeUInt32Array(
        const char physicalChannel[],
        int32 attribute,
        uInt32 value[],
        uInt32 size
    ) = 0;
    virtual int32
    GetReadAttributeBool(TaskHandle task, int32 attribute, bool32 *value) = 0;
    virtual int32
    GetReadAttributeDouble(TaskHandle task, int32 attribute, float64 *value) = 0;
    virtual int32
    GetReadAttributeInt32(TaskHandle task, int32 attribute, int32 *value) = 0;
    virtual int32 GetReadAttributeString(
        TaskHandle task,
        int32 attribute,
        char value[],
        uInt32 size
    ) = 0;
    virtual int32
    GetReadAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) = 0;
    virtual int32
    GetReadAttributeUInt64(TaskHandle task, int32 attribute, uInt64 *value) = 0;
    virtual int32
    GetRealTimeAttributeBool(TaskHandle task, int32 attribute, bool32 *value) = 0;
    virtual int32
    GetRealTimeAttributeInt32(TaskHandle task, int32 attribute, int32 *value) = 0;
    virtual int32
    GetRealTimeAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) = 0;
    virtual int32 GetRefTrigTimestampVal(TaskHandle task, CVIAbsoluteTime *data) = 0;
    virtual int32 GetScaleAttributeDouble(
        const char scaleName[],
        int32 attribute,
        float64 *value
    ) = 0;
    virtual int32 GetScaleAttributeDoubleArray(
        const char scaleName[],
        int32 attribute,
        float64 value[],
        uInt32 size
    ) = 0;
    virtual int32
    GetScaleAttributeInt32(const char scaleName[], int32 attribute, int32 *value) = 0;
    virtual int32 GetScaleAttributeString(
        const char scaleName[],
        int32 attribute,
        char value[],
        uInt32 size
    ) = 0;
    virtual int32 GetSelfCalLastDateAndTime(
        const char deviceName[],
        uInt32 *year,
        uInt32 *month,
        uInt32 *day,
        uInt32 *hour,
        uInt32 *minute
    ) = 0;
    virtual int32 GetStartTrigTimestampVal(TaskHandle task, CVIAbsoluteTime *data) = 0;
    virtual int32 GetStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime *data) = 0;
    virtual int32 GetSyncPulseTimeWhen(TaskHandle task, CVIAbsoluteTime *data) = 0;
    virtual int32
    GetSystemInfoAttributeString(int32 attribute, char value[], uInt32 size) = 0;
    virtual int32 GetSystemInfoAttributeUInt32(int32 attribute, uInt32 *value) = 0;
    virtual int32
    GetTaskAttributeBool(TaskHandle task, int32 attribute, bool32 *value) = 0;
    virtual int32 GetTaskAttributeString(
        TaskHandle task,
        int32 attribute,
        char value[],
        uInt32 size
    ) = 0;
    virtual int32
    GetTaskAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) = 0;
    virtual int32
    GetTimingAttributeBool(TaskHandle task, int32 attribute, bool32 *value) = 0;
    virtual int32
    GetTimingAttributeDouble(TaskHandle task, int32 attribute, float64 *value) = 0;
    virtual int32 GetTimingAttributeExBool(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        bool32 *value
    ) = 0;
    virtual int32 GetTimingAttributeExDouble(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        float64 *value
    ) = 0;
    virtual int32 GetTimingAttributeExInt32(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        int32 *value
    ) = 0;
    virtual int32 GetTimingAttributeExString(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        char value[],
        uInt32 size
    ) = 0;
    virtual int32 GetTimingAttributeExTimestamp(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        CVIAbsoluteTime *value
    ) = 0;
    virtual int32 GetTimingAttributeExUInt32(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        uInt32 *value
    ) = 0;
    virtual int32 GetTimingAttributeExUInt64(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        uInt64 *value
    ) = 0;
    virtual int32
    GetTimingAttributeInt32(TaskHandle task, int32 attribute, int32 *value) = 0;
    virtual int32 GetTimingAttributeString(
        TaskHandle task,
        int32 attribute,
        char value[],
        uInt32 size
    ) = 0;
    virtual int32 GetTimingAttributeTimestamp(
        TaskHandle task,
        int32 attribute,
        CVIAbsoluteTime *value
    ) = 0;
    virtual int32
    GetTimingAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) = 0;
    virtual int32
    GetTimingAttributeUInt64(TaskHandle task, int32 attribute, uInt64 *value) = 0;
    virtual int32
    GetTrigAttributeBool(TaskHandle task, int32 attribute, bool32 *value) = 0;
    virtual int32
    GetTrigAttributeDouble(TaskHandle task, int32 attribute, float64 *value) = 0;
    virtual int32 GetTrigAttributeDoubleArray(
        TaskHandle task,
        int32 attribute,
        float64 value[],
        uInt32 size
    ) = 0;
    virtual int32
    GetTrigAttributeInt32(TaskHandle task, int32 attribute, int32 *value) = 0;
    virtual int32 GetTrigAttributeInt32Array(
        TaskHandle task,
        int32 attribute,
        int32 value[],
        uInt32 size
    ) = 0;
    virtual int32 GetTrigAttributeString(
        TaskHandle task,
        int32 attribute,
        char value[],
        uInt32 size
    ) = 0;
    virtual int32 GetTrigAttributeTimestamp(
        TaskHandle task,
        int32 attribute,
        CVIAbsoluteTime *value
    ) = 0;
    virtual int32
    GetTrigAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) = 0;
    virtual int32 GetWatchdogAttributeBool(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        bool32 *value
    ) = 0;
    virtual int32 GetWatchdogAttributeDouble(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        float64 *value
    ) = 0;
    virtual int32 GetWatchdogAttributeInt32(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        int32 *value
    ) = 0;
    virtual int32 GetWatchdogAttributeString(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        char value[],
        uInt32 size
    ) = 0;
    virtual int32
    GetWriteAttributeBool(TaskHandle task, int32 attribute, bool32 *value) = 0;
    virtual int32
    GetWriteAttributeDouble(TaskHandle task, int32 attribute, float64 *value) = 0;
    virtual int32
    GetWriteAttributeInt32(TaskHandle task, int32 attribute, int32 *value) = 0;
    virtual int32 GetWriteAttributeString(
        TaskHandle task,
        int32 attribute,
        char value[],
        uInt32 size
    ) = 0;
    virtual int32
    GetWriteAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) = 0;
    virtual int32
    GetWriteAttributeUInt64(TaskHandle task, int32 attribute, uInt64 *value) = 0;
    virtual int32 IsTaskDone(TaskHandle task, bool32 *isTaskDone) = 0;
    virtual int32 LoadTask(const char sessionName[], TaskHandle *task) = 0;
    virtual int32 PerformBridgeOffsetNullingCalEx(
        TaskHandle task,
        const char channel[],
        bool32 skipUnsupportedChannels
    ) = 0;
    virtual int32 PerformBridgeShuntCalEx(
        TaskHandle task,
        const char channel[],
        float64 shuntResistorValue,
        int32 shuntResistorLocation,
        int32 shuntResistorSelect,
        int32 shuntResistorSource,
        float64 bridgeResistance,
        bool32 skipUnsupportedChannels
    ) = 0;
    virtual int32 PerformStrainShuntCalEx(
        TaskHandle task,
        const char channel[],
        float64 shuntResistorValue,
        int32 shuntResistorLocation,
        int32 shuntResistorSelect,
        int32 shuntResistorSource,
        bool32 skipUnsupportedChannels
    ) = 0;
    virtual int32 PerformThrmcplLeadOffsetNullingCal(
        TaskHandle task,
        const char channel[],
        bool32 skipUnsupportedChannels
    ) = 0;
    virtual int32 ReadAnalogF64(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        float64 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadAnalogScalarF64(
        TaskHandle task,
        float64 timeout,
        float64 *value,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadBinaryI16(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        int16 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadBinaryI32(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        int32 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadBinaryU16(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt16 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadBinaryU32(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt32 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadCounterF64(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        float64 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadCounterF64Ex(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        float64 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadCounterScalarF64(
        TaskHandle task,
        float64 timeout,
        float64 *value,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadCounterScalarU32(
        TaskHandle task,
        float64 timeout,
        uInt32 *value,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadCounterU32(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        uInt32 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadCounterU32Ex(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt32 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadCtrFreq(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 interleaved,
        float64 readArrayFrequency[],
        float64 readArrayDutyCycle[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadCtrFreqScalar(
        TaskHandle task,
        float64 timeout,
        float64 *frequency,
        float64 *dutyCycle,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadCtrTicks(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 interleaved,
        uInt32 readArrayHighTicks[],
        uInt32 readArrayLowTicks[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadCtrTicksScalar(
        TaskHandle task,
        float64 timeout,
        uInt32 *highTicks,
        uInt32 *lowTicks,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadCtrTime(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 interleaved,
        float64 readArrayHighTime[],
        float64 readArrayLowTime[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadCtrTimeScalar(
        TaskHandle task,
        float64 timeout,
        float64 *highTime,
        float64 *lowTime,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadDigitalLines(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt8 readArray[],
        uInt32 arraySizeInBytes,
        int32 *sampsPerChanRead,
        int32 *numBytesPerSamp,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadDigitalScalarU32(
        TaskHandle task,
        float64 timeout,
        uInt32 *value,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadDigitalU16(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt16 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadDigitalU32(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt32 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadDigitalU8(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt8 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadPowerBinaryI16(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        int16 readArrayVoltage[],
        int16 readArrayCurrent[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadPowerF64(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        float64 readArrayVoltage[],
        float64 readArrayCurrent[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadPowerScalarF64(
        TaskHandle task,
        float64 timeout,
        float64 *voltage,
        float64 *current,
        bool32 *reserved
    ) = 0;
    virtual int32 ReadRaw(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        uInt8 readArray[],
        uInt32 arraySizeInBytes,
        int32 *sampsRead,
        int32 *numBytesPerSamp,
        bool32 *reserved
    ) = 0;
    virtual int32 RegisterDoneEvent(
        TaskHandle task,
        uInt32 options,
        DAQmxDoneEventCallbackPtr callbackFunction,
        void *callbackData
    ) = 0;
    virtual int32 RegisterEveryNSamplesEvent(
        TaskHandle task,
        int32 everyNSamplesEventType,
        uInt32 nSamples,
        uInt32 options,
        DAQmxEveryNSamplesEventCallbackPtr callbackFunction,
        void *callbackData
    ) = 0;
    virtual int32 RegisterSignalEvent(
        TaskHandle task,
        int32 signalID,
        uInt32 options,
        DAQmxSignalEventCallbackPtr callbackFunction,
        void *callbackData
    ) = 0;
    virtual int32 RemoveCDAQSyncConnection(const char portList[]) = 0;
    virtual int32
    ReserveNetworkDevice(const char deviceName[], bool32 overrideReservation) = 0;
    virtual int32 ResetBufferAttribute(TaskHandle task, int32 attribute) = 0;
    virtual int32
    ResetChanAttribute(TaskHandle task, const char channel[], int32 attribute) = 0;
    virtual int32 ResetDevice(const char deviceName[]) = 0;
    virtual int32 ResetExportedSignalAttribute(TaskHandle task, int32 attribute) = 0;
    virtual int32 ResetReadAttribute(TaskHandle task, int32 attribute) = 0;
    virtual int32 ResetRealTimeAttribute(TaskHandle task, int32 attribute) = 0;
    virtual int32 ResetTimingAttribute(TaskHandle task, int32 attribute) = 0;
    virtual int32 ResetTimingAttributeEx(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute
    ) = 0;
    virtual int32 ResetTrigAttribute(TaskHandle task, int32 attribute) = 0;
    virtual int32
    ResetWatchdogAttribute(TaskHandle task, const char lines[], int32 attribute) = 0;
    virtual int32 ResetWriteAttribute(TaskHandle task, int32 attribute) = 0;
    virtual int32 RestoreLastExtCalConst(const char deviceName[]) = 0;
    virtual int32 SaveGlobalChan(
        TaskHandle task,
        const char channelName[],
        const char saveAs[],
        const char author[],
        uInt32 options
    ) = 0;
    virtual int32 SaveScale(
        const char scaleName[],
        const char saveAs[],
        const char author[],
        uInt32 options
    ) = 0;
    virtual int32 SaveTask(
        TaskHandle task,
        const char saveAs[],
        const char author[],
        uInt32 options
    ) = 0;
    virtual int32 SelfCal(const char deviceName[]) = 0;
    virtual int32 SelfTestDevice(const char deviceName[]) = 0;
    virtual int32 SetAIChanCalCalDate(
        TaskHandle task,
        const char channelName[],
        uInt32 year,
        uInt32 month,
        uInt32 day,
        uInt32 hour,
        uInt32 minute
    ) = 0;
    virtual int32 SetAIChanCalExpDate(
        TaskHandle task,
        const char channelName[],
        uInt32 year,
        uInt32 month,
        uInt32 day,
        uInt32 hour,
        uInt32 minute
    ) = 0;
    virtual int32 SetAnalogPowerUpStatesWithOutputType(
        const char channelNames[],
        const float64 stateArray[],
        const int32 channelTypeArray[],
        uInt32 arraySize
    ) = 0;
    virtual int32 SetArmStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime data) = 0;
    virtual int32
    SetBufferAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) = 0;
    virtual int32
    SetCalInfoAttributeBool(const char deviceName[], int32 attribute, bool32 value) = 0;
    virtual int32 SetCalInfoAttributeDouble(
        const char deviceName[],
        int32 attribute,
        float64 value
    ) = 0;
    virtual int32 SetCalInfoAttributeString(
        const char deviceName[],
        int32 attribute,
        const char value[]
    ) = 0;
    virtual int32 SetCalInfoAttributeUInt32(
        const char deviceName[],
        int32 attribute,
        uInt32 value
    ) = 0;
    virtual int32 SetChanAttributeBool(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        bool32 value
    ) = 0;
    virtual int32 SetChanAttributeDouble(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        float64 value
    ) = 0;
    virtual int32 SetChanAttributeDoubleArray(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        const float64 value[],
        uInt32 size
    ) = 0;
    virtual int32 SetChanAttributeInt32(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        int32 value
    ) = 0;
    virtual int32 SetChanAttributeString(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        const char value[]
    ) = 0;
    virtual int32 SetChanAttributeUInt32(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        uInt32 value
    ) = 0;
    virtual int32
    SetDigitalLogicFamilyPowerUpState(const char deviceName[], int32 logicFamily) = 0;
    virtual int32
    SetExportedSignalAttributeBool(TaskHandle task, int32 attribute, bool32 value) = 0;
    virtual int32 SetExportedSignalAttributeDouble(
        TaskHandle task,
        int32 attribute,
        float64 value
    ) = 0;
    virtual int32
    SetExportedSignalAttributeInt32(TaskHandle task, int32 attribute, int32 value) = 0;
    virtual int32 SetExportedSignalAttributeString(
        TaskHandle task,
        int32 attribute,
        const char value[]
    ) = 0;
    virtual int32 SetExportedSignalAttributeUInt32(
        TaskHandle task,
        int32 attribute,
        uInt32 value
    ) = 0;
    virtual int32 SetFirstSampClkWhen(TaskHandle task, CVIAbsoluteTime data) = 0;
    virtual int32
    SetReadAttributeBool(TaskHandle task, int32 attribute, bool32 value) = 0;
    virtual int32
    SetReadAttributeDouble(TaskHandle task, int32 attribute, float64 value) = 0;
    virtual int32
    SetReadAttributeInt32(TaskHandle task, int32 attribute, int32 value) = 0;
    virtual int32
    SetReadAttributeString(TaskHandle task, int32 attribute, const char value[]) = 0;
    virtual int32
    SetReadAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) = 0;
    virtual int32
    SetReadAttributeUInt64(TaskHandle task, int32 attribute, uInt64 value) = 0;
    virtual int32
    SetRealTimeAttributeBool(TaskHandle task, int32 attribute, bool32 value) = 0;
    virtual int32
    SetRealTimeAttributeInt32(TaskHandle task, int32 attribute, int32 value) = 0;
    virtual int32
    SetRealTimeAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) = 0;
    virtual int32 SetRuntimeEnvironment(
        const char environment[],
        const char environmentVersion[],
        const char reserved1[],
        const char reserved2[]
    ) = 0;
    virtual int32
    SetScaleAttributeDouble(const char scaleName[], int32 attribute, float64 value) = 0;
    virtual int32 SetScaleAttributeDoubleArray(
        const char scaleName[],
        int32 attribute,
        const float64 value[],
        uInt32 size
    ) = 0;
    virtual int32
    SetScaleAttributeInt32(const char scaleName[], int32 attribute, int32 value) = 0;
    virtual int32 SetScaleAttributeString(
        const char scaleName[],
        int32 attribute,
        const char value[]
    ) = 0;
    virtual int32 SetStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime data) = 0;
    virtual int32 SetSyncPulseTimeWhen(TaskHandle task, CVIAbsoluteTime data) = 0;
    virtual int32
    SetTimingAttributeBool(TaskHandle task, int32 attribute, bool32 value) = 0;
    virtual int32
    SetTimingAttributeDouble(TaskHandle task, int32 attribute, float64 value) = 0;
    virtual int32 SetTimingAttributeExBool(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        bool32 value
    ) = 0;
    virtual int32 SetTimingAttributeExDouble(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        float64 value
    ) = 0;
    virtual int32 SetTimingAttributeExInt32(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        int32 value
    ) = 0;
    virtual int32 SetTimingAttributeExString(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        const char value[]
    ) = 0;
    virtual int32 SetTimingAttributeExTimestamp(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        CVIAbsoluteTime value
    ) = 0;
    virtual int32 SetTimingAttributeExUInt32(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        uInt32 value
    ) = 0;
    virtual int32 SetTimingAttributeExUInt64(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        uInt64 value
    ) = 0;
    virtual int32
    SetTimingAttributeInt32(TaskHandle task, int32 attribute, int32 value) = 0;
    virtual int32
    SetTimingAttributeString(TaskHandle task, int32 attribute, const char value[]) = 0;
    virtual int32 SetTimingAttributeTimestamp(
        TaskHandle task,
        int32 attribute,
        CVIAbsoluteTime value
    ) = 0;
    virtual int32
    SetTimingAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) = 0;
    virtual int32
    SetTimingAttributeUInt64(TaskHandle task, int32 attribute, uInt64 value) = 0;
    virtual int32
    SetTrigAttributeBool(TaskHandle task, int32 attribute, bool32 value) = 0;
    virtual int32
    SetTrigAttributeDouble(TaskHandle task, int32 attribute, float64 value) = 0;
    virtual int32 SetTrigAttributeDoubleArray(
        TaskHandle task,
        int32 attribute,
        const float64 value[],
        uInt32 size
    ) = 0;
    virtual int32
    SetTrigAttributeInt32(TaskHandle task, int32 attribute, int32 value) = 0;
    virtual int32 SetTrigAttributeInt32Array(
        TaskHandle task,
        int32 attribute,
        const int32 value[],
        uInt32 size
    ) = 0;
    virtual int32
    SetTrigAttributeString(TaskHandle task, int32 attribute, const char value[]) = 0;
    virtual int32 SetTrigAttributeTimestamp(
        TaskHandle task,
        int32 attribute,
        CVIAbsoluteTime value
    ) = 0;
    virtual int32
    SetTrigAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) = 0;
    virtual int32 SetWatchdogAttributeBool(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        bool32 value
    ) = 0;
    virtual int32 SetWatchdogAttributeDouble(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        float64 value
    ) = 0;
    virtual int32 SetWatchdogAttributeInt32(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        int32 value
    ) = 0;
    virtual int32 SetWatchdogAttributeString(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        const char value[]
    ) = 0;
    virtual int32
    SetWriteAttributeBool(TaskHandle task, int32 attribute, bool32 value) = 0;
    virtual int32
    SetWriteAttributeDouble(TaskHandle task, int32 attribute, float64 value) = 0;
    virtual int32
    SetWriteAttributeInt32(TaskHandle task, int32 attribute, int32 value) = 0;
    virtual int32
    SetWriteAttributeString(TaskHandle task, int32 attribute, const char value[]) = 0;
    virtual int32
    SetWriteAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) = 0;
    virtual int32
    SetWriteAttributeUInt64(TaskHandle task, int32 attribute, uInt64 value) = 0;
    virtual int32 StartNewFile(TaskHandle task, const char filePath[]) = 0;
    virtual int32 StartTask(TaskHandle task) = 0;
    virtual int32 StopTask(TaskHandle task) = 0;
    virtual int32 TaskControl(TaskHandle task, int32 action) = 0;
    virtual int32 TristateOutputTerm(const char outputTerminal[]) = 0;
    virtual int32 UnregisterDoneEvent(
        TaskHandle task,
        uInt32 options,
        DAQmxDoneEventCallbackPtr callbackFunction,
        void *callbackData
    ) = 0;
    virtual int32 UnregisterEveryNSamplesEvent(
        TaskHandle task,
        int32 everyNSamplesEventType,
        uInt32 nSamples,
        uInt32 options,
        DAQmxEveryNSamplesEventCallbackPtr callbackFunction,
        void *callbackData
    ) = 0;
    virtual int32 UnregisterSignalEvent(
        TaskHandle task,
        int32 signalID,
        uInt32 options,
        DAQmxSignalEventCallbackPtr callbackFunction,
        void *callbackData
    ) = 0;
    virtual int32 UnreserveNetworkDevice(const char deviceName[]) = 0;
    virtual int32
    WaitForNextSampleClock(TaskHandle task, float64 timeout, bool32 *isLate) = 0;
    virtual int32 WaitForValidTimestamp(
        TaskHandle task,
        int32 timestampEvent,
        float64 timeout,
        CVIAbsoluteTime *timestamp
    ) = 0;
    virtual int32 WaitUntilTaskDone(TaskHandle task, float64 timeToWait) = 0;
    virtual int32 WriteAnalogF64(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const float64 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) = 0;
    virtual int32 WriteAnalogScalarF64(
        TaskHandle task,
        bool32 autoStart,
        float64 timeout,
        float64 value,
        bool32 *reserved
    ) = 0;
    virtual int32 WriteBinaryI16(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const int16 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) = 0;
    virtual int32 WriteBinaryI32(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const int32 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) = 0;
    virtual int32 WriteBinaryU16(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt16 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) = 0;
    virtual int32 WriteBinaryU32(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt32 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) = 0;
    virtual int32 WriteCtrFreq(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const float64 frequency[],
        const float64 dutyCycle[],
        int32 *numSampsPerChanWritten,
        bool32 *reserved
    ) = 0;
    virtual int32 WriteCtrFreqScalar(
        TaskHandle task,
        bool32 autoStart,
        float64 timeout,
        float64 frequency,
        float64 dutyCycle,
        bool32 *reserved
    ) = 0;
    virtual int32 WriteCtrTicks(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt32 highTicks[],
        const uInt32 lowTicks[],
        int32 *numSampsPerChanWritten,
        bool32 *reserved
    ) = 0;
    virtual int32 WriteCtrTicksScalar(
        TaskHandle task,
        bool32 autoStart,
        float64 timeout,
        uInt32 highTicks,
        uInt32 lowTicks,
        bool32 *reserved
    ) = 0;
    virtual int32 WriteCtrTime(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const float64 highTime[],
        const float64 lowTime[],
        int32 *numSampsPerChanWritten,
        bool32 *reserved
    ) = 0;
    virtual int32 WriteCtrTimeScalar(
        TaskHandle task,
        bool32 autoStart,
        float64 timeout,
        float64 highTime,
        float64 lowTime,
        bool32 *reserved
    ) = 0;
    virtual int32 WriteDigitalLines(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt8 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) = 0;
    virtual int32 WriteDigitalScalarU32(
        TaskHandle task,
        bool32 autoStart,
        float64 timeout,
        uInt32 value,
        bool32 *reserved
    ) = 0;
    virtual int32 WriteDigitalU16(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt16 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) = 0;
    virtual int32 WriteDigitalU32(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt32 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) = 0;
    virtual int32 WriteDigitalU8(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt8 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) = 0;
    virtual int32 WriteRaw(
        TaskHandle task,
        int32 numSamps,
        bool32 autoStart,
        float64 timeout,
        const uInt8 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) = 0;
    virtual int32 WriteToTEDSFromArray(
        const char physicalChannel[],
        const uInt8 bitStream[],
        uInt32 arraySize,
        int32 basicTEDSOptions
    ) = 0;
    virtual int32 WriteToTEDSFromFile(
        const char physicalChannel[],
        const char filePath[],
        int32 basicTEDSOptions
    ) = 0;
    virtual int32 SetReadRelativeTo(TaskHandle taskHandle, int32 data) = 0;
    virtual int32 SetReadOffset(TaskHandle taskHandle, int32 data) = 0;
    virtual int32 SetReadOverWrite(TaskHandle taskHandle, int32 data) = 0;
    virtual int32
    GetReadTotalSampPerChanAcquired(TaskHandle taskHandle, uInt64 *data) = 0;
};
}
