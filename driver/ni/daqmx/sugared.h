// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "x/cpp/errors/errors.h"

#include "driver/errors/errors.h"
#include "driver/ni/daqmx/api.h"

namespace daqmx {
const x::errors::Error CRITICAL_ERROR = driver::CRITICAL_HARDWARE_ERROR.sub("ni");
const x::errors::Error TEMPORARY_ERROR = driver::TEMPORARY_HARDWARE_ERROR.sub("ni");
const x::errors::Error FIELD_ERROR = CRITICAL_ERROR.sub("field");
const x::errors::Error ANALOG_WRITE_OUT_OF_BOUNDS = CRITICAL_ERROR.sub("200561");
const x::errors::Error APPLICATION_TOO_SLOW = CRITICAL_ERROR.sub("200729");
const x::errors::Error DEVICE_DISCONNECTED = CRITICAL_ERROR.sub("88710");
const x::errors::Error RESOURCE_NOT_AVAILABLE = CRITICAL_ERROR.sub("88708");
const x::errors::Error DEVICE_DISCONNECTED_2 = CRITICAL_ERROR.sub("88709");
const x::errors::Error ADC_CONVERSION_ERROR = CRITICAL_ERROR.sub("200019");
const x::errors::Error RESOURCE_RESERVED = CRITICAL_ERROR.sub("201105");
const x::errors::Error ROUTING_ERROR = CRITICAL_ERROR.sub("89130");
const auto TEMPORARILY_UNREACHABLE = x::errors::Error(
    TEMPORARY_ERROR,
    "The device is temporarily unreachable. Will keep trying"
);
const auto REQUIRES_RESTART = x::errors::Error(
    TEMPORARILY_UNREACHABLE,
    "Restarting task to recover"
);

class SugaredAPI {
    std::shared_ptr<API> dmx;

    [[nodiscard]] x::errors::Error process_error(int32 status) const;

public:
    explicit SugaredAPI(std::shared_ptr<API> dmx): dmx(std::move(dmx)) {}

    x::errors::Error AddCDAQSyncConnection(const char portList[]);
    x::errors::Error AddGlobalChansToTask(TaskHandle task, const char channelNames[]);
    x::errors::Error AddNetworkDevice(
        const char ipAddress[],
        const char deviceName[],
        bool32 attemptReservation,
        float64 timeout,
        char deviceNameOut[],
        uInt32 deviceNameOutBufferSize
    );
    x::errors::Error AreConfiguredCDAQSyncPortsDisconnected(
        const char chassisDevicesPorts[],
        float64 timeout,
        bool32 *disconnectedPortsExist
    );
    x::errors::Error
    AutoConfigureCDAQSyncConnections(const char chassisDevicesPorts[], float64 timeout);
    x::errors::Error CalculateReversePolyCoeff(
        const float64 forwardCoeffs[],
        uInt32 numForwardCoeffsIn,
        float64 minValX,
        float64 maxValX,
        int32 numPointsToCompute,
        int32 reversePolyOrder,
        float64 reverseCoeffs[]
    );
    x::errors::Error CfgAnlgEdgeRefTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerSlope,
        float64 triggerLevel,
        uInt32 pretriggerSamples
    );
    x::errors::Error CfgAnlgEdgeStartTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerSlope,
        float64 triggerLevel
    );
    x::errors::Error CfgAnlgMultiEdgeRefTrig(
        TaskHandle task,
        const char triggerSources[],
        const int32 triggerSlopeArray[],
        const float64 triggerLevelArray[],
        uInt32 pretriggerSamples,
        uInt32 arraySize
    );
    x::errors::Error CfgAnlgMultiEdgeStartTrig(
        TaskHandle task,
        const char triggerSources[],
        const int32 triggerSlopeArray[],
        const float64 triggerLevelArray[],
        uInt32 arraySize
    );
    x::errors::Error CfgAnlgWindowRefTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerWhen,
        float64 windowTop,
        float64 windowBottom,
        uInt32 pretriggerSamples
    );
    x::errors::Error CfgAnlgWindowStartTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerWhen,
        float64 windowTop,
        float64 windowBottom
    );
    x::errors::Error CfgBurstHandshakingTimingExportClock(
        TaskHandle task,
        int32 sampleMode,
        uInt64 sampsPerChan,
        float64 sampleClkRate,
        const char sampleClkOutpTerm[],
        int32 sampleClkPulsePolarity,
        int32 pauseWhen,
        int32 readyEventActiveLevel
    );
    x::errors::Error CfgBurstHandshakingTimingImportClock(
        TaskHandle task,
        int32 sampleMode,
        uInt64 sampsPerChan,
        float64 sampleClkRate,
        const char sampleClkSrc[],
        int32 sampleClkActiveEdge,
        int32 pauseWhen,
        int32 readyEventActiveLevel
    );
    x::errors::Error CfgChangeDetectionTiming(
        TaskHandle task,
        const char risingEdgeChan[],
        const char fallingEdgeChan[],
        int32 sampleMode,
        uInt64 sampsPerChan
    );
    x::errors::Error CfgDigEdgeRefTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerEdge,
        uInt32 pretriggerSamples
    );
    x::errors::Error
    CfgDigEdgeStartTrig(TaskHandle task, const char triggerSource[], int32 triggerEdge);
    x::errors::Error CfgDigPatternRefTrig(
        TaskHandle task,
        const char triggerSource[],
        const char triggerPattern[],
        int32 triggerWhen,
        uInt32 pretriggerSamples
    );
    x::errors::Error CfgDigPatternStartTrig(
        TaskHandle task,
        const char triggerSource[],
        const char triggerPattern[],
        int32 triggerWhen
    );
    x::errors::Error
    CfgHandshakingTiming(TaskHandle task, int32 sampleMode, uInt64 sampsPerChan);
    x::errors::Error
    CfgImplicitTiming(TaskHandle task, int32 sampleMode, uInt64 sampsPerChan);
    x::errors::Error CfgInputBuffer(TaskHandle task, uInt32 numSampsPerChan);
    x::errors::Error CfgOutputBuffer(TaskHandle task, uInt32 numSampsPerChan);
    x::errors::Error CfgPipelinedSampClkTiming(
        TaskHandle task,
        const char source[],
        float64 rate,
        int32 activeEdge,
        int32 sampleMode,
        uInt64 sampsPerChan
    );
    x::errors::Error CfgSampClkTiming(
        TaskHandle task,
        const char source[],
        float64 rate,
        int32 activeEdge,
        int32 sampleMode,
        uInt64 sampsPerChan
    );
    x::errors::Error
    CfgTimeStartTrig(TaskHandle task, CVIAbsoluteTime when, int32 timescale);
    x::errors::Error CfgWatchdogAOExpirStates(
        TaskHandle task,
        const char channelNames[],
        const float64 expirStateArray[],
        const int32 outputTypeArray[],
        uInt32 arraySize
    );
    x::errors::Error CfgWatchdogCOExpirStates(
        TaskHandle task,
        const char channelNames[],
        const int32 expirStateArray[],
        uInt32 arraySize
    );
    x::errors::Error CfgWatchdogDOExpirStates(
        TaskHandle task,
        const char channelNames[],
        const int32 expirStateArray[],
        uInt32 arraySize
    );
    x::errors::Error ClearTEDS(const char physicalChannel[]);
    x::errors::Error ClearTask(TaskHandle task);
    x::errors::Error ConfigureLogging(
        TaskHandle task,
        const char filePath[],
        int32 loggingMode,
        const char groupName[],
        int32 operation
    );
    x::errors::Error ConfigureTEDS(const char physicalChannel[], const char filePath[]);
    x::errors::Error ConnectTerms(
        const char sourceTerminal[],
        const char destinationTerminal[],
        int32 signalModifiers
    );
    x::errors::Error ControlWatchdogTask(TaskHandle task, int32 action);
    x::errors::Error CreateAIAccel4WireDCVoltageChan(
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
    );
    x::errors::Error CreateAIAccelChan(
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
    );
    x::errors::Error CreateAIAccelChargeChan(
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
    );
    x::errors::Error CreateAIBridgeChan(
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
    );
    x::errors::Error CreateAIChargeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    );
    x::errors::Error CreateAICurrentChan(
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
    );
    x::errors::Error CreateAICurrentRMSChan(
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
    );
    x::errors::Error CreateAIForceBridgePolynomialChan(
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
    );
    x::errors::Error CreateAIForceBridgeTableChan(
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
    );
    x::errors::Error CreateAIForceBridgeTwoPointLinChan(
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
    );
    x::errors::Error CreateAIForceIEPEChan(
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
    );
    x::errors::Error CreateAIFreqVoltageChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        float64 thresholdLevel,
        float64 hysteresis,
        const char customScaleName[]
    );
    x::errors::Error CreateAIMicrophoneChan(
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
    );
    x::errors::Error CreateAIPosEddyCurrProxProbeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        float64 sensitivity,
        int32 sensitivityUnits,
        const char customScaleName[]
    );
    x::errors::Error CreateAIPosLVDTChan(
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
    );
    x::errors::Error CreateAIPosRVDTChan(
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
    );
    x::errors::Error CreateAIPowerChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 voltageSetpoint,
        float64 currentSetpoint,
        bool32 outputEnable
    );
    x::errors::Error CreateAIPressureBridgePolynomialChan(
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
    );
    x::errors::Error CreateAIPressureBridgeTableChan(
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
    );
    x::errors::Error CreateAIPressureBridgeTwoPointLinChan(
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
    );
    x::errors::Error CreateAIRTDChan(
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
    );
    x::errors::Error CreateAIResistanceChan(
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
    );
    x::errors::Error CreateAIRosetteStrainGageChan(
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
    );
    x::errors::Error CreateAIStrainGageChan(
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
    );
    x::errors::Error CreateAITempBuiltInSensorChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 units
    );
    x::errors::Error CreateAIThrmcplChan(
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
    );
    x::errors::Error CreateAIThrmstrChanIex(
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
    );
    x::errors::Error CreateAIThrmstrChanVex(
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
    );
    x::errors::Error CreateAITorqueBridgePolynomialChan(
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
    );
    x::errors::Error CreateAITorqueBridgeTableChan(
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
    );
    x::errors::Error CreateAITorqueBridgeTwoPointLinChan(
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
    );
    x::errors::Error CreateAIVelocityIEPEChan(
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
    );
    x::errors::Error CreateAIVoltageChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    );
    x::errors::Error CreateAIVoltageChanWithExcit(
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
    );
    x::errors::Error CreateAIVoltageRMSChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    );
    x::errors::Error CreateAOCurrentChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    );
    x::errors::Error CreateAOFuncGenChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 type,
        float64 freq,
        float64 amplitude,
        float64 offset
    );
    x::errors::Error CreateAOVoltageChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    );
    x::errors::Error CreateCIAngEncoderChan(
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
    );
    x::errors::Error CreateCIAngVelocityChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 decodingType,
        int32 units,
        uInt32 pulsesPerRev,
        const char customScaleName[]
    );
    x::errors::Error CreateCICountEdgesChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        int32 edge,
        uInt32 initialCount,
        int32 countDirection
    );
    x::errors::Error CreateCIDutyCycleChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minFreq,
        float64 maxFreq,
        int32 edge,
        const char customScaleName[]
    );
    x::errors::Error CreateCIFreqChan(
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
    );
    x::errors::Error CreateCIGPSTimestampChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        int32 units,
        int32 syncMethod,
        const char customScaleName[]
    );
    x::errors::Error CreateCILinEncoderChan(
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
    );
    x::errors::Error CreateCILinVelocityChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 decodingType,
        int32 units,
        float64 distPerPulse,
        const char customScaleName[]
    );
    x::errors::Error CreateCIPeriodChan(
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
    );
    x::errors::Error CreateCIPulseChanFreq(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units
    );
    x::errors::Error CreateCIPulseChanTicks(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        const char sourceTerminal[],
        float64 minVal,
        float64 maxVal
    );
    x::errors::Error CreateCIPulseChanTime(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units
    );
    x::errors::Error CreateCIPulseWidthChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 startingEdge,
        const char customScaleName[]
    );
    x::errors::Error CreateCISemiPeriodChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    );
    x::errors::Error CreateCITwoEdgeSepChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 firstEdge,
        int32 secondEdge,
        const char customScaleName[]
    );
    x::errors::Error CreateCOPulseChanFreq(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        int32 units,
        int32 idleState,
        float64 initialDelay,
        float64 freq,
        float64 dutyCycle
    );
    x::errors::Error CreateCOPulseChanTicks(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        const char sourceTerminal[],
        int32 idleState,
        int32 initialDelay,
        int32 lowTicks,
        int32 highTicks
    );
    x::errors::Error CreateCOPulseChanTime(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        int32 units,
        int32 idleState,
        float64 initialDelay,
        float64 lowTime,
        float64 highTime
    );
    x::errors::Error CreateDIChan(
        TaskHandle task,
        const char lines[],
        const char nameToAssignToLines[],
        int32 lineGrouping
    );
    x::errors::Error CreateDOChan(
        TaskHandle task,
        const char lines[],
        const char nameToAssignToLines[],
        int32 lineGrouping
    );
    x::errors::Error CreateLinScale(
        const char name[],
        float64 slope,
        float64 yIntercept,
        int32 preScaledUnits,
        const char scaledUnits[]
    );
    x::errors::Error CreateMapScale(
        const char name[],
        float64 prescaledMin,
        float64 prescaledMax,
        float64 scaledMin,
        float64 scaledMax,
        int32 preScaledUnits,
        const char scaledUnits[]
    );
    x::errors::Error CreatePolynomialScale(
        const char name[],
        const float64 forwardCoeffs[],
        uInt32 numForwardCoeffsIn,
        const float64 reverseCoeffs[],
        uInt32 numReverseCoeffsIn,
        int32 preScaledUnits,
        const char scaledUnits[]
    );
    x::errors::Error CreateTEDSAIAccelChan(
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
    );
    x::errors::Error CreateTEDSAIBridgeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        const char customScaleName[]
    );
    x::errors::Error CreateTEDSAICurrentChan(
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
    );
    x::errors::Error CreateTEDSAIForceBridgeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        const char customScaleName[]
    );
    x::errors::Error CreateTEDSAIForceIEPEChan(
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
    );
    x::errors::Error CreateTEDSAIMicrophoneChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        int32 units,
        float64 maxSndPressLevel,
        int32 currentExcitSource,
        float64 currentExcitVal,
        const char customScaleName[]
    );
    x::errors::Error CreateTEDSAIPosLVDTChan(
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
    );
    x::errors::Error CreateTEDSAIPosRVDTChan(
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
    );
    x::errors::Error CreateTEDSAIPressureBridgeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        const char customScaleName[]
    );
    x::errors::Error CreateTEDSAIRTDChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 resistanceConfig,
        int32 currentExcitSource,
        float64 currentExcitVal
    );
    x::errors::Error CreateTEDSAIResistanceChan(
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
    );
    x::errors::Error CreateTEDSAIStrainGageChan(
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
    );
    x::errors::Error CreateTEDSAIThrmcplChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 cjcSource,
        float64 cjcVal,
        const char cjcChannel[]
    );
    x::errors::Error CreateTEDSAIThrmstrChanIex(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 resistanceConfig,
        int32 currentExcitSource,
        float64 currentExcitVal
    );
    x::errors::Error CreateTEDSAIThrmstrChanVex(
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
    );
    x::errors::Error CreateTEDSAITorqueBridgeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 voltageExcitSource,
        float64 voltageExcitVal,
        const char customScaleName[]
    );
    x::errors::Error CreateTEDSAIVoltageChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    );
    x::errors::Error CreateTEDSAIVoltageChanWithExcit(
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
    );
    x::errors::Error CreateTableScale(
        const char name[],
        const float64 prescaledVals[],
        uInt32 numPrescaledValsIn,
        const float64 scaledVals[],
        uInt32 numScaledValsIn,
        int32 preScaledUnits,
        const char scaledUnits[]
    );
    x::errors::Error CreateTask(const char sessionName[], TaskHandle *task);
    x::errors::Error CreateWatchdogTimerTaskEx(
        const char deviceName[],
        const char sessionName[],
        TaskHandle *task,
        float64 timeout
    );
    x::errors::Error DeleteNetworkDevice(const char deviceName[]);
    x::errors::Error DeleteSavedGlobalChan(const char channelName[]);
    x::errors::Error DeleteSavedScale(const char scaleName[]);
    x::errors::Error DeleteSavedTask(const char taskName[]);
    x::errors::Error DeviceSupportsCal(const char deviceName[], bool32 *calSupported);
    x::errors::Error DisableRefTrig(TaskHandle task);
    x::errors::Error DisableStartTrig(TaskHandle task);
    x::errors::Error
    DisconnectTerms(const char sourceTerminal[], const char destinationTerminal[]);
    x::errors::Error
    ExportSignal(TaskHandle task, int32 signalID, const char outputTerminal[]);
    x::errors::Error GetAIChanCalCalDate(
        TaskHandle task,
        const char channelName[],
        uInt32 *year,
        uInt32 *month,
        uInt32 *day,
        uInt32 *hour,
        uInt32 *minute
    );
    x::errors::Error GetAIChanCalExpDate(
        TaskHandle task,
        const char channelName[],
        uInt32 *year,
        uInt32 *month,
        uInt32 *day,
        uInt32 *hour,
        uInt32 *minute
    );
    x::errors::Error GetAnalogPowerUpStatesWithOutputType(
        const char channelNames[],
        float64 stateArray[],
        int32 channelTypeArray[],
        uInt32 *arraySize
    );
    x::errors::Error GetArmStartTrigTimestampVal(TaskHandle task, CVIAbsoluteTime *data);
    x::errors::Error GetArmStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime *data);
    x::errors::Error
    GetAutoConfiguredCDAQSyncConnections(char portList[], uInt32 portListSize);
    x::errors::Error
    GetBufferAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value);
    x::errors::Error
    GetCalInfoAttributeBool(const char deviceName[], int32 attribute, bool32 *value);
    x::errors::Error
    GetCalInfoAttributeDouble(const char deviceName[], int32 attribute, float64 *value);
    x::errors::Error GetCalInfoAttributeString(
        const char deviceName[],
        int32 attribute,
        char value[],
        uInt32 size
    );
    x::errors::Error
    GetCalInfoAttributeUInt32(const char deviceName[], int32 attribute, uInt32 *value);
    x::errors::Error GetChanAttributeBool(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        bool32 *value
    );
    x::errors::Error GetChanAttributeDouble(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        float64 *value
    );
    x::errors::Error GetChanAttributeDoubleArray(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        float64 value[],
        uInt32 size
    );
    x::errors::Error GetChanAttributeInt32(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        int32 *value
    );
    x::errors::Error GetChanAttributeString(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        char value[],
        uInt32 size
    );
    x::errors::Error GetChanAttributeUInt32(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        uInt32 *value
    );
    x::errors::Error
    GetDeviceAttributeBool(const char deviceName[], int32 attribute, bool32 *value);
    x::errors::Error
    GetDeviceAttributeDouble(const char deviceName[], int32 attribute, float64 *value);
    x::errors::Error GetDeviceAttributeDoubleArray(
        const char deviceName[],
        int32 attribute,
        float64 value[],
        uInt32 size
    );
    x::errors::Error
    GetDeviceAttributeInt32(const char deviceName[], int32 attribute, int32 *value);
    x::errors::Error GetDeviceAttributeInt32Array(
        const char deviceName[],
        int32 attribute,
        int32 value[],
        uInt32 size
    );
    x::errors::Error GetDeviceAttributeString(
        const char deviceName[],
        int32 attribute,
        char value[],
        uInt32 size
    );
    x::errors::Error
    GetDeviceAttributeUInt32(const char deviceName[], int32 attribute, uInt32 *value);
    x::errors::Error GetDeviceAttributeUInt32Array(
        const char deviceName[],
        int32 attribute,
        uInt32 value[],
        uInt32 size
    );
    x::errors::Error
    GetDigitalLogicFamilyPowerUpState(const char deviceName[], int32 *logicFamily);
    x::errors::Error GetDisconnectedCDAQSyncPorts(char portList[], uInt32 portListSize);
    x::errors::Error
    GetErrorString(int32 errorCode, char errorString[], uInt32 bufferSize);
    x::errors::Error
    GetExportedSignalAttributeBool(TaskHandle task, int32 attribute, bool32 *value);
    x::errors::Error
    GetExportedSignalAttributeDouble(TaskHandle task, int32 attribute, float64 *value);
    x::errors::Error
    GetExportedSignalAttributeInt32(TaskHandle task, int32 attribute, int32 *value);
    x::errors::Error GetExportedSignalAttributeString(
        TaskHandle task,
        int32 attribute,
        char value[],
        uInt32 size
    );
    x::errors::Error
    GetExportedSignalAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value);
    x::errors::Error GetExtCalLastDateAndTime(
        const char deviceName[],
        uInt32 *year,
        uInt32 *month,
        uInt32 *day,
        uInt32 *hour,
        uInt32 *minute
    );
    x::errors::Error GetExtendedErrorInfo(char errorString[], uInt32 bufferSize);
    x::errors::Error GetFirstSampClkWhen(TaskHandle task, CVIAbsoluteTime *data);
    x::errors::Error GetFirstSampTimestampVal(TaskHandle task, CVIAbsoluteTime *data);
    x::errors::Error
    GetNthTaskChannel(TaskHandle task, uInt32 index, char buffer[], int32 bufferSize);
    x::errors::Error
    GetNthTaskDevice(TaskHandle task, uInt32 index, char buffer[], int32 bufferSize);
    x::errors::Error GetNthTaskReadChannel(
        TaskHandle task,
        uInt32 index,
        char buffer[],
        int32 bufferSize
    );
    x::errors::Error
    GetPersistedChanAttributeBool(const char channel[], int32 attribute, bool32 *value);
    x::errors::Error GetPersistedChanAttributeString(
        const char channel[],
        int32 attribute,
        char value[],
        uInt32 size
    );
    x::errors::Error GetPersistedScaleAttributeBool(
        const char scaleName[],
        int32 attribute,
        bool32 *value
    );
    x::errors::Error GetPersistedScaleAttributeString(
        const char scaleName[],
        int32 attribute,
        char value[],
        uInt32 size
    );
    x::errors::Error GetPersistedTaskAttributeBool(
        const char taskName[],
        int32 attribute,
        bool32 *value
    );
    x::errors::Error GetPersistedTaskAttributeString(
        const char taskName[],
        int32 attribute,
        char value[],
        uInt32 size
    );
    x::errors::Error GetPhysicalChanAttributeBool(
        const char physicalChannel[],
        int32 attribute,
        bool32 *value
    );
    x::errors::Error GetPhysicalChanAttributeBytes(
        const char physicalChannel[],
        int32 attribute,
        uInt8 value[],
        uInt32 size
    );
    x::errors::Error GetPhysicalChanAttributeDouble(
        const char physicalChannel[],
        int32 attribute,
        float64 *value
    );
    x::errors::Error GetPhysicalChanAttributeDoubleArray(
        const char physicalChannel[],
        int32 attribute,
        float64 value[],
        uInt32 size
    );
    x::errors::Error GetPhysicalChanAttributeInt32(
        const char physicalChannel[],
        int32 attribute,
        int32 *value
    );
    x::errors::Error GetPhysicalChanAttributeInt32Array(
        const char physicalChannel[],
        int32 attribute,
        int32 value[],
        uInt32 size
    );
    x::errors::Error GetPhysicalChanAttributeString(
        const char physicalChannel[],
        int32 attribute,
        char value[],
        uInt32 size
    );
    x::errors::Error GetPhysicalChanAttributeUInt32(
        const char physicalChannel[],
        int32 attribute,
        uInt32 *value
    );
    x::errors::Error GetPhysicalChanAttributeUInt32Array(
        const char physicalChannel[],
        int32 attribute,
        uInt32 value[],
        uInt32 size
    );
    x::errors::Error
    GetReadAttributeBool(TaskHandle task, int32 attribute, bool32 *value);
    x::errors::Error
    GetReadAttributeDouble(TaskHandle task, int32 attribute, float64 *value);
    x::errors::Error
    GetReadAttributeInt32(TaskHandle task, int32 attribute, int32 *value);
    x::errors::Error
    GetReadAttributeString(TaskHandle task, int32 attribute, char value[], uInt32 size);
    x::errors::Error
    GetReadAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value);
    x::errors::Error
    GetReadAttributeUInt64(TaskHandle task, int32 attribute, uInt64 *value);
    x::errors::Error
    GetRealTimeAttributeBool(TaskHandle task, int32 attribute, bool32 *value);
    x::errors::Error
    GetRealTimeAttributeInt32(TaskHandle task, int32 attribute, int32 *value);
    x::errors::Error
    GetRealTimeAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value);
    x::errors::Error GetRefTrigTimestampVal(TaskHandle task, CVIAbsoluteTime *data);
    x::errors::Error
    GetScaleAttributeDouble(const char scaleName[], int32 attribute, float64 *value);
    x::errors::Error GetScaleAttributeDoubleArray(
        const char scaleName[],
        int32 attribute,
        float64 value[],
        uInt32 size
    );
    x::errors::Error
    GetScaleAttributeInt32(const char scaleName[], int32 attribute, int32 *value);
    x::errors::Error GetScaleAttributeString(
        const char scaleName[],
        int32 attribute,
        char value[],
        uInt32 size
    );
    x::errors::Error GetSelfCalLastDateAndTime(
        const char deviceName[],
        uInt32 *year,
        uInt32 *month,
        uInt32 *day,
        uInt32 *hour,
        uInt32 *minute
    );
    x::errors::Error GetStartTrigTimestampVal(TaskHandle task, CVIAbsoluteTime *data);
    x::errors::Error GetStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime *data);
    x::errors::Error GetSyncPulseTimeWhen(TaskHandle task, CVIAbsoluteTime *data);
    x::errors::Error
    GetSystemInfoAttributeString(int32 attribute, char value[], uInt32 size);
    x::errors::Error GetSystemInfoAttributeUInt32(int32 attribute, uInt32 *value);
    x::errors::Error
    GetTaskAttributeBool(TaskHandle task, int32 attribute, bool32 *value);
    x::errors::Error
    GetTaskAttributeString(TaskHandle task, int32 attribute, char value[], uInt32 size);
    x::errors::Error
    GetTaskAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value);
    x::errors::Error
    GetTimingAttributeBool(TaskHandle task, int32 attribute, bool32 *value);
    x::errors::Error
    GetTimingAttributeDouble(TaskHandle task, int32 attribute, float64 *value);
    x::errors::Error GetTimingAttributeExBool(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        bool32 *value
    );
    x::errors::Error GetTimingAttributeExDouble(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        float64 *value
    );
    x::errors::Error GetTimingAttributeExInt32(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        int32 *value
    );
    x::errors::Error GetTimingAttributeExString(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        char value[],
        uInt32 size
    );
    x::errors::Error GetTimingAttributeExTimestamp(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        CVIAbsoluteTime *value
    );
    x::errors::Error GetTimingAttributeExUInt32(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        uInt32 *value
    );
    x::errors::Error GetTimingAttributeExUInt64(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        uInt64 *value
    );
    x::errors::Error
    GetTimingAttributeInt32(TaskHandle task, int32 attribute, int32 *value);
    x::errors::Error GetTimingAttributeString(
        TaskHandle task,
        int32 attribute,
        char value[],
        uInt32 size
    );
    x::errors::Error GetTimingAttributeTimestamp(
        TaskHandle task,
        int32 attribute,
        CVIAbsoluteTime *value
    );
    x::errors::Error
    GetTimingAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value);
    x::errors::Error
    GetTimingAttributeUInt64(TaskHandle task, int32 attribute, uInt64 *value);
    x::errors::Error
    GetTrigAttributeBool(TaskHandle task, int32 attribute, bool32 *value);
    x::errors::Error
    GetTrigAttributeDouble(TaskHandle task, int32 attribute, float64 *value);
    x::errors::Error GetTrigAttributeDoubleArray(
        TaskHandle task,
        int32 attribute,
        float64 value[],
        uInt32 size
    );
    x::errors::Error
    GetTrigAttributeInt32(TaskHandle task, int32 attribute, int32 *value);
    x::errors::Error GetTrigAttributeInt32Array(
        TaskHandle task,
        int32 attribute,
        int32 value[],
        uInt32 size
    );
    x::errors::Error
    GetTrigAttributeString(TaskHandle task, int32 attribute, char value[], uInt32 size);
    x::errors::Error
    GetTrigAttributeTimestamp(TaskHandle task, int32 attribute, CVIAbsoluteTime *value);
    x::errors::Error
    GetTrigAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value);
    x::errors::Error GetWatchdogAttributeBool(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        bool32 *value
    );
    x::errors::Error GetWatchdogAttributeDouble(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        float64 *value
    );
    x::errors::Error GetWatchdogAttributeInt32(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        int32 *value
    );
    x::errors::Error GetWatchdogAttributeString(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        char value[],
        uInt32 size
    );
    x::errors::Error
    GetWriteAttributeBool(TaskHandle task, int32 attribute, bool32 *value);
    x::errors::Error
    GetWriteAttributeDouble(TaskHandle task, int32 attribute, float64 *value);
    x::errors::Error
    GetWriteAttributeInt32(TaskHandle task, int32 attribute, int32 *value);
    x::errors::Error GetWriteAttributeString(
        TaskHandle task,
        int32 attribute,
        char value[],
        uInt32 size
    );
    x::errors::Error
    GetWriteAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value);
    x::errors::Error
    GetWriteAttributeUInt64(TaskHandle task, int32 attribute, uInt64 *value);
    x::errors::Error IsTaskDone(TaskHandle task, bool32 *isTaskDone);
    x::errors::Error LoadTask(const char sessionName[], TaskHandle *task);
    x::errors::Error PerformBridgeOffsetNullingCalEx(
        TaskHandle task,
        const char channel[],
        bool32 skipUnsupportedChannels
    );
    x::errors::Error PerformBridgeShuntCalEx(
        TaskHandle task,
        const char channel[],
        float64 shuntResistorValue,
        int32 shuntResistorLocation,
        int32 shuntResistorSelect,
        int32 shuntResistorSource,
        float64 bridgeResistance,
        bool32 skipUnsupportedChannels
    );
    x::errors::Error PerformStrainShuntCalEx(
        TaskHandle task,
        const char channel[],
        float64 shuntResistorValue,
        int32 shuntResistorLocation,
        int32 shuntResistorSelect,
        int32 shuntResistorSource,
        bool32 skipUnsupportedChannels
    );
    x::errors::Error PerformThrmcplLeadOffsetNullingCal(
        TaskHandle task,
        const char channel[],
        bool32 skipUnsupportedChannels
    );
    x::errors::Error ReadAnalogF64(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        float64 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    x::errors::Error ReadAnalogScalarF64(
        TaskHandle task,
        float64 timeout,
        float64 *value,
        bool32 *reserved
    );
    x::errors::Error ReadBinaryI16(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        int16 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    x::errors::Error ReadBinaryI32(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        int32 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    x::errors::Error ReadBinaryU16(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt16 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    x::errors::Error ReadBinaryU32(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt32 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    x::errors::Error ReadCounterF64(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        float64 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    x::errors::Error ReadCounterF64Ex(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        float64 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    x::errors::Error ReadCounterScalarF64(
        TaskHandle task,
        float64 timeout,
        float64 *value,
        bool32 *reserved
    );
    x::errors::Error ReadCounterScalarU32(
        TaskHandle task,
        float64 timeout,
        uInt32 *value,
        bool32 *reserved
    );
    x::errors::Error ReadCounterU32(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        uInt32 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    x::errors::Error ReadCounterU32Ex(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt32 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    x::errors::Error ReadCtrFreq(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 interleaved,
        float64 readArrayFrequency[],
        float64 readArrayDutyCycle[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    x::errors::Error ReadCtrFreqScalar(
        TaskHandle task,
        float64 timeout,
        float64 *frequency,
        float64 *dutyCycle,
        bool32 *reserved
    );
    x::errors::Error ReadCtrTicks(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 interleaved,
        uInt32 readArrayHighTicks[],
        uInt32 readArrayLowTicks[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    x::errors::Error ReadCtrTicksScalar(
        TaskHandle task,
        float64 timeout,
        uInt32 *highTicks,
        uInt32 *lowTicks,
        bool32 *reserved
    );
    x::errors::Error ReadCtrTime(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 interleaved,
        float64 readArrayHighTime[],
        float64 readArrayLowTime[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    x::errors::Error ReadCtrTimeScalar(
        TaskHandle task,
        float64 timeout,
        float64 *highTime,
        float64 *lowTime,
        bool32 *reserved
    );
    x::errors::Error ReadDigitalLines(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt8 readArray[],
        uInt32 arraySizeInBytes,
        int32 *sampsPerChanRead,
        int32 *numBytesPerSamp,
        bool32 *reserved
    );
    x::errors::Error ReadDigitalScalarU32(
        TaskHandle task,
        float64 timeout,
        uInt32 *value,
        bool32 *reserved
    );
    x::errors::Error ReadDigitalU16(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt16 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    x::errors::Error ReadDigitalU32(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt32 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    x::errors::Error ReadDigitalU8(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt8 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    x::errors::Error ReadPowerBinaryI16(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        int16 readArrayVoltage[],
        int16 readArrayCurrent[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    x::errors::Error ReadPowerF64(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        float64 readArrayVoltage[],
        float64 readArrayCurrent[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    x::errors::Error ReadPowerScalarF64(
        TaskHandle task,
        float64 timeout,
        float64 *voltage,
        float64 *current,
        bool32 *reserved
    );
    x::errors::Error ReadRaw(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        uInt8 readArray[],
        uInt32 arraySizeInBytes,
        int32 *sampsRead,
        int32 *numBytesPerSamp,
        bool32 *reserved
    );
    x::errors::Error RegisterDoneEvent(
        TaskHandle task,
        uInt32 options,
        DAQmxDoneEventCallbackPtr callbackFunction,
        void *callbackData
    );
    x::errors::Error RegisterEveryNSamplesEvent(
        TaskHandle task,
        int32 everyNSamplesEventType,
        uInt32 nSamples,
        uInt32 options,
        DAQmxEveryNSamplesEventCallbackPtr callbackFunction,
        void *callbackData
    );
    x::errors::Error RegisterSignalEvent(
        TaskHandle task,
        int32 signalID,
        uInt32 options,
        DAQmxSignalEventCallbackPtr callbackFunction,
        void *callbackData
    );
    x::errors::Error RemoveCDAQSyncConnection(const char portList[]);
    x::errors::Error
    ReserveNetworkDevice(const char deviceName[], bool32 overrideReservation);
    x::errors::Error ResetBufferAttribute(TaskHandle task, int32 attribute);
    x::errors::Error
    ResetChanAttribute(TaskHandle task, const char channel[], int32 attribute);
    x::errors::Error ResetDevice(const char deviceName[]);
    x::errors::Error ResetExportedSignalAttribute(TaskHandle task, int32 attribute);
    x::errors::Error ResetReadAttribute(TaskHandle task, int32 attribute);
    x::errors::Error ResetRealTimeAttribute(TaskHandle task, int32 attribute);
    x::errors::Error ResetTimingAttribute(TaskHandle task, int32 attribute);
    x::errors::Error
    ResetTimingAttributeEx(TaskHandle task, const char deviceNames[], int32 attribute);
    x::errors::Error ResetTrigAttribute(TaskHandle task, int32 attribute);
    x::errors::Error
    ResetWatchdogAttribute(TaskHandle task, const char lines[], int32 attribute);
    x::errors::Error ResetWriteAttribute(TaskHandle task, int32 attribute);
    x::errors::Error RestoreLastExtCalConst(const char deviceName[]);
    x::errors::Error SaveGlobalChan(
        TaskHandle task,
        const char channelName[],
        const char saveAs[],
        const char author[],
        uInt32 options
    );
    x::errors::Error SaveScale(
        const char scaleName[],
        const char saveAs[],
        const char author[],
        uInt32 options
    );
    x::errors::Error
    SaveTask(TaskHandle task, const char saveAs[], const char author[], uInt32 options);
    x::errors::Error SelfCal(const char deviceName[]);
    x::errors::Error SelfTestDevice(const char deviceName[]);
    x::errors::Error SetAIChanCalCalDate(
        TaskHandle task,
        const char channelName[],
        uInt32 year,
        uInt32 month,
        uInt32 day,
        uInt32 hour,
        uInt32 minute
    );
    x::errors::Error SetAIChanCalExpDate(
        TaskHandle task,
        const char channelName[],
        uInt32 year,
        uInt32 month,
        uInt32 day,
        uInt32 hour,
        uInt32 minute
    );
    x::errors::Error SetAnalogPowerUpStatesWithOutputType(
        const char channelNames[],
        const float64 stateArray[],
        const int32 channelTypeArray[],
        uInt32 arraySize
    );
    x::errors::Error SetArmStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime data);
    x::errors::Error
    SetBufferAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value);
    x::errors::Error
    SetCalInfoAttributeBool(const char deviceName[], int32 attribute, bool32 value);
    x::errors::Error
    SetCalInfoAttributeDouble(const char deviceName[], int32 attribute, float64 value);
    x::errors::Error SetCalInfoAttributeString(
        const char deviceName[],
        int32 attribute,
        const char value[]
    );
    x::errors::Error
    SetCalInfoAttributeUInt32(const char deviceName[], int32 attribute, uInt32 value);
    x::errors::Error SetChanAttributeBool(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        bool32 value
    );
    x::errors::Error SetChanAttributeDouble(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        float64 value
    );
    x::errors::Error SetChanAttributeDoubleArray(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        const float64 value[],
        uInt32 size
    );
    x::errors::Error SetChanAttributeInt32(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        int32 value
    );
    x::errors::Error SetChanAttributeString(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        const char value[]
    );
    x::errors::Error SetChanAttributeUInt32(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        uInt32 value
    );
    x::errors::Error
    SetDigitalLogicFamilyPowerUpState(const char deviceName[], int32 logicFamily);
    x::errors::Error
    SetExportedSignalAttributeBool(TaskHandle task, int32 attribute, bool32 value);
    x::errors::Error
    SetExportedSignalAttributeDouble(TaskHandle task, int32 attribute, float64 value);
    x::errors::Error
    SetExportedSignalAttributeInt32(TaskHandle task, int32 attribute, int32 value);
    x::errors::Error SetExportedSignalAttributeString(
        TaskHandle task,
        int32 attribute,
        const char value[]
    );
    x::errors::Error
    SetExportedSignalAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value);
    x::errors::Error SetFirstSampClkWhen(TaskHandle task, CVIAbsoluteTime data);
    x::errors::Error SetReadAttributeBool(TaskHandle task, int32 attribute, bool32 value);
    x::errors::Error
    SetReadAttributeDouble(TaskHandle task, int32 attribute, float64 value);
    x::errors::Error SetReadAttributeInt32(TaskHandle task, int32 attribute, int32 value);
    x::errors::Error
    SetReadAttributeString(TaskHandle task, int32 attribute, const char value[]);
    x::errors::Error
    SetReadAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value);
    x::errors::Error
    SetReadAttributeUInt64(TaskHandle task, int32 attribute, uInt64 value);
    x::errors::Error
    SetRealTimeAttributeBool(TaskHandle task, int32 attribute, bool32 value);
    x::errors::Error
    SetRealTimeAttributeInt32(TaskHandle task, int32 attribute, int32 value);
    x::errors::Error
    SetRealTimeAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value);
    x::errors::Error SetRuntimeEnvironment(
        const char environment[],
        const char environmentVersion[],
        const char reserved1[],
        const char reserved2[]
    );
    x::errors::Error
    SetScaleAttributeDouble(const char scaleName[], int32 attribute, float64 value);
    x::errors::Error SetScaleAttributeDoubleArray(
        const char scaleName[],
        int32 attribute,
        const float64 value[],
        uInt32 size
    );
    x::errors::Error
    SetScaleAttributeInt32(const char scaleName[], int32 attribute, int32 value);
    x::errors::Error SetScaleAttributeString(
        const char scaleName[],
        int32 attribute,
        const char value[]
    );
    x::errors::Error SetStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime data);
    x::errors::Error SetSyncPulseTimeWhen(TaskHandle task, CVIAbsoluteTime data);
    x::errors::Error
    SetTimingAttributeBool(TaskHandle task, int32 attribute, bool32 value);
    x::errors::Error
    SetTimingAttributeDouble(TaskHandle task, int32 attribute, float64 value);
    x::errors::Error SetTimingAttributeExBool(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        bool32 value
    );
    x::errors::Error SetTimingAttributeExDouble(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        float64 value
    );
    x::errors::Error SetTimingAttributeExInt32(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        int32 value
    );
    x::errors::Error SetTimingAttributeExString(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        const char value[]
    );
    x::errors::Error SetTimingAttributeExTimestamp(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        CVIAbsoluteTime value
    );
    x::errors::Error SetTimingAttributeExUInt32(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        uInt32 value
    );
    x::errors::Error SetTimingAttributeExUInt64(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        uInt64 value
    );
    x::errors::Error
    SetTimingAttributeInt32(TaskHandle task, int32 attribute, int32 value);
    x::errors::Error
    SetTimingAttributeString(TaskHandle task, int32 attribute, const char value[]);
    x::errors::Error SetTimingAttributeTimestamp(
        TaskHandle task,
        int32 attribute,
        CVIAbsoluteTime value
    );
    x::errors::Error
    SetTimingAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value);
    x::errors::Error
    SetTimingAttributeUInt64(TaskHandle task, int32 attribute, uInt64 value);
    x::errors::Error SetTrigAttributeBool(TaskHandle task, int32 attribute, bool32 value);
    x::errors::Error
    SetTrigAttributeDouble(TaskHandle task, int32 attribute, float64 value);
    x::errors::Error SetTrigAttributeDoubleArray(
        TaskHandle task,
        int32 attribute,
        const float64 value[],
        uInt32 size
    );
    x::errors::Error SetTrigAttributeInt32(TaskHandle task, int32 attribute, int32 value);
    x::errors::Error SetTrigAttributeInt32Array(
        TaskHandle task,
        int32 attribute,
        const int32 value[],
        uInt32 size
    );
    x::errors::Error
    SetTrigAttributeString(TaskHandle task, int32 attribute, const char value[]);
    x::errors::Error
    SetTrigAttributeTimestamp(TaskHandle task, int32 attribute, CVIAbsoluteTime value);
    x::errors::Error
    SetTrigAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value);
    x::errors::Error SetWatchdogAttributeBool(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        bool32 value
    );
    x::errors::Error SetWatchdogAttributeDouble(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        float64 value
    );
    x::errors::Error SetWatchdogAttributeInt32(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        int32 value
    );
    x::errors::Error SetWatchdogAttributeString(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        const char value[]
    );
    x::errors::Error
    SetWriteAttributeBool(TaskHandle task, int32 attribute, bool32 value);
    x::errors::Error
    SetWriteAttributeDouble(TaskHandle task, int32 attribute, float64 value);
    x::errors::Error
    SetWriteAttributeInt32(TaskHandle task, int32 attribute, int32 value);
    x::errors::Error
    SetWriteAttributeString(TaskHandle task, int32 attribute, const char value[]);
    x::errors::Error
    SetWriteAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value);
    x::errors::Error
    SetWriteAttributeUInt64(TaskHandle task, int32 attribute, uInt64 value);
    x::errors::Error StartNewFile(TaskHandle task, const char filePath[]);
    x::errors::Error StartTask(TaskHandle task);
    x::errors::Error StopTask(TaskHandle task);
    x::errors::Error TaskControl(TaskHandle task, int32 action);
    x::errors::Error TristateOutputTerm(const char outputTerminal[]);
    x::errors::Error UnregisterDoneEvent(
        TaskHandle task,
        uInt32 options,
        DAQmxDoneEventCallbackPtr callbackFunction,
        void *callbackData
    );
    x::errors::Error UnregisterEveryNSamplesEvent(
        TaskHandle task,
        int32 everyNSamplesEventType,
        uInt32 nSamples,
        uInt32 options,
        DAQmxEveryNSamplesEventCallbackPtr callbackFunction,
        void *callbackData
    );
    x::errors::Error UnregisterSignalEvent(
        TaskHandle task,
        int32 signalID,
        uInt32 options,
        DAQmxSignalEventCallbackPtr callbackFunction,
        void *callbackData
    );
    x::errors::Error UnreserveNetworkDevice(const char deviceName[]);
    x::errors::Error
    WaitForNextSampleClock(TaskHandle task, float64 timeout, bool32 *isLate);
    x::errors::Error WaitForValidTimestamp(
        TaskHandle task,
        int32 timestampEvent,
        float64 timeout,
        CVIAbsoluteTime *timestamp
    );
    x::errors::Error WaitUntilTaskDone(TaskHandle task, float64 timeToWait);
    x::errors::Error WriteAnalogF64(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const float64 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    );
    x::errors::Error WriteAnalogScalarF64(
        TaskHandle task,
        bool32 autoStart,
        float64 timeout,
        float64 value,
        bool32 *reserved
    );
    x::errors::Error WriteBinaryI16(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const int16 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    );
    x::errors::Error WriteBinaryI32(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const int32 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    );
    x::errors::Error WriteBinaryU16(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt16 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    );
    x::errors::Error WriteBinaryU32(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt32 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    );
    x::errors::Error WriteCtrFreq(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const float64 frequency[],
        const float64 dutyCycle[],
        int32 *numSampsPerChanWritten,
        bool32 *reserved
    );
    x::errors::Error WriteCtrFreqScalar(
        TaskHandle task,
        bool32 autoStart,
        float64 timeout,
        float64 frequency,
        float64 dutyCycle,
        bool32 *reserved
    );
    x::errors::Error WriteCtrTicks(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt32 highTicks[],
        const uInt32 lowTicks[],
        int32 *numSampsPerChanWritten,
        bool32 *reserved
    );
    x::errors::Error WriteCtrTicksScalar(
        TaskHandle task,
        bool32 autoStart,
        float64 timeout,
        uInt32 highTicks,
        uInt32 lowTicks,
        bool32 *reserved
    );
    x::errors::Error WriteCtrTime(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const float64 highTime[],
        const float64 lowTime[],
        int32 *numSampsPerChanWritten,
        bool32 *reserved
    );
    x::errors::Error WriteCtrTimeScalar(
        TaskHandle task,
        bool32 autoStart,
        float64 timeout,
        float64 highTime,
        float64 lowTime,
        bool32 *reserved
    );
    x::errors::Error WriteDigitalLines(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt8 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    );
    x::errors::Error WriteDigitalScalarU32(
        TaskHandle task,
        bool32 autoStart,
        float64 timeout,
        uInt32 value,
        bool32 *reserved
    );
    x::errors::Error WriteDigitalU16(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt16 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    );
    x::errors::Error WriteDigitalU32(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt32 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    );
    x::errors::Error WriteDigitalU8(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt8 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    );
    x::errors::Error WriteRaw(
        TaskHandle task,
        int32 numSamps,
        bool32 autoStart,
        float64 timeout,
        const uInt8 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    );
    x::errors::Error WriteToTEDSFromArray(
        const char physicalChannel[],
        const uInt8 bitStream[],
        uInt32 arraySize,
        int32 basicTEDSOptions
    );
    x::errors::Error WriteToTEDSFromFile(
        const char physicalChannel[],
        const char filePath[],
        int32 basicTEDSOptions
    );
    x::errors::Error SetReadRelativeTo(TaskHandle taskHandle, int32 data);
    x::errors::Error SetReadOffset(TaskHandle taskHandle, int32 data);
    x::errors::Error SetReadOverWrite(TaskHandle taskHandle, int32 data);
    x::errors::Error GetReadTotalSampPerChanAcquired(TaskHandle taskHandle, uInt64 *data);
};
}
