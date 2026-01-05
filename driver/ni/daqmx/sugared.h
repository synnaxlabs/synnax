// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#pragma once

#include "x/cpp/xerrors/errors.h"

#include "driver/errors/errors.h"
#include "driver/ni/daqmx/api.h"

namespace daqmx {
const xerrors::Error CRITICAL_ERROR = driver::CRITICAL_HARDWARE_ERROR.sub("ni");
const xerrors::Error TEMPORARY_ERROR = driver::TEMPORARY_HARDWARE_ERROR.sub("ni");
const xerrors::Error FIELD_ERROR = CRITICAL_ERROR.sub("field");
const xerrors::Error ANALOG_WRITE_OUT_OF_BOUNDS = CRITICAL_ERROR.sub("200561");
const xerrors::Error APPLICATION_TOO_SLOW = CRITICAL_ERROR.sub("200729");
const xerrors::Error DEVICE_DISCONNECTED = CRITICAL_ERROR.sub("88710");
const xerrors::Error RESOURCE_NOT_AVAILABLE = CRITICAL_ERROR.sub("88708");
const xerrors::Error DEVICE_DISCONNECTED_2 = CRITICAL_ERROR.sub("88709");
const xerrors::Error ADC_CONVERSION_ERROR = CRITICAL_ERROR.sub("200019");
const xerrors::Error RESOURCE_RESERVED = CRITICAL_ERROR.sub("201105");
const xerrors::Error ROUTING_ERROR = CRITICAL_ERROR.sub("89130");
const auto TEMPORARILY_UNREACHABLE = xerrors::Error(
    TEMPORARY_ERROR,
    "The device is temporarily unreachable. Will keep trying"
);
const auto REQUIRES_RESTART = xerrors::Error(
    TEMPORARILY_UNREACHABLE,
    "Restarting task to recover"
);

class SugaredAPI {
    std::shared_ptr<API> dmx;

    [[nodiscard]] xerrors::Error process_error(int32 status) const;

public:
    explicit SugaredAPI(std::shared_ptr<API> dmx): dmx(std::move(dmx)) {}

    xerrors::Error AddCDAQSyncConnection(const char portList[]);
    xerrors::Error AddGlobalChansToTask(TaskHandle task, const char channelNames[]);
    xerrors::Error AddNetworkDevice(
        const char ipAddress[],
        const char deviceName[],
        bool32 attemptReservation,
        float64 timeout,
        char deviceNameOut[],
        uInt32 deviceNameOutBufferSize
    );
    xerrors::Error AreConfiguredCDAQSyncPortsDisconnected(
        const char chassisDevicesPorts[],
        float64 timeout,
        bool32 *disconnectedPortsExist
    );
    xerrors::Error
    AutoConfigureCDAQSyncConnections(const char chassisDevicesPorts[], float64 timeout);
    xerrors::Error CalculateReversePolyCoeff(
        const float64 forwardCoeffs[],
        uInt32 numForwardCoeffsIn,
        float64 minValX,
        float64 maxValX,
        int32 numPointsToCompute,
        int32 reversePolyOrder,
        float64 reverseCoeffs[]
    );
    xerrors::Error CfgAnlgEdgeRefTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerSlope,
        float64 triggerLevel,
        uInt32 pretriggerSamples
    );
    xerrors::Error CfgAnlgEdgeStartTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerSlope,
        float64 triggerLevel
    );
    xerrors::Error CfgAnlgMultiEdgeRefTrig(
        TaskHandle task,
        const char triggerSources[],
        const int32 triggerSlopeArray[],
        const float64 triggerLevelArray[],
        uInt32 pretriggerSamples,
        uInt32 arraySize
    );
    xerrors::Error CfgAnlgMultiEdgeStartTrig(
        TaskHandle task,
        const char triggerSources[],
        const int32 triggerSlopeArray[],
        const float64 triggerLevelArray[],
        uInt32 arraySize
    );
    xerrors::Error CfgAnlgWindowRefTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerWhen,
        float64 windowTop,
        float64 windowBottom,
        uInt32 pretriggerSamples
    );
    xerrors::Error CfgAnlgWindowStartTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerWhen,
        float64 windowTop,
        float64 windowBottom
    );
    xerrors::Error CfgBurstHandshakingTimingExportClock(
        TaskHandle task,
        int32 sampleMode,
        uInt64 sampsPerChan,
        float64 sampleClkRate,
        const char sampleClkOutpTerm[],
        int32 sampleClkPulsePolarity,
        int32 pauseWhen,
        int32 readyEventActiveLevel
    );
    xerrors::Error CfgBurstHandshakingTimingImportClock(
        TaskHandle task,
        int32 sampleMode,
        uInt64 sampsPerChan,
        float64 sampleClkRate,
        const char sampleClkSrc[],
        int32 sampleClkActiveEdge,
        int32 pauseWhen,
        int32 readyEventActiveLevel
    );
    xerrors::Error CfgChangeDetectionTiming(
        TaskHandle task,
        const char risingEdgeChan[],
        const char fallingEdgeChan[],
        int32 sampleMode,
        uInt64 sampsPerChan
    );
    xerrors::Error CfgDigEdgeRefTrig(
        TaskHandle task,
        const char triggerSource[],
        int32 triggerEdge,
        uInt32 pretriggerSamples
    );
    xerrors::Error
    CfgDigEdgeStartTrig(TaskHandle task, const char triggerSource[], int32 triggerEdge);
    xerrors::Error CfgDigPatternRefTrig(
        TaskHandle task,
        const char triggerSource[],
        const char triggerPattern[],
        int32 triggerWhen,
        uInt32 pretriggerSamples
    );
    xerrors::Error CfgDigPatternStartTrig(
        TaskHandle task,
        const char triggerSource[],
        const char triggerPattern[],
        int32 triggerWhen
    );
    xerrors::Error
    CfgHandshakingTiming(TaskHandle task, int32 sampleMode, uInt64 sampsPerChan);
    xerrors::Error
    CfgImplicitTiming(TaskHandle task, int32 sampleMode, uInt64 sampsPerChan);
    xerrors::Error CfgInputBuffer(TaskHandle task, uInt32 numSampsPerChan);
    xerrors::Error CfgOutputBuffer(TaskHandle task, uInt32 numSampsPerChan);
    xerrors::Error CfgPipelinedSampClkTiming(
        TaskHandle task,
        const char source[],
        float64 rate,
        int32 activeEdge,
        int32 sampleMode,
        uInt64 sampsPerChan
    );
    xerrors::Error CfgSampClkTiming(
        TaskHandle task,
        const char source[],
        float64 rate,
        int32 activeEdge,
        int32 sampleMode,
        uInt64 sampsPerChan
    );
    xerrors::Error
    CfgTimeStartTrig(TaskHandle task, CVIAbsoluteTime when, int32 timescale);
    xerrors::Error CfgWatchdogAOExpirStates(
        TaskHandle task,
        const char channelNames[],
        const float64 expirStateArray[],
        const int32 outputTypeArray[],
        uInt32 arraySize
    );
    xerrors::Error CfgWatchdogCOExpirStates(
        TaskHandle task,
        const char channelNames[],
        const int32 expirStateArray[],
        uInt32 arraySize
    );
    xerrors::Error CfgWatchdogDOExpirStates(
        TaskHandle task,
        const char channelNames[],
        const int32 expirStateArray[],
        uInt32 arraySize
    );
    xerrors::Error ClearTEDS(const char physicalChannel[]);
    xerrors::Error ClearTask(TaskHandle task);
    xerrors::Error ConfigureLogging(
        TaskHandle task,
        const char filePath[],
        int32 loggingMode,
        const char groupName[],
        int32 operation
    );
    xerrors::Error ConfigureTEDS(const char physicalChannel[], const char filePath[]);
    xerrors::Error ConnectTerms(
        const char sourceTerminal[],
        const char destinationTerminal[],
        int32 signalModifiers
    );
    xerrors::Error ControlWatchdogTask(TaskHandle task, int32 action);
    xerrors::Error CreateAIAccel4WireDCVoltageChan(
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
    xerrors::Error CreateAIAccelChan(
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
    xerrors::Error CreateAIAccelChargeChan(
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
    xerrors::Error CreateAIBridgeChan(
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
    xerrors::Error CreateAIChargeChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    );
    xerrors::Error CreateAICurrentChan(
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
    xerrors::Error CreateAICurrentRMSChan(
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
    xerrors::Error CreateAIForceBridgePolynomialChan(
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
    xerrors::Error CreateAIForceBridgeTableChan(
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
    xerrors::Error CreateAIForceBridgeTwoPointLinChan(
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
    xerrors::Error CreateAIForceIEPEChan(
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
    xerrors::Error CreateAIFreqVoltageChan(
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
    xerrors::Error CreateAIMicrophoneChan(
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
    xerrors::Error CreateAIPosEddyCurrProxProbeChan(
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
    xerrors::Error CreateAIPosLVDTChan(
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
    xerrors::Error CreateAIPosRVDTChan(
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
    xerrors::Error CreateAIPowerChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 voltageSetpoint,
        float64 currentSetpoint,
        bool32 outputEnable
    );
    xerrors::Error CreateAIPressureBridgePolynomialChan(
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
    xerrors::Error CreateAIPressureBridgeTableChan(
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
    xerrors::Error CreateAIPressureBridgeTwoPointLinChan(
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
    xerrors::Error CreateAIRTDChan(
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
    xerrors::Error CreateAIResistanceChan(
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
    xerrors::Error CreateAIRosetteStrainGageChan(
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
    xerrors::Error CreateAIStrainGageChan(
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
    xerrors::Error CreateAITempBuiltInSensorChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 units
    );
    xerrors::Error CreateAIThrmcplChan(
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
    xerrors::Error CreateAIThrmstrChanIex(
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
    xerrors::Error CreateAIThrmstrChanVex(
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
    xerrors::Error CreateAITorqueBridgePolynomialChan(
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
    xerrors::Error CreateAITorqueBridgeTableChan(
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
    xerrors::Error CreateAITorqueBridgeTwoPointLinChan(
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
    xerrors::Error CreateAIVelocityIEPEChan(
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
    xerrors::Error CreateAIVoltageChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    );
    xerrors::Error CreateAIVoltageChanWithExcit(
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
    xerrors::Error CreateAIVoltageRMSChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    );
    xerrors::Error CreateAOCurrentChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    );
    xerrors::Error CreateAOFuncGenChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 type,
        float64 freq,
        float64 amplitude,
        float64 offset
    );
    xerrors::Error CreateAOVoltageChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    );
    xerrors::Error CreateCIAngEncoderChan(
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
    xerrors::Error CreateCIAngVelocityChan(
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
    xerrors::Error CreateCICountEdgesChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        int32 edge,
        uInt32 initialCount,
        int32 countDirection
    );
    xerrors::Error CreateCIDutyCycleChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minFreq,
        float64 maxFreq,
        int32 edge,
        const char customScaleName[]
    );
    xerrors::Error CreateCIFreqChan(
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
    xerrors::Error CreateCIGPSTimestampChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        int32 units,
        int32 syncMethod,
        const char customScaleName[]
    );
    xerrors::Error CreateCILinEncoderChan(
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
    xerrors::Error CreateCILinVelocityChan(
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
    xerrors::Error CreateCIPeriodChan(
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
    xerrors::Error CreateCIPulseChanFreq(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units
    );
    xerrors::Error CreateCIPulseChanTicks(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        const char sourceTerminal[],
        float64 minVal,
        float64 maxVal
    );
    xerrors::Error CreateCIPulseChanTime(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units
    );
    xerrors::Error CreateCIPulseWidthChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        int32 startingEdge,
        const char customScaleName[]
    );
    xerrors::Error CreateCISemiPeriodChan(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    );
    xerrors::Error CreateCITwoEdgeSepChan(
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
    xerrors::Error CreateCOPulseChanFreq(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        int32 units,
        int32 idleState,
        float64 initialDelay,
        float64 freq,
        float64 dutyCycle
    );
    xerrors::Error CreateCOPulseChanTicks(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        const char sourceTerminal[],
        int32 idleState,
        int32 initialDelay,
        int32 lowTicks,
        int32 highTicks
    );
    xerrors::Error CreateCOPulseChanTime(
        TaskHandle task,
        const char counter[],
        const char nameToAssignToChannel[],
        int32 units,
        int32 idleState,
        float64 initialDelay,
        float64 lowTime,
        float64 highTime
    );
    xerrors::Error CreateDIChan(
        TaskHandle task,
        const char lines[],
        const char nameToAssignToLines[],
        int32 lineGrouping
    );
    xerrors::Error CreateDOChan(
        TaskHandle task,
        const char lines[],
        const char nameToAssignToLines[],
        int32 lineGrouping
    );
    xerrors::Error CreateLinScale(
        const char name[],
        float64 slope,
        float64 yIntercept,
        int32 preScaledUnits,
        const char scaledUnits[]
    );
    xerrors::Error CreateMapScale(
        const char name[],
        float64 prescaledMin,
        float64 prescaledMax,
        float64 scaledMin,
        float64 scaledMax,
        int32 preScaledUnits,
        const char scaledUnits[]
    );
    xerrors::Error CreatePolynomialScale(
        const char name[],
        const float64 forwardCoeffs[],
        uInt32 numForwardCoeffsIn,
        const float64 reverseCoeffs[],
        uInt32 numReverseCoeffsIn,
        int32 preScaledUnits,
        const char scaledUnits[]
    );
    xerrors::Error CreateTEDSAIAccelChan(
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
    xerrors::Error CreateTEDSAIBridgeChan(
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
    xerrors::Error CreateTEDSAICurrentChan(
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
    xerrors::Error CreateTEDSAIForceBridgeChan(
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
    xerrors::Error CreateTEDSAIForceIEPEChan(
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
    xerrors::Error CreateTEDSAIMicrophoneChan(
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
    xerrors::Error CreateTEDSAIPosLVDTChan(
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
    xerrors::Error CreateTEDSAIPosRVDTChan(
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
    xerrors::Error CreateTEDSAIPressureBridgeChan(
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
    xerrors::Error CreateTEDSAIRTDChan(
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
    xerrors::Error CreateTEDSAIResistanceChan(
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
    xerrors::Error CreateTEDSAIStrainGageChan(
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
    xerrors::Error CreateTEDSAIThrmcplChan(
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
    xerrors::Error CreateTEDSAIThrmstrChanIex(
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
    xerrors::Error CreateTEDSAIThrmstrChanVex(
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
    xerrors::Error CreateTEDSAITorqueBridgeChan(
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
    xerrors::Error CreateTEDSAIVoltageChan(
        TaskHandle task,
        const char physicalChannel[],
        const char nameToAssignToChannel[],
        int32 terminalConfig,
        float64 minVal,
        float64 maxVal,
        int32 units,
        const char customScaleName[]
    );
    xerrors::Error CreateTEDSAIVoltageChanWithExcit(
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
    xerrors::Error CreateTableScale(
        const char name[],
        const float64 prescaledVals[],
        uInt32 numPrescaledValsIn,
        const float64 scaledVals[],
        uInt32 numScaledValsIn,
        int32 preScaledUnits,
        const char scaledUnits[]
    );
    xerrors::Error CreateTask(const char sessionName[], TaskHandle *task);
    xerrors::Error CreateWatchdogTimerTaskEx(
        const char deviceName[],
        const char sessionName[],
        TaskHandle *task,
        float64 timeout
    );
    xerrors::Error DeleteNetworkDevice(const char deviceName[]);
    xerrors::Error DeleteSavedGlobalChan(const char channelName[]);
    xerrors::Error DeleteSavedScale(const char scaleName[]);
    xerrors::Error DeleteSavedTask(const char taskName[]);
    xerrors::Error DeviceSupportsCal(const char deviceName[], bool32 *calSupported);
    xerrors::Error DisableRefTrig(TaskHandle task);
    xerrors::Error DisableStartTrig(TaskHandle task);
    xerrors::Error
    DisconnectTerms(const char sourceTerminal[], const char destinationTerminal[]);
    xerrors::Error
    ExportSignal(TaskHandle task, int32 signalID, const char outputTerminal[]);
    xerrors::Error GetAIChanCalCalDate(
        TaskHandle task,
        const char channelName[],
        uInt32 *year,
        uInt32 *month,
        uInt32 *day,
        uInt32 *hour,
        uInt32 *minute
    );
    xerrors::Error GetAIChanCalExpDate(
        TaskHandle task,
        const char channelName[],
        uInt32 *year,
        uInt32 *month,
        uInt32 *day,
        uInt32 *hour,
        uInt32 *minute
    );
    xerrors::Error GetAnalogPowerUpStatesWithOutputType(
        const char channelNames[],
        float64 stateArray[],
        int32 channelTypeArray[],
        uInt32 *arraySize
    );
    xerrors::Error GetArmStartTrigTimestampVal(TaskHandle task, CVIAbsoluteTime *data);
    xerrors::Error GetArmStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime *data);
    xerrors::Error
    GetAutoConfiguredCDAQSyncConnections(char portList[], uInt32 portListSize);
    xerrors::Error
    GetBufferAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value);
    xerrors::Error
    GetCalInfoAttributeBool(const char deviceName[], int32 attribute, bool32 *value);
    xerrors::Error
    GetCalInfoAttributeDouble(const char deviceName[], int32 attribute, float64 *value);
    xerrors::Error GetCalInfoAttributeString(
        const char deviceName[],
        int32 attribute,
        char value[],
        uInt32 size
    );
    xerrors::Error
    GetCalInfoAttributeUInt32(const char deviceName[], int32 attribute, uInt32 *value);
    xerrors::Error GetChanAttributeBool(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        bool32 *value
    );
    xerrors::Error GetChanAttributeDouble(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        float64 *value
    );
    xerrors::Error GetChanAttributeDoubleArray(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        float64 value[],
        uInt32 size
    );
    xerrors::Error GetChanAttributeInt32(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        int32 *value
    );
    xerrors::Error GetChanAttributeString(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        char value[],
        uInt32 size
    );
    xerrors::Error GetChanAttributeUInt32(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        uInt32 *value
    );
    xerrors::Error
    GetDeviceAttributeBool(const char deviceName[], int32 attribute, bool32 *value);
    xerrors::Error
    GetDeviceAttributeDouble(const char deviceName[], int32 attribute, float64 *value);
    xerrors::Error GetDeviceAttributeDoubleArray(
        const char deviceName[],
        int32 attribute,
        float64 value[],
        uInt32 size
    );
    xerrors::Error
    GetDeviceAttributeInt32(const char deviceName[], int32 attribute, int32 *value);
    xerrors::Error GetDeviceAttributeInt32Array(
        const char deviceName[],
        int32 attribute,
        int32 value[],
        uInt32 size
    );
    xerrors::Error GetDeviceAttributeString(
        const char deviceName[],
        int32 attribute,
        char value[],
        uInt32 size
    );
    xerrors::Error
    GetDeviceAttributeUInt32(const char deviceName[], int32 attribute, uInt32 *value);
    xerrors::Error GetDeviceAttributeUInt32Array(
        const char deviceName[],
        int32 attribute,
        uInt32 value[],
        uInt32 size
    );
    xerrors::Error
    GetDigitalLogicFamilyPowerUpState(const char deviceName[], int32 *logicFamily);
    xerrors::Error GetDisconnectedCDAQSyncPorts(char portList[], uInt32 portListSize);
    xerrors::Error
    GetErrorString(int32 errorCode, char errorString[], uInt32 bufferSize);
    xerrors::Error
    GetExportedSignalAttributeBool(TaskHandle task, int32 attribute, bool32 *value);
    xerrors::Error
    GetExportedSignalAttributeDouble(TaskHandle task, int32 attribute, float64 *value);
    xerrors::Error
    GetExportedSignalAttributeInt32(TaskHandle task, int32 attribute, int32 *value);
    xerrors::Error GetExportedSignalAttributeString(
        TaskHandle task,
        int32 attribute,
        char value[],
        uInt32 size
    );
    xerrors::Error
    GetExportedSignalAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value);
    xerrors::Error GetExtCalLastDateAndTime(
        const char deviceName[],
        uInt32 *year,
        uInt32 *month,
        uInt32 *day,
        uInt32 *hour,
        uInt32 *minute
    );
    xerrors::Error GetExtendedErrorInfo(char errorString[], uInt32 bufferSize);
    xerrors::Error GetFirstSampClkWhen(TaskHandle task, CVIAbsoluteTime *data);
    xerrors::Error GetFirstSampTimestampVal(TaskHandle task, CVIAbsoluteTime *data);
    xerrors::Error
    GetNthTaskChannel(TaskHandle task, uInt32 index, char buffer[], int32 bufferSize);
    xerrors::Error
    GetNthTaskDevice(TaskHandle task, uInt32 index, char buffer[], int32 bufferSize);
    xerrors::Error GetNthTaskReadChannel(
        TaskHandle task,
        uInt32 index,
        char buffer[],
        int32 bufferSize
    );
    xerrors::Error
    GetPersistedChanAttributeBool(const char channel[], int32 attribute, bool32 *value);
    xerrors::Error GetPersistedChanAttributeString(
        const char channel[],
        int32 attribute,
        char value[],
        uInt32 size
    );
    xerrors::Error GetPersistedScaleAttributeBool(
        const char scaleName[],
        int32 attribute,
        bool32 *value
    );
    xerrors::Error GetPersistedScaleAttributeString(
        const char scaleName[],
        int32 attribute,
        char value[],
        uInt32 size
    );
    xerrors::Error GetPersistedTaskAttributeBool(
        const char taskName[],
        int32 attribute,
        bool32 *value
    );
    xerrors::Error GetPersistedTaskAttributeString(
        const char taskName[],
        int32 attribute,
        char value[],
        uInt32 size
    );
    xerrors::Error GetPhysicalChanAttributeBool(
        const char physicalChannel[],
        int32 attribute,
        bool32 *value
    );
    xerrors::Error GetPhysicalChanAttributeBytes(
        const char physicalChannel[],
        int32 attribute,
        uInt8 value[],
        uInt32 size
    );
    xerrors::Error GetPhysicalChanAttributeDouble(
        const char physicalChannel[],
        int32 attribute,
        float64 *value
    );
    xerrors::Error GetPhysicalChanAttributeDoubleArray(
        const char physicalChannel[],
        int32 attribute,
        float64 value[],
        uInt32 size
    );
    xerrors::Error GetPhysicalChanAttributeInt32(
        const char physicalChannel[],
        int32 attribute,
        int32 *value
    );
    xerrors::Error GetPhysicalChanAttributeInt32Array(
        const char physicalChannel[],
        int32 attribute,
        int32 value[],
        uInt32 size
    );
    xerrors::Error GetPhysicalChanAttributeString(
        const char physicalChannel[],
        int32 attribute,
        char value[],
        uInt32 size
    );
    xerrors::Error GetPhysicalChanAttributeUInt32(
        const char physicalChannel[],
        int32 attribute,
        uInt32 *value
    );
    xerrors::Error GetPhysicalChanAttributeUInt32Array(
        const char physicalChannel[],
        int32 attribute,
        uInt32 value[],
        uInt32 size
    );
    xerrors::Error
    GetReadAttributeBool(TaskHandle task, int32 attribute, bool32 *value);
    xerrors::Error
    GetReadAttributeDouble(TaskHandle task, int32 attribute, float64 *value);
    xerrors::Error
    GetReadAttributeInt32(TaskHandle task, int32 attribute, int32 *value);
    xerrors::Error
    GetReadAttributeString(TaskHandle task, int32 attribute, char value[], uInt32 size);
    xerrors::Error
    GetReadAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value);
    xerrors::Error
    GetReadAttributeUInt64(TaskHandle task, int32 attribute, uInt64 *value);
    xerrors::Error
    GetRealTimeAttributeBool(TaskHandle task, int32 attribute, bool32 *value);
    xerrors::Error
    GetRealTimeAttributeInt32(TaskHandle task, int32 attribute, int32 *value);
    xerrors::Error
    GetRealTimeAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value);
    xerrors::Error GetRefTrigTimestampVal(TaskHandle task, CVIAbsoluteTime *data);
    xerrors::Error
    GetScaleAttributeDouble(const char scaleName[], int32 attribute, float64 *value);
    xerrors::Error GetScaleAttributeDoubleArray(
        const char scaleName[],
        int32 attribute,
        float64 value[],
        uInt32 size
    );
    xerrors::Error
    GetScaleAttributeInt32(const char scaleName[], int32 attribute, int32 *value);
    xerrors::Error GetScaleAttributeString(
        const char scaleName[],
        int32 attribute,
        char value[],
        uInt32 size
    );
    xerrors::Error GetSelfCalLastDateAndTime(
        const char deviceName[],
        uInt32 *year,
        uInt32 *month,
        uInt32 *day,
        uInt32 *hour,
        uInt32 *minute
    );
    xerrors::Error GetStartTrigTimestampVal(TaskHandle task, CVIAbsoluteTime *data);
    xerrors::Error GetStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime *data);
    xerrors::Error GetSyncPulseTimeWhen(TaskHandle task, CVIAbsoluteTime *data);
    xerrors::Error
    GetSystemInfoAttributeString(int32 attribute, char value[], uInt32 size);
    xerrors::Error GetSystemInfoAttributeUInt32(int32 attribute, uInt32 *value);
    xerrors::Error
    GetTaskAttributeBool(TaskHandle task, int32 attribute, bool32 *value);
    xerrors::Error
    GetTaskAttributeString(TaskHandle task, int32 attribute, char value[], uInt32 size);
    xerrors::Error
    GetTaskAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value);
    xerrors::Error
    GetTimingAttributeBool(TaskHandle task, int32 attribute, bool32 *value);
    xerrors::Error
    GetTimingAttributeDouble(TaskHandle task, int32 attribute, float64 *value);
    xerrors::Error GetTimingAttributeExBool(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        bool32 *value
    );
    xerrors::Error GetTimingAttributeExDouble(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        float64 *value
    );
    xerrors::Error GetTimingAttributeExInt32(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        int32 *value
    );
    xerrors::Error GetTimingAttributeExString(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        char value[],
        uInt32 size
    );
    xerrors::Error GetTimingAttributeExTimestamp(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        CVIAbsoluteTime *value
    );
    xerrors::Error GetTimingAttributeExUInt32(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        uInt32 *value
    );
    xerrors::Error GetTimingAttributeExUInt64(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        uInt64 *value
    );
    xerrors::Error
    GetTimingAttributeInt32(TaskHandle task, int32 attribute, int32 *value);
    xerrors::Error GetTimingAttributeString(
        TaskHandle task,
        int32 attribute,
        char value[],
        uInt32 size
    );
    xerrors::Error GetTimingAttributeTimestamp(
        TaskHandle task,
        int32 attribute,
        CVIAbsoluteTime *value
    );
    xerrors::Error
    GetTimingAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value);
    xerrors::Error
    GetTimingAttributeUInt64(TaskHandle task, int32 attribute, uInt64 *value);
    xerrors::Error
    GetTrigAttributeBool(TaskHandle task, int32 attribute, bool32 *value);
    xerrors::Error
    GetTrigAttributeDouble(TaskHandle task, int32 attribute, float64 *value);
    xerrors::Error GetTrigAttributeDoubleArray(
        TaskHandle task,
        int32 attribute,
        float64 value[],
        uInt32 size
    );
    xerrors::Error
    GetTrigAttributeInt32(TaskHandle task, int32 attribute, int32 *value);
    xerrors::Error GetTrigAttributeInt32Array(
        TaskHandle task,
        int32 attribute,
        int32 value[],
        uInt32 size
    );
    xerrors::Error
    GetTrigAttributeString(TaskHandle task, int32 attribute, char value[], uInt32 size);
    xerrors::Error
    GetTrigAttributeTimestamp(TaskHandle task, int32 attribute, CVIAbsoluteTime *value);
    xerrors::Error
    GetTrigAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value);
    xerrors::Error GetWatchdogAttributeBool(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        bool32 *value
    );
    xerrors::Error GetWatchdogAttributeDouble(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        float64 *value
    );
    xerrors::Error GetWatchdogAttributeInt32(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        int32 *value
    );
    xerrors::Error GetWatchdogAttributeString(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        char value[],
        uInt32 size
    );
    xerrors::Error
    GetWriteAttributeBool(TaskHandle task, int32 attribute, bool32 *value);
    xerrors::Error
    GetWriteAttributeDouble(TaskHandle task, int32 attribute, float64 *value);
    xerrors::Error
    GetWriteAttributeInt32(TaskHandle task, int32 attribute, int32 *value);
    xerrors::Error GetWriteAttributeString(
        TaskHandle task,
        int32 attribute,
        char value[],
        uInt32 size
    );
    xerrors::Error
    GetWriteAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value);
    xerrors::Error
    GetWriteAttributeUInt64(TaskHandle task, int32 attribute, uInt64 *value);
    xerrors::Error IsTaskDone(TaskHandle task, bool32 *isTaskDone);
    xerrors::Error LoadTask(const char sessionName[], TaskHandle *task);
    xerrors::Error PerformBridgeOffsetNullingCalEx(
        TaskHandle task,
        const char channel[],
        bool32 skipUnsupportedChannels
    );
    xerrors::Error PerformBridgeShuntCalEx(
        TaskHandle task,
        const char channel[],
        float64 shuntResistorValue,
        int32 shuntResistorLocation,
        int32 shuntResistorSelect,
        int32 shuntResistorSource,
        float64 bridgeResistance,
        bool32 skipUnsupportedChannels
    );
    xerrors::Error PerformStrainShuntCalEx(
        TaskHandle task,
        const char channel[],
        float64 shuntResistorValue,
        int32 shuntResistorLocation,
        int32 shuntResistorSelect,
        int32 shuntResistorSource,
        bool32 skipUnsupportedChannels
    );
    xerrors::Error PerformThrmcplLeadOffsetNullingCal(
        TaskHandle task,
        const char channel[],
        bool32 skipUnsupportedChannels
    );
    xerrors::Error ReadAnalogF64(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        float64 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    xerrors::Error ReadAnalogScalarF64(
        TaskHandle task,
        float64 timeout,
        float64 *value,
        bool32 *reserved
    );
    xerrors::Error ReadBinaryI16(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        int16 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    xerrors::Error ReadBinaryI32(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        int32 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    xerrors::Error ReadBinaryU16(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt16 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    xerrors::Error ReadBinaryU32(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt32 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    xerrors::Error ReadCounterF64(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        float64 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    xerrors::Error ReadCounterF64Ex(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        float64 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    xerrors::Error ReadCounterScalarF64(
        TaskHandle task,
        float64 timeout,
        float64 *value,
        bool32 *reserved
    );
    xerrors::Error ReadCounterScalarU32(
        TaskHandle task,
        float64 timeout,
        uInt32 *value,
        bool32 *reserved
    );
    xerrors::Error ReadCounterU32(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        uInt32 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    xerrors::Error ReadCounterU32Ex(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt32 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    xerrors::Error ReadCtrFreq(
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
    xerrors::Error ReadCtrFreqScalar(
        TaskHandle task,
        float64 timeout,
        float64 *frequency,
        float64 *dutyCycle,
        bool32 *reserved
    );
    xerrors::Error ReadCtrTicks(
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
    xerrors::Error ReadCtrTicksScalar(
        TaskHandle task,
        float64 timeout,
        uInt32 *highTicks,
        uInt32 *lowTicks,
        bool32 *reserved
    );
    xerrors::Error ReadCtrTime(
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
    xerrors::Error ReadCtrTimeScalar(
        TaskHandle task,
        float64 timeout,
        float64 *highTime,
        float64 *lowTime,
        bool32 *reserved
    );
    xerrors::Error ReadDigitalLines(
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
    xerrors::Error ReadDigitalScalarU32(
        TaskHandle task,
        float64 timeout,
        uInt32 *value,
        bool32 *reserved
    );
    xerrors::Error ReadDigitalU16(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt16 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    xerrors::Error ReadDigitalU32(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt32 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    xerrors::Error ReadDigitalU8(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        int32 fillMode,
        uInt8 readArray[],
        uInt32 arraySizeInSamps,
        int32 *sampsPerChanRead,
        bool32 *reserved
    );
    xerrors::Error ReadPowerBinaryI16(
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
    xerrors::Error ReadPowerF64(
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
    xerrors::Error ReadPowerScalarF64(
        TaskHandle task,
        float64 timeout,
        float64 *voltage,
        float64 *current,
        bool32 *reserved
    );
    xerrors::Error ReadRaw(
        TaskHandle task,
        int32 numSampsPerChan,
        float64 timeout,
        uInt8 readArray[],
        uInt32 arraySizeInBytes,
        int32 *sampsRead,
        int32 *numBytesPerSamp,
        bool32 *reserved
    );
    xerrors::Error RegisterDoneEvent(
        TaskHandle task,
        uInt32 options,
        DAQmxDoneEventCallbackPtr callbackFunction,
        void *callbackData
    );
    xerrors::Error RegisterEveryNSamplesEvent(
        TaskHandle task,
        int32 everyNSamplesEventType,
        uInt32 nSamples,
        uInt32 options,
        DAQmxEveryNSamplesEventCallbackPtr callbackFunction,
        void *callbackData
    );
    xerrors::Error RegisterSignalEvent(
        TaskHandle task,
        int32 signalID,
        uInt32 options,
        DAQmxSignalEventCallbackPtr callbackFunction,
        void *callbackData
    );
    xerrors::Error RemoveCDAQSyncConnection(const char portList[]);
    xerrors::Error
    ReserveNetworkDevice(const char deviceName[], bool32 overrideReservation);
    xerrors::Error ResetBufferAttribute(TaskHandle task, int32 attribute);
    xerrors::Error
    ResetChanAttribute(TaskHandle task, const char channel[], int32 attribute);
    xerrors::Error ResetDevice(const char deviceName[]);
    xerrors::Error ResetExportedSignalAttribute(TaskHandle task, int32 attribute);
    xerrors::Error ResetReadAttribute(TaskHandle task, int32 attribute);
    xerrors::Error ResetRealTimeAttribute(TaskHandle task, int32 attribute);
    xerrors::Error ResetTimingAttribute(TaskHandle task, int32 attribute);
    xerrors::Error
    ResetTimingAttributeEx(TaskHandle task, const char deviceNames[], int32 attribute);
    xerrors::Error ResetTrigAttribute(TaskHandle task, int32 attribute);
    xerrors::Error
    ResetWatchdogAttribute(TaskHandle task, const char lines[], int32 attribute);
    xerrors::Error ResetWriteAttribute(TaskHandle task, int32 attribute);
    xerrors::Error RestoreLastExtCalConst(const char deviceName[]);
    xerrors::Error SaveGlobalChan(
        TaskHandle task,
        const char channelName[],
        const char saveAs[],
        const char author[],
        uInt32 options
    );
    xerrors::Error SaveScale(
        const char scaleName[],
        const char saveAs[],
        const char author[],
        uInt32 options
    );
    xerrors::Error
    SaveTask(TaskHandle task, const char saveAs[], const char author[], uInt32 options);
    xerrors::Error SelfCal(const char deviceName[]);
    xerrors::Error SelfTestDevice(const char deviceName[]);
    xerrors::Error SetAIChanCalCalDate(
        TaskHandle task,
        const char channelName[],
        uInt32 year,
        uInt32 month,
        uInt32 day,
        uInt32 hour,
        uInt32 minute
    );
    xerrors::Error SetAIChanCalExpDate(
        TaskHandle task,
        const char channelName[],
        uInt32 year,
        uInt32 month,
        uInt32 day,
        uInt32 hour,
        uInt32 minute
    );
    xerrors::Error SetAnalogPowerUpStatesWithOutputType(
        const char channelNames[],
        const float64 stateArray[],
        const int32 channelTypeArray[],
        uInt32 arraySize
    );
    xerrors::Error SetArmStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime data);
    xerrors::Error
    SetBufferAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value);
    xerrors::Error
    SetCalInfoAttributeBool(const char deviceName[], int32 attribute, bool32 value);
    xerrors::Error
    SetCalInfoAttributeDouble(const char deviceName[], int32 attribute, float64 value);
    xerrors::Error SetCalInfoAttributeString(
        const char deviceName[],
        int32 attribute,
        const char value[]
    );
    xerrors::Error
    SetCalInfoAttributeUInt32(const char deviceName[], int32 attribute, uInt32 value);
    xerrors::Error SetChanAttributeBool(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        bool32 value
    );
    xerrors::Error SetChanAttributeDouble(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        float64 value
    );
    xerrors::Error SetChanAttributeDoubleArray(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        const float64 value[],
        uInt32 size
    );
    xerrors::Error SetChanAttributeInt32(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        int32 value
    );
    xerrors::Error SetChanAttributeString(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        const char value[]
    );
    xerrors::Error SetChanAttributeUInt32(
        TaskHandle task,
        const char channel[],
        int32 attribute,
        uInt32 value
    );
    xerrors::Error
    SetDigitalLogicFamilyPowerUpState(const char deviceName[], int32 logicFamily);
    xerrors::Error
    SetExportedSignalAttributeBool(TaskHandle task, int32 attribute, bool32 value);
    xerrors::Error
    SetExportedSignalAttributeDouble(TaskHandle task, int32 attribute, float64 value);
    xerrors::Error
    SetExportedSignalAttributeInt32(TaskHandle task, int32 attribute, int32 value);
    xerrors::Error SetExportedSignalAttributeString(
        TaskHandle task,
        int32 attribute,
        const char value[]
    );
    xerrors::Error
    SetExportedSignalAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value);
    xerrors::Error SetFirstSampClkWhen(TaskHandle task, CVIAbsoluteTime data);
    xerrors::Error SetReadAttributeBool(TaskHandle task, int32 attribute, bool32 value);
    xerrors::Error
    SetReadAttributeDouble(TaskHandle task, int32 attribute, float64 value);
    xerrors::Error SetReadAttributeInt32(TaskHandle task, int32 attribute, int32 value);
    xerrors::Error
    SetReadAttributeString(TaskHandle task, int32 attribute, const char value[]);
    xerrors::Error
    SetReadAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value);
    xerrors::Error
    SetReadAttributeUInt64(TaskHandle task, int32 attribute, uInt64 value);
    xerrors::Error
    SetRealTimeAttributeBool(TaskHandle task, int32 attribute, bool32 value);
    xerrors::Error
    SetRealTimeAttributeInt32(TaskHandle task, int32 attribute, int32 value);
    xerrors::Error
    SetRealTimeAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value);
    xerrors::Error SetRuntimeEnvironment(
        const char environment[],
        const char environmentVersion[],
        const char reserved1[],
        const char reserved2[]
    );
    xerrors::Error
    SetScaleAttributeDouble(const char scaleName[], int32 attribute, float64 value);
    xerrors::Error SetScaleAttributeDoubleArray(
        const char scaleName[],
        int32 attribute,
        const float64 value[],
        uInt32 size
    );
    xerrors::Error
    SetScaleAttributeInt32(const char scaleName[], int32 attribute, int32 value);
    xerrors::Error SetScaleAttributeString(
        const char scaleName[],
        int32 attribute,
        const char value[]
    );
    xerrors::Error SetStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime data);
    xerrors::Error SetSyncPulseTimeWhen(TaskHandle task, CVIAbsoluteTime data);
    xerrors::Error
    SetTimingAttributeBool(TaskHandle task, int32 attribute, bool32 value);
    xerrors::Error
    SetTimingAttributeDouble(TaskHandle task, int32 attribute, float64 value);
    xerrors::Error SetTimingAttributeExBool(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        bool32 value
    );
    xerrors::Error SetTimingAttributeExDouble(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        float64 value
    );
    xerrors::Error SetTimingAttributeExInt32(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        int32 value
    );
    xerrors::Error SetTimingAttributeExString(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        const char value[]
    );
    xerrors::Error SetTimingAttributeExTimestamp(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        CVIAbsoluteTime value
    );
    xerrors::Error SetTimingAttributeExUInt32(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        uInt32 value
    );
    xerrors::Error SetTimingAttributeExUInt64(
        TaskHandle task,
        const char deviceNames[],
        int32 attribute,
        uInt64 value
    );
    xerrors::Error
    SetTimingAttributeInt32(TaskHandle task, int32 attribute, int32 value);
    xerrors::Error
    SetTimingAttributeString(TaskHandle task, int32 attribute, const char value[]);
    xerrors::Error SetTimingAttributeTimestamp(
        TaskHandle task,
        int32 attribute,
        CVIAbsoluteTime value
    );
    xerrors::Error
    SetTimingAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value);
    xerrors::Error
    SetTimingAttributeUInt64(TaskHandle task, int32 attribute, uInt64 value);
    xerrors::Error SetTrigAttributeBool(TaskHandle task, int32 attribute, bool32 value);
    xerrors::Error
    SetTrigAttributeDouble(TaskHandle task, int32 attribute, float64 value);
    xerrors::Error SetTrigAttributeDoubleArray(
        TaskHandle task,
        int32 attribute,
        const float64 value[],
        uInt32 size
    );
    xerrors::Error SetTrigAttributeInt32(TaskHandle task, int32 attribute, int32 value);
    xerrors::Error SetTrigAttributeInt32Array(
        TaskHandle task,
        int32 attribute,
        const int32 value[],
        uInt32 size
    );
    xerrors::Error
    SetTrigAttributeString(TaskHandle task, int32 attribute, const char value[]);
    xerrors::Error
    SetTrigAttributeTimestamp(TaskHandle task, int32 attribute, CVIAbsoluteTime value);
    xerrors::Error
    SetTrigAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value);
    xerrors::Error SetWatchdogAttributeBool(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        bool32 value
    );
    xerrors::Error SetWatchdogAttributeDouble(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        float64 value
    );
    xerrors::Error SetWatchdogAttributeInt32(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        int32 value
    );
    xerrors::Error SetWatchdogAttributeString(
        TaskHandle task,
        const char lines[],
        int32 attribute,
        const char value[]
    );
    xerrors::Error
    SetWriteAttributeBool(TaskHandle task, int32 attribute, bool32 value);
    xerrors::Error
    SetWriteAttributeDouble(TaskHandle task, int32 attribute, float64 value);
    xerrors::Error
    SetWriteAttributeInt32(TaskHandle task, int32 attribute, int32 value);
    xerrors::Error
    SetWriteAttributeString(TaskHandle task, int32 attribute, const char value[]);
    xerrors::Error
    SetWriteAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value);
    xerrors::Error
    SetWriteAttributeUInt64(TaskHandle task, int32 attribute, uInt64 value);
    xerrors::Error StartNewFile(TaskHandle task, const char filePath[]);
    xerrors::Error StartTask(TaskHandle task);
    xerrors::Error StopTask(TaskHandle task);
    xerrors::Error TaskControl(TaskHandle task, int32 action);
    xerrors::Error TristateOutputTerm(const char outputTerminal[]);
    xerrors::Error UnregisterDoneEvent(
        TaskHandle task,
        uInt32 options,
        DAQmxDoneEventCallbackPtr callbackFunction,
        void *callbackData
    );
    xerrors::Error UnregisterEveryNSamplesEvent(
        TaskHandle task,
        int32 everyNSamplesEventType,
        uInt32 nSamples,
        uInt32 options,
        DAQmxEveryNSamplesEventCallbackPtr callbackFunction,
        void *callbackData
    );
    xerrors::Error UnregisterSignalEvent(
        TaskHandle task,
        int32 signalID,
        uInt32 options,
        DAQmxSignalEventCallbackPtr callbackFunction,
        void *callbackData
    );
    xerrors::Error UnreserveNetworkDevice(const char deviceName[]);
    xerrors::Error
    WaitForNextSampleClock(TaskHandle task, float64 timeout, bool32 *isLate);
    xerrors::Error WaitForValidTimestamp(
        TaskHandle task,
        int32 timestampEvent,
        float64 timeout,
        CVIAbsoluteTime *timestamp
    );
    xerrors::Error WaitUntilTaskDone(TaskHandle task, float64 timeToWait);
    xerrors::Error WriteAnalogF64(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const float64 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    );
    xerrors::Error WriteAnalogScalarF64(
        TaskHandle task,
        bool32 autoStart,
        float64 timeout,
        float64 value,
        bool32 *reserved
    );
    xerrors::Error WriteBinaryI16(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const int16 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    );
    xerrors::Error WriteBinaryI32(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const int32 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    );
    xerrors::Error WriteBinaryU16(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt16 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    );
    xerrors::Error WriteBinaryU32(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt32 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    );
    xerrors::Error WriteCtrFreq(
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
    xerrors::Error WriteCtrFreqScalar(
        TaskHandle task,
        bool32 autoStart,
        float64 timeout,
        float64 frequency,
        float64 dutyCycle,
        bool32 *reserved
    );
    xerrors::Error WriteCtrTicks(
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
    xerrors::Error WriteCtrTicksScalar(
        TaskHandle task,
        bool32 autoStart,
        float64 timeout,
        uInt32 highTicks,
        uInt32 lowTicks,
        bool32 *reserved
    );
    xerrors::Error WriteCtrTime(
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
    xerrors::Error WriteCtrTimeScalar(
        TaskHandle task,
        bool32 autoStart,
        float64 timeout,
        float64 highTime,
        float64 lowTime,
        bool32 *reserved
    );
    xerrors::Error WriteDigitalLines(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt8 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    );
    xerrors::Error WriteDigitalScalarU32(
        TaskHandle task,
        bool32 autoStart,
        float64 timeout,
        uInt32 value,
        bool32 *reserved
    );
    xerrors::Error WriteDigitalU16(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt16 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    );
    xerrors::Error WriteDigitalU32(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt32 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    );
    xerrors::Error WriteDigitalU8(
        TaskHandle task,
        int32 numSampsPerChan,
        bool32 autoStart,
        float64 timeout,
        int32 dataLayout,
        const uInt8 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    );
    xerrors::Error WriteRaw(
        TaskHandle task,
        int32 numSamps,
        bool32 autoStart,
        float64 timeout,
        const uInt8 writeArray[],
        int32 *sampsPerChanWritten,
        bool32 *reserved
    );
    xerrors::Error WriteToTEDSFromArray(
        const char physicalChannel[],
        const uInt8 bitStream[],
        uInt32 arraySize,
        int32 basicTEDSOptions
    );
    xerrors::Error WriteToTEDSFromFile(
        const char physicalChannel[],
        const char filePath[],
        int32 basicTEDSOptions
    );
    xerrors::Error SetReadRelativeTo(TaskHandle taskHandle, int32 data);
    xerrors::Error SetReadOffset(TaskHandle taskHandle, int32 data);
    xerrors::Error SetReadOverWrite(TaskHandle taskHandle, int32 data);
    xerrors::Error GetReadTotalSampPerChanAcquired(TaskHandle taskHandle, uInt64 *data);
};
}
