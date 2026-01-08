// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include <memory>

#include "x/cpp/errors/errors.h"
#include "x/cpp/lib/xlib.h"

#include "driver/ni/daqmx/api.h"
#include "driver/ni/daqmx/nidaqmx.h"

namespace daqmx {
class ProdAPI final : public API {
public:
    explicit ProdAPI(std::unique_ptr<x::lib::Shared> &lib_);

    ~ProdAPI() override;

    static std::pair<std::shared_ptr<API>, x::errors::Error> load();

    int32 AddCDAQSyncConnection(const char portList[]) override;

    int32 AddGlobalChansToTask(TaskHandle task, const char channelNames[]) override;

    int32 AddNetworkDevice(
        const char ipAddress[],
        const char deviceName[],
        bool32 attemptReservation,
        float64 timeout,
        char deviceNameOut[],
        uInt32 deviceNameOutBufferSize
    ) override;

    int32 AreConfiguredCDAQSyncPortsDisconnected(
        const char chassisDevicesPorts[],
        float64 timeout,
        bool32 *disconnectedPortsExist
    ) override;

    int32 AutoConfigureCDAQSyncConnections(
        const char chassisDevicesPorts[],
        float64 timeout
    ) override;

    int32 CalculateReversePolyCoeff(
        const float64 forwardCoeffs[],
        uInt32 numForwardCoeffsIn,
        float64 minValX,
        float64 maxValX,
        int32 numPointsToCompute,
        int32 reversePolyOrder,
        float64 reverseCoeffs[]
    ) override;

    int32 CfgAnlgEdgeRefTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerSlope,
        float64 triggerLevel,
        uInt32 pretriggerSamples
    ) override;

    int32 CfgAnlgEdgeStartTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerSlope,
        float64 triggerLevel
    ) override;

    int32 CfgAnlgMultiEdgeRefTrig(
        TaskHandle task,
        const char triggerSources[],
        const int32 triggerSlopeArray[],
        const float64 triggerLevelArray[],
        uInt32 pretriggerSamples,
        uInt32 arraySize
    ) override;

    int32 CfgAnlgMultiEdgeStartTrig(
        TaskHandle task,
        const char triggerSources[],
        const int32 triggerSlopeArray[],
        const float64 triggerLevelArray[],
        uInt32 arraySize
    ) override;

    int32 CfgAnlgWindowRefTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerWhen,
        float64 windowTop,
        float64 windowBottom,
        uInt32 pretriggerSamples
    ) override;

    int32 CfgAnlgWindowStartTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerWhen,
        float64 windowTop,
        float64 windowBottom
    ) override;

    int32 CfgBurstHandshakingTimingExportClock(
        TaskHandle task,
        int32 sampleMode,
        uInt64 sampsPerChan,
        float64 sampleClkRate,
        const char sampleClkOutpTerm[],
        int32 sampleClkPulsePolarity,
        int32 pauseWhen,
        int32 readyEventActiveLevel
    ) override;

    int32 CfgBurstHandshakingTimingImportClock(
        TaskHandle task,
        int32 sampleMode,
        uInt64 sampsPerChan,
        float64 sampleClkRate,
        const char sampleClkSrc[],
        int32 sampleClkActiveEdge,
        int32 pauseWhen,
        int32 readyEventActiveLevel
    ) override;

    int32 CfgChangeDetectionTiming(
        TaskHandle task,
        const char risingEdgeChan[],
        const char fallingEdgeChan[],
        int32 sampleMode,
        uInt64 sampsPerChan
    ) override;

    int32 CfgDigEdgeRefTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerEdge,
        uInt32 pretriggerSamples
    ) override;

    int32 CfgDigEdgeStartTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerEdge
    ) override;

    int32 CfgDigPatternRefTrig(
        TaskHandle task,
        const char triggerSource[],
        const char triggerPattern[],
        int32 triggerWhen,
        uInt32 pretriggerSamples
    ) override;

    int32 CfgDigPatternStartTrig(
        TaskHandle task,
        const char triggerSource[],
        const char triggerPattern[],
        int32 triggerWhen
    ) override;

    int32 CfgHandshakingTiming(
        TaskHandle task,
        int32 sampleMode,
        uInt64 sampsPerChan
    ) override;

    int32
    CfgImplicitTiming(TaskHandle task, int32 sampleMode, uInt64 sampsPerChan) override;

    int32 CfgInputBuffer(TaskHandle task, uInt32 numSampsPerChan) override;

    int32 CfgOutputBuffer(TaskHandle task, uInt32 numSampsPerChan) override;

    int32 CfgPipelinedSampClkTiming(
        TaskHandle task,
        const char source[],
        float64 rate,
        int32 activeEdge,
        int32 sampleMode,
        uInt64 sampsPerChan
    ) override;

    int32 CfgSampClkTiming(
        TaskHandle task,
        const char source[],
        float64 rate,
        int32 activeEdge,
        int32 sampleMode,
        uInt64 sampsPerChan
    ) override;

    int32
    CfgTimeStartTrig(TaskHandle task, CVIAbsoluteTime when, int32 timescale) override;

    int32 CfgWatchdogAOExpirStates(
        TaskHandle task,
        const char channelNames[],
        const float64 expirStateArray[],
        const int32 outputTypeArray[],
        uInt32 arraySize
    ) override;

    int32 CfgWatchdogCOExpirStates(
        TaskHandle task,
        const char channelNames[],
        const int32 expirStateArray[],
        uInt32 arraySize
    ) override;

    int32 CfgWatchdogDOExpirStates(
        TaskHandle task,
        const char channelNames[],
        const int32 expirStateArray[],
        uInt32 arraySize
    ) override;

    int32 ClearTEDS(const char physicalChannel[]) override;

    int32 ClearTask(TaskHandle task) override;

    int32 ConfigureLogging(
        TaskHandle task,
        const char filePath[],
        int32 loggingMode,
        const char groupName[],
        int32 operation
    ) override;

    int32 ConfigureTEDS(const char physicalChannel[], const char filePath[]) override;

    int32 ConnectTerms(
        const char sourceTerminal[],
        const char destinationTerminal[],
        int32 signalModifiers
    ) override;

    int32 ControlWatchdogTask(TaskHandle task, int32 action) override;

    int32 CreateAIAccel4WireDCVoltageChan(
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
    ) override;

    int32 CreateAIAccelChan(
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
    ) override;

    int32 CreateAIAccelChargeChan(
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
    ) override;

    int32 CreateAIBridgeChan(
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
    ) override;

    int32 CreateAIChargeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    ) override;

    int32 CreateAICurrentChan(
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
    ) override;

    int32 CreateAICurrentRMSChan(
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
    ) override;

    int32 CreateAIForceBridgePolynomialChan(
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
    ) override;

    int32 CreateAIForceBridgeTableChan(
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
    ) override;

    int32 CreateAIForceBridgeTwoPointLinChan(
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
    ) override;

    int32 CreateAIForceIEPEChan(
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
    ) override;

    int32 CreateAIFreqVoltageChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        float64 thresholdLevel,
        float64 hysteresis,
        const char customScaleName[]
    ) override;

    int32 CreateAIMicrophoneChan(
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
    ) override;

    int32 CreateAIPosEddyCurrProxProbeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        float64 sensitivity,
        int32 sensitivityUnits,
        const char customScaleName[]
    ) override;

    int32 CreateAIPosLVDTChan(
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
    ) override;

    int32 CreateAIPosRVDTChan(
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
    ) override;

    int32 CreateAIPowerChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 voltageSetpoint,
        float64 currentSetpoint,
        bool32 outputEnable
    ) override;

    int32 CreateAIPressureBridgePolynomialChan(
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
    ) override;

    int32 CreateAIPressureBridgeTableChan(
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
    ) override;

    int32 CreateAIPressureBridgeTwoPointLinChan(
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
    ) override;

    int32 CreateAIRTDChan(
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
    ) override;

    int32 CreateAIResistanceChan(
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
    ) override;

    int32 CreateAIRosetteStrainGageChan(
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
    ) override;

    int32 CreateAIStrainGageChan(
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
    ) override;

    int32 CreateAITempBuiltInSensorChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 units
    ) override;

    int32 CreateAIThrmcplChan(
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
    ) override;

    int32 CreateAIThrmstrChanIex(
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
    ) override;

    int32 CreateAIThrmstrChanVex(
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
    ) override;

    int32 CreateAITorqueBridgePolynomialChan(
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
    ) override;

    int32 CreateAITorqueBridgeTableChan(
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
    ) override;

    int32 CreateAITorqueBridgeTwoPointLinChan(
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
    ) override;

    int32 CreateAIVelocityIEPEChan(
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
    ) override;

    int32 CreateAIVoltageChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    ) override;

    int32 CreateAIVoltageChanWithExcit(
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
    ) override;

    int32 CreateAIVoltageRMSChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    ) override;

    int32 CreateAOCurrentChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    ) override;

    int32 CreateAOFuncGenChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 type,
        float64 freq,
        float64 amplitude,
        float64 offset
    ) override;

    int32 CreateAOVoltageChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    ) override;

    int32 CreateCIAngEncoderChan(
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
    ) override;

    int32 CreateCIAngVelocityChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 decodingType,
        int32 units,
        uInt32 pulsesPerRev,
        const char customScaleName[]
    ) override;

    int32 CreateCICountEdgesChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        int32 edge,
        uInt32 initialCount,
        int32 countDirection
    ) override;

    int32 CreateCIDutyCycleChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minFreq,
        float64 maxFreq,
        int32 edge,
        const char customScaleName[]
    ) override;

    int32 CreateCIFreqChan(
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
    ) override;

    int32 CreateCIGPSTimestampChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        int32 units,
        int32 syncMethod,
        const char customScaleName[]
    ) override;

    int32 CreateCILinEncoderChan(
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
    ) override;

    int32 CreateCILinVelocityChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 decodingType,
        int32 units,
        float64 distPerPulse,
        const char customScaleName[]
    ) override;

    int32 CreateCIPeriodChan(
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
    ) override;

    int32 CreateCIPulseChanFreq(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units
    ) override;

    int32 CreateCIPulseChanTicks(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        const char sourceTerminal[],
        float64 minVal,
        float64 maxVal
    ) override;

    int32 CreateCIPulseChanTime(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units
    ) override;

    int32 CreateCIPulseWidthChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 startingEdge,
        const char customScaleName[]
    ) override;

    int32 CreateCISemiPeriodChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    ) override;

    int32 CreateCITwoEdgeSepChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 firstEdge,
        int32 secondEdge,
        const char customScaleName[]
    ) override;

    int32 CreateCOPulseChanFreq(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        int32 units,
        int32 idleState,
        float64 initialDelay,
        float64 freq,
        float64 dutyCycle
    ) override;

    int32 CreateCOPulseChanTicks(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        const char sourceTerminal[],
        int32 idleState,
        int32 initialDelay,
        int32 lowTicks,
        int32 highTicks
    ) override;

    int32 CreateCOPulseChanTime(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        int32 units,
        int32 idleState,
        float64 initialDelay,
        float64 lowTime,
        float64 highTime
    ) override;

    int32 CreateDIChan(
        TaskHandle task,
        const char lines[],
        const char nameToAssignToLines[],
        int32 lineGrouping
    ) override;

    int32 CreateDOChan(
        TaskHandle task,
        const char lines[],
        const char nameToAssignToLines[],
        int32 lineGrouping
    ) override;

    int32 CreateLinScale(
        const char name[],
        float64 slope,
        float64 yIntercept,
        int32 preScaledUnits,
        const char scaledUnits[]
    ) override;

    int32 CreateMapScale(
        const char name[],
        float64 prescaledMin,
        float64 prescaledMax,
        float64 scaledMin,
        float64 scaledMax,
        int32 preScaledUnits,
        const char scaledUnits[]
    ) override;

    int32 CreatePolynomialScale(
        const char name[],
        const float64 forwardCoeffs[],
        uInt32 numForwardCoeffsIn,
        const float64 reverseCoeffs[],
        uInt32 numReverseCoeffsIn,
        int32 preScaledUnits,
        const char scaledUnits[]
    ) override;

    int32 CreateTEDSAIAccelChan(
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
    ) override;

    int32 CreateTEDSAIBridgeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        const char customScaleName[]
    ) override;

    int32 CreateTEDSAICurrentChan(
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
    ) override;

    int32 CreateTEDSAIForceBridgeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        const char customScaleName[]
    ) override;

    int32 CreateTEDSAIForceIEPEChan(
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
    ) override;

    int32 CreateTEDSAIMicrophoneChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        int32 units,
        float64 maxSndPressLevel,
        int32 currentExcitSource,
        float64 currentExcitVal,
        const char customScaleName[]
    ) override;

    int32 CreateTEDSAIPosLVDTChan(
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
    ) override;

    int32 CreateTEDSAIPosRVDTChan(
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
    ) override;

    int32 CreateTEDSAIPressureBridgeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        const char customScaleName[]
    ) override;

    int32 CreateTEDSAIRTDChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 resistanceConfig,
        int32 currentExcitSource,
        float64 currentExcitVal
    ) override;

    int32 CreateTEDSAIResistanceChan(
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
    ) override;

    int32 CreateTEDSAIStrainGageChan(
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
    ) override;

    int32 CreateTEDSAIThrmcplChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 cjcSource,
        float64 cjcVal,
        const char cjcChannel[]
    ) override;

    int32 CreateTEDSAIThrmstrChanIex(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 resistanceConfig,
        int32 currentExcitSource,
        float64 currentExcitVal
    ) override;

    int32 CreateTEDSAIThrmstrChanVex(
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
    ) override;

    int32 CreateTEDSAITorqueBridgeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        const char customScaleName[]
    ) override;

    int32 CreateTEDSAIVoltageChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    ) override;

    int32 CreateTEDSAIVoltageChanWithExcit(
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
    ) override;

    int32 CreateTableScale(
        const char name[],
        const float64 prescaledVals[],
        uInt32 numPrescaledValsIn,
        const float64 scaledVals[],
        uInt32 numScaledValsIn,
        int32 preScaledUnits,
        const char scaledUnits[]
    ) override;

    int32 CreateTask(const char sessionName[], TaskHandle *task) override;

    int32 CreateWatchdogTimerTask(
        const char deviceName[],
        const char sessionName[],
        TaskHandle *task,
        float64 timeout,
        const char lines[],
        int32 expState,
        const char lines0[],
        int32 expState0,
        const char lines1[],
        int32 expState1,
        const char lines2[],
        int32 expState2,
        const char lines3[],
        int32 expState3,
        const char lines4[],
        int32 expState4,
        const char lines5[],
        int32 expState5,
        const char lines6[],
        int32 expState6,
        const char lines7[],
        int32 expState7,
        const char lines8[],
        int32 expState8,
        const char lines9[],
        int32 expState9,
        const char lines10[],
        int32 expState10,
        const char lines11[],
        int32 expState11,
        const char lines12[],
        int32 expState12,
        const char lines13[],
        int32 expState13,
        const char lines14[],
        int32 expState14,
        const char lines15[],
        int32 expState15,
        const char lines16[],
        int32 expState16,
        const char lines17[],
        int32 expState17,
        const char lines18[],
        int32 expState18,
        const char lines19[],
        int32 expState19,
        const char lines20[],
        int32 expState20,
        const char lines21[],
        int32 expState21,
        const char lines22[],
        int32 expState22,
        const char lines23[],
        int32 expState23,
        const char lines24[],
        int32 expState24,
        const char lines25[],
        int32 expState25,
        const char lines26[],
        int32 expState26,
        const char lines27[],
        int32 expState27,
        const char lines28[],
        int32 expState28,
        const char lines29[],
        int32 expState29,
        const char lines30[],
        int32 expState30,
        const char lines31[],
        int32 expState31,
        const char lines32[],
        int32 expState32,
        const char lines33[],
        int32 expState33,
        const char lines34[],
        int32 expState34,
        const char lines35[],
        int32 expState35,
        const char lines36[],
        int32 expState36,
        const char lines37[],
        int32 expState37,
        const char lines38[],
        int32 expState38,
        const char lines39[],
        int32 expState39,
        const char lines40[],
        int32 expState40,
        const char lines41[],
        int32 expState41,
        const char lines42[],
        int32 expState42,
        const char lines43[],
        int32 expState43,
        const char lines44[],
        int32 expState44,
        const char lines45[],
        int32 expState45,
        const char lines46[],
        int32 expState46,
        const char lines47[],
        int32 expState47,
        const char lines48[],
        int32 expState48,
        const char lines49[],
        int32 expState49,
        const char lines50[],
        int32 expState50,
        const char lines51[],
        int32 expState51,
        const char lines52[],
        int32 expState52,
        const char lines53[],
        int32 expState53,
        const char lines54[],
        int32 expState54,
        const char lines55[],
        int32 expState55,
        const char lines56[],
        int32 expState56,
        const char lines57[],
        int32 expState57,
        const char lines58[],
        int32 expState58,
        const char lines59[],
        int32 expState59,
        const char lines60[],
        int32 expState60,
        const char lines61[],
        int32 expState61,
        const char lines62[],
        int32 expState62,
        const char lines63[],
        int32 expState63,
        const char lines64[],
        int32 expState64,
        const char lines65[],
        int32 expState65,
        const char lines66[],
        int32 expState66,
        const char lines67[],
        int32 expState67,
        const char lines68[],
        int32 expState68,
        const char lines69[],
        int32 expState69,
        const char lines70[],
        int32 expState70,
        const char lines71[],
        int32 expState71,
        const char lines72[],
        int32 expState72,
        const char lines73[],
        int32 expState73,
        const char lines74[],
        int32 expState74,
        const char lines75[],
        int32 expState75,
        const char lines76[],
        int32 expState76,
        const char lines77[],
        int32 expState77,
        const char lines78[],
        int32 expState78,
        const char lines79[],
        int32 expState79,
        const char lines80[],
        int32 expState80,
        const char lines81[],
        int32 expState81,
        const char lines82[],
        int32 expState82,
        const char lines83[],
        int32 expState83,
        const char lines84[],
        int32 expState84,
        const char lines85[],
        int32 expState85,
        const char lines86[],
        int32 expState86,
        const char lines87[],
        int32 expState87,
        const char lines88[],
        int32 expState88,
        const char lines89[],
        int32 expState89,
        const char lines90[],
        int32 expState90,
        const char lines91[],
        int32 expState91,
        const char lines92[],
        int32 expState92,
        const char lines93[],
        int32 expState93,
        const char lines94[],
        int32 expState94,
        const char lines95[],
        int32 expState95
    );

    int32 CreateWatchdogTimerTaskEx(
        const char deviceName[],
        const char sessionName[],
        TaskHandle *task,
        float64 timeout
    ) override;

    int32 DeleteNetworkDevice(const char deviceName[]) override;

    int32 DeleteSavedGlobalChan(const char channelName[]) override;

    int32 DeleteSavedScale(const char scaleName[]) override;

    int32 DeleteSavedTask(const char taskName[]) override;

    int32 DeviceSupportsCal(const char deviceName[], bool32 *calSupported) override;

    int32 DisableRefTrig(TaskHandle task) override;

    int32 DisableStartTrig(TaskHandle task) override;

    int32 DisconnectTerms(
        const char sourceTerminal[],
        const char destinationTerminal[]
    ) override;

    int32
    ExportSignal(TaskHandle task, int32 signalID, const char outputTerminal[]) override;

    int32 GetAIChanCalCalDate(
        TaskHandle task,
        const char channelName[],
        uInt32 *year,
        uInt32 *month,
        uInt32 *day,
        uInt32 *hour,
        uInt32 *minute
    ) override;

    int32 GetAIChanCalExpDate(
        TaskHandle task,
        const char channelName[],
        uInt32 *year,
        uInt32 *month,
        uInt32 *day,
        uInt32 *hour,
        uInt32 *minute
    ) override;

    int32 GetAnalogPowerUpStates(
        const char deviceName[],
        const char channelName[],
        float64 *state,
        int32 channelType,
        const char channelName0[],
        float64 *state0,
        int32 channelType0,
        const char channelName1[],
        float64 *state1,
        int32 channelType1,
        const char channelName2[],
        float64 *state2,
        int32 channelType2,
        const char channelName3[],
        float64 *state3,
        int32 channelType3,
        const char channelName4[],
        float64 *state4,
        int32 channelType4,
        const char channelName5[],
        float64 *state5,
        int32 channelType5,
        const char channelName6[],
        float64 *state6,
        int32 channelType6,
        const char channelName7[],
        float64 *state7,
        int32 channelType7,
        const char channelName8[],
        float64 *state8,
        int32 channelType8,
        const char channelName9[],
        float64 *state9,
        int32 channelType9,
        const char channelName10[],
        float64 *state10,
        int32 channelType10,
        const char channelName11[],
        float64 *state11,
        int32 channelType11,
        const char channelName12[],
        float64 *state12,
        int32 channelType12,
        const char channelName13[],
        float64 *state13,
        int32 channelType13,
        const char channelName14[],
        float64 *state14,
        int32 channelType14,
        const char channelName15[],
        float64 *state15,
        int32 channelType15,
        const char channelName16[],
        float64 *state16,
        int32 channelType16,
        const char channelName17[],
        float64 *state17,
        int32 channelType17,
        const char channelName18[],
        float64 *state18,
        int32 channelType18,
        const char channelName19[],
        float64 *state19,
        int32 channelType19,
        const char channelName20[],
        float64 *state20,
        int32 channelType20,
        const char channelName21[],
        float64 *state21,
        int32 channelType21,
        const char channelName22[],
        float64 *state22,
        int32 channelType22,
        const char channelName23[],
        float64 *state23,
        int32 channelType23,
        const char channelName24[],
        float64 *state24,
        int32 channelType24,
        const char channelName25[],
        float64 *state25,
        int32 channelType25,
        const char channelName26[],
        float64 *state26,
        int32 channelType26,
        const char channelName27[],
        float64 *state27,
        int32 channelType27,
        const char channelName28[],
        float64 *state28,
        int32 channelType28,
        const char channelName29[],
        float64 *state29,
        int32 channelType29,
        const char channelName30[],
        float64 *state30,
        int32 channelType30,
        const char channelName31[],
        float64 *state31,
        int32 channelType31,
        const char channelName32[],
        float64 *state32,
        int32 channelType32,
        const char channelName33[],
        float64 *state33,
        int32 channelType33,
        const char channelName34[],
        float64 *state34,
        int32 channelType34,
        const char channelName35[],
        float64 *state35,
        int32 channelType35,
        const char channelName36[],
        float64 *state36,
        int32 channelType36,
        const char channelName37[],
        float64 *state37,
        int32 channelType37,
        const char channelName38[],
        float64 *state38,
        int32 channelType38,
        const char channelName39[],
        float64 *state39,
        int32 channelType39,
        const char channelName40[],
        float64 *state40,
        int32 channelType40,
        const char channelName41[],
        float64 *state41,
        int32 channelType41,
        const char channelName42[],
        float64 *state42,
        int32 channelType42,
        const char channelName43[],
        float64 *state43,
        int32 channelType43,
        const char channelName44[],
        float64 *state44,
        int32 channelType44,
        const char channelName45[],
        float64 *state45,
        int32 channelType45,
        const char channelName46[],
        float64 *state46,
        int32 channelType46,
        const char channelName47[],
        float64 *state47,
        int32 channelType47,
        const char channelName48[],
        float64 *state48,
        int32 channelType48,
        const char channelName49[],
        float64 *state49,
        int32 channelType49,
        const char channelName50[],
        float64 *state50,
        int32 channelType50,
        const char channelName51[],
        float64 *state51,
        int32 channelType51,
        const char channelName52[],
        float64 *state52,
        int32 channelType52,
        const char channelName53[],
        float64 *state53,
        int32 channelType53,
        const char channelName54[],
        float64 *state54,
        int32 channelType54,
        const char channelName55[],
        float64 *state55,
        int32 channelType55,
        const char channelName56[],
        float64 *state56,
        int32 channelType56,
        const char channelName57[],
        float64 *state57,
        int32 channelType57,
        const char channelName58[],
        float64 *state58,
        int32 channelType58,
        const char channelName59[],
        float64 *state59,
        int32 channelType59,
        const char channelName60[],
        float64 *state60,
        int32 channelType60,
        const char channelName61[],
        float64 *state61,
        int32 channelType61,
        const char channelName62[],
        float64 *state62,
        int32 channelType62,
        const char channelName63[],
        float64 *state63,
        int32 channelType63,
        const char channelName64[],
        float64 *state64,
        int32 channelType64,
        const char channelName65[],
        float64 *state65,
        int32 channelType65,
        const char channelName66[],
        float64 *state66,
        int32 channelType66,
        const char channelName67[],
        float64 *state67,
        int32 channelType67,
        const char channelName68[],
        float64 *state68,
        int32 channelType68,
        const char channelName69[],
        float64 *state69,
        int32 channelType69,
        const char channelName70[],
        float64 *state70,
        int32 channelType70,
        const char channelName71[],
        float64 *state71,
        int32 channelType71,
        const char channelName72[],
        float64 *state72,
        int32 channelType72,
        const char channelName73[],
        float64 *state73,
        int32 channelType73,
        const char channelName74[],
        float64 *state74,
        int32 channelType74,
        const char channelName75[],
        float64 *state75,
        int32 channelType75,
        const char channelName76[],
        float64 *state76,
        int32 channelType76,
        const char channelName77[],
        float64 *state77,
        int32 channelType77,
        const char channelName78[],
        float64 *state78,
        int32 channelType78,
        const char channelName79[],
        float64 *state79,
        int32 channelType79,
        const char channelName80[],
        float64 *state80,
        int32 channelType80,
        const char channelName81[],
        float64 *state81,
        int32 channelType81,
        const char channelName82[],
        float64 *state82,
        int32 channelType82,
        const char channelName83[],
        float64 *state83,
        int32 channelType83,
        const char channelName84[],
        float64 *state84,
        int32 channelType84,
        const char channelName85[],
        float64 *state85,
        int32 channelType85,
        const char channelName86[],
        float64 *state86,
        int32 channelType86,
        const char channelName87[],
        float64 *state87,
        int32 channelType87,
        const char channelName88[],
        float64 *state88,
        int32 channelType88,
        const char channelName89[],
        float64 *state89,
        int32 channelType89,
        const char channelName90[],
        float64 *state90,
        int32 channelType90,
        const char channelName91[],
        float64 *state91,
        int32 channelType91,
        const char channelName92[],
        float64 *state92,
        int32 channelType92,
        const char channelName93[],
        float64 *state93,
        int32 channelType93,
        const char channelName94[],
        float64 *state94,
        int32 channelType94,
        const char channelName95[],
        float64 *state95,
        int32 channelType95
    );

    int32 GetAnalogPowerUpStatesWithOutputType(
        const char channelNames[],
        float64 stateArray[],
        int32 channelTypeArray[],
        uInt32 *arraySize
    ) override;

    int32 GetArmStartTrigTimestampVal(TaskHandle task, CVIAbsoluteTime *data) override;

    int32 GetArmStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime *data) override;

    int32
    GetAutoConfiguredCDAQSyncConnections(char portList[], uInt32 portListSize) override;

    int32
    GetBufferAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) override;

    int32 GetCalInfoAttributeBool(
        const char deviceName[],
        int32 attribute,
        bool32 *value
    ) override;

    int32 GetCalInfoAttributeDouble(
        const char deviceName[],
        int32 attribute,
        float64 *value
    ) override;

    int32 GetCalInfoAttributeString(
        const char deviceName[],
        int32 attribute,
        char value[],
        uInt32 size
    ) override;

    int32 GetCalInfoAttributeUInt32(
        const char deviceName[],
        int32 attribute,
        uInt32 *value
    ) override;

    int32 GetChanAttributeBool(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        bool32 *value
    ) override;

    int32 GetChanAttributeDouble(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        float64 *value
    ) override;

    int32 GetChanAttributeDoubleArray(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        float64 value[],
        uInt32 size
    ) override;

    int32 GetChanAttributeInt32(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        int32 *value
    ) override;

    int32 GetChanAttributeString(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        char value[],
        uInt32 size
    ) override;

    int32 GetChanAttributeUInt32(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        uInt32 *value
    ) override;

    int32 GetDeviceAttributeBool(
        const char deviceName[],
        int32 attribute,
        bool32 *value
    ) override;

    int32 GetDeviceAttributeDouble(
        const char deviceName[],
        int32 attribute,
        float64 *value
    ) override;

    int32 GetDeviceAttributeDoubleArray(
        const char deviceName[],
        int32 attribute,
        float64 value[],
        uInt32 size
    ) override;

    int32 GetDeviceAttributeInt32(
        const char deviceName[],
        int32 attribute,
        int32 *value
    ) override;

    int32 GetDeviceAttributeInt32Array(
        const char deviceName[],
        int32 attribute,
        int32 value[],
        uInt32 size
    ) override;

    int32 GetDeviceAttributeString(
        const char deviceName[],
        int32 attribute,
        char value[],
        uInt32 size
    ) override;

    int32 GetDeviceAttributeUInt32(
        const char deviceName[],
        int32 attribute,
        uInt32 *value
    ) override;

    int32 GetDeviceAttributeUInt32Array(
        const char deviceName[],
        int32 attribute,
        uInt32 value[],
        uInt32 size
    ) override;

    int32 GetDigitalLogicFamilyPowerUpState(
        const char deviceName[],
        int32 *logicFamily
    ) override;

    int32 GetDigitalPowerUpStates(
        const char deviceName[],
        const char channelName[],
        int32 *state,
        const char channelName0[],
        int32 *state0,
        const char channelName1[],
        int32 *state1,
        const char channelName2[],
        int32 *state2,
        const char channelName3[],
        int32 *state3,
        const char channelName4[],
        int32 *state4,
        const char channelName5[],
        int32 *state5,
        const char channelName6[],
        int32 *state6,
        const char channelName7[],
        int32 *state7,
        const char channelName8[],
        int32 *state8,
        const char channelName9[],
        int32 *state9,
        const char channelName10[],
        int32 *state10,
        const char channelName11[],
        int32 *state11,
        const char channelName12[],
        int32 *state12,
        const char channelName13[],
        int32 *state13,
        const char channelName14[],
        int32 *state14,
        const char channelName15[],
        int32 *state15,
        const char channelName16[],
        int32 *state16,
        const char channelName17[],
        int32 *state17,
        const char channelName18[],
        int32 *state18,
        const char channelName19[],
        int32 *state19,
        const char channelName20[],
        int32 *state20,
        const char channelName21[],
        int32 *state21,
        const char channelName22[],
        int32 *state22,
        const char channelName23[],
        int32 *state23,
        const char channelName24[],
        int32 *state24,
        const char channelName25[],
        int32 *state25,
        const char channelName26[],
        int32 *state26,
        const char channelName27[],
        int32 *state27,
        const char channelName28[],
        int32 *state28,
        const char channelName29[],
        int32 *state29,
        const char channelName30[],
        int32 *state30,
        const char channelName31[],
        int32 *state31,
        const char channelName32[],
        int32 *state32,
        const char channelName33[],
        int32 *state33,
        const char channelName34[],
        int32 *state34,
        const char channelName35[],
        int32 *state35,
        const char channelName36[],
        int32 *state36,
        const char channelName37[],
        int32 *state37,
        const char channelName38[],
        int32 *state38,
        const char channelName39[],
        int32 *state39,
        const char channelName40[],
        int32 *state40,
        const char channelName41[],
        int32 *state41,
        const char channelName42[],
        int32 *state42,
        const char channelName43[],
        int32 *state43,
        const char channelName44[],
        int32 *state44,
        const char channelName45[],
        int32 *state45,
        const char channelName46[],
        int32 *state46,
        const char channelName47[],
        int32 *state47,
        const char channelName48[],
        int32 *state48,
        const char channelName49[],
        int32 *state49,
        const char channelName50[],
        int32 *state50,
        const char channelName51[],
        int32 *state51,
        const char channelName52[],
        int32 *state52,
        const char channelName53[],
        int32 *state53,
        const char channelName54[],
        int32 *state54,
        const char channelName55[],
        int32 *state55,
        const char channelName56[],
        int32 *state56,
        const char channelName57[],
        int32 *state57,
        const char channelName58[],
        int32 *state58,
        const char channelName59[],
        int32 *state59,
        const char channelName60[],
        int32 *state60,
        const char channelName61[],
        int32 *state61,
        const char channelName62[],
        int32 *state62,
        const char channelName63[],
        int32 *state63,
        const char channelName64[],
        int32 *state64,
        const char channelName65[],
        int32 *state65,
        const char channelName66[],
        int32 *state66,
        const char channelName67[],
        int32 *state67,
        const char channelName68[],
        int32 *state68,
        const char channelName69[],
        int32 *state69,
        const char channelName70[],
        int32 *state70,
        const char channelName71[],
        int32 *state71,
        const char channelName72[],
        int32 *state72,
        const char channelName73[],
        int32 *state73,
        const char channelName74[],
        int32 *state74,
        const char channelName75[],
        int32 *state75,
        const char channelName76[],
        int32 *state76,
        const char channelName77[],
        int32 *state77,
        const char channelName78[],
        int32 *state78,
        const char channelName79[],
        int32 *state79,
        const char channelName80[],
        int32 *state80,
        const char channelName81[],
        int32 *state81,
        const char channelName82[],
        int32 *state82,
        const char channelName83[],
        int32 *state83,
        const char channelName84[],
        int32 *state84,
        const char channelName85[],
        int32 *state85,
        const char channelName86[],
        int32 *state86,
        const char channelName87[],
        int32 *state87,
        const char channelName88[],
        int32 *state88,
        const char channelName89[],
        int32 *state89,
        const char channelName90[],
        int32 *state90,
        const char channelName91[],
        int32 *state91,
        const char channelName92[],
        int32 *state92,
        const char channelName93[],
        int32 *state93,
        const char channelName94[],
        int32 *state94,
        const char channelName95[],
        int32 *state95
    );

    int32 GetDigitalPullUpPullDownStates(
        const char deviceName[],
        const char channelName[],
        int32 *state,
        const char channelName0[],
        int32 *state0,
        const char channelName1[],
        int32 *state1,
        const char channelName2[],
        int32 *state2,
        const char channelName3[],
        int32 *state3,
        const char channelName4[],
        int32 *state4,
        const char channelName5[],
        int32 *state5,
        const char channelName6[],
        int32 *state6,
        const char channelName7[],
        int32 *state7,
        const char channelName8[],
        int32 *state8,
        const char channelName9[],
        int32 *state9,
        const char channelName10[],
        int32 *state10,
        const char channelName11[],
        int32 *state11,
        const char channelName12[],
        int32 *state12,
        const char channelName13[],
        int32 *state13,
        const char channelName14[],
        int32 *state14,
        const char channelName15[],
        int32 *state15,
        const char channelName16[],
        int32 *state16,
        const char channelName17[],
        int32 *state17,
        const char channelName18[],
        int32 *state18,
        const char channelName19[],
        int32 *state19,
        const char channelName20[],
        int32 *state20,
        const char channelName21[],
        int32 *state21,
        const char channelName22[],
        int32 *state22,
        const char channelName23[],
        int32 *state23,
        const char channelName24[],
        int32 *state24,
        const char channelName25[],
        int32 *state25,
        const char channelName26[],
        int32 *state26,
        const char channelName27[],
        int32 *state27,
        const char channelName28[],
        int32 *state28,
        const char channelName29[],
        int32 *state29,
        const char channelName30[],
        int32 *state30,
        const char channelName31[],
        int32 *state31,
        const char channelName32[],
        int32 *state32,
        const char channelName33[],
        int32 *state33,
        const char channelName34[],
        int32 *state34,
        const char channelName35[],
        int32 *state35,
        const char channelName36[],
        int32 *state36,
        const char channelName37[],
        int32 *state37,
        const char channelName38[],
        int32 *state38,
        const char channelName39[],
        int32 *state39,
        const char channelName40[],
        int32 *state40,
        const char channelName41[],
        int32 *state41,
        const char channelName42[],
        int32 *state42,
        const char channelName43[],
        int32 *state43,
        const char channelName44[],
        int32 *state44,
        const char channelName45[],
        int32 *state45,
        const char channelName46[],
        int32 *state46,
        const char channelName47[],
        int32 *state47,
        const char channelName48[],
        int32 *state48,
        const char channelName49[],
        int32 *state49,
        const char channelName50[],
        int32 *state50,
        const char channelName51[],
        int32 *state51,
        const char channelName52[],
        int32 *state52,
        const char channelName53[],
        int32 *state53,
        const char channelName54[],
        int32 *state54,
        const char channelName55[],
        int32 *state55,
        const char channelName56[],
        int32 *state56,
        const char channelName57[],
        int32 *state57,
        const char channelName58[],
        int32 *state58,
        const char channelName59[],
        int32 *state59,
        const char channelName60[],
        int32 *state60,
        const char channelName61[],
        int32 *state61,
        const char channelName62[],
        int32 *state62,
        const char channelName63[],
        int32 *state63,
        const char channelName64[],
        int32 *state64,
        const char channelName65[],
        int32 *state65,
        const char channelName66[],
        int32 *state66,
        const char channelName67[],
        int32 *state67,
        const char channelName68[],
        int32 *state68,
        const char channelName69[],
        int32 *state69,
        const char channelName70[],
        int32 *state70,
        const char channelName71[],
        int32 *state71,
        const char channelName72[],
        int32 *state72,
        const char channelName73[],
        int32 *state73,
        const char channelName74[],
        int32 *state74,
        const char channelName75[],
        int32 *state75,
        const char channelName76[],
        int32 *state76,
        const char channelName77[],
        int32 *state77,
        const char channelName78[],
        int32 *state78,
        const char channelName79[],
        int32 *state79,
        const char channelName80[],
        int32 *state80,
        const char channelName81[],
        int32 *state81,
        const char channelName82[],
        int32 *state82,
        const char channelName83[],
        int32 *state83,
        const char channelName84[],
        int32 *state84,
        const char channelName85[],
        int32 *state85,
        const char channelName86[],
        int32 *state86,
        const char channelName87[],
        int32 *state87,
        const char channelName88[],
        int32 *state88,
        const char channelName89[],
        int32 *state89,
        const char channelName90[],
        int32 *state90,
        const char channelName91[],
        int32 *state91,
        const char channelName92[],
        int32 *state92,
        const char channelName93[],
        int32 *state93,
        const char channelName94[],
        int32 *state94,
        const char channelName95[],
        int32 *state95
    );

    int32 GetDisconnectedCDAQSyncPorts(char portList[], uInt32 portListSize) override;

    int32
    GetErrorString(int32 errorCode, char errorString[], uInt32 bufferSize) override;

    int32 GetExportedSignalAttributeBool(
        TaskHandle task,
        int32 attribute,
        bool32 *value
    ) override;

    int32 GetExportedSignalAttributeDouble(
        TaskHandle task,
        int32 attribute,
        float64 *value
    ) override;

    int32 GetExportedSignalAttributeInt32(
        TaskHandle task,
        int32 attribute,
        int32 *value
    ) override;

    int32 GetExportedSignalAttributeString(
        TaskHandle task,
        int32 attribute,
        char value[],
        uInt32 size
    ) override;

    int32 GetExportedSignalAttributeUInt32(
        TaskHandle task,
        int32 attribute,
        uInt32 *value
    ) override;

    int32 GetExtCalLastDateAndTime(
        const char deviceName[],
        uInt32 *year,
        uInt32 *month,
        uInt32 *day,
        uInt32 *hour,
        uInt32 *minute
    ) override;

    int32 GetExtendedErrorInfo(char errorString[], uInt32 bufferSize) override;

    int32 GetFirstSampClkWhen(TaskHandle task, CVIAbsoluteTime *data) override;

    int32 GetFirstSampTimestampVal(TaskHandle task, CVIAbsoluteTime *data) override;

    int32 GetNthTaskChannel(
        TaskHandle task,
        uInt32 index,
        char buffer[],
        int32 bufferSize
    ) override;

    int32 GetNthTaskDevice(
        TaskHandle task,
        uInt32 index,
        char buffer[],
        int32 bufferSize
    ) override;

    int32 GetNthTaskReadChannel(
        TaskHandle task,
        uInt32 index,
        char buffer[],
        int32 bufferSize
    ) override;

    int32 GetPersistedChanAttributeBool(
        const char channel[],
        int32 attribute,
        bool32 *value
    ) override;

    int32 GetPersistedChanAttributeString(
        const char channel[],
        int32 attribute,
        char value[],
        uInt32 size
    ) override;

    int32 GetPersistedScaleAttributeBool(
        const char scaleName[],
        int32 attribute,
        bool32 *value
    ) override;

    int32 GetPersistedScaleAttributeString(
        const char scaleName[],
        int32 attribute,
        char value[],
        uInt32 size
    ) override;

    int32 GetPersistedTaskAttributeBool(
        const char taskName[],
        int32 attribute,
        bool32 *value
    ) override;

    int32 GetPersistedTaskAttributeString(
        const char taskName[],
        int32 attribute,
        char value[],
        uInt32 size
    ) override;

    int32 GetPhysicalChanAttributeBool(
        const char physicalChannel[],
        int32 attribute,
        bool32 *value
    ) override;

    int32 GetPhysicalChanAttributeBytes(
        const char physicalChannel[],
        int32 attribute,
        uInt8 value[],
        uInt32 size
    ) override;

    int32 GetPhysicalChanAttributeDouble(
        const char physicalChannel[],
        int32 attribute,
        float64 *value
    ) override;

    int32 GetPhysicalChanAttributeDoubleArray(
        const char physicalChannel[],
        int32 attribute,
        float64 value[],
        uInt32 size
    ) override;

    int32 GetPhysicalChanAttributeInt32(
        const char physicalChannel[],
        int32 attribute,
        int32 *value
    ) override;

    int32 GetPhysicalChanAttributeInt32Array(
        const char physicalChannel[],
        int32 attribute,
        int32 value[],
        uInt32 size
    ) override;

    int32 GetPhysicalChanAttributeString(
        const char physicalChannel[],
        int32 attribute,
        char value[],
        uInt32 size
    ) override;

    int32 GetPhysicalChanAttributeUInt32(
        const char physicalChannel[],
        int32 attribute,
        uInt32 *value
    ) override;

    int32 GetPhysicalChanAttributeUInt32Array(
        const char physicalChannel[],
        int32 attribute,
        uInt32 value[],
        uInt32 size
    ) override;

    int32
    GetReadAttributeBool(TaskHandle task, int32 attribute, bool32 *value) override;

    int32
    GetReadAttributeDouble(TaskHandle task, int32 attribute, float64 *value) override;

    int32
    GetReadAttributeInt32(TaskHandle task, int32 attribute, int32 *value) override;

    int32 GetReadAttributeString(
        TaskHandle task,
        int32 attribute,
        char value[],
        uInt32 size
    ) override;

    int32
    GetReadAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) override;

    int32
    GetReadAttributeUInt64(TaskHandle task, int32 attribute, uInt64 *value) override;

    int32
    GetRealTimeAttributeBool(TaskHandle task, int32 attribute, bool32 *value) override;

    int32
    GetRealTimeAttributeInt32(TaskHandle task, int32 attribute, int32 *value) override;

    int32 GetRealTimeAttributeUInt32(
        TaskHandle task,
        int32 attribute,
        uInt32 *value
    ) override;

    int32 GetRefTrigTimestampVal(TaskHandle task, CVIAbsoluteTime *data) override;

    int32 GetScaleAttributeDouble(
        const char scaleName[],
        int32 attribute,
        float64 *value
    ) override;

    int32 GetScaleAttributeDoubleArray(
        const char scaleName[],
        int32 attribute,
        float64 value[],
        uInt32 size
    ) override;

    int32 GetScaleAttributeInt32(
        const char scaleName[],
        int32 attribute,
        int32 *value
    ) override;

    int32 GetScaleAttributeString(
        const char scaleName[],
        int32 attribute,
        char value[],
        uInt32 size
    ) override;

    int32 GetSelfCalLastDateAndTime(
        const char deviceName[],
        uInt32 *year,
        uInt32 *month,
        uInt32 *day,
        uInt32 *hour,
        uInt32 *minute
    ) override;

    int32 GetStartTrigTimestampVal(TaskHandle task, CVIAbsoluteTime *data) override;

    int32 GetStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime *data) override;

    int32 GetSyncPulseTimeWhen(TaskHandle task, CVIAbsoluteTime *data) override;

    int32
    GetSystemInfoAttributeString(int32 attribute, char value[], uInt32 size) override;

    int32 GetSystemInfoAttributeUInt32(int32 attribute, uInt32 *value) override;

    int32
    GetTaskAttributeBool(TaskHandle task, int32 attribute, bool32 *value) override;

    int32 GetTaskAttributeString(
        TaskHandle task,
        int32 attribute,
        char value[],
        uInt32 size
    ) override;

    int32
    GetTaskAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) override;

    int32
    GetTimingAttributeBool(TaskHandle task, int32 attribute, bool32 *value) override;

    int32
    GetTimingAttributeDouble(TaskHandle task, int32 attribute, float64 *value) override;

    int32 GetTimingAttributeExBool(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        bool32 *value
    ) override;

    int32 GetTimingAttributeExDouble(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        float64 *value
    ) override;

    int32 GetTimingAttributeExInt32(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        int32 *value
    ) override;

    int32 GetTimingAttributeExString(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        char value[],
        uInt32 size
    ) override;

    int32 GetTimingAttributeExTimestamp(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        CVIAbsoluteTime *value
    ) override;

    int32 GetTimingAttributeExUInt32(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        uInt32 *value
    ) override;

    int32 GetTimingAttributeExUInt64(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        uInt64 *value
    ) override;

    int32
    GetTimingAttributeInt32(TaskHandle task, int32 attribute, int32 *value) override;

    int32 GetTimingAttributeString(
        TaskHandle task,
        int32 attribute,
        char value[],
        uInt32 size
    ) override;

    int32 GetTimingAttributeTimestamp(
        TaskHandle task,
        int32 attribute,
        CVIAbsoluteTime *value
    ) override;

    int32
    GetTimingAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) override;

    int32
    GetTimingAttributeUInt64(TaskHandle task, int32 attribute, uInt64 *value) override;

    int32
    GetTrigAttributeBool(TaskHandle task, int32 attribute, bool32 *value) override;

    int32
    GetTrigAttributeDouble(TaskHandle task, int32 attribute, float64 *value) override;

    int32 GetTrigAttributeDoubleArray(
        TaskHandle task,
        int32 attribute,
        float64 value[],
        uInt32 size
    ) override;

    int32
    GetTrigAttributeInt32(TaskHandle task, int32 attribute, int32 *value) override;

    int32 GetTrigAttributeInt32Array(
        TaskHandle task,
        int32 attribute,
        int32 value[],
        uInt32 size
    ) override;

    int32 GetTrigAttributeString(
        TaskHandle task,
        int32 attribute,
        char value[],
        uInt32 size
    ) override;

    int32 GetTrigAttributeTimestamp(
        TaskHandle task,
        int32 attribute,
        CVIAbsoluteTime *value
    ) override;

    int32
    GetTrigAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) override;

    int32 GetWatchdogAttributeBool(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        bool32 *value
    ) override;

    int32 GetWatchdogAttributeDouble(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        float64 *value
    ) override;

    int32 GetWatchdogAttributeInt32(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        int32 *value
    ) override;

    int32 GetWatchdogAttributeString(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        char value[],
        uInt32 size
    ) override;

    int32
    GetWriteAttributeBool(TaskHandle task, int32 attribute, bool32 *value) override;

    int32
    GetWriteAttributeDouble(TaskHandle task, int32 attribute, float64 *value) override;

    int32
    GetWriteAttributeInt32(TaskHandle task, int32 attribute, int32 *value) override;

    int32 GetWriteAttributeString(
        TaskHandle task,
        int32 attribute,
        char value[],
        uInt32 size
    ) override;

    int32
    GetWriteAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) override;

    int32
    GetWriteAttributeUInt64(TaskHandle task, int32 attribute, uInt64 *value) override;

    int32 IsTaskDone(TaskHandle task, bool32 *isTaskDone) override;

    int32 LoadTask(const char sessionName[], TaskHandle *task) override;

    int32 PerformBridgeOffsetNullingCalEx(
        TaskHandle task,
        const char channel[],
        bool32 skipUnsupportedChannels
    ) override;

    int32 PerformBridgeShuntCalEx(
        TaskHandle task,
        const char channel[],
        float64 shuntResistorValue,
        int32 shuntResistorLocation,
        int32 shuntResistorSelect,
        int32 shuntResistorSource,
        float64 bridgeResistance,
        bool32 skipUnsupportedChannels
    ) override;

    int32 PerformStrainShuntCalEx(
        TaskHandle task,
        const char channel[],
        float64 shuntResistorValue,
        int32 shuntResistorLocation,
        int32 shuntResistorSelect,
        int32 shuntResistorSource,
        bool32 skipUnsupportedChannels
    ) override;

    int32 PerformThrmcplLeadOffsetNullingCal(
        TaskHandle task,
        const char channel[],
        bool32 skipUnsupportedChannels
    ) override;

    int32 ReadAnalogF64(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        float64 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override;

    int32 ReadAnalogScalarF64(
        TaskHandle task,
        float64 timeout,
        float64 *value,
        bool32 *reserved
    ) override;

    int32 ReadBinaryI16(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        int16 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override;

    int32 ReadBinaryI32(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        int32 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override;

    int32 ReadBinaryU16(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt16 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override;

    int32 ReadBinaryU32(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt32 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override;

    int32 ReadCounterF64(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        float64 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override;

    int32 ReadCounterF64Ex(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        float64 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override;

    int32 ReadCounterScalarF64(
        TaskHandle task,
        float64 timeout,
        float64 *value,
        bool32 *reserved
    ) override;

    int32 ReadCounterScalarU32(
        TaskHandle task,
        float64 timeout,
        uInt32 *value,
        bool32 *reserved
    ) override;

    int32 ReadCounterU32(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        uInt32 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override;

    int32 ReadCounterU32Ex(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt32 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override;

    int32 ReadCtrFreq(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 interleaved,
        float64 readArrayFrequency[],
        float64 readArrayDutyCycle[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override;

    int32 ReadCtrFreqScalar(
        TaskHandle task,
        float64 timeout,
        float64 *frequency,
        float64 *dutyCycle,
        bool32 *reserved
    ) override;

    int32 ReadCtrTicks(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 interleaved,
        uInt32 readArrayHighTicks[],
        uInt32 readArrayLowTicks[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override;

    int32 ReadCtrTicksScalar(
        TaskHandle task,
        float64 timeout,
        uInt32 *highTicks,
        uInt32 *lowTicks,
        bool32 *reserved
    ) override;

    int32 ReadCtrTime(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 interleaved,
        float64 readArrayHighTime[],
        float64 readArrayLowTime[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override;

    int32 ReadCtrTimeScalar(
        TaskHandle task,
        float64 timeout,
        float64 *highTime,
        float64 *lowTime,
        bool32 *reserved
    ) override;

    int32 ReadDigitalLines(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt8 readArray[],
        uInt32 arraySizeInBytes,
        int32 *sampsPerChanRead,
        int32 *numBytesPerSamp,
        bool32 *reserved
    ) override;

    int32 ReadDigitalScalarU32(
        TaskHandle task,
        float64 timeout,
        uInt32 *value,
        bool32 *reserved
    ) override;

    int32 ReadDigitalU16(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt16 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override;

    int32 ReadDigitalU32(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt32 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override;

    int32 ReadDigitalU8(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt8 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override;

    int32 ReadPowerBinaryI16(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        int16 readArrayVoltage[],
        int16 readArrayCurrent[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override;

    int32 ReadPowerF64(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        float64 readArrayVoltage[],
        float64 readArrayCurrent[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    ) override;

    int32 ReadPowerScalarF64(
        TaskHandle task,
        float64 timeout,
        float64 *voltage,
        float64 *current,
        bool32 *reserved
    ) override;

    int32 ReadRaw(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        uInt8 readArray[],
        uInt32 arraySizeInBytes,
        int32 *sampsRead,
        int32 *numBytesPerSamp,
        bool32 *reserved
    ) override;

    int32 RegisterDoneEvent(
        TaskHandle task,
        uInt32 options,
        DAQmxDoneEventCallbackPtr callbackFunction,
        void *callbackData
    ) override;

    int32 RegisterEveryNSamplesEvent(
        TaskHandle task,
        int32 everyNSamplesEventType,
        uInt32 nSamples,
        uInt32 options,
        DAQmxEveryNSamplesEventCallbackPtr callbackFunction,
        void *callbackData
    ) override;

    int32 RegisterSignalEvent(
        TaskHandle task,
        int32 signalID,
        uInt32 options,
        DAQmxSignalEventCallbackPtr callbackFunction,
        void *callbackData
    ) override;

    int32 RemoveCDAQSyncConnection(const char portList[]) override;

    int32
    ReserveNetworkDevice(const char deviceName[], bool32 overrideReservation) override;

    int32 ResetBufferAttribute(TaskHandle task, int32 attribute) override;

    int32
    ResetChanAttribute(TaskHandle task, const char channel[], int32 attribute) override;

    int32 ResetDevice(const char deviceName[]) override;

    int32 ResetExportedSignalAttribute(TaskHandle task, int32 attribute) override;

    int32 ResetReadAttribute(TaskHandle task, int32 attribute) override;

    int32 ResetRealTimeAttribute(TaskHandle task, int32 attribute) override;

    int32 ResetTimingAttribute(TaskHandle task, int32 attribute) override;

    int32 ResetTimingAttributeEx(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute
    ) override;

    int32 ResetTrigAttribute(TaskHandle task, int32 attribute) override;

    int32 ResetWatchdogAttribute(
        TaskHandle task,
        const char lines[],
        int32 attribute
    ) override;

    int32 ResetWriteAttribute(TaskHandle task, int32 attribute) override;

    int32 RestoreLastExtCalConst(const char deviceName[]) override;

    int32 SaveGlobalChan(
        TaskHandle task,
        const char channelName[],
        const char saveAs[],
        const char author[],
        uInt32 options
    ) override;

    int32 SaveScale(
        const char scaleName[],
        const char saveAs[],
        const char author[],
        uInt32 options
    ) override;

    int32 SaveTask(
        TaskHandle task,
        const char saveAs[],
        const char author[],
        uInt32 options
    ) override;

    int32 SelfCal(const char deviceName[]) override;

    int32 SelfTestDevice(const char deviceName[]) override;

    int32 SetAIChanCalCalDate(
        TaskHandle task,
        const char channelName[],
        uInt32 year,
        uInt32 month,
        uInt32 day,
        uInt32 hour,
        uInt32 minute
    ) override;

    int32 SetAIChanCalExpDate(
        TaskHandle task,
        const char channelName[],
        uInt32 year,
        uInt32 month,
        uInt32 day,
        uInt32 hour,
        uInt32 minute
    ) override;

    int32 SetAnalogPowerUpStates(
        const char deviceName[],
        const char channelNames[],
        float64 state,
        int32 channelType,
        const char channelNames0[],
        float64 state0,
        int32 channelType0,
        const char channelNames1[],
        float64 state1,
        int32 channelType1,
        const char channelNames2[],
        float64 state2,
        int32 channelType2,
        const char channelNames3[],
        float64 state3,
        int32 channelType3,
        const char channelNames4[],
        float64 state4,
        int32 channelType4,
        const char channelNames5[],
        float64 state5,
        int32 channelType5,
        const char channelNames6[],
        float64 state6,
        int32 channelType6,
        const char channelNames7[],
        float64 state7,
        int32 channelType7,
        const char channelNames8[],
        float64 state8,
        int32 channelType8,
        const char channelNames9[],
        float64 state9,
        int32 channelType9,
        const char channelNames10[],
        float64 state10,
        int32 channelType10,
        const char channelNames11[],
        float64 state11,
        int32 channelType11,
        const char channelNames12[],
        float64 state12,
        int32 channelType12,
        const char channelNames13[],
        float64 state13,
        int32 channelType13,
        const char channelNames14[],
        float64 state14,
        int32 channelType14,
        const char channelNames15[],
        float64 state15,
        int32 channelType15,
        const char channelNames16[],
        float64 state16,
        int32 channelType16,
        const char channelNames17[],
        float64 state17,
        int32 channelType17,
        const char channelNames18[],
        float64 state18,
        int32 channelType18,
        const char channelNames19[],
        float64 state19,
        int32 channelType19,
        const char channelNames20[],
        float64 state20,
        int32 channelType20,
        const char channelNames21[],
        float64 state21,
        int32 channelType21,
        const char channelNames22[],
        float64 state22,
        int32 channelType22,
        const char channelNames23[],
        float64 state23,
        int32 channelType23,
        const char channelNames24[],
        float64 state24,
        int32 channelType24,
        const char channelNames25[],
        float64 state25,
        int32 channelType25,
        const char channelNames26[],
        float64 state26,
        int32 channelType26,
        const char channelNames27[],
        float64 state27,
        int32 channelType27,
        const char channelNames28[],
        float64 state28,
        int32 channelType28,
        const char channelNames29[],
        float64 state29,
        int32 channelType29,
        const char channelNames30[],
        float64 state30,
        int32 channelType30,
        const char channelNames31[],
        float64 state31,
        int32 channelType31,
        const char channelNames32[],
        float64 state32,
        int32 channelType32,
        const char channelNames33[],
        float64 state33,
        int32 channelType33,
        const char channelNames34[],
        float64 state34,
        int32 channelType34,
        const char channelNames35[],
        float64 state35,
        int32 channelType35,
        const char channelNames36[],
        float64 state36,
        int32 channelType36,
        const char channelNames37[],
        float64 state37,
        int32 channelType37,
        const char channelNames38[],
        float64 state38,
        int32 channelType38,
        const char channelNames39[],
        float64 state39,
        int32 channelType39,
        const char channelNames40[],
        float64 state40,
        int32 channelType40,
        const char channelNames41[],
        float64 state41,
        int32 channelType41,
        const char channelNames42[],
        float64 state42,
        int32 channelType42,
        const char channelNames43[],
        float64 state43,
        int32 channelType43,
        const char channelNames44[],
        float64 state44,
        int32 channelType44,
        const char channelNames45[],
        float64 state45,
        int32 channelType45,
        const char channelNames46[],
        float64 state46,
        int32 channelType46,
        const char channelNames47[],
        float64 state47,
        int32 channelType47,
        const char channelNames48[],
        float64 state48,
        int32 channelType48,
        const char channelNames49[],
        float64 state49,
        int32 channelType49,
        const char channelNames50[],
        float64 state50,
        int32 channelType50,
        const char channelNames51[],
        float64 state51,
        int32 channelType51,
        const char channelNames52[],
        float64 state52,
        int32 channelType52,
        const char channelNames53[],
        float64 state53,
        int32 channelType53,
        const char channelNames54[],
        float64 state54,
        int32 channelType54,
        const char channelNames55[],
        float64 state55,
        int32 channelType55,
        const char channelNames56[],
        float64 state56,
        int32 channelType56,
        const char channelNames57[],
        float64 state57,
        int32 channelType57,
        const char channelNames58[],
        float64 state58,
        int32 channelType58,
        const char channelNames59[],
        float64 state59,
        int32 channelType59,
        const char channelNames60[],
        float64 state60,
        int32 channelType60,
        const char channelNames61[],
        float64 state61,
        int32 channelType61,
        const char channelNames62[],
        float64 state62,
        int32 channelType62,
        const char channelNames63[],
        float64 state63,
        int32 channelType63,
        const char channelNames64[],
        float64 state64,
        int32 channelType64,
        const char channelNames65[],
        float64 state65,
        int32 channelType65,
        const char channelNames66[],
        float64 state66,
        int32 channelType66,
        const char channelNames67[],
        float64 state67,
        int32 channelType67,
        const char channelNames68[],
        float64 state68,
        int32 channelType68,
        const char channelNames69[],
        float64 state69,
        int32 channelType69,
        const char channelNames70[],
        float64 state70,
        int32 channelType70,
        const char channelNames71[],
        float64 state71,
        int32 channelType71,
        const char channelNames72[],
        float64 state72,
        int32 channelType72,
        const char channelNames73[],
        float64 state73,
        int32 channelType73,
        const char channelNames74[],
        float64 state74,
        int32 channelType74,
        const char channelNames75[],
        float64 state75,
        int32 channelType75,
        const char channelNames76[],
        float64 state76,
        int32 channelType76,
        const char channelNames77[],
        float64 state77,
        int32 channelType77,
        const char channelNames78[],
        float64 state78,
        int32 channelType78,
        const char channelNames79[],
        float64 state79,
        int32 channelType79,
        const char channelNames80[],
        float64 state80,
        int32 channelType80,
        const char channelNames81[],
        float64 state81,
        int32 channelType81,
        const char channelNames82[],
        float64 state82,
        int32 channelType82,
        const char channelNames83[],
        float64 state83,
        int32 channelType83,
        const char channelNames84[],
        float64 state84,
        int32 channelType84,
        const char channelNames85[],
        float64 state85,
        int32 channelType85,
        const char channelNames86[],
        float64 state86,
        int32 channelType86,
        const char channelNames87[],
        float64 state87,
        int32 channelType87,
        const char channelNames88[],
        float64 state88,
        int32 channelType88,
        const char channelNames89[],
        float64 state89,
        int32 channelType89,
        const char channelNames90[],
        float64 state90,
        int32 channelType90,
        const char channelNames91[],
        float64 state91,
        int32 channelType91,
        const char channelNames92[],
        float64 state92,
        int32 channelType92,
        const char channelNames93[],
        float64 state93,
        int32 channelType93,
        const char channelNames94[],
        float64 state94,
        int32 channelType94,
        const char channelNames95[],
        float64 state95,
        int32 channelType95
    );

    int32 SetAnalogPowerUpStatesWithOutputType(
        const char channelNames[],
        const float64 stateArray[],
        const int32 channelTypeArray[],
        uInt32 arraySize
    ) override;

    int32 SetArmStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime data) override;

    int32
    SetBufferAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) override;

    int32 SetCalInfoAttributeBool(
        const char deviceName[],
        int32 attribute,
        bool32 value
    ) override;

    int32 SetCalInfoAttributeDouble(
        const char deviceName[],
        int32 attribute,
        float64 value
    ) override;

    int32 SetCalInfoAttributeString(
        const char deviceName[],
        int32 attribute,
        const char value[]
    ) override;

    int32 SetCalInfoAttributeUInt32(
        const char deviceName[],
        int32 attribute,
        uInt32 value
    ) override;

    int32 SetChanAttributeBool(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        bool32 value
    ) override;

    int32 SetChanAttributeDouble(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        float64 value
    ) override;

    int32 SetChanAttributeDoubleArray(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        const float64 value[],
        uInt32 size
    ) override;

    int32 SetChanAttributeInt32(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        int32 value
    ) override;

    int32 SetChanAttributeString(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        const char value[]
    ) override;

    int32 SetChanAttributeUInt32(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        uInt32 value
    ) override;

    int32 SetDigitalLogicFamilyPowerUpState(
        const char deviceName[],
        int32 logicFamily
    ) override;

    int32 SetDigitalPowerUpStates(
        const char deviceName[],
        const char channelNames[],
        int32 state,
        const char channelNames0[],
        int32 state0,
        const char channelNames1[],
        int32 state1,
        const char channelNames2[],
        int32 state2,
        const char channelNames3[],
        int32 state3,
        const char channelNames4[],
        int32 state4,
        const char channelNames5[],
        int32 state5,
        const char channelNames6[],
        int32 state6,
        const char channelNames7[],
        int32 state7,
        const char channelNames8[],
        int32 state8,
        const char channelNames9[],
        int32 state9,
        const char channelNames10[],
        int32 state10,
        const char channelNames11[],
        int32 state11,
        const char channelNames12[],
        int32 state12,
        const char channelNames13[],
        int32 state13,
        const char channelNames14[],
        int32 state14,
        const char channelNames15[],
        int32 state15,
        const char channelNames16[],
        int32 state16,
        const char channelNames17[],
        int32 state17,
        const char channelNames18[],
        int32 state18,
        const char channelNames19[],
        int32 state19,
        const char channelNames20[],
        int32 state20,
        const char channelNames21[],
        int32 state21,
        const char channelNames22[],
        int32 state22,
        const char channelNames23[],
        int32 state23,
        const char channelNames24[],
        int32 state24,
        const char channelNames25[],
        int32 state25,
        const char channelNames26[],
        int32 state26,
        const char channelNames27[],
        int32 state27,
        const char channelNames28[],
        int32 state28,
        const char channelNames29[],
        int32 state29,
        const char channelNames30[],
        int32 state30,
        const char channelNames31[],
        int32 state31,
        const char channelNames32[],
        int32 state32,
        const char channelNames33[],
        int32 state33,
        const char channelNames34[],
        int32 state34,
        const char channelNames35[],
        int32 state35,
        const char channelNames36[],
        int32 state36,
        const char channelNames37[],
        int32 state37,
        const char channelNames38[],
        int32 state38,
        const char channelNames39[],
        int32 state39,
        const char channelNames40[],
        int32 state40,
        const char channelNames41[],
        int32 state41,
        const char channelNames42[],
        int32 state42,
        const char channelNames43[],
        int32 state43,
        const char channelNames44[],
        int32 state44,
        const char channelNames45[],
        int32 state45,
        const char channelNames46[],
        int32 state46,
        const char channelNames47[],
        int32 state47,
        const char channelNames48[],
        int32 state48,
        const char channelNames49[],
        int32 state49,
        const char channelNames50[],
        int32 state50,
        const char channelNames51[],
        int32 state51,
        const char channelNames52[],
        int32 state52,
        const char channelNames53[],
        int32 state53,
        const char channelNames54[],
        int32 state54,
        const char channelNames55[],
        int32 state55,
        const char channelNames56[],
        int32 state56,
        const char channelNames57[],
        int32 state57,
        const char channelNames58[],
        int32 state58,
        const char channelNames59[],
        int32 state59,
        const char channelNames60[],
        int32 state60,
        const char channelNames61[],
        int32 state61,
        const char channelNames62[],
        int32 state62,
        const char channelNames63[],
        int32 state63,
        const char channelNames64[],
        int32 state64,
        const char channelNames65[],
        int32 state65,
        const char channelNames66[],
        int32 state66,
        const char channelNames67[],
        int32 state67,
        const char channelNames68[],
        int32 state68,
        const char channelNames69[],
        int32 state69,
        const char channelNames70[],
        int32 state70,
        const char channelNames71[],
        int32 state71,
        const char channelNames72[],
        int32 state72,
        const char channelNames73[],
        int32 state73,
        const char channelNames74[],
        int32 state74,
        const char channelNames75[],
        int32 state75,
        const char channelNames76[],
        int32 state76,
        const char channelNames77[],
        int32 state77,
        const char channelNames78[],
        int32 state78,
        const char channelNames79[],
        int32 state79,
        const char channelNames80[],
        int32 state80,
        const char channelNames81[],
        int32 state81,
        const char channelNames82[],
        int32 state82,
        const char channelNames83[],
        int32 state83,
        const char channelNames84[],
        int32 state84,
        const char channelNames85[],
        int32 state85,
        const char channelNames86[],
        int32 state86,
        const char channelNames87[],
        int32 state87,
        const char channelNames88[],
        int32 state88,
        const char channelNames89[],
        int32 state89,
        const char channelNames90[],
        int32 state90,
        const char channelNames91[],
        int32 state91,
        const char channelNames92[],
        int32 state92,
        const char channelNames93[],
        int32 state93,
        const char channelNames94[],
        int32 state94,
        const char channelNames95[],
        int32 state95
    );

    int32 SetDigitalPullUpPullDownStates(
        const char deviceName[],
        const char channelNames[],
        int32 state,
        const char channelNames0[],
        int32 state0,
        const char channelNames1[],
        int32 state1,
        const char channelNames2[],
        int32 state2,
        const char channelNames3[],
        int32 state3,
        const char channelNames4[],
        int32 state4,
        const char channelNames5[],
        int32 state5,
        const char channelNames6[],
        int32 state6,
        const char channelNames7[],
        int32 state7,
        const char channelNames8[],
        int32 state8,
        const char channelNames9[],
        int32 state9,
        const char channelNames10[],
        int32 state10,
        const char channelNames11[],
        int32 state11,
        const char channelNames12[],
        int32 state12,
        const char channelNames13[],
        int32 state13,
        const char channelNames14[],
        int32 state14,
        const char channelNames15[],
        int32 state15,
        const char channelNames16[],
        int32 state16,
        const char channelNames17[],
        int32 state17,
        const char channelNames18[],
        int32 state18,
        const char channelNames19[],
        int32 state19,
        const char channelNames20[],
        int32 state20,
        const char channelNames21[],
        int32 state21,
        const char channelNames22[],
        int32 state22,
        const char channelNames23[],
        int32 state23,
        const char channelNames24[],
        int32 state24,
        const char channelNames25[],
        int32 state25,
        const char channelNames26[],
        int32 state26,
        const char channelNames27[],
        int32 state27,
        const char channelNames28[],
        int32 state28,
        const char channelNames29[],
        int32 state29,
        const char channelNames30[],
        int32 state30,
        const char channelNames31[],
        int32 state31,
        const char channelNames32[],
        int32 state32,
        const char channelNames33[],
        int32 state33,
        const char channelNames34[],
        int32 state34,
        const char channelNames35[],
        int32 state35,
        const char channelNames36[],
        int32 state36,
        const char channelNames37[],
        int32 state37,
        const char channelNames38[],
        int32 state38,
        const char channelNames39[],
        int32 state39,
        const char channelNames40[],
        int32 state40,
        const char channelNames41[],
        int32 state41,
        const char channelNames42[],
        int32 state42,
        const char channelNames43[],
        int32 state43,
        const char channelNames44[],
        int32 state44,
        const char channelNames45[],
        int32 state45,
        const char channelNames46[],
        int32 state46,
        const char channelNames47[],
        int32 state47,
        const char channelNames48[],
        int32 state48,
        const char channelNames49[],
        int32 state49,
        const char channelNames50[],
        int32 state50,
        const char channelNames51[],
        int32 state51,
        const char channelNames52[],
        int32 state52,
        const char channelNames53[],
        int32 state53,
        const char channelNames54[],
        int32 state54,
        const char channelNames55[],
        int32 state55,
        const char channelNames56[],
        int32 state56,
        const char channelNames57[],
        int32 state57,
        const char channelNames58[],
        int32 state58,
        const char channelNames59[],
        int32 state59,
        const char channelNames60[],
        int32 state60,
        const char channelNames61[],
        int32 state61,
        const char channelNames62[],
        int32 state62,
        const char channelNames63[],
        int32 state63,
        const char channelNames64[],
        int32 state64,
        const char channelNames65[],
        int32 state65,
        const char channelNames66[],
        int32 state66,
        const char channelNames67[],
        int32 state67,
        const char channelNames68[],
        int32 state68,
        const char channelNames69[],
        int32 state69,
        const char channelNames70[],
        int32 state70,
        const char channelNames71[],
        int32 state71,
        const char channelNames72[],
        int32 state72,
        const char channelNames73[],
        int32 state73,
        const char channelNames74[],
        int32 state74,
        const char channelNames75[],
        int32 state75,
        const char channelNames76[],
        int32 state76,
        const char channelNames77[],
        int32 state77,
        const char channelNames78[],
        int32 state78,
        const char channelNames79[],
        int32 state79,
        const char channelNames80[],
        int32 state80,
        const char channelNames81[],
        int32 state81,
        const char channelNames82[],
        int32 state82,
        const char channelNames83[],
        int32 state83,
        const char channelNames84[],
        int32 state84,
        const char channelNames85[],
        int32 state85,
        const char channelNames86[],
        int32 state86,
        const char channelNames87[],
        int32 state87,
        const char channelNames88[],
        int32 state88,
        const char channelNames89[],
        int32 state89,
        const char channelNames90[],
        int32 state90,
        const char channelNames91[],
        int32 state91,
        const char channelNames92[],
        int32 state92,
        const char channelNames93[],
        int32 state93,
        const char channelNames94[],
        int32 state94,
        const char channelNames95[],
        int32 state95
    );

    int32 SetExportedSignalAttributeBool(
        TaskHandle task,
        int32 attribute,
        bool32 value
    ) override;

    int32 SetExportedSignalAttributeDouble(
        TaskHandle task,
        int32 attribute,
        float64 value
    ) override;

    int32 SetExportedSignalAttributeInt32(
        TaskHandle task,
        int32 attribute,
        int32 value
    ) override;

    int32 SetExportedSignalAttributeString(
        TaskHandle task,
        int32 attribute,
        const char value[]
    ) override;

    int32 SetExportedSignalAttributeUInt32(
        TaskHandle task,
        int32 attribute,
        uInt32 value
    ) override;

    int32 SetFirstSampClkWhen(TaskHandle task, CVIAbsoluteTime data) override;

    int32 SetReadAttributeBool(TaskHandle task, int32 attribute, bool32 value) override;

    int32
    SetReadAttributeDouble(TaskHandle task, int32 attribute, float64 value) override;

    int32 SetReadAttributeInt32(TaskHandle task, int32 attribute, int32 value) override;

    int32 SetReadAttributeString(
        TaskHandle task,
        int32 attribute,
        const char value[]
    ) override;

    int32
    SetReadAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) override;

    int32
    SetReadAttributeUInt64(TaskHandle task, int32 attribute, uInt64 value) override;

    int32
    SetRealTimeAttributeBool(TaskHandle task, int32 attribute, bool32 value) override;

    int32
    SetRealTimeAttributeInt32(TaskHandle task, int32 attribute, int32 value) override;

    int32
    SetRealTimeAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) override;

    int32 SetRuntimeEnvironment(
        const char environment[],
        const char environmentVersion[],
        const char reserved1[],
        const char reserved2[]
    ) override;

    int32 SetScaleAttributeDouble(
        const char scaleName[],
        int32 attribute,
        float64 value
    ) override;

    int32 SetScaleAttributeDoubleArray(
        const char scaleName[],
        int32 attribute,
        const float64 value[],
        uInt32 size
    ) override;

    int32 SetScaleAttributeInt32(
        const char scaleName[],
        int32 attribute,
        int32 value
    ) override;

    int32 SetScaleAttributeString(
        const char scaleName[],
        int32 attribute,
        const char value[]
    ) override;

    int32 SetStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime data) override;

    int32 SetSyncPulseTimeWhen(TaskHandle task, CVIAbsoluteTime data) override;

    int32
    SetTimingAttributeBool(TaskHandle task, int32 attribute, bool32 value) override;

    int32
    SetTimingAttributeDouble(TaskHandle task, int32 attribute, float64 value) override;

    int32 SetTimingAttributeExBool(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        bool32 value
    ) override;

    int32 SetTimingAttributeExDouble(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        float64 value
    ) override;

    int32 SetTimingAttributeExInt32(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        int32 value
    ) override;

    int32 SetTimingAttributeExString(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        const char value[]
    ) override;

    int32 SetTimingAttributeExTimestamp(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        CVIAbsoluteTime value
    ) override;

    int32 SetTimingAttributeExUInt32(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        uInt32 value
    ) override;

    int32 SetTimingAttributeExUInt64(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        uInt64 value
    ) override;

    int32
    SetTimingAttributeInt32(TaskHandle task, int32 attribute, int32 value) override;

    int32 SetTimingAttributeString(
        TaskHandle task,
        int32 attribute,
        const char value[]
    ) override;

    int32 SetTimingAttributeTimestamp(
        TaskHandle task,
        int32 attribute,
        CVIAbsoluteTime value
    ) override;

    int32
    SetTimingAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) override;

    int32
    SetTimingAttributeUInt64(TaskHandle task, int32 attribute, uInt64 value) override;

    int32 SetTrigAttributeBool(TaskHandle task, int32 attribute, bool32 value) override;

    int32
    SetTrigAttributeDouble(TaskHandle task, int32 attribute, float64 value) override;

    int32 SetTrigAttributeDoubleArray(
        TaskHandle task,
        int32 attribute,
        const float64 value[],
        uInt32 size
    ) override;

    int32 SetTrigAttributeInt32(TaskHandle task, int32 attribute, int32 value) override;

    int32 SetTrigAttributeInt32Array(
        TaskHandle task,
        int32 attribute,
        const int32 value[],
        uInt32 size
    ) override;

    int32 SetTrigAttributeString(
        TaskHandle task,
        int32 attribute,
        const char value[]
    ) override;

    int32 SetTrigAttributeTimestamp(
        TaskHandle task,
        int32 attribute,
        CVIAbsoluteTime value
    ) override;

    int32
    SetTrigAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) override;

    int32 SetWatchdogAttributeBool(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        bool32 value
    ) override;

    int32 SetWatchdogAttributeDouble(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        float64 value
    ) override;

    int32 SetWatchdogAttributeInt32(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        int32 value
    ) override;

    int32 SetWatchdogAttributeString(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        const char value[]
    ) override;

    int32
    SetWriteAttributeBool(TaskHandle task, int32 attribute, bool32 value) override;

    int32
    SetWriteAttributeDouble(TaskHandle task, int32 attribute, float64 value) override;

    int32
    SetWriteAttributeInt32(TaskHandle task, int32 attribute, int32 value) override;

    int32 SetWriteAttributeString(
        TaskHandle task,
        int32 attribute,
        const char value[]
    ) override;

    int32
    SetWriteAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) override;

    int32
    SetWriteAttributeUInt64(TaskHandle task, int32 attribute, uInt64 value) override;

    int32 StartNewFile(TaskHandle task, const char filePath[]) override;

    int32 StartTask(TaskHandle task) override;

    int32 StopTask(TaskHandle task) override;

    int32 TaskControl(TaskHandle task, int32 action) override;

    int32 TristateOutputTerm(const char outputTerminal[]) override;

    int32 UnregisterDoneEvent(
        TaskHandle task,
        uInt32 options,
        DAQmxDoneEventCallbackPtr callbackFunction,
        void *callbackData
    ) override;

    int32 UnregisterEveryNSamplesEvent(
        TaskHandle task,
        int32 everyNSamplesEventType,
        uInt32 nSamples,
        uInt32 options,
        DAQmxEveryNSamplesEventCallbackPtr callbackFunction,
        void *callbackData
    ) override;

    int32 UnregisterSignalEvent(
        TaskHandle task,
        int32 signalID,
        uInt32 options,
        DAQmxSignalEventCallbackPtr callbackFunction,
        void *callbackData
    ) override;

    int32 UnreserveNetworkDevice(const char deviceName[]) override;

    int32
    WaitForNextSampleClock(TaskHandle task, float64 timeout, bool32 *isLate) override;

    int32 WaitForValidTimestamp(
        TaskHandle task,
        int32 timestampEvent,
        float64 timeout,
        CVIAbsoluteTime *timestamp
    ) override;

    int32 WaitUntilTaskDone(TaskHandle task, float64 timeToWait) override;

    int32 WriteAnalogF64(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const float64 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) override;

    int32 WriteAnalogScalarF64(
        TaskHandle task,
        bool32 autoStart,
        float64 timeout,
        float64 value,
        bool32 *reserved
    ) override;

    int32 WriteBinaryI16(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const int16 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) override;

    int32 WriteBinaryI32(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const int32 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) override;

    int32 WriteBinaryU16(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt16 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) override;

    int32 WriteBinaryU32(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt32 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) override;

    int32 WriteCtrFreq(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const float64 frequency[],
        const float64 dutyCycle[],
        int32 *numSampsPerChanWritten,
        bool32 *reserved
    ) override;

    int32 WriteCtrFreqScalar(
        TaskHandle task,
        bool32 autoStart,
        float64 timeout,
        float64 frequency,
        float64 dutyCycle,
        bool32 *reserved
    ) override;

    int32 WriteCtrTicks(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt32 highTicks[],
        const uInt32 lowTicks[],
        int32 *numSampsPerChanWritten,
        bool32 *reserved
    ) override;

    int32 WriteCtrTicksScalar(
        TaskHandle task,
        bool32 autoStart,
        float64 timeout,
        uInt32 highTicks,
        uInt32 lowTicks,
        bool32 *reserved
    ) override;

    int32 WriteCtrTime(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const float64 highTime[],
        const float64 lowTime[],
        int32 *numSampsPerChanWritten,
        bool32 *reserved
    ) override;

    int32 WriteCtrTimeScalar(
        TaskHandle task,
        bool32 autoStart,
        float64 timeout,
        float64 highTime,
        float64 lowTime,
        bool32 *reserved
    ) override;

    int32 WriteDigitalLines(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt8 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) override;

    int32 WriteDigitalScalarU32(
        TaskHandle task,
        bool32 autoStart,
        float64 timeout,
        uInt32 value,
        bool32 *reserved
    ) override;

    int32 WriteDigitalU16(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt16 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) override;

    int32 WriteDigitalU32(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt32 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) override;

    int32 WriteDigitalU8(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt8 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) override;

    int32 WriteRaw(
        TaskHandle task,
        int32 numSamps,
        bool32 autoStart,
        float64 timeout,
        const uInt8 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    ) override;

    int32 WriteToTEDSFromArray(
        const char physicalChannel[],
        const uInt8 bitStream[],
        uInt32 arraySize,
        int32 basicTEDSOptions
    ) override;

    int32 WriteToTEDSFromFile(
        const char physicalChannel[],
        const char filePath[],
        int32 basicTEDSOptions
    ) override;
    int32 SetReadRelativeTo(TaskHandle taskHandle, int32 data) override;
    int32 SetReadOffset(TaskHandle taskHandle, int32 data) override;
    int32 SetReadOverWrite(TaskHandle taskHandle, int32 data) override;
    int32 GetReadTotalSampPerChanAcquired(TaskHandle taskHandle, uInt64 *data) override;

private:
    using AddCDAQSyncConnectionPtr = decltype(&DAQmxAddCDAQSyncConnection);
    using AddGlobalChansToTaskPtr = decltype(&DAQmxAddGlobalChansToTask);
    using AddNetworkDevicePtr = decltype(&DAQmxAddNetworkDevice);
    using AreConfiguredCDAQSyncPortsDisconnectedPtr =
        decltype(&DAQmxAreConfiguredCDAQSyncPortsDisconnected);
    using AutoConfigureCDAQSyncConnectionsPtr =
        decltype(&DAQmxAutoConfigureCDAQSyncConnections);
    using CalculateReversePolyCoeffPtr = decltype(&DAQmxCalculateReversePolyCoeff);
    using CfgAnlgEdgeRefTrigPtr = decltype(&DAQmxCfgAnlgEdgeRefTrig);
    using CfgAnlgEdgeStartTrigPtr = decltype(&DAQmxCfgAnlgEdgeStartTrig);
    using CfgAnlgMultiEdgeRefTrigPtr = decltype(&DAQmxCfgAnlgMultiEdgeRefTrig);
    using CfgAnlgMultiEdgeStartTrigPtr = decltype(&DAQmxCfgAnlgMultiEdgeStartTrig);
    using CfgAnlgWindowRefTrigPtr = decltype(&DAQmxCfgAnlgWindowRefTrig);
    using CfgAnlgWindowStartTrigPtr = decltype(&DAQmxCfgAnlgWindowStartTrig);
    using CfgBurstHandshakingTimingExportClockPtr =
        decltype(&DAQmxCfgBurstHandshakingTimingExportClock);
    using CfgBurstHandshakingTimingImportClockPtr =
        decltype(&DAQmxCfgBurstHandshakingTimingImportClock);
    using CfgChangeDetectionTimingPtr = decltype(&DAQmxCfgChangeDetectionTiming);
    using CfgDigEdgeRefTrigPtr = decltype(&DAQmxCfgDigEdgeRefTrig);
    using CfgDigEdgeStartTrigPtr = decltype(&DAQmxCfgDigEdgeStartTrig);
    using CfgDigPatternRefTrigPtr = decltype(&DAQmxCfgDigPatternRefTrig);
    using CfgDigPatternStartTrigPtr = decltype(&DAQmxCfgDigPatternStartTrig);
    using CfgHandshakingTimingPtr = decltype(&DAQmxCfgHandshakingTiming);
    using CfgImplicitTimingPtr = decltype(&DAQmxCfgImplicitTiming);
    using CfgInputBufferPtr = decltype(&DAQmxCfgInputBuffer);
    using CfgOutputBufferPtr = decltype(&DAQmxCfgOutputBuffer);
    using CfgPipelinedSampClkTimingPtr = decltype(&DAQmxCfgPipelinedSampClkTiming);
    using CfgSampClkTimingPtr = decltype(&DAQmxCfgSampClkTiming);
    using CfgTimeStartTrigPtr = decltype(&DAQmxCfgTimeStartTrig);
    using CfgWatchdogAOExpirStatesPtr = decltype(&DAQmxCfgWatchdogAOExpirStates);
    using CfgWatchdogCOExpirStatesPtr = decltype(&DAQmxCfgWatchdogCOExpirStates);
    using CfgWatchdogDOExpirStatesPtr = decltype(&DAQmxCfgWatchdogDOExpirStates);
    using ClearTEDSPtr = decltype(&DAQmxClearTEDS);
    using ClearTaskPtr = decltype(&DAQmxClearTask);
    using ConfigureLoggingPtr = decltype(&DAQmxConfigureLogging);
    using ConfigureTEDSPtr = decltype(&DAQmxConfigureTEDS);
    using ConnectTermsPtr = decltype(&DAQmxConnectTerms);
    using ControlWatchdogTaskPtr = decltype(&DAQmxControlWatchdogTask);
    using CreateAIAccel4WireDCVoltageChanPtr =
        decltype(&DAQmxCreateAIAccel4WireDCVoltageChan);
    using CreateAIAccelChanPtr = decltype(&DAQmxCreateAIAccelChan);
    using CreateAIAccelChargeChanPtr = decltype(&DAQmxCreateAIAccelChargeChan);
    using CreateAIBridgeChanPtr = decltype(&DAQmxCreateAIBridgeChan);
    using CreateAIChargeChanPtr = decltype(&DAQmxCreateAIChargeChan);
    using CreateAICurrentChanPtr = decltype(&DAQmxCreateAICurrentChan);
    using CreateAICurrentRMSChanPtr = decltype(&DAQmxCreateAICurrentRMSChan);
    using CreateAIForceBridgePolynomialChanPtr =
        decltype(&DAQmxCreateAIForceBridgePolynomialChan);
    using CreateAIForceBridgeTableChanPtr =
        decltype(&DAQmxCreateAIForceBridgeTableChan);
    using CreateAIForceBridgeTwoPointLinChanPtr =
        decltype(&DAQmxCreateAIForceBridgeTwoPointLinChan);
    using CreateAIForceIEPEChanPtr = decltype(&DAQmxCreateAIForceIEPEChan);
    using CreateAIFreqVoltageChanPtr = decltype(&DAQmxCreateAIFreqVoltageChan);
    using CreateAIMicrophoneChanPtr = decltype(&DAQmxCreateAIMicrophoneChan);
    using CreateAIPosEddyCurrProxProbeChanPtr =
        decltype(&DAQmxCreateAIPosEddyCurrProxProbeChan);
    using CreateAIPosLVDTChanPtr = decltype(&DAQmxCreateAIPosLVDTChan);
    using CreateAIPosRVDTChanPtr = decltype(&DAQmxCreateAIPosRVDTChan);
    using CreateAIPressureBridgePolynomialChanPtr =
        decltype(&DAQmxCreateAIPressureBridgePolynomialChan);
    using CreateAIPressureBridgeTableChanPtr =
        decltype(&DAQmxCreateAIPressureBridgeTableChan);
    using CreateAIPressureBridgeTwoPointLinChanPtr =
        decltype(&DAQmxCreateAIPressureBridgeTwoPointLinChan);
    using CreateAIRTDChanPtr = decltype(&DAQmxCreateAIRTDChan);
    using CreateAIResistanceChanPtr = decltype(&DAQmxCreateAIResistanceChan);
    using CreateAIRosetteStrainGageChanPtr =
        decltype(&DAQmxCreateAIRosetteStrainGageChan);
    using CreateAIStrainGageChanPtr = decltype(&DAQmxCreateAIStrainGageChan);
    using CreateAITempBuiltInSensorChanPtr =
        decltype(&DAQmxCreateAITempBuiltInSensorChan);
    using CreateAIThrmcplChanPtr = decltype(&DAQmxCreateAIThrmcplChan);
    using CreateAIThrmstrChanIexPtr = decltype(&DAQmxCreateAIThrmstrChanIex);
    using CreateAIThrmstrChanVexPtr = decltype(&DAQmxCreateAIThrmstrChanVex);
    using CreateAITorqueBridgePolynomialChanPtr =
        decltype(&DAQmxCreateAITorqueBridgePolynomialChan);
    using CreateAITorqueBridgeTableChanPtr =
        decltype(&DAQmxCreateAITorqueBridgeTableChan);
    using CreateAITorqueBridgeTwoPointLinChanPtr =
        decltype(&DAQmxCreateAITorqueBridgeTwoPointLinChan);
    using CreateAIVelocityIEPEChanPtr = decltype(&DAQmxCreateAIVelocityIEPEChan);
    using CreateAIVoltageChanPtr = decltype(&DAQmxCreateAIVoltageChan);
    using CreateAIVoltageChanWithExcitPtr =
        decltype(&DAQmxCreateAIVoltageChanWithExcit);
    using CreateAIVoltageRMSChanPtr = decltype(&DAQmxCreateAIVoltageRMSChan);
    using CreateAOCurrentChanPtr = decltype(&DAQmxCreateAOCurrentChan);
    using CreateAOFuncGenChanPtr = decltype(&DAQmxCreateAOFuncGenChan);
    using CreateAOVoltageChanPtr = decltype(&DAQmxCreateAOVoltageChan);
    using CreateCIAngEncoderChanPtr = decltype(&DAQmxCreateCIAngEncoderChan);
    using CreateCIAngVelocityChanPtr = decltype(&DAQmxCreateCIAngVelocityChan);
    using CreateCICountEdgesChanPtr = decltype(&DAQmxCreateCICountEdgesChan);
    using CreateCIDutyCycleChanPtr = decltype(&DAQmxCreateCIDutyCycleChan);
    using CreateCIFreqChanPtr = decltype(&DAQmxCreateCIFreqChan);
    using CreateCIGPSTimestampChanPtr = decltype(&DAQmxCreateCIGPSTimestampChan);
    using CreateCILinEncoderChanPtr = decltype(&DAQmxCreateCILinEncoderChan);
    using CreateCILinVelocityChanPtr = decltype(&DAQmxCreateCILinVelocityChan);
    using CreateCIPeriodChanPtr = decltype(&DAQmxCreateCIPeriodChan);
    using CreateCIPulseChanFreqPtr = decltype(&DAQmxCreateCIPulseChanFreq);
    using CreateCIPulseChanTicksPtr = decltype(&DAQmxCreateCIPulseChanTicks);
    using CreateCIPulseChanTimePtr = decltype(&DAQmxCreateCIPulseChanTime);
    using CreateCIPulseWidthChanPtr = decltype(&DAQmxCreateCIPulseWidthChan);
    using CreateCISemiPeriodChanPtr = decltype(&DAQmxCreateCISemiPeriodChan);
    using CreateCITwoEdgeSepChanPtr = decltype(&DAQmxCreateCITwoEdgeSepChan);
    using CreateCOPulseChanFreqPtr = decltype(&DAQmxCreateCOPulseChanFreq);
    using CreateCOPulseChanTicksPtr = decltype(&DAQmxCreateCOPulseChanTicks);
    using CreateCOPulseChanTimePtr = decltype(&DAQmxCreateCOPulseChanTime);
    using CreateDIChanPtr = decltype(&DAQmxCreateDIChan);
    using CreateDOChanPtr = decltype(&DAQmxCreateDOChan);
    using CreateLinScalePtr = decltype(&DAQmxCreateLinScale);
    using CreateMapScalePtr = decltype(&DAQmxCreateMapScale);
    using CreatePolynomialScalePtr = decltype(&DAQmxCreatePolynomialScale);
    using CreateTEDSAIAccelChanPtr = decltype(&DAQmxCreateTEDSAIAccelChan);
    using CreateTEDSAIBridgeChanPtr = decltype(&DAQmxCreateTEDSAIBridgeChan);
    using CreateTEDSAICurrentChanPtr = decltype(&DAQmxCreateTEDSAICurrentChan);
    using CreateTEDSAIForceBridgeChanPtr = decltype(&DAQmxCreateTEDSAIForceBridgeChan);
    using CreateTEDSAIForceIEPEChanPtr = decltype(&DAQmxCreateTEDSAIForceIEPEChan);
    using CreateTEDSAIMicrophoneChanPtr = decltype(&DAQmxCreateTEDSAIMicrophoneChan);
    using CreateTEDSAIPosLVDTChanPtr = decltype(&DAQmxCreateTEDSAIPosLVDTChan);
    using CreateTEDSAIPosRVDTChanPtr = decltype(&DAQmxCreateTEDSAIPosRVDTChan);
    using CreateTEDSAIPressureBridgeChanPtr =
        decltype(&DAQmxCreateTEDSAIPressureBridgeChan);
    using CreateTEDSAIRTDChanPtr = decltype(&DAQmxCreateTEDSAIRTDChan);
    using CreateTEDSAIResistanceChanPtr = decltype(&DAQmxCreateTEDSAIResistanceChan);
    using CreateTEDSAIStrainGageChanPtr = decltype(&DAQmxCreateTEDSAIStrainGageChan);
    using CreateTEDSAIThrmcplChanPtr = decltype(&DAQmxCreateTEDSAIThrmcplChan);
    using CreateTEDSAIThrmstrChanIexPtr = decltype(&DAQmxCreateTEDSAIThrmstrChanIex);
    using CreateTEDSAIThrmstrChanVexPtr = decltype(&DAQmxCreateTEDSAIThrmstrChanVex);
    using CreateTEDSAITorqueBridgeChanPtr =
        decltype(&DAQmxCreateTEDSAITorqueBridgeChan);
    using CreateTEDSAIVoltageChanPtr = decltype(&DAQmxCreateTEDSAIVoltageChan);
    using CreateTEDSAIVoltageChanWithExcitPtr =
        decltype(&DAQmxCreateTEDSAIVoltageChanWithExcit);
    using CreateTableScalePtr = decltype(&DAQmxCreateTableScale);
    using CreateTaskPtr = decltype(&DAQmxCreateTask);
    using CreateWatchdogTimerTaskPtr = decltype(&DAQmxCreateWatchdogTimerTask);
    using CreateWatchdogTimerTaskExPtr = decltype(&DAQmxCreateWatchdogTimerTaskEx);
    using DeleteNetworkDevicePtr = decltype(&DAQmxDeleteNetworkDevice);
    using DeleteSavedGlobalChanPtr = decltype(&DAQmxDeleteSavedGlobalChan);
    using DeleteSavedScalePtr = decltype(&DAQmxDeleteSavedScale);
    using DeleteSavedTaskPtr = decltype(&DAQmxDeleteSavedTask);
    using DeviceSupportsCalPtr = decltype(&DAQmxDeviceSupportsCal);
    using DisableRefTrigPtr = decltype(&DAQmxDisableRefTrig);
    using DisableStartTrigPtr = decltype(&DAQmxDisableStartTrig);
    using DisconnectTermsPtr = decltype(&DAQmxDisconnectTerms);
    using ExportSignalPtr = decltype(&DAQmxExportSignal);
    using GetAIChanCalCalDatePtr = decltype(&DAQmxGetAIChanCalCalDate);
    using GetAIChanCalExpDatePtr = decltype(&DAQmxGetAIChanCalExpDate);
    using GetAnalogPowerUpStatesPtr = decltype(&DAQmxGetAnalogPowerUpStates);
    using GetAnalogPowerUpStatesWithOutputTypePtr =
        decltype(&DAQmxGetAnalogPowerUpStatesWithOutputType);
    using GetArmStartTrigTimestampValPtr = decltype(&DAQmxGetArmStartTrigTimestampVal);
    using GetArmStartTrigTrigWhenPtr = decltype(&DAQmxGetArmStartTrigTrigWhen);
    using GetAutoConfiguredCDAQSyncConnectionsPtr =
        decltype(&DAQmxGetAutoConfiguredCDAQSyncConnections);
    using GetBufferAttributeUInt32Ptr = decltype(&DAQmxGetBufferAttribute);
    using GetCalInfoAttributeBoolPtr = decltype(&DAQmxGetCalInfoAttribute);
    using GetCalInfoAttributeDoublePtr = decltype(&DAQmxGetCalInfoAttribute);
    using GetCalInfoAttributeStringPtr = decltype(&DAQmxGetCalInfoAttribute);
    using GetCalInfoAttributeUInt32Ptr = decltype(&DAQmxGetCalInfoAttribute);
    using GetChanAttributeBoolPtr = decltype(&DAQmxGetChanAttribute);
    using GetChanAttributeDoublePtr = decltype(&DAQmxGetChanAttribute);
    using GetChanAttributeDoubleArrayPtr = decltype(&DAQmxGetChanAttribute);
    using GetChanAttributeInt32Ptr = decltype(&DAQmxGetChanAttribute);
    using GetChanAttributeStringPtr = decltype(&DAQmxGetChanAttribute);
    using GetChanAttributeUInt32Ptr = decltype(&DAQmxGetChanAttribute);
    using GetDeviceAttributeBoolPtr = decltype(&DAQmxGetDeviceAttribute);
    using GetDeviceAttributeDoublePtr = decltype(&DAQmxGetDeviceAttribute);
    using GetDeviceAttributeDoubleArrayPtr = decltype(&DAQmxGetDeviceAttribute);
    using GetDeviceAttributeInt32Ptr = decltype(&DAQmxGetDeviceAttribute);
    using GetDeviceAttributeInt32ArrayPtr = decltype(&DAQmxGetDeviceAttribute);
    using GetDeviceAttributeStringPtr = decltype(&DAQmxGetDeviceAttribute);
    using GetDeviceAttributeUInt32Ptr = decltype(&DAQmxGetDeviceAttribute);
    using GetDeviceAttributeUInt32ArrayPtr = decltype(&DAQmxGetDeviceAttribute);
    using GetDigitalLogicFamilyPowerUpStatePtr =
        decltype(&DAQmxGetDigitalLogicFamilyPowerUpState);
    using GetDigitalPowerUpStatesPtr = decltype(&DAQmxGetDigitalPowerUpStates);
    using GetDigitalPullUpPullDownStatesPtr =
        decltype(&DAQmxGetDigitalPullUpPullDownStates);
    using GetDisconnectedCDAQSyncPortsPtr =
        decltype(&DAQmxGetDisconnectedCDAQSyncPorts);
    using GetErrorStringPtr = decltype(&DAQmxGetErrorString);
    using GetExportedSignalAttributeBoolPtr =
        decltype(&DAQmxGetExportedSignalAttribute);
    using GetExportedSignalAttributeDoublePtr =
        decltype(&DAQmxGetExportedSignalAttribute);
    using GetExportedSignalAttributeInt32Ptr =
        decltype(&DAQmxGetExportedSignalAttribute);
    using GetExportedSignalAttributeStringPtr =
        decltype(&DAQmxGetExportedSignalAttribute);
    using GetExportedSignalAttributeUInt32Ptr =
        decltype(&DAQmxGetExportedSignalAttribute);
    using GetExtCalLastDateAndTimePtr = decltype(&DAQmxGetExtCalLastDateAndTime);
    using GetExtendedErrorInfoPtr = int32 (*)(char errorString[], uInt32 bufferSize);
    using GetFirstSampClkWhenPtr = decltype(&DAQmxGetFirstSampClkWhen);
    using GetFirstSampTimestampValPtr = decltype(&DAQmxGetFirstSampTimestampVal);
    using GetNthTaskChannelPtr = decltype(&DAQmxGetNthTaskChannel);
    using GetNthTaskDevicePtr = decltype(&DAQmxGetNthTaskDevice);
    using GetNthTaskReadChannelPtr = decltype(&DAQmxGetNthTaskReadChannel);
    using GetPersistedChanAttributeBoolPtr = decltype(&DAQmxGetPersistedChanAttribute);
    using GetPersistedChanAttributeStringPtr =
        decltype(&DAQmxGetPersistedChanAttribute);
    using GetPersistedScaleAttributeBoolPtr =
        decltype(&DAQmxGetPersistedScaleAttribute);
    using GetPersistedScaleAttributeStringPtr =
        decltype(&DAQmxGetPersistedScaleAttribute);
    using GetPersistedTaskAttributeBoolPtr = decltype(&DAQmxGetPersistedTaskAttribute);
    using GetPersistedTaskAttributeStringPtr =
        decltype(&DAQmxGetPersistedTaskAttribute);
    using GetPhysicalChanAttributeBoolPtr = decltype(&DAQmxGetPhysicalChanAttribute);
    using GetPhysicalChanAttributeBytesPtr = decltype(&DAQmxGetPhysicalChanAttribute);
    using GetPhysicalChanAttributeDoublePtr = decltype(&DAQmxGetPhysicalChanAttribute);
    using GetPhysicalChanAttributeDoubleArrayPtr =
        decltype(&DAQmxGetPhysicalChanAttribute);
    using GetPhysicalChanAttributeInt32Ptr = decltype(&DAQmxGetPhysicalChanAttribute);
    using GetPhysicalChanAttributeInt32ArrayPtr =
        decltype(&DAQmxGetPhysicalChanAttribute);
    using GetPhysicalChanAttributeStringPtr = decltype(&DAQmxGetPhysicalChanAttribute);
    using GetPhysicalChanAttributeUInt32Ptr = decltype(&DAQmxGetPhysicalChanAttribute);
    using GetPhysicalChanAttributeUInt32ArrayPtr =
        decltype(&DAQmxGetPhysicalChanAttribute);
    using GetReadAttributeBoolPtr = decltype(&DAQmxGetReadAttribute);
    using GetReadAttributeDoublePtr = decltype(&DAQmxGetReadAttribute);
    using GetReadAttributeInt32Ptr = decltype(&DAQmxGetReadAttribute);
    using GetReadAttributeStringPtr = decltype(&DAQmxGetReadAttribute);
    using GetReadAttributeUInt32Ptr = decltype(&DAQmxGetReadAttribute);
    using GetReadAttributeUInt64Ptr = decltype(&DAQmxGetReadAttribute);
    using GetRealTimeAttributeBoolPtr = decltype(&DAQmxGetRealTimeAttribute);
    using GetRealTimeAttributeInt32Ptr = decltype(&DAQmxGetRealTimeAttribute);
    using GetRealTimeAttributeUInt32Ptr = decltype(&DAQmxGetRealTimeAttribute);
    using GetRefTrigTimestampValPtr = decltype(&DAQmxGetRefTrigTimestampVal);
    using GetScaleAttributeDoublePtr = decltype(&DAQmxGetScaleAttribute);
    using GetScaleAttributeDoubleArrayPtr = decltype(&DAQmxGetScaleAttribute);
    using GetScaleAttributeInt32Ptr = decltype(&DAQmxGetScaleAttribute);
    using GetScaleAttributeStringPtr = decltype(&DAQmxGetScaleAttribute);
    using GetSelfCalLastDateAndTimePtr = decltype(&DAQmxGetSelfCalLastDateAndTime);
    using GetStartTrigTimestampValPtr = decltype(&DAQmxGetStartTrigTimestampVal);
    using GetStartTrigTrigWhenPtr = decltype(&DAQmxGetStartTrigTrigWhen);
    using GetSyncPulseTimeWhenPtr = decltype(&DAQmxGetSyncPulseTimeWhen);
    using GetSystemInfoAttributeStringPtr = decltype(&DAQmxGetSystemInfoAttribute);
    using GetSystemInfoAttributeUInt32Ptr = decltype(&DAQmxGetSystemInfoAttribute);
    using GetTaskAttributeBoolPtr = decltype(&DAQmxGetTaskAttribute);
    using GetTaskAttributeStringPtr = decltype(&DAQmxGetTaskAttribute);
    using GetTaskAttributeUInt32Ptr = decltype(&DAQmxGetTaskAttribute);
    using GetTimingAttributeBoolPtr = decltype(&DAQmxGetTimingAttribute);
    using GetTimingAttributeDoublePtr = decltype(&DAQmxGetTimingAttribute);
    using GetTimingAttributeExBoolPtr = decltype(&DAQmxGetTimingAttributeEx);
    using GetTimingAttributeExDoublePtr = decltype(&DAQmxGetTimingAttributeEx);
    using GetTimingAttributeExInt32Ptr = decltype(&DAQmxGetTimingAttributeEx);
    using GetTimingAttributeExStringPtr = decltype(&DAQmxGetTimingAttributeEx);
    using GetTimingAttributeExTimestampPtr = decltype(&DAQmxGetTimingAttributeEx);
    using GetTimingAttributeExUInt32Ptr = decltype(&DAQmxGetTimingAttributeEx);
    using GetTimingAttributeExUInt64Ptr = decltype(&DAQmxGetTimingAttributeEx);
    using GetTimingAttributeInt32Ptr = decltype(&DAQmxGetTimingAttribute);
    using GetTimingAttributeStringPtr = decltype(&DAQmxGetTimingAttribute);
    using GetTimingAttributeTimestampPtr = decltype(&DAQmxGetTimingAttribute);
    using GetTimingAttributeUInt32Ptr = decltype(&DAQmxGetTimingAttribute);
    using GetTimingAttributeUInt64Ptr = decltype(&DAQmxGetTimingAttribute);
    using GetTrigAttributeBoolPtr = decltype(&DAQmxGetTrigAttribute);
    using GetTrigAttributeDoublePtr = decltype(&DAQmxGetTrigAttribute);
    using GetTrigAttributeDoubleArrayPtr = decltype(&DAQmxGetTrigAttribute);
    using GetTrigAttributeInt32Ptr = decltype(&DAQmxGetTrigAttribute);
    using GetTrigAttributeInt32ArrayPtr = decltype(&DAQmxGetTrigAttribute);
    using GetTrigAttributeStringPtr = decltype(&DAQmxGetTrigAttribute);
    using GetTrigAttributeTimestampPtr = decltype(&DAQmxGetTrigAttribute);
    using GetTrigAttributeUInt32Ptr = decltype(&DAQmxGetTrigAttribute);
    using GetWatchdogAttributeBoolPtr = decltype(&DAQmxGetWatchdogAttribute);
    using GetWatchdogAttributeDoublePtr = decltype(&DAQmxGetWatchdogAttribute);
    using GetWatchdogAttributeInt32Ptr = decltype(&DAQmxGetWatchdogAttribute);
    using GetWatchdogAttributeStringPtr = decltype(&DAQmxGetWatchdogAttribute);
    using GetWriteAttributeBoolPtr = decltype(&DAQmxGetWriteAttribute);
    using GetWriteAttributeDoublePtr = decltype(&DAQmxGetWriteAttribute);
    using GetWriteAttributeInt32Ptr = decltype(&DAQmxGetWriteAttribute);
    using GetWriteAttributeStringPtr = decltype(&DAQmxGetWriteAttribute);
    using GetWriteAttributeUInt32Ptr = decltype(&DAQmxGetWriteAttribute);
    using GetWriteAttributeUInt64Ptr = decltype(&DAQmxGetWriteAttribute);
    using IsTaskDonePtr = decltype(&DAQmxIsTaskDone);
    using LoadTaskPtr = decltype(&DAQmxLoadTask);
    using PerformBridgeOffsetNullingCalExPtr =
        decltype(&DAQmxPerformBridgeOffsetNullingCalEx);
    using PerformBridgeShuntCalExPtr = decltype(&DAQmxPerformBridgeShuntCalEx);
    using PerformStrainShuntCalExPtr = decltype(&DAQmxPerformStrainShuntCalEx);
    using PerformThrmcplLeadOffsetNullingCalPtr =
        decltype(&DAQmxPerformThrmcplLeadOffsetNullingCal);
    using ReadAnalogF64Ptr = decltype(&DAQmxReadAnalogF64);
    using ReadAnalogScalarF64Ptr = decltype(&DAQmxReadAnalogScalarF64);
    using ReadBinaryI16Ptr = decltype(&DAQmxReadBinaryI16);
    using ReadBinaryI32Ptr = decltype(&DAQmxReadBinaryI32);
    using ReadBinaryU16Ptr = decltype(&DAQmxReadBinaryU16);
    using ReadBinaryU32Ptr = decltype(&DAQmxReadBinaryU32);
    using ReadCounterF64Ptr = decltype(&DAQmxReadCounterF64);
    using ReadCounterF64ExPtr = decltype(&DAQmxReadCounterF64Ex);
    using ReadCounterScalarF64Ptr = decltype(&DAQmxReadCounterScalarF64);
    using ReadCounterScalarU32Ptr = decltype(&DAQmxReadCounterScalarU32);
    using ReadCounterU32Ptr = decltype(&DAQmxReadCounterU32);
    using ReadCounterU32ExPtr = decltype(&DAQmxReadCounterU32Ex);
    using ReadCtrFreqPtr = decltype(&DAQmxReadCtrFreq);
    using ReadCtrFreqScalarPtr = decltype(&DAQmxReadCtrFreqScalar);
    using ReadCtrTicksPtr = decltype(&DAQmxReadCtrTicks);
    using ReadCtrTicksScalarPtr = decltype(&DAQmxReadCtrTicksScalar);
    using ReadCtrTimePtr = decltype(&DAQmxReadCtrTime);
    using ReadCtrTimeScalarPtr = decltype(&DAQmxReadCtrTimeScalar);
    using ReadDigitalLinesPtr = decltype(&DAQmxReadDigitalLines);
    using ReadDigitalScalarU32Ptr = decltype(&DAQmxReadDigitalScalarU32);
    using ReadDigitalU16Ptr = decltype(&DAQmxReadDigitalU16);
    using ReadDigitalU32Ptr = decltype(&DAQmxReadDigitalU32);
    using ReadDigitalU8Ptr = decltype(&DAQmxReadDigitalU8);
    using ReadRawPtr = decltype(&DAQmxReadRaw);
    using RegisterDoneEventPtr = decltype(&DAQmxRegisterDoneEvent);
    using RegisterEveryNSamplesEventPtr = decltype(&DAQmxRegisterEveryNSamplesEvent);
    using RegisterSignalEventPtr = decltype(&DAQmxRegisterSignalEvent);
    using RemoveCDAQSyncConnectionPtr = decltype(&DAQmxRemoveCDAQSyncConnection);
    using ReserveNetworkDevicePtr = decltype(&DAQmxReserveNetworkDevice);
    using ResetBufferAttributePtr = decltype(&DAQmxResetBufferAttribute);
    using ResetChanAttributePtr = decltype(&DAQmxResetChanAttribute);
    using ResetDevicePtr = decltype(&DAQmxResetDevice);
    using ResetExportedSignalAttributePtr =
        decltype(&DAQmxResetExportedSignalAttribute);
    using ResetReadAttributePtr = decltype(&DAQmxResetReadAttribute);
    using ResetRealTimeAttributePtr = decltype(&DAQmxResetRealTimeAttribute);
    using ResetTimingAttributePtr = decltype(&DAQmxResetTimingAttribute);
    using ResetTimingAttributeExPtr = decltype(&DAQmxResetTimingAttributeEx);
    using ResetTrigAttributePtr = decltype(&DAQmxResetTrigAttribute);
    using ResetWatchdogAttributePtr = decltype(&DAQmxResetWatchdogAttribute);
    using ResetWriteAttributePtr = decltype(&DAQmxResetWriteAttribute);
    using RestoreLastExtCalConstPtr = decltype(&DAQmxRestoreLastExtCalConst);
    using SaveGlobalChanPtr = decltype(&DAQmxSaveGlobalChan);
    using SaveScalePtr = decltype(&DAQmxSaveScale);
    using SaveTaskPtr = decltype(&DAQmxSaveTask);
    using SelfCalPtr = decltype(&DAQmxSelfCal);
    using SelfTestDevicePtr = decltype(&DAQmxSelfTestDevice);
    using SetAIChanCalCalDatePtr = decltype(&DAQmxSetAIChanCalCalDate);
    using SetAIChanCalExpDatePtr = decltype(&DAQmxSetAIChanCalExpDate);
    using SetAnalogPowerUpStatesPtr = decltype(&DAQmxSetAnalogPowerUpStates);
    using SetAnalogPowerUpStatesWithOutputTypePtr =
        decltype(&DAQmxSetAnalogPowerUpStatesWithOutputType);
    using SetArmStartTrigTrigWhenPtr = decltype(&DAQmxSetArmStartTrigTrigWhen);
    using SetBufferAttributeUInt32Ptr = decltype(&DAQmxSetBufferAttribute);
    using SetCalInfoAttributeBoolPtr = decltype(&DAQmxSetCalInfoAttribute);
    using SetCalInfoAttributeDoublePtr = decltype(&DAQmxSetCalInfoAttribute);
    using SetCalInfoAttributeStringPtr = decltype(&DAQmxSetCalInfoAttribute);
    using SetCalInfoAttributeUInt32Ptr = decltype(&DAQmxSetCalInfoAttribute);
    using SetChanAttributeBoolPtr = decltype(&DAQmxSetChanAttribute);
    using SetChanAttributeDoublePtr = decltype(&DAQmxSetChanAttribute);
    using SetChanAttributeDoubleArrayPtr = decltype(&DAQmxSetChanAttribute);
    using SetChanAttributeInt32Ptr = decltype(&DAQmxSetChanAttribute);
    using SetChanAttributeStringPtr = decltype(&DAQmxSetChanAttribute);
    using SetChanAttributeUInt32Ptr = decltype(&DAQmxSetChanAttribute);
    using SetDigitalLogicFamilyPowerUpStatePtr =
        decltype(&DAQmxSetDigitalLogicFamilyPowerUpState);
    using SetDigitalPowerUpStatesPtr = decltype(&DAQmxSetDigitalPowerUpStates);
    using SetDigitalPullUpPullDownStatesPtr =
        decltype(&DAQmxSetDigitalPullUpPullDownStates);
    using SetExportedSignalAttributeBoolPtr =
        decltype(&DAQmxSetExportedSignalAttribute);
    using SetExportedSignalAttributeDoublePtr =
        decltype(&DAQmxSetExportedSignalAttribute);
    using SetExportedSignalAttributeInt32Ptr =
        decltype(&DAQmxSetExportedSignalAttribute);
    using SetExportedSignalAttributeStringPtr =
        decltype(&DAQmxSetExportedSignalAttribute);
    using SetExportedSignalAttributeUInt32Ptr =
        decltype(&DAQmxSetExportedSignalAttribute);
    using SetFirstSampClkWhenPtr = decltype(&DAQmxSetFirstSampClkWhen);
    using SetReadAttributeBoolPtr = decltype(&DAQmxSetReadAttribute);
    using SetReadAttributeDoublePtr = decltype(&DAQmxSetReadAttribute);
    using SetReadAttributeInt32Ptr = decltype(&DAQmxSetReadAttribute);
    using SetReadAttributeStringPtr = decltype(&DAQmxSetReadAttribute);
    using SetReadAttributeUInt32Ptr = decltype(&DAQmxSetReadAttribute);
    using SetReadAttributeUInt64Ptr = decltype(&DAQmxSetReadAttribute);
    using SetRealTimeAttributeBoolPtr = decltype(&DAQmxSetRealTimeAttribute);
    using SetRealTimeAttributeInt32Ptr = decltype(&DAQmxSetRealTimeAttribute);
    using SetRealTimeAttributeUInt32Ptr = decltype(&DAQmxSetRealTimeAttribute);
    using SetRuntimeEnvironmentPtr = int32 (*)(
        const char environment[],
        const char environmentVersion[],
        const char reserved1[],
        const char reserved2[]
    );
    using SetScaleAttributeDoublePtr = decltype(&DAQmxSetScaleAttribute);
    using SetScaleAttributeDoubleArrayPtr = decltype(&DAQmxSetScaleAttribute);
    using SetScaleAttributeInt32Ptr = decltype(&DAQmxSetScaleAttribute);
    using SetScaleAttributeStringPtr = decltype(&DAQmxSetScaleAttribute);
    using SetStartTrigTrigWhenPtr = decltype(&DAQmxSetStartTrigTrigWhen);
    using SetSyncPulseTimeWhenPtr = decltype(&DAQmxSetSyncPulseTimeWhen);
    using SetTimingAttributeBoolPtr = decltype(&DAQmxSetTimingAttribute);
    using SetTimingAttributeDoublePtr = decltype(&DAQmxSetTimingAttribute);
    using SetTimingAttributeExBoolPtr = decltype(&DAQmxSetTimingAttributeEx);
    using SetTimingAttributeExDoublePtr = decltype(&DAQmxSetTimingAttributeEx);
    using SetTimingAttributeExInt32Ptr = decltype(&DAQmxSetTimingAttributeEx);
    using SetTimingAttributeExStringPtr = decltype(&DAQmxSetTimingAttributeEx);
    using SetTimingAttributeExTimestampPtr = decltype(&DAQmxSetTimingAttributeEx);
    using SetTimingAttributeExUInt32Ptr = decltype(&DAQmxSetTimingAttributeEx);
    using SetTimingAttributeExUInt64Ptr = decltype(&DAQmxSetTimingAttributeEx);
    using SetTimingAttributeInt32Ptr = decltype(&DAQmxSetTimingAttribute);
    using SetTimingAttributeStringPtr = decltype(&DAQmxSetTimingAttribute);
    using SetTimingAttributeTimestampPtr = decltype(&DAQmxSetTimingAttribute);
    using SetTimingAttributeUInt32Ptr = decltype(&DAQmxSetTimingAttribute);
    using SetTimingAttributeUInt64Ptr = decltype(&DAQmxSetTimingAttribute);
    using SetTrigAttributeBoolPtr = decltype(&DAQmxSetTrigAttribute);
    using SetTrigAttributeDoublePtr = decltype(&DAQmxSetTrigAttribute);
    using SetTrigAttributeDoubleArrayPtr = decltype(&DAQmxSetTrigAttribute);
    using SetTrigAttributeInt32Ptr = decltype(&DAQmxSetTrigAttribute);
    using SetTrigAttributeInt32ArrayPtr = decltype(&DAQmxSetTrigAttribute);
    using SetTrigAttributeStringPtr = decltype(&DAQmxSetTrigAttribute);
    using SetTrigAttributeTimestampPtr = decltype(&DAQmxSetTrigAttribute);
    using SetTrigAttributeUInt32Ptr = decltype(&DAQmxSetTrigAttribute);
    using SetWatchdogAttributeBoolPtr = decltype(&DAQmxSetWatchdogAttribute);
    using SetWatchdogAttributeDoublePtr = decltype(&DAQmxSetWatchdogAttribute);
    using SetWatchdogAttributeInt32Ptr = decltype(&DAQmxSetWatchdogAttribute);
    using SetWatchdogAttributeStringPtr = decltype(&DAQmxSetWatchdogAttribute);
    using SetWriteAttributeBoolPtr = decltype(&DAQmxSetWriteAttribute);
    using SetWriteAttributeDoublePtr = decltype(&DAQmxSetWriteAttribute);
    using SetWriteAttributeInt32Ptr = decltype(&DAQmxSetWriteAttribute);
    using SetWriteAttributeStringPtr = decltype(&DAQmxSetWriteAttribute);
    using SetWriteAttributeUInt32Ptr = decltype(&DAQmxSetWriteAttribute);
    using SetWriteAttributeUInt64Ptr = decltype(&DAQmxSetWriteAttribute);
    using StartNewFilePtr = decltype(&DAQmxStartNewFile);
    using StartTaskPtr = decltype(&DAQmxStartTask);
    using StopTaskPtr = decltype(&DAQmxStopTask);
    using TaskControlPtr = decltype(&DAQmxTaskControl);
    using TristateOutputTermPtr = decltype(&DAQmxTristateOutputTerm);
    using UnregisterDoneEventPtr = decltype(&DAQmxRegisterDoneEvent);
    using UnregisterEveryNSamplesEventPtr = decltype(&DAQmxRegisterEveryNSamplesEvent);
    using UnregisterSignalEventPtr = decltype(&DAQmxRegisterSignalEvent);
    using UnreserveNetworkDevicePtr = decltype(&DAQmxUnreserveNetworkDevice);
    using WaitForNextSampleClockPtr = decltype(&DAQmxWaitForNextSampleClock);
    using WaitForValidTimestampPtr = decltype(&DAQmxWaitForValidTimestamp);
    using WaitUntilTaskDonePtr = decltype(&DAQmxWaitUntilTaskDone);
    using WriteAnalogF64Ptr = decltype(&DAQmxWriteAnalogF64);
    using WriteAnalogScalarF64Ptr = decltype(&DAQmxWriteAnalogScalarF64);
    using WriteBinaryI16Ptr = decltype(&DAQmxWriteBinaryI16);
    using WriteBinaryI32Ptr = decltype(&DAQmxWriteBinaryI32);
    using WriteBinaryU16Ptr = decltype(&DAQmxWriteBinaryU16);
    using WriteBinaryU32Ptr = decltype(&DAQmxWriteBinaryU32);
    using WriteCtrFreqPtr = decltype(&DAQmxWriteCtrFreq);
    using WriteCtrFreqScalarPtr = decltype(&DAQmxWriteCtrFreqScalar);
    using WriteCtrTicksPtr = decltype(&DAQmxWriteCtrTicks);
    using WriteCtrTicksScalarPtr = decltype(&DAQmxWriteCtrTicksScalar);
    using WriteCtrTimePtr = decltype(&DAQmxWriteCtrTime);
    using WriteCtrTimeScalarPtr = decltype(&DAQmxWriteCtrTimeScalar);
    using WriteDigitalLinesPtr = decltype(&DAQmxWriteDigitalLines);
    using WriteDigitalScalarU32Ptr = decltype(&DAQmxWriteDigitalScalarU32);
    using WriteDigitalU16Ptr = decltype(&DAQmxWriteDigitalU16);
    using WriteDigitalU32Ptr = decltype(&DAQmxWriteDigitalU32);
    using WriteDigitalU8Ptr = decltype(&DAQmxWriteDigitalU8);
    using WriteRawPtr = decltype(&DAQmxWriteRaw);
    using WriteToTEDSFromArrayPtr = decltype(&DAQmxWriteToTEDSFromArray);
    using WriteToTEDSFromFilePtr = decltype(&DAQmxWriteToTEDSFromFile);
    using SetReadRelativeToPtr = decltype(&DAQmxSetReadRelativeTo);
    using SetReadOffsetPtr = decltype(&DAQmxSetReadOffset);
    using SetReadOverWritePtr = decltype(&DAQmxSetReadOverWrite);
    using GetReadTotalSampPerChanAcquiredPtr =
        decltype(&DAQmxGetReadTotalSampPerChanAcquired);

    typedef struct FunctionPointers {
        AddCDAQSyncConnectionPtr AddCDAQSyncConnection;
        AddGlobalChansToTaskPtr AddGlobalChansToTask;
        AddNetworkDevicePtr AddNetworkDevice;
        AreConfiguredCDAQSyncPortsDisconnectedPtr
            AreConfiguredCDAQSyncPortsDisconnected;
        AutoConfigureCDAQSyncConnectionsPtr AutoConfigureCDAQSyncConnections;
        CalculateReversePolyCoeffPtr CalculateReversePolyCoeff;
        CfgAnlgEdgeRefTrigPtr CfgAnlgEdgeRefTrig;
        CfgAnlgEdgeStartTrigPtr CfgAnlgEdgeStartTrig;
        CfgAnlgMultiEdgeRefTrigPtr CfgAnlgMultiEdgeRefTrig;
        CfgAnlgMultiEdgeStartTrigPtr CfgAnlgMultiEdgeStartTrig;
        CfgAnlgWindowRefTrigPtr CfgAnlgWindowRefTrig;
        CfgAnlgWindowStartTrigPtr CfgAnlgWindowStartTrig;
        CfgBurstHandshakingTimingExportClockPtr CfgBurstHandshakingTimingExportClock;
        CfgBurstHandshakingTimingImportClockPtr CfgBurstHandshakingTimingImportClock;
        CfgChangeDetectionTimingPtr CfgChangeDetectionTiming;
        CfgDigEdgeRefTrigPtr CfgDigEdgeRefTrig;
        CfgDigEdgeStartTrigPtr CfgDigEdgeStartTrig;
        CfgDigPatternRefTrigPtr CfgDigPatternRefTrig;
        CfgDigPatternStartTrigPtr CfgDigPatternStartTrig;
        CfgHandshakingTimingPtr CfgHandshakingTiming;
        CfgImplicitTimingPtr CfgImplicitTiming;
        CfgInputBufferPtr CfgInputBuffer;
        CfgOutputBufferPtr CfgOutputBuffer;
        CfgPipelinedSampClkTimingPtr CfgPipelinedSampClkTiming;
        CfgSampClkTimingPtr CfgSampClkTiming;
        CfgTimeStartTrigPtr CfgTimeStartTrig;
        CfgWatchdogAOExpirStatesPtr CfgWatchdogAOExpirStates;
        CfgWatchdogCOExpirStatesPtr CfgWatchdogCOExpirStates;
        CfgWatchdogDOExpirStatesPtr CfgWatchdogDOExpirStates;
        ClearTEDSPtr ClearTEDS;
        ClearTaskPtr ClearTask;
        ConfigureLoggingPtr ConfigureLogging;
        ConfigureTEDSPtr ConfigureTEDS;
        ConnectTermsPtr ConnectTerms;
        ControlWatchdogTaskPtr ControlWatchdogTask;
        CreateAIAccel4WireDCVoltageChanPtr CreateAIAccel4WireDCVoltageChan;
        CreateAIAccelChanPtr CreateAIAccelChan;
        CreateAIAccelChargeChanPtr CreateAIAccelChargeChan;
        CreateAIBridgeChanPtr CreateAIBridgeChan;
        CreateAIChargeChanPtr CreateAIChargeChan;
        CreateAICurrentChanPtr CreateAICurrentChan;
        CreateAICurrentRMSChanPtr CreateAICurrentRMSChan;
        CreateAIForceBridgePolynomialChanPtr CreateAIForceBridgePolynomialChan;
        CreateAIForceBridgeTableChanPtr CreateAIForceBridgeTableChan;
        CreateAIForceBridgeTwoPointLinChanPtr CreateAIForceBridgeTwoPointLinChan;
        CreateAIForceIEPEChanPtr CreateAIForceIEPEChan;
        CreateAIFreqVoltageChanPtr CreateAIFreqVoltageChan;
        CreateAIMicrophoneChanPtr CreateAIMicrophoneChan;
        CreateAIPosEddyCurrProxProbeChanPtr CreateAIPosEddyCurrProxProbeChan;
        CreateAIPosLVDTChanPtr CreateAIPosLVDTChan;
        CreateAIPosRVDTChanPtr CreateAIPosRVDTChan;
        CreateAIPressureBridgePolynomialChanPtr CreateAIPressureBridgePolynomialChan;
        CreateAIPressureBridgeTableChanPtr CreateAIPressureBridgeTableChan;
        CreateAIPressureBridgeTwoPointLinChanPtr CreateAIPressureBridgeTwoPointLinChan;
        CreateAIRTDChanPtr CreateAIRTDChan;
        CreateAIResistanceChanPtr CreateAIResistanceChan;
        CreateAIRosetteStrainGageChanPtr CreateAIRosetteStrainGageChan;
        CreateAIStrainGageChanPtr CreateAIStrainGageChan;
        CreateAITempBuiltInSensorChanPtr CreateAITempBuiltInSensorChan;
        CreateAIThrmcplChanPtr CreateAIThrmcplChan;
        CreateAIThrmstrChanIexPtr CreateAIThrmstrChanIex;
        CreateAIThrmstrChanVexPtr CreateAIThrmstrChanVex;
        CreateAITorqueBridgePolynomialChanPtr CreateAITorqueBridgePolynomialChan;
        CreateAITorqueBridgeTableChanPtr CreateAITorqueBridgeTableChan;
        CreateAITorqueBridgeTwoPointLinChanPtr CreateAITorqueBridgeTwoPointLinChan;
        CreateAIVelocityIEPEChanPtr CreateAIVelocityIEPEChan;
        CreateAIVoltageChanPtr CreateAIVoltageChan;
        CreateAIVoltageChanWithExcitPtr CreateAIVoltageChanWithExcit;
        CreateAIVoltageRMSChanPtr CreateAIVoltageRMSChan;
        CreateAOCurrentChanPtr CreateAOCurrentChan;
        CreateAOFuncGenChanPtr CreateAOFuncGenChan;
        CreateAOVoltageChanPtr CreateAOVoltageChan;
        CreateCIAngEncoderChanPtr CreateCIAngEncoderChan;
        CreateCIAngVelocityChanPtr CreateCIAngVelocityChan;
        CreateCICountEdgesChanPtr CreateCICountEdgesChan;
        CreateCIDutyCycleChanPtr CreateCIDutyCycleChan;
        CreateCIFreqChanPtr CreateCIFreqChan;
        CreateCIGPSTimestampChanPtr CreateCIGPSTimestampChan;
        CreateCILinEncoderChanPtr CreateCILinEncoderChan;
        CreateCILinVelocityChanPtr CreateCILinVelocityChan;
        CreateCIPeriodChanPtr CreateCIPeriodChan;
        CreateCIPulseChanFreqPtr CreateCIPulseChanFreq;
        CreateCIPulseChanTicksPtr CreateCIPulseChanTicks;
        CreateCIPulseChanTimePtr CreateCIPulseChanTime;
        CreateCIPulseWidthChanPtr CreateCIPulseWidthChan;
        CreateCISemiPeriodChanPtr CreateCISemiPeriodChan;
        CreateCITwoEdgeSepChanPtr CreateCITwoEdgeSepChan;
        CreateCOPulseChanFreqPtr CreateCOPulseChanFreq;
        CreateCOPulseChanTicksPtr CreateCOPulseChanTicks;
        CreateCOPulseChanTimePtr CreateCOPulseChanTime;
        CreateDIChanPtr CreateDIChan;
        CreateDOChanPtr CreateDOChan;
        CreateLinScalePtr CreateLinScale;
        CreateMapScalePtr CreateMapScale;
        CreatePolynomialScalePtr CreatePolynomialScale;
        CreateTEDSAIAccelChanPtr CreateTEDSAIAccelChan;
        CreateTEDSAIBridgeChanPtr CreateTEDSAIBridgeChan;
        CreateTEDSAICurrentChanPtr CreateTEDSAICurrentChan;
        CreateTEDSAIForceBridgeChanPtr CreateTEDSAIForceBridgeChan;
        CreateTEDSAIForceIEPEChanPtr CreateTEDSAIForceIEPEChan;
        CreateTEDSAIMicrophoneChanPtr CreateTEDSAIMicrophoneChan;
        CreateTEDSAIPosLVDTChanPtr CreateTEDSAIPosLVDTChan;
        CreateTEDSAIPosRVDTChanPtr CreateTEDSAIPosRVDTChan;
        CreateTEDSAIPressureBridgeChanPtr CreateTEDSAIPressureBridgeChan;
        CreateTEDSAIRTDChanPtr CreateTEDSAIRTDChan;
        CreateTEDSAIResistanceChanPtr CreateTEDSAIResistanceChan;
        CreateTEDSAIStrainGageChanPtr CreateTEDSAIStrainGageChan;
        CreateTEDSAIThrmcplChanPtr CreateTEDSAIThrmcplChan;
        CreateTEDSAIThrmstrChanIexPtr CreateTEDSAIThrmstrChanIex;
        CreateTEDSAIThrmstrChanVexPtr CreateTEDSAIThrmstrChanVex;
        CreateTEDSAITorqueBridgeChanPtr CreateTEDSAITorqueBridgeChan;
        CreateTEDSAIVoltageChanPtr CreateTEDSAIVoltageChan;
        CreateTEDSAIVoltageChanWithExcitPtr CreateTEDSAIVoltageChanWithExcit;
        CreateTableScalePtr CreateTableScale;
        CreateTaskPtr CreateTask;
        CreateWatchdogTimerTaskPtr CreateWatchdogTimerTask;
        CreateWatchdogTimerTaskExPtr CreateWatchdogTimerTaskEx;
        DeleteNetworkDevicePtr DeleteNetworkDevice;
        DeleteSavedGlobalChanPtr DeleteSavedGlobalChan;
        DeleteSavedScalePtr DeleteSavedScale;
        DeleteSavedTaskPtr DeleteSavedTask;
        DeviceSupportsCalPtr DeviceSupportsCal;
        DisableRefTrigPtr DisableRefTrig;
        DisableStartTrigPtr DisableStartTrig;
        DisconnectTermsPtr DisconnectTerms;
        ExportSignalPtr ExportSignal;
        GetAIChanCalCalDatePtr GetAIChanCalCalDate;
        GetAIChanCalExpDatePtr GetAIChanCalExpDate;
        GetAnalogPowerUpStatesPtr GetAnalogPowerUpStates;
        GetAnalogPowerUpStatesWithOutputTypePtr GetAnalogPowerUpStatesWithOutputType;
        GetArmStartTrigTimestampValPtr GetArmStartTrigTimestampVal;
        GetArmStartTrigTrigWhenPtr GetArmStartTrigTrigWhen;
        GetAutoConfiguredCDAQSyncConnectionsPtr GetAutoConfiguredCDAQSyncConnections;
        GetBufferAttributeUInt32Ptr GetBufferAttributeUInt32;
        GetCalInfoAttributeBoolPtr GetCalInfoAttributeBool;
        GetCalInfoAttributeDoublePtr GetCalInfoAttributeDouble;
        GetCalInfoAttributeStringPtr GetCalInfoAttributeString;
        GetCalInfoAttributeUInt32Ptr GetCalInfoAttributeUInt32;
        GetChanAttributeBoolPtr GetChanAttributeBool;
        GetChanAttributeDoublePtr GetChanAttributeDouble;
        GetChanAttributeDoubleArrayPtr GetChanAttributeDoubleArray;
        GetChanAttributeInt32Ptr GetChanAttributeInt32;
        GetChanAttributeStringPtr GetChanAttributeString;
        GetChanAttributeUInt32Ptr GetChanAttributeUInt32;
        GetDeviceAttributeBoolPtr GetDeviceAttributeBool;
        GetDeviceAttributeDoublePtr GetDeviceAttributeDouble;
        GetDeviceAttributeDoubleArrayPtr GetDeviceAttributeDoubleArray;
        GetDeviceAttributeInt32Ptr GetDeviceAttributeInt32;
        GetDeviceAttributeInt32ArrayPtr GetDeviceAttributeInt32Array;
        GetDeviceAttributeStringPtr GetDeviceAttributeString;
        GetDeviceAttributeUInt32Ptr GetDeviceAttributeUInt32;
        GetDeviceAttributeUInt32ArrayPtr GetDeviceAttributeUInt32Array;
        GetDigitalLogicFamilyPowerUpStatePtr GetDigitalLogicFamilyPowerUpState;
        GetDigitalPowerUpStatesPtr GetDigitalPowerUpStates;
        GetDigitalPullUpPullDownStatesPtr GetDigitalPullUpPullDownStates;
        GetDisconnectedCDAQSyncPortsPtr GetDisconnectedCDAQSyncPorts;
        GetErrorStringPtr GetErrorString;
        GetExportedSignalAttributeBoolPtr GetExportedSignalAttributeBool;
        GetExportedSignalAttributeDoublePtr GetExportedSignalAttributeDouble;
        GetExportedSignalAttributeInt32Ptr GetExportedSignalAttributeInt32;
        GetExportedSignalAttributeStringPtr GetExportedSignalAttributeString;
        GetExportedSignalAttributeUInt32Ptr GetExportedSignalAttributeUInt32;
        GetExtCalLastDateAndTimePtr GetExtCalLastDateAndTime;
        GetExtendedErrorInfoPtr GetExtendedErrorInfo;
        GetFirstSampClkWhenPtr GetFirstSampClkWhen;
        GetFirstSampTimestampValPtr GetFirstSampTimestampVal;
        GetNthTaskChannelPtr GetNthTaskChannel;
        GetNthTaskDevicePtr GetNthTaskDevice;
        GetNthTaskReadChannelPtr GetNthTaskReadChannel;
        GetPersistedChanAttributeBoolPtr GetPersistedChanAttributeBool;
        GetPersistedChanAttributeStringPtr GetPersistedChanAttributeString;
        GetPersistedScaleAttributeBoolPtr GetPersistedScaleAttributeBool;
        GetPersistedScaleAttributeStringPtr GetPersistedScaleAttributeString;
        GetPersistedTaskAttributeBoolPtr GetPersistedTaskAttributeBool;
        GetPersistedTaskAttributeStringPtr GetPersistedTaskAttributeString;
        GetPhysicalChanAttributeBoolPtr GetPhysicalChanAttributeBool;
        GetPhysicalChanAttributeBytesPtr GetPhysicalChanAttributeBytes;
        GetPhysicalChanAttributeDoublePtr GetPhysicalChanAttributeDouble;
        GetPhysicalChanAttributeDoubleArrayPtr GetPhysicalChanAttributeDoubleArray;
        GetPhysicalChanAttributeInt32Ptr GetPhysicalChanAttributeInt32;
        GetPhysicalChanAttributeInt32ArrayPtr GetPhysicalChanAttributeInt32Array;
        GetPhysicalChanAttributeStringPtr GetPhysicalChanAttributeString;
        GetPhysicalChanAttributeUInt32Ptr GetPhysicalChanAttributeUInt32;
        GetPhysicalChanAttributeUInt32ArrayPtr GetPhysicalChanAttributeUInt32Array;
        GetReadAttributeBoolPtr GetReadAttributeBool;
        GetReadAttributeDoublePtr GetReadAttributeDouble;
        GetReadAttributeInt32Ptr GetReadAttributeInt32;
        GetReadAttributeStringPtr GetReadAttributeString;
        GetReadAttributeUInt32Ptr GetReadAttributeUInt32;
        GetReadAttributeUInt64Ptr GetReadAttributeUInt64;
        GetRealTimeAttributeBoolPtr GetRealTimeAttributeBool;
        GetRealTimeAttributeInt32Ptr GetRealTimeAttributeInt32;
        GetRealTimeAttributeUInt32Ptr GetRealTimeAttributeUInt32;
        GetRefTrigTimestampValPtr GetRefTrigTimestampVal;
        GetScaleAttributeDoublePtr GetScaleAttributeDouble;
        GetScaleAttributeDoubleArrayPtr GetScaleAttributeDoubleArray;
        GetScaleAttributeInt32Ptr GetScaleAttributeInt32;
        GetScaleAttributeStringPtr GetScaleAttributeString;
        GetSelfCalLastDateAndTimePtr GetSelfCalLastDateAndTime;
        GetStartTrigTimestampValPtr GetStartTrigTimestampVal;
        GetStartTrigTrigWhenPtr GetStartTrigTrigWhen;
        GetSyncPulseTimeWhenPtr GetSyncPulseTimeWhen;
        GetSystemInfoAttributeStringPtr GetSystemInfoAttributeString;
        GetSystemInfoAttributeUInt32Ptr GetSystemInfoAttributeUInt32;
        GetTaskAttributeBoolPtr GetTaskAttributeBool;
        GetTaskAttributeStringPtr GetTaskAttributeString;
        GetTaskAttributeUInt32Ptr GetTaskAttributeUInt32;
        GetTimingAttributeBoolPtr GetTimingAttributeBool;
        GetTimingAttributeDoublePtr GetTimingAttributeDouble;
        GetTimingAttributeExBoolPtr GetTimingAttributeExBool;
        GetTimingAttributeExDoublePtr GetTimingAttributeExDouble;
        GetTimingAttributeExInt32Ptr GetTimingAttributeExInt32;
        GetTimingAttributeExStringPtr GetTimingAttributeExString;
        GetTimingAttributeExTimestampPtr GetTimingAttributeExTimestamp;
        GetTimingAttributeExUInt32Ptr GetTimingAttributeExUInt32;
        GetTimingAttributeExUInt64Ptr GetTimingAttributeExUInt64;
        GetTimingAttributeInt32Ptr GetTimingAttributeInt32;
        GetTimingAttributeStringPtr GetTimingAttributeString;
        GetTimingAttributeTimestampPtr GetTimingAttributeTimestamp;
        GetTimingAttributeUInt32Ptr GetTimingAttributeUInt32;
        GetTimingAttributeUInt64Ptr GetTimingAttributeUInt64;
        GetTrigAttributeBoolPtr GetTrigAttributeBool;
        GetTrigAttributeDoublePtr GetTrigAttributeDouble;
        GetTrigAttributeDoubleArrayPtr GetTrigAttributeDoubleArray;
        GetTrigAttributeInt32Ptr GetTrigAttributeInt32;
        GetTrigAttributeInt32ArrayPtr GetTrigAttributeInt32Array;
        GetTrigAttributeStringPtr GetTrigAttributeString;
        GetTrigAttributeTimestampPtr GetTrigAttributeTimestamp;
        GetTrigAttributeUInt32Ptr GetTrigAttributeUInt32;
        GetWatchdogAttributeBoolPtr GetWatchdogAttributeBool;
        GetWatchdogAttributeDoublePtr GetWatchdogAttributeDouble;
        GetWatchdogAttributeInt32Ptr GetWatchdogAttributeInt32;
        GetWatchdogAttributeStringPtr GetWatchdogAttributeString;
        GetWriteAttributeBoolPtr GetWriteAttributeBool;
        GetWriteAttributeDoublePtr GetWriteAttributeDouble;
        GetWriteAttributeInt32Ptr GetWriteAttributeInt32;
        GetWriteAttributeStringPtr GetWriteAttributeString;
        GetWriteAttributeUInt32Ptr GetWriteAttributeUInt32;
        GetWriteAttributeUInt64Ptr GetWriteAttributeUInt64;
        IsTaskDonePtr IsTaskDone;
        LoadTaskPtr LoadTask;
        PerformBridgeOffsetNullingCalExPtr PerformBridgeOffsetNullingCalEx;
        PerformBridgeShuntCalExPtr PerformBridgeShuntCalEx;
        PerformStrainShuntCalExPtr PerformStrainShuntCalEx;
        PerformThrmcplLeadOffsetNullingCalPtr PerformThrmcplLeadOffsetNullingCal;
        ReadAnalogF64Ptr ReadAnalogF64;
        ReadAnalogScalarF64Ptr ReadAnalogScalarF64;
        ReadBinaryI16Ptr ReadBinaryI16;
        ReadBinaryI32Ptr ReadBinaryI32;
        ReadBinaryU16Ptr ReadBinaryU16;
        ReadBinaryU32Ptr ReadBinaryU32;
        ReadCounterF64Ptr ReadCounterF64;
        ReadCounterF64ExPtr ReadCounterF64Ex;
        ReadCounterScalarF64Ptr ReadCounterScalarF64;
        ReadCounterScalarU32Ptr ReadCounterScalarU32;
        ReadCounterU32Ptr ReadCounterU32;
        ReadCounterU32ExPtr ReadCounterU32Ex;
        ReadCtrFreqPtr ReadCtrFreq;
        ReadCtrFreqScalarPtr ReadCtrFreqScalar;
        ReadCtrTicksPtr ReadCtrTicks;
        ReadCtrTicksScalarPtr ReadCtrTicksScalar;
        ReadCtrTimePtr ReadCtrTime;
        ReadCtrTimeScalarPtr ReadCtrTimeScalar;
        ReadDigitalLinesPtr ReadDigitalLines;
        ReadDigitalScalarU32Ptr ReadDigitalScalarU32;
        ReadDigitalU16Ptr ReadDigitalU16;
        ReadDigitalU32Ptr ReadDigitalU32;
        ReadDigitalU8Ptr ReadDigitalU8;
        ReadRawPtr ReadRaw;
        RegisterDoneEventPtr RegisterDoneEvent;
        RegisterEveryNSamplesEventPtr RegisterEveryNSamplesEvent;
        RegisterSignalEventPtr RegisterSignalEvent;
        RemoveCDAQSyncConnectionPtr RemoveCDAQSyncConnection;
        ReserveNetworkDevicePtr ReserveNetworkDevice;
        ResetBufferAttributePtr ResetBufferAttribute;
        ResetChanAttributePtr ResetChanAttribute;
        ResetDevicePtr ResetDevice;
        ResetExportedSignalAttributePtr ResetExportedSignalAttribute;
        ResetReadAttributePtr ResetReadAttribute;
        ResetRealTimeAttributePtr ResetRealTimeAttribute;
        ResetTimingAttributePtr ResetTimingAttribute;
        ResetTimingAttributeExPtr ResetTimingAttributeEx;
        ResetTrigAttributePtr ResetTrigAttribute;
        ResetWatchdogAttributePtr ResetWatchdogAttribute;
        ResetWriteAttributePtr ResetWriteAttribute;
        RestoreLastExtCalConstPtr RestoreLastExtCalConst;
        SaveGlobalChanPtr SaveGlobalChan;
        SaveScalePtr SaveScale;
        SaveTaskPtr SaveTask;
        SelfCalPtr SelfCal;
        SelfTestDevicePtr SelfTestDevice;
        SetAIChanCalCalDatePtr SetAIChanCalCalDate;
        SetAIChanCalExpDatePtr SetAIChanCalExpDate;
        SetAnalogPowerUpStatesPtr SetAnalogPowerUpStates;
        SetAnalogPowerUpStatesWithOutputTypePtr SetAnalogPowerUpStatesWithOutputType;
        SetArmStartTrigTrigWhenPtr SetArmStartTrigTrigWhen;
        SetBufferAttributeUInt32Ptr SetBufferAttributeUInt32;
        SetCalInfoAttributeBoolPtr SetCalInfoAttributeBool;
        SetCalInfoAttributeDoublePtr SetCalInfoAttributeDouble;
        SetCalInfoAttributeStringPtr SetCalInfoAttributeString;
        SetCalInfoAttributeUInt32Ptr SetCalInfoAttributeUInt32;
        SetChanAttributeBoolPtr SetChanAttributeBool;
        SetChanAttributeDoublePtr SetChanAttributeDouble;
        SetChanAttributeDoubleArrayPtr SetChanAttributeDoubleArray;
        SetChanAttributeInt32Ptr SetChanAttributeInt32;
        SetChanAttributeStringPtr SetChanAttributeString;
        SetChanAttributeUInt32Ptr SetChanAttributeUInt32;
        SetDigitalLogicFamilyPowerUpStatePtr SetDigitalLogicFamilyPowerUpState;
        SetDigitalPowerUpStatesPtr SetDigitalPowerUpStates;
        SetDigitalPullUpPullDownStatesPtr SetDigitalPullUpPullDownStates;
        SetExportedSignalAttributeBoolPtr SetExportedSignalAttributeBool;
        SetExportedSignalAttributeDoublePtr SetExportedSignalAttributeDouble;
        SetExportedSignalAttributeInt32Ptr SetExportedSignalAttributeInt32;
        SetExportedSignalAttributeStringPtr SetExportedSignalAttributeString;
        SetExportedSignalAttributeUInt32Ptr SetExportedSignalAttributeUInt32;
        SetFirstSampClkWhenPtr SetFirstSampClkWhen;
        SetReadAttributeBoolPtr SetReadAttributeBool;
        SetReadAttributeDoublePtr SetReadAttributeDouble;
        SetReadAttributeInt32Ptr SetReadAttributeInt32;
        SetReadAttributeStringPtr SetReadAttributeString;
        SetReadAttributeUInt32Ptr SetReadAttributeUInt32;
        SetReadAttributeUInt64Ptr SetReadAttributeUInt64;
        SetRealTimeAttributeBoolPtr SetRealTimeAttributeBool;
        SetRealTimeAttributeInt32Ptr SetRealTimeAttributeInt32;
        SetRealTimeAttributeUInt32Ptr SetRealTimeAttributeUInt32;
        SetRuntimeEnvironmentPtr SetRuntimeEnvironment;
        SetScaleAttributeDoublePtr SetScaleAttributeDouble;
        SetScaleAttributeDoubleArrayPtr SetScaleAttributeDoubleArray;
        SetScaleAttributeInt32Ptr SetScaleAttributeInt32;
        SetScaleAttributeStringPtr SetScaleAttributeString;
        SetStartTrigTrigWhenPtr SetStartTrigTrigWhen;
        SetSyncPulseTimeWhenPtr SetSyncPulseTimeWhen;
        SetTimingAttributeBoolPtr SetTimingAttributeBool;
        SetTimingAttributeDoublePtr SetTimingAttributeDouble;
        SetTimingAttributeExBoolPtr SetTimingAttributeExBool;
        SetTimingAttributeExDoublePtr SetTimingAttributeExDouble;
        SetTimingAttributeExInt32Ptr SetTimingAttributeExInt32;
        SetTimingAttributeExStringPtr SetTimingAttributeExString;
        SetTimingAttributeExTimestampPtr SetTimingAttributeExTimestamp;
        SetTimingAttributeExUInt32Ptr SetTimingAttributeExUInt32;
        SetTimingAttributeExUInt64Ptr SetTimingAttributeExUInt64;
        SetTimingAttributeInt32Ptr SetTimingAttributeInt32;
        SetTimingAttributeStringPtr SetTimingAttributeString;
        SetTimingAttributeTimestampPtr SetTimingAttributeTimestamp;
        SetTimingAttributeUInt32Ptr SetTimingAttributeUInt32;
        SetTimingAttributeUInt64Ptr SetTimingAttributeUInt64;
        SetTrigAttributeBoolPtr SetTrigAttributeBool;
        SetTrigAttributeDoublePtr SetTrigAttributeDouble;
        SetTrigAttributeDoubleArrayPtr SetTrigAttributeDoubleArray;
        SetTrigAttributeInt32Ptr SetTrigAttributeInt32;
        SetTrigAttributeInt32ArrayPtr SetTrigAttributeInt32Array;
        SetTrigAttributeStringPtr SetTrigAttributeString;
        SetTrigAttributeTimestampPtr SetTrigAttributeTimestamp;
        SetTrigAttributeUInt32Ptr SetTrigAttributeUInt32;
        SetWatchdogAttributeBoolPtr SetWatchdogAttributeBool;
        SetWatchdogAttributeDoublePtr SetWatchdogAttributeDouble;
        SetWatchdogAttributeInt32Ptr SetWatchdogAttributeInt32;
        SetWatchdogAttributeStringPtr SetWatchdogAttributeString;
        SetWriteAttributeBoolPtr SetWriteAttributeBool;
        SetWriteAttributeDoublePtr SetWriteAttributeDouble;
        SetWriteAttributeInt32Ptr SetWriteAttributeInt32;
        SetWriteAttributeStringPtr SetWriteAttributeString;
        SetWriteAttributeUInt32Ptr SetWriteAttributeUInt32;
        SetWriteAttributeUInt64Ptr SetWriteAttributeUInt64;
        StartNewFilePtr StartNewFile;
        StartTaskPtr StartTask;
        StopTaskPtr StopTask;
        TaskControlPtr TaskControl;
        TristateOutputTermPtr TristateOutputTerm;
        UnregisterDoneEventPtr UnregisterDoneEvent;
        UnregisterEveryNSamplesEventPtr UnregisterEveryNSamplesEvent;
        UnregisterSignalEventPtr UnregisterSignalEvent;
        UnreserveNetworkDevicePtr UnreserveNetworkDevice;
        WaitForNextSampleClockPtr WaitForNextSampleClock;
        WaitForValidTimestampPtr WaitForValidTimestamp;
        WaitUntilTaskDonePtr WaitUntilTaskDone;
        WriteAnalogF64Ptr WriteAnalogF64;
        WriteAnalogScalarF64Ptr WriteAnalogScalarF64;
        WriteBinaryI16Ptr WriteBinaryI16;
        WriteBinaryI32Ptr WriteBinaryI32;
        WriteBinaryU16Ptr WriteBinaryU16;
        WriteBinaryU32Ptr WriteBinaryU32;
        WriteCtrFreqPtr WriteCtrFreq;
        WriteCtrFreqScalarPtr WriteCtrFreqScalar;
        WriteCtrTicksPtr WriteCtrTicks;
        WriteCtrTicksScalarPtr WriteCtrTicksScalar;
        WriteCtrTimePtr WriteCtrTime;
        WriteCtrTimeScalarPtr WriteCtrTimeScalar;
        WriteDigitalLinesPtr WriteDigitalLines;
        WriteDigitalScalarU32Ptr WriteDigitalScalarU32;
        WriteDigitalU16Ptr WriteDigitalU16;
        WriteDigitalU32Ptr WriteDigitalU32;
        WriteDigitalU8Ptr WriteDigitalU8;
        WriteRawPtr WriteRaw;
        WriteToTEDSFromArrayPtr WriteToTEDSFromArray;
        WriteToTEDSFromFilePtr WriteToTEDSFromFile;
        SetReadRelativeToPtr SetReadRelativeTo;
        SetReadOffsetPtr SetReadOffset;
        SetReadOverWritePtr SetReadOverWrite;
        GetReadTotalSampPerChanAcquiredPtr GetReadTotalSampPerChanAcquired;
    } FunctionLoadStatus;

    FunctionPointers function_pointers_{};
    std::unique_ptr<x::lib::Shared> lib;
};
}
