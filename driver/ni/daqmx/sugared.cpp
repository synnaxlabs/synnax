// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/ni/daqmx/sugared.h"

namespace daqmx {
using Status = int32;

inline std::string get_error_msg(const std::shared_ptr<API> &dmx) {
    const size_t bytes_to_alloc = dmx->GetExtendedErrorInfo(nullptr, 0);
    std::vector<char> err_buf(bytes_to_alloc);
    dmx->GetExtendedErrorInfo(err_buf.data(), err_buf.size());
    return {err_buf.data()};
}

static x::errors::Error
parse_error(const std::shared_ptr<API> &dmx, const Status status) {
    if (status == 0) return x::errors::NIL;
    const auto err_msg = get_error_msg(dmx);
    return x::errors::Error{
        CRITICAL_ERROR.sub(std::to_string(status).substr(1)),
        err_msg
    };
}

x::errors::Error SugaredAPI::process_error(const int32 status) const {
    return parse_error(dmx, status);
}

x::errors::Error SugaredAPI::AddCDAQSyncConnection(const char portList[]) {
    return process_error(dmx->AddCDAQSyncConnection(portList));
}

x::errors::Error
SugaredAPI::AddGlobalChansToTask(TaskHandle task, const char channelNames[]) {
    return process_error(dmx->AddGlobalChansToTask(task, channelNames));
}

x::errors::Error SugaredAPI::AddNetworkDevice(
    const char ipAddress[],
    const char deviceName[],
    bool32 attemptReservation,
    float64 timeout,
    char deviceNameOut[],
    uInt32 deviceNameOutBufferSize
) {
    return process_error(dmx->AddNetworkDevice(
        ipAddress,
        deviceName,
        attemptReservation,
        timeout,
        deviceNameOut,
        deviceNameOutBufferSize
    ));
}

x::errors::Error SugaredAPI::AreConfiguredCDAQSyncPortsDisconnected(
    const char chassisDevicesPorts[],
    float64 timeout,
    bool32 *disconnectedPortsExist
) {
    return process_error(dmx->AreConfiguredCDAQSyncPortsDisconnected(
        chassisDevicesPorts,
        timeout,
        disconnectedPortsExist
    ));
}

x::errors::Error SugaredAPI::AutoConfigureCDAQSyncConnections(
    const char chassisDevicesPorts[],
    float64 timeout
) {
    return process_error(
        dmx->AutoConfigureCDAQSyncConnections(chassisDevicesPorts, timeout)
    );
}

x::errors::Error SugaredAPI::CalculateReversePolyCoeff(
    const float64 forwardCoeffs[],
    uInt32 numForwardCoeffsIn,
    float64 minValX,
    float64 maxValX,
    int32 numPointsToCompute,
    int32 reversePolyOrder,
    float64 reverseCoeffs[]
) {
    return process_error(dmx->CalculateReversePolyCoeff(
        forwardCoeffs,
        numForwardCoeffsIn,
        minValX,
        maxValX,
        numPointsToCompute,
        reversePolyOrder,
        reverseCoeffs
    ));
}

x::errors::Error SugaredAPI::CfgAnlgEdgeRefTrig(
    TaskHandle task,
    const char triggerSource[],
    int32 triggerSlope,
    float64 triggerLevel,
    uInt32 pretriggerSamples
) {
    return process_error(dmx->CfgAnlgEdgeRefTrig(
        task,
        triggerSource,
        triggerSlope,
        triggerLevel,
        pretriggerSamples
    ));
}

x::errors::Error SugaredAPI::CfgAnlgEdgeStartTrig(
    TaskHandle task,
    const char triggerSource[],
    int32 triggerSlope,
    float64 triggerLevel
) {
    return process_error(
        dmx->CfgAnlgEdgeStartTrig(task, triggerSource, triggerSlope, triggerLevel)
    );
}

x::errors::Error SugaredAPI::CfgAnlgMultiEdgeRefTrig(
    TaskHandle task,
    const char triggerSources[],
    const int32 triggerSlopeArray[],
    const float64 triggerLevelArray[],
    uInt32 pretriggerSamples,
    uInt32 arraySize
) {
    return process_error(dmx->CfgAnlgMultiEdgeRefTrig(
        task,
        triggerSources,
        triggerSlopeArray,
        triggerLevelArray,
        pretriggerSamples,
        arraySize
    ));
}

x::errors::Error SugaredAPI::CfgAnlgMultiEdgeStartTrig(
    TaskHandle task,
    const char triggerSources[],
    const int32 triggerSlopeArray[],
    const float64 triggerLevelArray[],
    uInt32 arraySize
) {
    return process_error(dmx->CfgAnlgMultiEdgeStartTrig(
        task,
        triggerSources,
        triggerSlopeArray,
        triggerLevelArray,
        arraySize
    ));
}

x::errors::Error SugaredAPI::CfgAnlgWindowRefTrig(
    TaskHandle task,
    const char triggerSource[],
    int32 triggerWhen,
    float64 windowTop,
    float64 windowBottom,
    uInt32 pretriggerSamples
) {
    return process_error(dmx->CfgAnlgWindowRefTrig(
        task,
        triggerSource,
        triggerWhen,
        windowTop,
        windowBottom,
        pretriggerSamples
    ));
}

x::errors::Error SugaredAPI::CfgAnlgWindowStartTrig(
    TaskHandle task,
    const char triggerSource[],
    int32 triggerWhen,
    float64 windowTop,
    float64 windowBottom
) {
    return process_error(dmx->CfgAnlgWindowStartTrig(
        task,
        triggerSource,
        triggerWhen,
        windowTop,
        windowBottom
    ));
}

x::errors::Error SugaredAPI::CfgBurstHandshakingTimingExportClock(
    TaskHandle task,
    int32 sampleMode,
    uInt64 sampsPerChan,
    float64 sampleClkRate,
    const char sampleClkOutpTerm[],
    int32 sampleClkPulsePolarity,
    int32 pauseWhen,
    int32 readyEventActiveLevel
) {
    return process_error(dmx->CfgBurstHandshakingTimingExportClock(
        task,
        sampleMode,
        sampsPerChan,
        sampleClkRate,
        sampleClkOutpTerm,
        sampleClkPulsePolarity,
        pauseWhen,
        readyEventActiveLevel
    ));
}

x::errors::Error SugaredAPI::CfgBurstHandshakingTimingImportClock(
    TaskHandle task,
    int32 sampleMode,
    uInt64 sampsPerChan,
    float64 sampleClkRate,
    const char sampleClkSrc[],
    int32 sampleClkActiveEdge,
    int32 pauseWhen,
    int32 readyEventActiveLevel
) {
    return process_error(dmx->CfgBurstHandshakingTimingImportClock(
        task,
        sampleMode,
        sampsPerChan,
        sampleClkRate,
        sampleClkSrc,
        sampleClkActiveEdge,
        pauseWhen,
        readyEventActiveLevel
    ));
}

x::errors::Error SugaredAPI::CfgChangeDetectionTiming(
    TaskHandle task,
    const char risingEdgeChan[],
    const char fallingEdgeChan[],
    int32 sampleMode,
    uInt64 sampsPerChan
) {
    return process_error(dmx->CfgChangeDetectionTiming(
        task,
        risingEdgeChan,
        fallingEdgeChan,
        sampleMode,
        sampsPerChan
    ));
}

x::errors::Error SugaredAPI::CfgDigEdgeRefTrig(
    TaskHandle task,
    const char triggerSource[],
    int32 triggerEdge,
    uInt32 pretriggerSamples
) {
    return process_error(
        dmx->CfgDigEdgeRefTrig(task, triggerSource, triggerEdge, pretriggerSamples)
    );
}

x::errors::Error SugaredAPI::CfgDigEdgeStartTrig(
    TaskHandle task,
    const char triggerSource[],
    int32 triggerEdge
) {
    return process_error(dmx->CfgDigEdgeStartTrig(task, triggerSource, triggerEdge));
}

x::errors::Error SugaredAPI::CfgDigPatternRefTrig(
    TaskHandle task,
    const char triggerSource[],
    const char triggerPattern[],
    int32 triggerWhen,
    uInt32 pretriggerSamples
) {
    return process_error(dmx->CfgDigPatternRefTrig(
        task,
        triggerSource,
        triggerPattern,
        triggerWhen,
        pretriggerSamples
    ));
}

x::errors::Error SugaredAPI::CfgDigPatternStartTrig(
    TaskHandle task,
    const char triggerSource[],
    const char triggerPattern[],
    int32 triggerWhen
) {
    return process_error(
        dmx->CfgDigPatternStartTrig(task, triggerSource, triggerPattern, triggerWhen)
    );
}

x::errors::Error SugaredAPI::CfgHandshakingTiming(
    TaskHandle task,
    int32 sampleMode,
    uInt64 sampsPerChan
) {
    return process_error(dmx->CfgHandshakingTiming(task, sampleMode, sampsPerChan));
}

x::errors::Error
SugaredAPI::CfgImplicitTiming(TaskHandle task, int32 sampleMode, uInt64 sampsPerChan) {
    return process_error(dmx->CfgImplicitTiming(task, sampleMode, sampsPerChan));
}

x::errors::Error SugaredAPI::CfgInputBuffer(TaskHandle task, uInt32 numSampsPerChan) {
    return process_error(dmx->CfgInputBuffer(task, numSampsPerChan));
}

x::errors::Error SugaredAPI::CfgOutputBuffer(TaskHandle task, uInt32 numSampsPerChan) {
    return process_error(dmx->CfgOutputBuffer(task, numSampsPerChan));
}

x::errors::Error SugaredAPI::CfgPipelinedSampClkTiming(
    TaskHandle task,
    const char source[],
    float64 rate,
    int32 activeEdge,
    int32 sampleMode,
    uInt64 sampsPerChan
) {
    return process_error(dmx->CfgPipelinedSampClkTiming(
        task,
        source,
        rate,
        activeEdge,
        sampleMode,
        sampsPerChan
    ));
}

x::errors::Error SugaredAPI::CfgSampClkTiming(
    TaskHandle task,
    const char source[],
    float64 rate,
    int32 activeEdge,
    int32 sampleMode,
    uInt64 sampsPerChan
) {
    return process_error(
        dmx->CfgSampClkTiming(task, source, rate, activeEdge, sampleMode, sampsPerChan)
    );
}

x::errors::Error
SugaredAPI::CfgTimeStartTrig(TaskHandle task, CVIAbsoluteTime when, int32 timescale) {
    return process_error(dmx->CfgTimeStartTrig(task, when, timescale));
}

x::errors::Error SugaredAPI::CfgWatchdogAOExpirStates(
    TaskHandle task,
    const char channelNames[],
    const float64 expirStateArray[],
    const int32 outputTypeArray[],
    uInt32 arraySize
) {
    return process_error(dmx->CfgWatchdogAOExpirStates(
        task,
        channelNames,
        expirStateArray,
        outputTypeArray,
        arraySize
    ));
}

x::errors::Error SugaredAPI::CfgWatchdogCOExpirStates(
    TaskHandle task,
    const char channelNames[],
    const int32 expirStateArray[],
    uInt32 arraySize
) {
    return process_error(
        dmx->CfgWatchdogCOExpirStates(task, channelNames, expirStateArray, arraySize)
    );
}

x::errors::Error SugaredAPI::CfgWatchdogDOExpirStates(
    TaskHandle task,
    const char channelNames[],
    const int32 expirStateArray[],
    uInt32 arraySize
) {
    return process_error(
        dmx->CfgWatchdogDOExpirStates(task, channelNames, expirStateArray, arraySize)
    );
}

x::errors::Error SugaredAPI::ClearTEDS(const char physicalChannel[]) {
    return process_error(dmx->ClearTEDS(physicalChannel));
}

x::errors::Error SugaredAPI::ClearTask(TaskHandle task) {
    return process_error(dmx->ClearTask(task));
}

x::errors::Error SugaredAPI::ConfigureLogging(
    TaskHandle task,
    const char filePath[],
    int32 loggingMode,
    const char groupName[],
    int32 operation
) {
    return process_error(
        dmx->ConfigureLogging(task, filePath, loggingMode, groupName, operation)
    );
}

x::errors::Error
SugaredAPI::ConfigureTEDS(const char physicalChannel[], const char filePath[]) {
    return process_error(dmx->ConfigureTEDS(physicalChannel, filePath));
}

x::errors::Error SugaredAPI::ConnectTerms(
    const char sourceTerminal[],
    const char destinationTerminal[],
    int32 signalModifiers
) {
    return process_error(
        dmx->ConnectTerms(sourceTerminal, destinationTerminal, signalModifiers)
    );
}

x::errors::Error SugaredAPI::ControlWatchdogTask(TaskHandle task, int32 action) {
    return process_error(dmx->ControlWatchdogTask(task, action));
}

x::errors::Error SugaredAPI::CreateAIAccel4WireDCVoltageChan(
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
) {
    return process_error(dmx->CreateAIAccel4WireDCVoltageChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        sensitivity,
        sensitivityUnits,
        voltageExcitSource,
        voltageExcitVal,
        useExcitForScaling,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAIAccelChan(
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
) {
    return process_error(dmx->CreateAIAccelChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        sensitivity,
        sensitivityUnits,
        currentExcitSource,
        currentExcitVal,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAIAccelChargeChan(
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
) {
    return process_error(dmx->CreateAIAccelChargeChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        sensitivity,
        sensitivityUnits,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAIBridgeChan(
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
) {
    return process_error(dmx->CreateAIBridgeChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        bridgeConfig,
        voltageExcitSource,
        voltageExcitVal,
        nominalBridgeResistance,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAIChargeChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 terminalConfig,
    float64 minVal,
    float64 maxVal,
    int32 units,
    const char customScaleName[]
) {
    return process_error(dmx->CreateAIChargeChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAICurrentChan(
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
) {
    return process_error(dmx->CreateAICurrentChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        shuntResistorLoc,
        extShuntResistorVal,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAICurrentRMSChan(
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
) {
    return process_error(dmx->CreateAICurrentRMSChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        shuntResistorLoc,
        extShuntResistorVal,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAIForceBridgePolynomialChan(
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
) {
    return process_error(dmx->CreateAIForceBridgePolynomialChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        bridgeConfig,
        voltageExcitSource,
        voltageExcitVal,
        nominalBridgeResistance,
        forwardCoeffs,
        numForwardCoeffs,
        reverseCoeffs,
        numReverseCoeffs,
        electricalUnits,
        physicalUnits,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAIForceBridgeTableChan(
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
) {
    return process_error(dmx->CreateAIForceBridgeTableChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        bridgeConfig,
        voltageExcitSource,
        voltageExcitVal,
        nominalBridgeResistance,
        electricalVals,
        numElectricalVals,
        electricalUnits,
        physicalVals,
        numPhysicalVals,
        physicalUnits,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAIForceBridgeTwoPointLinChan(
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
) {
    return process_error(dmx->CreateAIForceBridgeTwoPointLinChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        bridgeConfig,
        voltageExcitSource,
        voltageExcitVal,
        nominalBridgeResistance,
        firstElectricalVal,
        secondElectricalVal,
        electricalUnits,
        firstPhysicalVal,
        secondPhysicalVal,
        physicalUnits,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAIForceIEPEChan(
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
) {
    return process_error(dmx->CreateAIForceIEPEChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        sensitivity,
        sensitivityUnits,
        currentExcitSource,
        currentExcitVal,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAIFreqVoltageChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    float64 thresholdLevel,
    float64 hysteresis,
    const char customScaleName[]
) {
    return process_error(dmx->CreateAIFreqVoltageChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        thresholdLevel,
        hysteresis,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAIMicrophoneChan(
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
) {
    return process_error(dmx->CreateAIMicrophoneChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        units,
        micSensitivity,
        maxSndPressLevel,
        currentExcitSource,
        currentExcitVal,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAIPosEddyCurrProxProbeChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    float64 sensitivity,
    int32 sensitivityUnits,
    const char customScaleName[]
) {
    return process_error(dmx->CreateAIPosEddyCurrProxProbeChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        sensitivity,
        sensitivityUnits,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAIPosLVDTChan(
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
) {
    return process_error(dmx->CreateAIPosLVDTChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        sensitivity,
        sensitivityUnits,
        voltageExcitSource,
        voltageExcitVal,
        voltageExcitFreq,
        acExcitWireMode,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAIPosRVDTChan(
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
) {
    return process_error(dmx->CreateAIPosRVDTChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        sensitivity,
        sensitivityUnits,
        voltageExcitSource,
        voltageExcitVal,
        voltageExcitFreq,
        acExcitWireMode,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAIPowerChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 voltageSetpoint,
    float64 currentSetpoint,
    bool32 outputEnable
) {
    return process_error(dmx->CreateAIPowerChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        voltageSetpoint,
        currentSetpoint,
        outputEnable
    ));
}

x::errors::Error SugaredAPI::CreateAIPressureBridgePolynomialChan(
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
) {
    return process_error(dmx->CreateAIPressureBridgePolynomialChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        bridgeConfig,
        voltageExcitSource,
        voltageExcitVal,
        nominalBridgeResistance,
        forwardCoeffs,
        numForwardCoeffs,
        reverseCoeffs,
        numReverseCoeffs,
        electricalUnits,
        physicalUnits,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAIPressureBridgeTableChan(
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
) {
    return process_error(dmx->CreateAIPressureBridgeTableChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        bridgeConfig,
        voltageExcitSource,
        voltageExcitVal,
        nominalBridgeResistance,
        electricalVals,
        numElectricalVals,
        electricalUnits,
        physicalVals,
        numPhysicalVals,
        physicalUnits,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAIPressureBridgeTwoPointLinChan(
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
) {
    return process_error(dmx->CreateAIPressureBridgeTwoPointLinChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        bridgeConfig,
        voltageExcitSource,
        voltageExcitVal,
        nominalBridgeResistance,
        firstElectricalVal,
        secondElectricalVal,
        electricalUnits,
        firstPhysicalVal,
        secondPhysicalVal,
        physicalUnits,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAIRTDChan(
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
) {
    return process_error(dmx->CreateAIRTDChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        rtdType,
        resistanceConfig,
        currentExcitSource,
        currentExcitVal,
        r0
    ));
}

x::errors::Error SugaredAPI::CreateAIResistanceChan(
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
) {
    return process_error(dmx->CreateAIResistanceChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        resistanceConfig,
        currentExcitSource,
        currentExcitVal,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAIRosetteStrainGageChan(
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
) {
    return process_error(dmx->CreateAIRosetteStrainGageChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        rosetteType,
        gageOrientation,
        rosetteMeasTypes,
        numRosetteMeasTypes,
        strainConfig,
        voltageExcitSource,
        voltageExcitVal,
        gageFactor,
        nominalGageResistance,
        poissonRatio,
        leadWireResistance
    ));
}

x::errors::Error SugaredAPI::CreateAIStrainGageChan(
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
) {
    return process_error(dmx->CreateAIStrainGageChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        strainConfig,
        voltageExcitSource,
        voltageExcitVal,
        gageFactor,
        initialBridgeVoltage,
        nominalGageResistance,
        poissonRatio,
        leadWireResistance,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAITempBuiltInSensorChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 units
) {
    return process_error(dmx->CreateAITempBuiltInSensorChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        units
    ));
}

x::errors::Error SugaredAPI::CreateAIThrmcplChan(
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
) {
    return process_error(dmx->CreateAIThrmcplChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        thermocoupleType,
        cjcSource,
        cjcVal,
        cjcChannel
    ));
}

x::errors::Error SugaredAPI::CreateAIThrmstrChanIex(
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
) {
    return process_error(dmx->CreateAIThrmstrChanIex(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        resistanceConfig,
        currentExcitSource,
        currentExcitVal,
        a,
        b,
        c
    ));
}

x::errors::Error SugaredAPI::SetScaleAttributeDouble(
    const char scaleName[],
    int32 attribute,
    float64 value
) {
    return process_error(dmx->SetScaleAttributeDouble(scaleName, attribute, value));
}

x::errors::Error SugaredAPI::SetScaleAttributeDoubleArray(
    const char scaleName[],
    int32 attribute,
    const float64 value[],
    uInt32 size
) {
    return process_error(
        dmx->SetScaleAttributeDoubleArray(scaleName, attribute, value, size)
    );
}

x::errors::Error SugaredAPI::SetScaleAttributeInt32(
    const char scaleName[],
    int32 attribute,
    int32 value
) {
    return process_error(dmx->SetScaleAttributeInt32(scaleName, attribute, value));
}

x::errors::Error SugaredAPI::SetScaleAttributeString(
    const char scaleName[],
    int32 attribute,
    const char value[]
) {
    return process_error(dmx->SetScaleAttributeString(scaleName, attribute, value));
}

x::errors::Error
SugaredAPI::SetStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime data) {
    return process_error(dmx->SetStartTrigTrigWhen(task, data));
}

x::errors::Error
SugaredAPI::SetSyncPulseTimeWhen(TaskHandle task, CVIAbsoluteTime data) {
    return process_error(dmx->SetSyncPulseTimeWhen(task, data));
}

x::errors::Error
SugaredAPI::SetTimingAttributeBool(TaskHandle task, int32 attribute, bool32 value) {
    return process_error(dmx->SetTimingAttributeBool(task, attribute, value));
}

x::errors::Error
SugaredAPI::SetTimingAttributeDouble(TaskHandle task, int32 attribute, float64 value) {
    return process_error(dmx->SetTimingAttributeDouble(task, attribute, value));
}

x::errors::Error SugaredAPI::SetTimingAttributeExBool(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    bool32 value
) {
    return process_error(
        dmx->SetTimingAttributeExBool(task, deviceNames, attribute, value)
    );
}

x::errors::Error SugaredAPI::SetTimingAttributeExDouble(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    float64 value
) {
    return process_error(
        dmx->SetTimingAttributeExDouble(task, deviceNames, attribute, value)
    );
}

x::errors::Error SugaredAPI::SetTimingAttributeExInt32(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    int32 value
) {
    return process_error(
        dmx->SetTimingAttributeExInt32(task, deviceNames, attribute, value)
    );
}

x::errors::Error SugaredAPI::SetTimingAttributeExString(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    const char value[]
) {
    return process_error(
        dmx->SetTimingAttributeExString(task, deviceNames, attribute, value)
    );
}

x::errors::Error SugaredAPI::SetTimingAttributeExTimestamp(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    CVIAbsoluteTime value
) {
    return process_error(
        dmx->SetTimingAttributeExTimestamp(task, deviceNames, attribute, value)
    );
}

x::errors::Error SugaredAPI::SetTimingAttributeExUInt32(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    uInt32 value
) {
    return process_error(
        dmx->SetTimingAttributeExUInt32(task, deviceNames, attribute, value)
    );
}

x::errors::Error SugaredAPI::SetTimingAttributeExUInt64(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    uInt64 value
) {
    return process_error(
        dmx->SetTimingAttributeExUInt64(task, deviceNames, attribute, value)
    );
}

x::errors::Error
SugaredAPI::SetTimingAttributeInt32(TaskHandle task, int32 attribute, int32 value) {
    return process_error(dmx->SetTimingAttributeInt32(task, attribute, value));
}

x::errors::Error SugaredAPI::SetTimingAttributeString(
    TaskHandle task,
    int32 attribute,
    const char value[]
) {
    return process_error(dmx->SetTimingAttributeString(task, attribute, value));
}

x::errors::Error SugaredAPI::SetTimingAttributeTimestamp(
    TaskHandle task,
    int32 attribute,
    CVIAbsoluteTime value
) {
    return process_error(dmx->SetTimingAttributeTimestamp(task, attribute, value));
}

x::errors::Error
SugaredAPI::SetTimingAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) {
    return process_error(dmx->SetTimingAttributeUInt32(task, attribute, value));
}

x::errors::Error
SugaredAPI::SetTimingAttributeUInt64(TaskHandle task, int32 attribute, uInt64 value) {
    return process_error(dmx->SetTimingAttributeUInt64(task, attribute, value));
}

x::errors::Error
SugaredAPI::SetTrigAttributeBool(TaskHandle task, int32 attribute, bool32 value) {
    return process_error(dmx->SetTrigAttributeBool(task, attribute, value));
}

x::errors::Error
SugaredAPI::SetTrigAttributeDouble(TaskHandle task, int32 attribute, float64 value) {
    return process_error(dmx->SetTrigAttributeDouble(task, attribute, value));
}

x::errors::Error SugaredAPI::SetTrigAttributeDoubleArray(
    TaskHandle task,
    int32 attribute,
    const float64 value[],
    uInt32 size
) {
    return process_error(
        dmx->SetTrigAttributeDoubleArray(task, attribute, value, size)
    );
}

x::errors::Error
SugaredAPI::SetTrigAttributeInt32(TaskHandle task, int32 attribute, int32 value) {
    return process_error(dmx->SetTrigAttributeInt32(task, attribute, value));
}

x::errors::Error SugaredAPI::SetTrigAttributeInt32Array(
    TaskHandle task,
    int32 attribute,
    const int32 value[],
    uInt32 size
) {
    return process_error(dmx->SetTrigAttributeInt32Array(task, attribute, value, size));
}

x::errors::Error SugaredAPI::SetTrigAttributeString(
    TaskHandle task,
    int32 attribute,
    const char value[]
) {
    return process_error(dmx->SetTrigAttributeString(task, attribute, value));
}

x::errors::Error SugaredAPI::SetTrigAttributeTimestamp(
    TaskHandle task,
    int32 attribute,
    CVIAbsoluteTime value
) {
    return process_error(dmx->SetTrigAttributeTimestamp(task, attribute, value));
}

x::errors::Error
SugaredAPI::SetTrigAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) {
    return process_error(dmx->SetTrigAttributeUInt32(task, attribute, value));
}

x::errors::Error SugaredAPI::SetWatchdogAttributeBool(
    TaskHandle task,
    const char lines[],
    int32 attribute,
    bool32 value
) {
    return process_error(dmx->SetWatchdogAttributeBool(task, lines, attribute, value));
}

x::errors::Error SugaredAPI::SetWatchdogAttributeDouble(
    TaskHandle task,
    const char lines[],
    int32 attribute,
    float64 value
) {
    return process_error(
        dmx->SetWatchdogAttributeDouble(task, lines, attribute, value)
    );
}

x::errors::Error SugaredAPI::SetWatchdogAttributeInt32(
    TaskHandle task,
    const char lines[],
    int32 attribute,
    int32 value
) {
    return process_error(dmx->SetWatchdogAttributeInt32(task, lines, attribute, value));
}

x::errors::Error SugaredAPI::SetWatchdogAttributeString(
    TaskHandle task,
    const char lines[],
    int32 attribute,
    const char value[]
) {
    return process_error(
        dmx->SetWatchdogAttributeString(task, lines, attribute, value)
    );
}

x::errors::Error
SugaredAPI::SetWriteAttributeBool(TaskHandle task, int32 attribute, bool32 value) {
    return process_error(dmx->SetWriteAttributeBool(task, attribute, value));
}

x::errors::Error
SugaredAPI::SetWriteAttributeDouble(TaskHandle task, int32 attribute, float64 value) {
    return process_error(dmx->SetWriteAttributeDouble(task, attribute, value));
}

x::errors::Error
SugaredAPI::SetWriteAttributeInt32(TaskHandle task, int32 attribute, int32 value) {
    return process_error(dmx->SetWriteAttributeInt32(task, attribute, value));
}

x::errors::Error SugaredAPI::SetWriteAttributeString(
    TaskHandle task,
    int32 attribute,
    const char value[]
) {
    return process_error(dmx->SetWriteAttributeString(task, attribute, value));
}

x::errors::Error
SugaredAPI::SetWriteAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) {
    return process_error(dmx->SetWriteAttributeUInt32(task, attribute, value));
}

x::errors::Error
SugaredAPI::SetWriteAttributeUInt64(TaskHandle task, int32 attribute, uInt64 value) {
    return process_error(dmx->SetWriteAttributeUInt64(task, attribute, value));
}

x::errors::Error SugaredAPI::StartNewFile(TaskHandle task, const char filePath[]) {
    return process_error(dmx->StartNewFile(task, filePath));
}

x::errors::Error SugaredAPI::StartTask(TaskHandle task) {
    return process_error(dmx->StartTask(task));
}

x::errors::Error SugaredAPI::StopTask(TaskHandle task) {
    return process_error(dmx->StopTask(task));
}

x::errors::Error SugaredAPI::TaskControl(TaskHandle task, int32 action) {
    return process_error(dmx->TaskControl(task, action));
}

x::errors::Error SugaredAPI::TristateOutputTerm(const char outputTerminal[]) {
    return process_error(dmx->TristateOutputTerm(outputTerminal));
}

x::errors::Error SugaredAPI::UnregisterDoneEvent(
    TaskHandle task,
    uInt32 options,
    DAQmxDoneEventCallbackPtr callbackFunction,
    void *callbackData
) {
    return process_error(
        dmx->UnregisterDoneEvent(task, options, callbackFunction, callbackData)
    );
}

x::errors::Error SugaredAPI::UnregisterEveryNSamplesEvent(
    TaskHandle task,
    int32 everyNSamplesEventType,
    uInt32 nSamples,
    uInt32 options,
    DAQmxEveryNSamplesEventCallbackPtr callbackFunction,
    void *callbackData
) {
    return process_error(dmx->UnregisterEveryNSamplesEvent(
        task,
        everyNSamplesEventType,
        nSamples,
        options,
        callbackFunction,
        callbackData
    ));
}

x::errors::Error SugaredAPI::UnregisterSignalEvent(
    TaskHandle task,
    int32 signalID,
    uInt32 options,
    DAQmxSignalEventCallbackPtr callbackFunction,
    void *callbackData
) {
    return process_error(dmx->UnregisterSignalEvent(
        task,
        signalID,
        options,
        callbackFunction,
        callbackData
    ));
}

x::errors::Error SugaredAPI::UnreserveNetworkDevice(const char deviceName[]) {
    return process_error(dmx->UnreserveNetworkDevice(deviceName));
}

x::errors::Error
SugaredAPI::WaitForNextSampleClock(TaskHandle task, float64 timeout, bool32 *isLate) {
    return process_error(dmx->WaitForNextSampleClock(task, timeout, isLate));
}

x::errors::Error SugaredAPI::WaitForValidTimestamp(
    TaskHandle task,
    int32 timestampEvent,
    float64 timeout,
    CVIAbsoluteTime *timestamp
) {
    return process_error(
        dmx->WaitForValidTimestamp(task, timestampEvent, timeout, timestamp)
    );
}

x::errors::Error SugaredAPI::WaitUntilTaskDone(TaskHandle task, float64 timeToWait) {
    return process_error(dmx->WaitUntilTaskDone(task, timeToWait));
}

x::errors::Error SugaredAPI::WriteAnalogF64(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const float64 writeArray[],
    int32 *sampsPerChanWritten,
    bool32 *reserved
) {
    return process_error(dmx->WriteAnalogF64(
        task,
        numSampsPerChan,
        autoStart,
        timeout,
        dataLayout,
        writeArray,
        sampsPerChanWritten,
        reserved
    ));
}

x::errors::Error SugaredAPI::WriteAnalogScalarF64(
    TaskHandle task,
    bool32 autoStart,
    float64 timeout,
    float64 value,
    bool32 *reserved
) {
    return process_error(
        dmx->WriteAnalogScalarF64(task, autoStart, timeout, value, reserved)
    );
}

x::errors::Error SugaredAPI::WriteBinaryI16(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const int16 writeArray[],
    int32 *sampsPerChanWritten,
    bool32 *reserved
) {
    return process_error(dmx->WriteBinaryI16(
        task,
        numSampsPerChan,
        autoStart,
        timeout,
        dataLayout,
        writeArray,
        sampsPerChanWritten,
        reserved
    ));
}

x::errors::Error SugaredAPI::WriteBinaryI32(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const int32 writeArray[],
    int32 *sampsPerChanWritten,
    bool32 *reserved
) {
    return process_error(dmx->WriteBinaryI32(
        task,
        numSampsPerChan,
        autoStart,
        timeout,
        dataLayout,
        writeArray,
        sampsPerChanWritten,
        reserved
    ));
}

x::errors::Error SugaredAPI::WriteBinaryU16(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const uInt16 writeArray[],
    int32 *sampsPerChanWritten,
    bool32 *reserved
) {
    return process_error(dmx->WriteBinaryU16(
        task,
        numSampsPerChan,
        autoStart,
        timeout,
        dataLayout,
        writeArray,
        sampsPerChanWritten,
        reserved
    ));
}

x::errors::Error SugaredAPI::WriteBinaryU32(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const uInt32 writeArray[],
    int32 *sampsPerChanWritten,
    bool32 *reserved
) {
    return process_error(dmx->WriteBinaryU32(
        task,
        numSampsPerChan,
        autoStart,
        timeout,
        dataLayout,
        writeArray,
        sampsPerChanWritten,
        reserved
    ));
}

x::errors::Error SugaredAPI::WriteCtrFreq(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const float64 frequency[],
    const float64 dutyCycle[],
    int32 *numSampsPerChanWritten,
    bool32 *reserved
) {
    return process_error(dmx->WriteCtrFreq(
        task,
        numSampsPerChan,
        autoStart,
        timeout,
        dataLayout,
        frequency,
        dutyCycle,
        numSampsPerChanWritten,
        reserved
    ));
}

x::errors::Error SugaredAPI::WriteCtrFreqScalar(
    TaskHandle task,
    bool32 autoStart,
    float64 timeout,
    float64 frequency,
    float64 dutyCycle,
    bool32 *reserved
) {
    return process_error(dmx->WriteCtrFreqScalar(
        task,
        autoStart,
        timeout,
        frequency,
        dutyCycle,
        reserved
    ));
}

x::errors::Error SugaredAPI::WriteCtrTicks(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const uInt32 highTicks[],
    const uInt32 lowTicks[],
    int32 *numSampsPerChanWritten,
    bool32 *reserved
) {
    return process_error(dmx->WriteCtrTicks(
        task,
        numSampsPerChan,
        autoStart,
        timeout,
        dataLayout,
        highTicks,
        lowTicks,
        numSampsPerChanWritten,
        reserved
    ));
}

x::errors::Error SugaredAPI::WriteCtrTicksScalar(
    TaskHandle task,
    bool32 autoStart,
    float64 timeout,
    uInt32 highTicks,
    uInt32 lowTicks,
    bool32 *reserved
) {
    return process_error(dmx->WriteCtrTicksScalar(
        task,
        autoStart,
        timeout,
        highTicks,
        lowTicks,
        reserved
    ));
}

x::errors::Error SugaredAPI::WriteCtrTime(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const float64 highTime[],
    const float64 lowTime[],
    int32 *numSampsPerChanWritten,
    bool32 *reserved
) {
    return process_error(dmx->WriteCtrTime(
        task,
        numSampsPerChan,
        autoStart,
        timeout,
        dataLayout,
        highTime,
        lowTime,
        numSampsPerChanWritten,
        reserved
    ));
}

x::errors::Error SugaredAPI::WriteCtrTimeScalar(
    TaskHandle task,
    bool32 autoStart,
    float64 timeout,
    float64 highTime,
    float64 lowTime,
    bool32 *reserved
) {
    return process_error(
        dmx->WriteCtrTimeScalar(task, autoStart, timeout, highTime, lowTime, reserved)
    );
}

x::errors::Error SugaredAPI::WriteDigitalLines(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const uInt8 writeArray[],
    int32 *sampsPerChanWritten,
    bool32 *reserved
) {
    return process_error(dmx->WriteDigitalLines(
        task,
        numSampsPerChan,
        autoStart,
        timeout,
        dataLayout,
        writeArray,
        sampsPerChanWritten,
        reserved
    ));
}

x::errors::Error SugaredAPI::WriteDigitalScalarU32(
    TaskHandle task,
    bool32 autoStart,
    float64 timeout,
    uInt32 value,
    bool32 *reserved
) {
    return process_error(
        dmx->WriteDigitalScalarU32(task, autoStart, timeout, value, reserved)
    );
}

x::errors::Error SugaredAPI::WriteDigitalU16(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const uInt16 writeArray[],
    int32 *sampsPerChanWritten,
    bool32 *reserved
) {
    return process_error(dmx->WriteDigitalU16(
        task,
        numSampsPerChan,
        autoStart,
        timeout,
        dataLayout,
        writeArray,
        sampsPerChanWritten,
        reserved
    ));
}

x::errors::Error SugaredAPI::WriteDigitalU32(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const uInt32 writeArray[],
    int32 *sampsPerChanWritten,
    bool32 *reserved
) {
    return process_error(dmx->WriteDigitalU32(
        task,
        numSampsPerChan,
        autoStart,
        timeout,
        dataLayout,
        writeArray,
        sampsPerChanWritten,
        reserved
    ));
}

x::errors::Error SugaredAPI::WriteDigitalU8(
    TaskHandle task,
    int32 numSampsPerChan,
    bool32 autoStart,
    float64 timeout,
    int32 dataLayout,
    const uInt8 writeArray[],
    int32 *sampsPerChanWritten,
    bool32 *reserved
) {
    return process_error(dmx->WriteDigitalU8(
        task,
        numSampsPerChan,
        autoStart,
        timeout,
        dataLayout,
        writeArray,
        sampsPerChanWritten,
        reserved
    ));
}

x::errors::Error SugaredAPI::WriteRaw(
    TaskHandle task,
    int32 numSamps,
    bool32 autoStart,
    float64 timeout,
    const uInt8 writeArray[],
    int32 *sampsPerChanWritten,
    bool32 *reserved
) {
    return process_error(dmx->WriteRaw(
        task,
        numSamps,
        autoStart,
        timeout,
        writeArray,
        sampsPerChanWritten,
        reserved
    ));
}

x::errors::Error SugaredAPI::WriteToTEDSFromArray(
    const char physicalChannel[],
    const uInt8 bitStream[],
    uInt32 arraySize,
    int32 basicTEDSOptions
) {
    return process_error(dmx->WriteToTEDSFromArray(
        physicalChannel,
        bitStream,
        arraySize,
        basicTEDSOptions
    ));
}

x::errors::Error SugaredAPI::WriteToTEDSFromFile(
    const char physicalChannel[],
    const char filePath[],
    int32 basicTEDSOptions
) {
    return process_error(
        dmx->WriteToTEDSFromFile(physicalChannel, filePath, basicTEDSOptions)
    );
}

x::errors::Error SugaredAPI::SetReadRelativeTo(TaskHandle taskHandle, int32 data) {
    return process_error(dmx->SetReadRelativeTo(taskHandle, data));
}

x::errors::Error SugaredAPI::SetReadOffset(TaskHandle taskHandle, int32 data) {
    return process_error(dmx->SetReadOffset(taskHandle, data));
}

x::errors::Error SugaredAPI::SetReadOverWrite(TaskHandle taskHandle, int32 data) {
    return process_error(dmx->SetReadOverWrite(taskHandle, data));
}

x::errors::Error
SugaredAPI::GetReadTotalSampPerChanAcquired(TaskHandle taskHandle, uInt64 *data) {
    return process_error(dmx->GetReadTotalSampPerChanAcquired(taskHandle, data));
}

x::errors::Error SugaredAPI::CreateLinScale(
    const char scaleName[],
    float64 slope,
    float64 yIntercept,
    int32 preScaledUnits,
    const char customScaleName[]
) {
    return process_error(dmx->CreateLinScale(
        scaleName,
        slope,
        yIntercept,
        preScaledUnits,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateMapScale(
    const char scaleName[],
    float64 prescaledMin,
    float64 prescaledMax,
    float64 scaledMin,
    float64 scaledMax,
    int32 preScaledUnits,
    const char customScaleName[]
) {
    return process_error(dmx->CreateMapScale(
        scaleName,
        prescaledMin,
        prescaledMax,
        scaledMin,
        scaledMax,
        preScaledUnits,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateTableScale(
    const char scaleName[],
    const float64 prescaledVals[],
    uInt32 numPrescaledVals,
    const float64 scaledVals[],
    uInt32 numScaledVals,
    int32 preScaledUnits,
    const char customScaleName[]
) {
    return process_error(dmx->CreateTableScale(
        scaleName,
        prescaledVals,
        numPrescaledVals,
        scaledVals,
        numScaledVals,
        preScaledUnits,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAIVoltageChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 terminalConfig,
    float64 minVal,
    float64 maxVal,
    int32 units,
    const char customScaleName[]
) {
    return process_error(dmx->CreateAIVoltageChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAOCurrentChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    const char customScaleName[]
) {
    return process_error(dmx->CreateAOCurrentChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAOFuncGenChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 type,
    float64 freq,
    float64 amplitude,
    float64 offset
) {
    return process_error(dmx->CreateAOFuncGenChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        type,
        freq,
        amplitude,
        offset
    ));
}

x::errors::Error SugaredAPI::CreateAOVoltageChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    const char customScaleName[]
) {
    return process_error(dmx->CreateAOVoltageChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreatePolynomialScale(
    const char scaleName[],
    const float64 forwardCoeffs[],
    uInt32 numForwardCoeffs,
    const float64 reverseCoeffs[],
    uInt32 numReverseCoeffs,
    int32 preScaledUnits,
    const char customScaleName[]
) {
    return process_error(dmx->CreatePolynomialScale(
        scaleName,
        forwardCoeffs,
        numForwardCoeffs,
        reverseCoeffs,
        numReverseCoeffs,
        preScaledUnits,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAIVelocityIEPEChan(
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
) {
    return process_error(dmx->CreateAIVelocityIEPEChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        sensitivity,
        sensitivityUnits,
        currentExcitSource,
        currentExcitVal,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAITorqueBridgeTableChan(
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
) {
    return process_error(dmx->CreateAITorqueBridgeTableChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        bridgeConfig,
        voltageExcitSource,
        voltageExcitVal,
        nominalBridgeResistance,
        electricalVals,
        numElectricalVals,
        electricalUnits,
        physicalVals,
        numPhysicalVals,
        physicalUnits,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAITorqueBridgePolynomialChan(
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
) {
    return process_error(dmx->CreateAITorqueBridgePolynomialChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        bridgeConfig,
        voltageExcitSource,
        voltageExcitVal,
        nominalBridgeResistance,
        forwardCoeffs,
        numForwardCoeffs,
        reverseCoeffs,
        numReverseCoeffs,
        electricalUnits,
        physicalUnits,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAITorqueBridgeTwoPointLinChan(
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
) {
    return process_error(dmx->CreateAITorqueBridgeTwoPointLinChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        bridgeConfig,
        voltageExcitSource,
        voltageExcitVal,
        nominalBridgeResistance,
        firstElectricalVal,
        secondElectricalVal,
        electricalUnits,
        firstPhysicalVal,
        secondPhysicalVal,
        physicalUnits,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateTask(const char sessionName[], TaskHandle *task) {
    return process_error(dmx->CreateTask(sessionName, task));
}

x::errors::Error SugaredAPI::CreateWatchdogTimerTaskEx(
    const char deviceName[],
    const char sessionName[],
    TaskHandle *task,
    float64 timeout
) {
    return process_error(
        dmx->CreateWatchdogTimerTaskEx(deviceName, sessionName, task, timeout)
    );
}

x::errors::Error SugaredAPI::DeleteNetworkDevice(const char deviceName[]) {
    return process_error(dmx->DeleteNetworkDevice(deviceName));
}

x::errors::Error SugaredAPI::DeleteSavedGlobalChan(const char channelName[]) {
    return process_error(dmx->DeleteSavedGlobalChan(channelName));
}

x::errors::Error SugaredAPI::DeleteSavedScale(const char scaleName[]) {
    return process_error(dmx->DeleteSavedScale(scaleName));
}

x::errors::Error SugaredAPI::DeleteSavedTask(const char taskName[]) {
    return process_error(dmx->DeleteSavedTask(taskName));
}

x::errors::Error
SugaredAPI::DeviceSupportsCal(const char deviceName[], bool32 *calSupported) {
    return process_error(dmx->DeviceSupportsCal(deviceName, calSupported));
}

x::errors::Error SugaredAPI::DisableRefTrig(TaskHandle task) {
    return process_error(dmx->DisableRefTrig(task));
}

x::errors::Error SugaredAPI::DisableStartTrig(TaskHandle task) {
    return process_error(dmx->DisableStartTrig(task));
}

x::errors::Error SugaredAPI::DisconnectTerms(
    const char sourceTerminal[],
    const char destinationTerminal[]
) {
    return process_error(dmx->DisconnectTerms(sourceTerminal, destinationTerminal));
}

x::errors::Error
SugaredAPI::ExportSignal(TaskHandle task, int32 signalID, const char outputTerminal[]) {
    return process_error(dmx->ExportSignal(task, signalID, outputTerminal));
}

x::errors::Error SugaredAPI::GetAIChanCalCalDate(
    TaskHandle task,
    const char channelName[],
    uInt32 *year,
    uInt32 *month,
    uInt32 *day,
    uInt32 *hour,
    uInt32 *minute
) {
    return process_error(
        dmx->GetAIChanCalCalDate(task, channelName, year, month, day, hour, minute)
    );
}

x::errors::Error SugaredAPI::GetAIChanCalExpDate(
    TaskHandle task,
    const char channelName[],
    uInt32 *year,
    uInt32 *month,
    uInt32 *day,
    uInt32 *hour,
    uInt32 *minute
) {
    return process_error(
        dmx->GetAIChanCalExpDate(task, channelName, year, month, day, hour, minute)
    );
}

x::errors::Error SugaredAPI::GetAnalogPowerUpStatesWithOutputType(
    const char channelNames[],
    float64 stateArray[],
    int32 channelTypeArray[],
    uInt32 *arraySize
) {
    return process_error(dmx->GetAnalogPowerUpStatesWithOutputType(
        channelNames,
        stateArray,
        channelTypeArray,
        arraySize
    ));
}

x::errors::Error
SugaredAPI::GetArmStartTrigTimestampVal(TaskHandle task, CVIAbsoluteTime *data) {
    return process_error(dmx->GetArmStartTrigTimestampVal(task, data));
}

x::errors::Error
SugaredAPI::GetArmStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime *data) {
    return process_error(dmx->GetArmStartTrigTrigWhen(task, data));
}

x::errors::Error
SugaredAPI::GetAutoConfiguredCDAQSyncConnections(char portList[], uInt32 portListSize) {
    return process_error(
        dmx->GetAutoConfiguredCDAQSyncConnections(portList, portListSize)
    );
}

x::errors::Error
SugaredAPI::GetBufferAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) {
    return process_error(dmx->GetBufferAttributeUInt32(task, attribute, value));
}

x::errors::Error SugaredAPI::GetCalInfoAttributeBool(
    const char deviceName[],
    int32 attribute,
    bool32 *value
) {
    return process_error(dmx->GetCalInfoAttributeBool(deviceName, attribute, value));
}

x::errors::Error SugaredAPI::GetCalInfoAttributeDouble(
    const char deviceName[],
    int32 attribute,
    float64 *value
) {
    return process_error(dmx->GetCalInfoAttributeDouble(deviceName, attribute, value));
}

x::errors::Error SugaredAPI::GetCalInfoAttributeString(
    const char deviceName[],
    int32 attribute,
    char value[],
    uInt32 size
) {
    return process_error(
        dmx->GetCalInfoAttributeString(deviceName, attribute, value, size)
    );
}

x::errors::Error SugaredAPI::GetCalInfoAttributeUInt32(
    const char deviceName[],
    int32 attribute,
    uInt32 *value
) {
    return process_error(dmx->GetCalInfoAttributeUInt32(deviceName, attribute, value));
}

x::errors::Error SugaredAPI::GetChanAttributeBool(
    TaskHandle task,
    const char channel[],
    int32 attribute,
    bool32 *value
) {
    return process_error(dmx->GetChanAttributeBool(task, channel, attribute, value));
}

x::errors::Error SugaredAPI::GetChanAttributeDouble(
    TaskHandle task,
    const char channel[],
    int32 attribute,
    float64 *value
) {
    return process_error(dmx->GetChanAttributeDouble(task, channel, attribute, value));
}

x::errors::Error SugaredAPI::GetChanAttributeDoubleArray(
    TaskHandle task,
    const char channel[],
    int32 attribute,
    float64 value[],
    uInt32 size
) {
    return process_error(
        dmx->GetChanAttributeDoubleArray(task, channel, attribute, value, size)
    );
}

x::errors::Error SugaredAPI::GetChanAttributeInt32(
    TaskHandle task,
    const char channel[],
    int32 attribute,
    int32 *value
) {
    return process_error(dmx->GetChanAttributeInt32(task, channel, attribute, value));
}

x::errors::Error SugaredAPI::GetChanAttributeString(
    TaskHandle task,
    const char channel[],
    int32 attribute,
    char value[],
    uInt32 size
) {
    return process_error(
        dmx->GetChanAttributeString(task, channel, attribute, value, size)
    );
}

x::errors::Error SugaredAPI::GetChanAttributeUInt32(
    TaskHandle task,
    const char channel[],
    int32 attribute,
    uInt32 *value
) {
    return process_error(dmx->GetChanAttributeUInt32(task, channel, attribute, value));
}

x::errors::Error SugaredAPI::GetDeviceAttributeBool(
    const char deviceName[],
    int32 attribute,
    bool32 *value
) {
    return process_error(dmx->GetDeviceAttributeBool(deviceName, attribute, value));
}

x::errors::Error SugaredAPI::GetDeviceAttributeDouble(
    const char deviceName[],
    int32 attribute,
    float64 *value
) {
    return process_error(dmx->GetDeviceAttributeDouble(deviceName, attribute, value));
}

x::errors::Error SugaredAPI::GetDeviceAttributeDoubleArray(
    const char deviceName[],
    int32 attribute,
    float64 value[],
    uInt32 size
) {
    return process_error(
        dmx->GetDeviceAttributeDoubleArray(deviceName, attribute, value, size)
    );
}

x::errors::Error SugaredAPI::GetDeviceAttributeInt32(
    const char deviceName[],
    int32 attribute,
    int32 *value
) {
    return process_error(dmx->GetDeviceAttributeInt32(deviceName, attribute, value));
}

x::errors::Error SugaredAPI::GetDeviceAttributeInt32Array(
    const char deviceName[],
    int32 attribute,
    int32 value[],
    uInt32 size
) {
    return process_error(
        dmx->GetDeviceAttributeInt32Array(deviceName, attribute, value, size)
    );
}

x::errors::Error SugaredAPI::GetDeviceAttributeString(
    const char deviceName[],
    int32 attribute,
    char value[],
    uInt32 size
) {
    return process_error(
        dmx->GetDeviceAttributeString(deviceName, attribute, value, size)
    );
}

x::errors::Error SugaredAPI::GetDeviceAttributeUInt32(
    const char deviceName[],
    int32 attribute,
    uInt32 *value
) {
    return process_error(dmx->GetDeviceAttributeUInt32(deviceName, attribute, value));
}

x::errors::Error SugaredAPI::GetDeviceAttributeUInt32Array(
    const char deviceName[],
    int32 attribute,
    uInt32 value[],
    uInt32 size
) {
    return process_error(
        dmx->GetDeviceAttributeUInt32Array(deviceName, attribute, value, size)
    );
}

x::errors::Error SugaredAPI::GetDigitalLogicFamilyPowerUpState(
    const char deviceName[],
    int32 *logicFamily
) {
    return process_error(
        dmx->GetDigitalLogicFamilyPowerUpState(deviceName, logicFamily)
    );
}

x::errors::Error
SugaredAPI::GetDisconnectedCDAQSyncPorts(char portList[], uInt32 portListSize) {
    return process_error(dmx->GetDisconnectedCDAQSyncPorts(portList, portListSize));
}

x::errors::Error SugaredAPI::GetExportedSignalAttributeBool(
    TaskHandle task,
    int32 attribute,
    bool32 *value
) {
    return process_error(dmx->GetExportedSignalAttributeBool(task, attribute, value));
}

x::errors::Error SugaredAPI::GetExportedSignalAttributeDouble(
    TaskHandle task,
    int32 attribute,
    float64 *value
) {
    return process_error(dmx->GetExportedSignalAttributeDouble(task, attribute, value));
}

x::errors::Error SugaredAPI::GetExportedSignalAttributeInt32(
    TaskHandle task,
    int32 attribute,
    int32 *value
) {
    return process_error(dmx->GetExportedSignalAttributeInt32(task, attribute, value));
}

x::errors::Error SugaredAPI::GetExportedSignalAttributeString(
    TaskHandle task,
    int32 attribute,
    char value[],
    uInt32 size
) {
    return process_error(
        dmx->GetExportedSignalAttributeString(task, attribute, value, size)
    );
}

x::errors::Error SugaredAPI::GetExportedSignalAttributeUInt32(
    TaskHandle task,
    int32 attribute,
    uInt32 *value
) {
    return process_error(dmx->GetExportedSignalAttributeUInt32(task, attribute, value));
}

x::errors::Error SugaredAPI::GetExtCalLastDateAndTime(
    const char deviceName[],
    uInt32 *year,
    uInt32 *month,
    uInt32 *day,
    uInt32 *hour,
    uInt32 *minute
) {
    return process_error(
        dmx->GetExtCalLastDateAndTime(deviceName, year, month, day, hour, minute)
    );
}

x::errors::Error
SugaredAPI::GetExtendedErrorInfo(char errorString[], uInt32 bufferSize) {
    return process_error(dmx->GetExtendedErrorInfo(errorString, bufferSize));
}

x::errors::Error
SugaredAPI::GetFirstSampClkWhen(TaskHandle task, CVIAbsoluteTime *data) {
    return process_error(dmx->GetFirstSampClkWhen(task, data));
}

x::errors::Error
SugaredAPI::GetFirstSampTimestampVal(TaskHandle task, CVIAbsoluteTime *data) {
    return process_error(dmx->GetFirstSampTimestampVal(task, data));
}

x::errors::Error SugaredAPI::GetNthTaskChannel(
    TaskHandle task,
    uInt32 index,
    char buffer[],
    int32 bufferSize
) {
    return process_error(dmx->GetNthTaskChannel(task, index, buffer, bufferSize));
}

x::errors::Error SugaredAPI::GetNthTaskDevice(
    TaskHandle task,
    uInt32 index,
    char buffer[],
    int32 bufferSize
) {
    return process_error(dmx->GetNthTaskDevice(task, index, buffer, bufferSize));
}

x::errors::Error SugaredAPI::GetNthTaskReadChannel(
    TaskHandle task,
    uInt32 index,
    char buffer[],
    int32 bufferSize
) {
    return process_error(dmx->GetNthTaskReadChannel(task, index, buffer, bufferSize));
}

x::errors::Error SugaredAPI::GetPersistedChanAttributeBool(
    const char channel[],
    int32 attribute,
    bool32 *value
) {
    return process_error(dmx->GetPersistedChanAttributeBool(channel, attribute, value));
}

x::errors::Error SugaredAPI::GetPersistedChanAttributeString(
    const char channel[],
    int32 attribute,
    char value[],
    uInt32 size
) {
    return process_error(
        dmx->GetPersistedChanAttributeString(channel, attribute, value, size)
    );
}

x::errors::Error SugaredAPI::GetPersistedScaleAttributeBool(
    const char scaleName[],
    int32 attribute,
    bool32 *value
) {
    return process_error(
        dmx->GetPersistedScaleAttributeBool(scaleName, attribute, value)
    );
}

x::errors::Error SugaredAPI::GetPersistedScaleAttributeString(
    const char scaleName[],
    int32 attribute,
    char value[],
    uInt32 size
) {
    return process_error(
        dmx->GetPersistedScaleAttributeString(scaleName, attribute, value, size)
    );
}

x::errors::Error SugaredAPI::GetPersistedTaskAttributeBool(
    const char taskName[],
    int32 attribute,
    bool32 *value
) {
    return process_error(
        dmx->GetPersistedTaskAttributeBool(taskName, attribute, value)
    );
}

x::errors::Error SugaredAPI::GetPersistedTaskAttributeString(
    const char taskName[],
    int32 attribute,
    char value[],
    uInt32 size
) {
    return process_error(
        dmx->GetPersistedTaskAttributeString(taskName, attribute, value, size)
    );
}

x::errors::Error SugaredAPI::GetPhysicalChanAttributeBool(
    const char physicalChannel[],
    int32 attribute,
    bool32 *value
) {
    return process_error(
        dmx->GetPhysicalChanAttributeBool(physicalChannel, attribute, value)
    );
}

x::errors::Error SugaredAPI::GetPhysicalChanAttributeBytes(
    const char physicalChannel[],
    int32 attribute,
    uInt8 value[],
    uInt32 size
) {
    return process_error(
        dmx->GetPhysicalChanAttributeBytes(physicalChannel, attribute, value, size)
    );
}

x::errors::Error SugaredAPI::GetPhysicalChanAttributeDouble(
    const char physicalChannel[],
    int32 attribute,
    float64 *value
) {
    return process_error(
        dmx->GetPhysicalChanAttributeDouble(physicalChannel, attribute, value)
    );
}

x::errors::Error SugaredAPI::GetPhysicalChanAttributeDoubleArray(
    const char physicalChannel[],
    int32 attribute,
    float64 value[],
    uInt32 size
) {
    return process_error(dmx->GetPhysicalChanAttributeDoubleArray(
        physicalChannel,
        attribute,
        value,
        size
    ));
}

x::errors::Error SugaredAPI::GetPhysicalChanAttributeInt32(
    const char physicalChannel[],
    int32 attribute,
    int32 *value
) {
    return process_error(
        dmx->GetPhysicalChanAttributeInt32(physicalChannel, attribute, value)
    );
}

x::errors::Error SugaredAPI::GetPhysicalChanAttributeInt32Array(
    const char physicalChannel[],
    int32 attribute,
    int32 value[],
    uInt32 size
) {
    return process_error(
        dmx->GetPhysicalChanAttributeInt32Array(physicalChannel, attribute, value, size)
    );
}

x::errors::Error SugaredAPI::GetPhysicalChanAttributeString(
    const char physicalChannel[],
    int32 attribute,
    char value[],
    uInt32 size
) {
    return process_error(
        dmx->GetPhysicalChanAttributeString(physicalChannel, attribute, value, size)
    );
}

x::errors::Error SugaredAPI::GetPhysicalChanAttributeUInt32(
    const char physicalChannel[],
    int32 attribute,
    uInt32 *value
) {
    return process_error(
        dmx->GetPhysicalChanAttributeUInt32(physicalChannel, attribute, value)
    );
}

x::errors::Error SugaredAPI::GetPhysicalChanAttributeUInt32Array(
    const char physicalChannel[],
    int32 attribute,
    uInt32 value[],
    uInt32 size
) {
    return process_error(dmx->GetPhysicalChanAttributeUInt32Array(
        physicalChannel,
        attribute,
        value,
        size
    ));
}

x::errors::Error
SugaredAPI::GetReadAttributeBool(TaskHandle task, int32 attribute, bool32 *value) {
    return process_error(dmx->GetReadAttributeBool(task, attribute, value));
}

x::errors::Error
SugaredAPI::GetReadAttributeDouble(TaskHandle task, int32 attribute, float64 *value) {
    return process_error(dmx->GetReadAttributeDouble(task, attribute, value));
}

x::errors::Error
SugaredAPI::GetReadAttributeInt32(TaskHandle task, int32 attribute, int32 *value) {
    return process_error(dmx->GetReadAttributeInt32(task, attribute, value));
}

x::errors::Error SugaredAPI::GetReadAttributeString(
    TaskHandle task,
    int32 attribute,
    char value[],
    uInt32 size
) {
    return process_error(dmx->GetReadAttributeString(task, attribute, value, size));
}

x::errors::Error
SugaredAPI::GetReadAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) {
    return process_error(dmx->GetReadAttributeUInt32(task, attribute, value));
}

x::errors::Error
SugaredAPI::GetReadAttributeUInt64(TaskHandle task, int32 attribute, uInt64 *value) {
    return process_error(dmx->GetReadAttributeUInt64(task, attribute, value));
}

x::errors::Error
SugaredAPI::GetRealTimeAttributeBool(TaskHandle task, int32 attribute, bool32 *value) {
    return process_error(dmx->GetRealTimeAttributeBool(task, attribute, value));
}

x::errors::Error
SugaredAPI::GetRealTimeAttributeInt32(TaskHandle task, int32 attribute, int32 *value) {
    return process_error(dmx->GetRealTimeAttributeInt32(task, attribute, value));
}

x::errors::Error SugaredAPI::GetRealTimeAttributeUInt32(
    TaskHandle task,
    int32 attribute,
    uInt32 *value
) {
    return process_error(dmx->GetRealTimeAttributeUInt32(task, attribute, value));
}

x::errors::Error
SugaredAPI::GetRefTrigTimestampVal(TaskHandle task, CVIAbsoluteTime *data) {
    return process_error(dmx->GetRefTrigTimestampVal(task, data));
}

x::errors::Error SugaredAPI::GetScaleAttributeDoubleArray(
    const char scaleName[],
    int32 attribute,
    float64 value[],
    uInt32 size
) {
    return process_error(
        dmx->GetScaleAttributeDoubleArray(scaleName, attribute, value, size)
    );
}

x::errors::Error SugaredAPI::GetScaleAttributeInt32(
    const char scaleName[],
    int32 attribute,
    int32 *value
) {
    return process_error(dmx->GetScaleAttributeInt32(scaleName, attribute, value));
}

x::errors::Error SugaredAPI::GetScaleAttributeString(
    const char scaleName[],
    int32 attribute,
    char value[],
    uInt32 size
) {
    return process_error(
        dmx->GetScaleAttributeString(scaleName, attribute, value, size)
    );
}

x::errors::Error SugaredAPI::GetSelfCalLastDateAndTime(
    const char deviceName[],
    uInt32 *year,
    uInt32 *month,
    uInt32 *day,
    uInt32 *hour,
    uInt32 *minute
) {
    return process_error(
        dmx->GetSelfCalLastDateAndTime(deviceName, year, month, day, hour, minute)
    );
}

x::errors::Error
SugaredAPI::GetStartTrigTimestampVal(TaskHandle task, CVIAbsoluteTime *data) {
    return process_error(dmx->GetStartTrigTimestampVal(task, data));
}

x::errors::Error
SugaredAPI::GetStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime *data) {
    return process_error(dmx->GetStartTrigTrigWhen(task, data));
}

x::errors::Error
SugaredAPI::GetSyncPulseTimeWhen(TaskHandle task, CVIAbsoluteTime *data) {
    return process_error(dmx->GetSyncPulseTimeWhen(task, data));
}

x::errors::Error
SugaredAPI::GetSystemInfoAttributeString(int32 attribute, char value[], uInt32 size) {
    return process_error(dmx->GetSystemInfoAttributeString(attribute, value, size));
}

x::errors::Error
SugaredAPI::GetSystemInfoAttributeUInt32(int32 attribute, uInt32 *value) {
    return process_error(dmx->GetSystemInfoAttributeUInt32(attribute, value));
}

x::errors::Error
SugaredAPI::GetTaskAttributeBool(TaskHandle task, int32 attribute, bool32 *value) {
    return process_error(dmx->GetTaskAttributeBool(task, attribute, value));
}

x::errors::Error SugaredAPI::GetTaskAttributeString(
    TaskHandle task,
    int32 attribute,
    char value[],
    uInt32 size
) {
    return process_error(dmx->GetTaskAttributeString(task, attribute, value, size));
}

x::errors::Error
SugaredAPI::GetTaskAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) {
    return process_error(dmx->GetTaskAttributeUInt32(task, attribute, value));
}

x::errors::Error
SugaredAPI::GetTimingAttributeBool(TaskHandle task, int32 attribute, bool32 *value) {
    return process_error(dmx->GetTimingAttributeBool(task, attribute, value));
}

x::errors::Error
SugaredAPI::GetTimingAttributeDouble(TaskHandle task, int32 attribute, float64 *value) {
    return process_error(dmx->GetTimingAttributeDouble(task, attribute, value));
}

x::errors::Error SugaredAPI::GetTimingAttributeExBool(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    bool32 *value
) {
    return process_error(
        dmx->GetTimingAttributeExBool(task, deviceNames, attribute, value)
    );
}

x::errors::Error SugaredAPI::GetTimingAttributeExDouble(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    float64 *value
) {
    return process_error(
        dmx->GetTimingAttributeExDouble(task, deviceNames, attribute, value)
    );
}

x::errors::Error SugaredAPI::GetTimingAttributeExInt32(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    int32 *value
) {
    return process_error(
        dmx->GetTimingAttributeExInt32(task, deviceNames, attribute, value)
    );
}

x::errors::Error SugaredAPI::GetTimingAttributeExString(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    char value[],
    uInt32 size
) {
    return process_error(
        dmx->GetTimingAttributeExString(task, deviceNames, attribute, value, size)
    );
}

x::errors::Error SugaredAPI::GetTimingAttributeExTimestamp(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    CVIAbsoluteTime *value
) {
    return process_error(
        dmx->GetTimingAttributeExTimestamp(task, deviceNames, attribute, value)
    );
}

x::errors::Error SugaredAPI::GetTimingAttributeExUInt32(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    uInt32 *value
) {
    return process_error(
        dmx->GetTimingAttributeExUInt32(task, deviceNames, attribute, value)
    );
}

x::errors::Error SugaredAPI::GetTimingAttributeExUInt64(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute,
    uInt64 *value
) {
    return process_error(
        dmx->GetTimingAttributeExUInt64(task, deviceNames, attribute, value)
    );
}

x::errors::Error
SugaredAPI::GetTimingAttributeInt32(TaskHandle task, int32 attribute, int32 *value) {
    return process_error(dmx->GetTimingAttributeInt32(task, attribute, value));
}

x::errors::Error SugaredAPI::GetTimingAttributeString(
    TaskHandle task,
    int32 attribute,
    char value[],
    uInt32 size
) {
    return process_error(dmx->GetTimingAttributeString(task, attribute, value, size));
}

x::errors::Error SugaredAPI::GetTimingAttributeTimestamp(
    TaskHandle task,
    int32 attribute,
    CVIAbsoluteTime *value
) {
    return process_error(dmx->GetTimingAttributeTimestamp(task, attribute, value));
}

x::errors::Error
SugaredAPI::GetTimingAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) {
    return process_error(dmx->GetTimingAttributeUInt32(task, attribute, value));
}

x::errors::Error
SugaredAPI::GetTimingAttributeUInt64(TaskHandle task, int32 attribute, uInt64 *value) {
    return process_error(dmx->GetTimingAttributeUInt64(task, attribute, value));
}

x::errors::Error
SugaredAPI::GetTrigAttributeBool(TaskHandle task, int32 attribute, bool32 *value) {
    return process_error(dmx->GetTrigAttributeBool(task, attribute, value));
}

x::errors::Error
SugaredAPI::GetTrigAttributeDouble(TaskHandle task, int32 attribute, float64 *value) {
    return process_error(dmx->GetTrigAttributeDouble(task, attribute, value));
}

x::errors::Error SugaredAPI::GetTrigAttributeDoubleArray(
    TaskHandle task,
    int32 attribute,
    float64 value[],
    uInt32 size
) {
    return process_error(
        dmx->GetTrigAttributeDoubleArray(task, attribute, value, size)
    );
}

x::errors::Error
SugaredAPI::GetTrigAttributeInt32(TaskHandle task, int32 attribute, int32 *value) {
    return process_error(dmx->GetTrigAttributeInt32(task, attribute, value));
}

x::errors::Error SugaredAPI::GetTrigAttributeInt32Array(
    TaskHandle task,
    int32 attribute,
    int32 value[],
    uInt32 size
) {
    return process_error(dmx->GetTrigAttributeInt32Array(task, attribute, value, size));
}

x::errors::Error SugaredAPI::GetTrigAttributeString(
    TaskHandle task,
    int32 attribute,
    char value[],
    uInt32 size
) {
    return process_error(dmx->GetTrigAttributeString(task, attribute, value, size));
}

x::errors::Error SugaredAPI::GetTrigAttributeTimestamp(
    TaskHandle task,
    int32 attribute,
    CVIAbsoluteTime *value
) {
    return process_error(dmx->GetTrigAttributeTimestamp(task, attribute, value));
}

x::errors::Error
SugaredAPI::GetTrigAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) {
    return process_error(dmx->GetTrigAttributeUInt32(task, attribute, value));
}

x::errors::Error SugaredAPI::GetWatchdogAttributeBool(
    TaskHandle task,
    const char lines[],
    int32 attribute,
    bool32 *value
) {
    return process_error(dmx->GetWatchdogAttributeBool(task, lines, attribute, value));
}

x::errors::Error SugaredAPI::GetWatchdogAttributeDouble(
    TaskHandle task,
    const char lines[],
    int32 attribute,
    float64 *value
) {
    return process_error(
        dmx->GetWatchdogAttributeDouble(task, lines, attribute, value)
    );
}

x::errors::Error SugaredAPI::GetWatchdogAttributeInt32(
    TaskHandle task,
    const char lines[],
    int32 attribute,
    int32 *value
) {
    return process_error(dmx->GetWatchdogAttributeInt32(task, lines, attribute, value));
}

x::errors::Error SugaredAPI::GetWatchdogAttributeString(
    TaskHandle task,
    const char lines[],
    int32 attribute,
    char value[],
    uInt32 size
) {
    return process_error(
        dmx->GetWatchdogAttributeString(task, lines, attribute, value, size)
    );
}

x::errors::Error
SugaredAPI::GetWriteAttributeBool(TaskHandle task, int32 attribute, bool32 *value) {
    return process_error(dmx->GetWriteAttributeBool(task, attribute, value));
}

x::errors::Error
SugaredAPI::GetWriteAttributeDouble(TaskHandle task, int32 attribute, float64 *value) {
    return process_error(dmx->GetWriteAttributeDouble(task, attribute, value));
}

x::errors::Error
SugaredAPI::GetWriteAttributeInt32(TaskHandle task, int32 attribute, int32 *value) {
    return process_error(dmx->GetWriteAttributeInt32(task, attribute, value));
}

x::errors::Error SugaredAPI::GetWriteAttributeString(
    TaskHandle task,
    int32 attribute,
    char value[],
    uInt32 size
) {
    return process_error(dmx->GetWriteAttributeString(task, attribute, value, size));
}

x::errors::Error
SugaredAPI::GetWriteAttributeUInt32(TaskHandle task, int32 attribute, uInt32 *value) {
    return process_error(dmx->GetWriteAttributeUInt32(task, attribute, value));
}

x::errors::Error
SugaredAPI::GetWriteAttributeUInt64(TaskHandle task, int32 attribute, uInt64 *value) {
    return process_error(dmx->GetWriteAttributeUInt64(task, attribute, value));
}

x::errors::Error SugaredAPI::IsTaskDone(TaskHandle task, bool32 *isTaskDone) {
    return process_error(dmx->IsTaskDone(task, isTaskDone));
}

x::errors::Error SugaredAPI::LoadTask(const char sessionName[], TaskHandle *task) {
    return process_error(dmx->LoadTask(sessionName, task));
}

x::errors::Error SugaredAPI::PerformBridgeOffsetNullingCalEx(
    TaskHandle task,
    const char channel[],
    bool32 skipUnsupportedChannels
) {
    return process_error(
        dmx->PerformBridgeOffsetNullingCalEx(task, channel, skipUnsupportedChannels)
    );
}

x::errors::Error SugaredAPI::PerformBridgeShuntCalEx(
    TaskHandle task,
    const char channel[],
    float64 shuntResistorValue,
    int32 shuntResistorLocation,
    int32 shuntResistorSelect,
    int32 shuntResistorSource,
    float64 bridgeResistance,
    bool32 skipUnsupportedChannels
) {
    return process_error(dmx->PerformBridgeShuntCalEx(
        task,
        channel,
        shuntResistorValue,
        shuntResistorLocation,
        shuntResistorSelect,
        shuntResistorSource,
        bridgeResistance,
        skipUnsupportedChannels
    ));
}

x::errors::Error SugaredAPI::PerformStrainShuntCalEx(
    TaskHandle task,
    const char channel[],
    float64 shuntResistorValue,
    int32 shuntResistorLocation,
    int32 shuntResistorSelect,
    int32 shuntResistorSource,
    bool32 skipUnsupportedChannels
) {
    return process_error(dmx->PerformStrainShuntCalEx(
        task,
        channel,
        shuntResistorValue,
        shuntResistorLocation,
        shuntResistorSelect,
        shuntResistorSource,
        skipUnsupportedChannels
    ));
}

x::errors::Error SugaredAPI::PerformThrmcplLeadOffsetNullingCal(
    TaskHandle task,
    const char channel[],
    bool32 skipUnsupportedChannels
) {
    return process_error(
        dmx->PerformThrmcplLeadOffsetNullingCal(task, channel, skipUnsupportedChannels)
    );
}

x::errors::Error SugaredAPI::ReadAnalogF64(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    float64 readArray[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return process_error(dmx->ReadAnalogF64(
        task,
        numSampsPerChan,
        timeout,
        fillMode,
        readArray,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    ));
}

x::errors::Error SugaredAPI::ReadAnalogScalarF64(
    TaskHandle task,
    float64 timeout,
    float64 *value,
    bool32 *reserved
) {
    return process_error(dmx->ReadAnalogScalarF64(task, timeout, value, reserved));
}

x::errors::Error SugaredAPI::ReadBinaryI16(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    int16 readArray[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return process_error(dmx->ReadBinaryI16(
        task,
        numSampsPerChan,
        timeout,
        fillMode,
        readArray,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    ));
}

x::errors::Error SugaredAPI::ReadBinaryI32(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    int32 readArray[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return process_error(dmx->ReadBinaryI32(
        task,
        numSampsPerChan,
        timeout,
        fillMode,
        readArray,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    ));
}

x::errors::Error SugaredAPI::ReadBinaryU16(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    uInt16 readArray[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return process_error(dmx->ReadBinaryU16(
        task,
        numSampsPerChan,
        timeout,
        fillMode,
        readArray,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    ));
}

x::errors::Error SugaredAPI::ReadBinaryU32(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    uInt32 readArray[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return process_error(dmx->ReadBinaryU32(
        task,
        numSampsPerChan,
        timeout,
        fillMode,
        readArray,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    ));
}

x::errors::Error SugaredAPI::ReadCounterF64(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    float64 readArray[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return process_error(dmx->ReadCounterF64(
        task,
        numSampsPerChan,
        timeout,
        readArray,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    ));
}

x::errors::Error SugaredAPI::ReadCounterF64Ex(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    float64 readArray[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return process_error(dmx->ReadCounterF64Ex(
        task,
        numSampsPerChan,
        timeout,
        fillMode,
        readArray,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    ));
}

x::errors::Error SugaredAPI::ReadCounterScalarF64(
    TaskHandle task,
    float64 timeout,
    float64 *value,
    bool32 *reserved
) {
    return process_error(dmx->ReadCounterScalarF64(task, timeout, value, reserved));
}

x::errors::Error SugaredAPI::ReadCounterScalarU32(
    TaskHandle task,
    float64 timeout,
    uInt32 *value,
    bool32 *reserved
) {
    return process_error(dmx->ReadCounterScalarU32(task, timeout, value, reserved));
}

x::errors::Error SugaredAPI::ReadCounterU32(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    uInt32 readArray[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return process_error(dmx->ReadCounterU32(
        task,
        numSampsPerChan,
        timeout,
        readArray,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    ));
}

x::errors::Error SugaredAPI::ReadCounterU32Ex(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    uInt32 readArray[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return process_error(dmx->ReadCounterU32Ex(
        task,
        numSampsPerChan,
        timeout,
        fillMode,
        readArray,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    ));
}

x::errors::Error SugaredAPI::ReadCtrFreq(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 interleaved,
    float64 readArrayFrequency[],
    float64 readArrayDutyCycle[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return process_error(dmx->ReadCtrFreq(
        task,
        numSampsPerChan,
        timeout,
        interleaved,
        readArrayFrequency,
        readArrayDutyCycle,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    ));
}

x::errors::Error SugaredAPI::ReadCtrFreqScalar(
    TaskHandle task,
    float64 timeout,
    float64 *frequency,
    float64 *dutyCycle,
    bool32 *reserved
) {
    return process_error(
        dmx->ReadCtrFreqScalar(task, timeout, frequency, dutyCycle, reserved)
    );
}

x::errors::Error SugaredAPI::ReadCtrTicks(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 interleaved,
    uInt32 readArrayHighTicks[],
    uInt32 readArrayLowTicks[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return process_error(dmx->ReadCtrTicks(
        task,
        numSampsPerChan,
        timeout,
        interleaved,
        readArrayHighTicks,
        readArrayLowTicks,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    ));
}

x::errors::Error SugaredAPI::ReadCtrTicksScalar(
    TaskHandle task,
    float64 timeout,
    uInt32 *highTicks,
    uInt32 *lowTicks,
    bool32 *reserved
) {
    return process_error(
        dmx->ReadCtrTicksScalar(task, timeout, highTicks, lowTicks, reserved)
    );
}

x::errors::Error SugaredAPI::ReadCtrTime(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 interleaved,
    float64 readArrayHighTime[],
    float64 readArrayLowTime[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return process_error(dmx->ReadCtrTime(
        task,
        numSampsPerChan,
        timeout,
        interleaved,
        readArrayHighTime,
        readArrayLowTime,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    ));
}

x::errors::Error SugaredAPI::ReadCtrTimeScalar(
    TaskHandle task,
    float64 timeout,
    float64 *highTime,
    float64 *lowTime,
    bool32 *reserved
) {
    return process_error(
        dmx->ReadCtrTimeScalar(task, timeout, highTime, lowTime, reserved)
    );
}

x::errors::Error SugaredAPI::ReadDigitalLines(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    uInt8 readArray[],
    uInt32 arraySizeInBytes,
    int32 *sampsPerChanRead,
    int32 *numBytesPerSamp,
    bool32 *reserved
) {
    return process_error(dmx->ReadDigitalLines(
        task,
        numSampsPerChan,
        timeout,
        fillMode,
        readArray,
        arraySizeInBytes,
        sampsPerChanRead,
        numBytesPerSamp,
        reserved
    ));
}

x::errors::Error SugaredAPI::ReadDigitalScalarU32(
    TaskHandle task,
    float64 timeout,
    uInt32 *value,
    bool32 *reserved
) {
    return process_error(dmx->ReadDigitalScalarU32(task, timeout, value, reserved));
}

x::errors::Error SugaredAPI::ReadDigitalU16(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    uInt16 readArray[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return process_error(dmx->ReadDigitalU16(
        task,
        numSampsPerChan,
        timeout,
        fillMode,
        readArray,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    ));
}

x::errors::Error SugaredAPI::ReadDigitalU32(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    uInt32 readArray[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return process_error(dmx->ReadDigitalU32(
        task,
        numSampsPerChan,
        timeout,
        fillMode,
        readArray,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    ));
}

x::errors::Error SugaredAPI::ReadDigitalU8(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    uInt8 readArray[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return process_error(dmx->ReadDigitalU8(
        task,
        numSampsPerChan,
        timeout,
        fillMode,
        readArray,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    ));
}

x::errors::Error SugaredAPI::ReadPowerBinaryI16(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    int16 readArrayVoltage[],
    int16 readArrayCurrent[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return process_error(dmx->ReadPowerBinaryI16(
        task,
        numSampsPerChan,
        timeout,
        fillMode,
        readArrayVoltage,
        readArrayCurrent,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    ));
}

x::errors::Error SugaredAPI::ReadPowerF64(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    int32 fillMode,
    float64 readArrayVoltage[],
    float64 readArrayCurrent[],
    uInt32 arraySizeInSamps,
    int32 *sampsPerChanRead,
    bool32 *reserved
) {
    return process_error(dmx->ReadPowerF64(
        task,
        numSampsPerChan,
        timeout,
        fillMode,
        readArrayVoltage,
        readArrayCurrent,
        arraySizeInSamps,
        sampsPerChanRead,
        reserved
    ));
}

x::errors::Error SugaredAPI::ReadPowerScalarF64(
    TaskHandle task,
    float64 timeout,
    float64 *voltage,
    float64 *current,
    bool32 *reserved
) {
    return process_error(
        dmx->ReadPowerScalarF64(task, timeout, voltage, current, reserved)
    );
}

x::errors::Error SugaredAPI::ReadRaw(
    TaskHandle task,
    int32 numSampsPerChan,
    float64 timeout,
    uInt8 readArray[],
    uInt32 arraySizeInBytes,
    int32 *sampsRead,
    int32 *numBytesPerSamp,
    bool32 *reserved
) {
    return process_error(dmx->ReadRaw(
        task,
        numSampsPerChan,
        timeout,
        readArray,
        arraySizeInBytes,
        sampsRead,
        numBytesPerSamp,
        reserved
    ));
}

x::errors::Error SugaredAPI::RegisterDoneEvent(
    TaskHandle task,
    uInt32 options,
    DAQmxDoneEventCallbackPtr callbackFunction,
    void *callbackData
) {
    return process_error(
        dmx->RegisterDoneEvent(task, options, callbackFunction, callbackData)
    );
}

x::errors::Error SugaredAPI::RegisterEveryNSamplesEvent(
    TaskHandle task,
    int32 everyNSamplesEventType,
    uInt32 nSamples,
    uInt32 options,
    DAQmxEveryNSamplesEventCallbackPtr callbackFunction,
    void *callbackData
) {
    return process_error(dmx->RegisterEveryNSamplesEvent(
        task,
        everyNSamplesEventType,
        nSamples,
        options,
        callbackFunction,
        callbackData
    ));
}

x::errors::Error SugaredAPI::RegisterSignalEvent(
    TaskHandle task,
    int32 signalID,
    uInt32 options,
    DAQmxSignalEventCallbackPtr callbackFunction,
    void *callbackData
) {
    return process_error(dmx->RegisterSignalEvent(
        task,
        signalID,
        options,
        callbackFunction,
        callbackData
    ));
}

x::errors::Error SugaredAPI::RemoveCDAQSyncConnection(const char portList[]) {
    return process_error(dmx->RemoveCDAQSyncConnection(portList));
}

x::errors::Error
SugaredAPI::ReserveNetworkDevice(const char deviceName[], bool32 overrideReservation) {
    return process_error(dmx->ReserveNetworkDevice(deviceName, overrideReservation));
}

x::errors::Error SugaredAPI::ResetBufferAttribute(TaskHandle task, int32 attribute) {
    return process_error(dmx->ResetBufferAttribute(task, attribute));
}

x::errors::Error
SugaredAPI::ResetChanAttribute(TaskHandle task, const char channel[], int32 attribute) {
    return process_error(dmx->ResetChanAttribute(task, channel, attribute));
}

x::errors::Error SugaredAPI::ResetDevice(const char deviceName[]) {
    return process_error(dmx->ResetDevice(deviceName));
}

x::errors::Error SugaredAPI::ResetRealTimeAttribute(TaskHandle task, int32 attribute) {
    return process_error(dmx->ResetRealTimeAttribute(task, attribute));
}

x::errors::Error SugaredAPI::ResetTimingAttribute(TaskHandle task, int32 attribute) {
    return process_error(dmx->ResetTimingAttribute(task, attribute));
}

x::errors::Error SugaredAPI::ResetTimingAttributeEx(
    TaskHandle task,
    const char deviceNames[],
    int32 attribute
) {
    return process_error(dmx->ResetTimingAttributeEx(task, deviceNames, attribute));
}

x::errors::Error SugaredAPI::ResetTrigAttribute(TaskHandle task, int32 attribute) {
    return process_error(dmx->ResetTrigAttribute(task, attribute));
}

x::errors::Error SugaredAPI::ResetWriteAttribute(TaskHandle task, int32 attribute) {
    return process_error(dmx->ResetWriteAttribute(task, attribute));
}

x::errors::Error SugaredAPI::RestoreLastExtCalConst(const char deviceName[]) {
    return process_error(dmx->RestoreLastExtCalConst(deviceName));
}

x::errors::Error SugaredAPI::SaveGlobalChan(
    TaskHandle task,
    const char channelName[],
    const char saveAs[],
    const char author[],
    uInt32 options
) {
    return process_error(
        dmx->SaveGlobalChan(task, channelName, saveAs, author, options)
    );
}

x::errors::Error SugaredAPI::SaveTask(
    TaskHandle task,
    const char saveAs[],
    const char author[],
    uInt32 options
) {
    return process_error(dmx->SaveTask(task, saveAs, author, options));
}

x::errors::Error SugaredAPI::SelfCal(const char deviceName[]) {
    return process_error(dmx->SelfCal(deviceName));
}

x::errors::Error SugaredAPI::SelfTestDevice(const char deviceName[]) {
    return process_error(dmx->SelfTestDevice(deviceName));
}

x::errors::Error SugaredAPI::SetAIChanCalCalDate(
    TaskHandle task,
    const char channelName[],
    uInt32 year,
    uInt32 month,
    uInt32 day,
    uInt32 hour,
    uInt32 minute
) {
    return process_error(
        dmx->SetAIChanCalCalDate(task, channelName, year, month, day, hour, minute)
    );
}

x::errors::Error SugaredAPI::SetAIChanCalExpDate(
    TaskHandle task,
    const char channelName[],
    uInt32 year,
    uInt32 month,
    uInt32 day,
    uInt32 hour,
    uInt32 minute
) {
    return process_error(
        dmx->SetAIChanCalExpDate(task, channelName, year, month, day, hour, minute)
    );
}

x::errors::Error SugaredAPI::SetAnalogPowerUpStatesWithOutputType(
    const char channelNames[],
    const float64 stateArray[],
    const int32 channelTypeArray[],
    uInt32 arraySize
) {
    return process_error(dmx->SetAnalogPowerUpStatesWithOutputType(
        channelNames,
        stateArray,
        channelTypeArray,
        arraySize
    ));
}

x::errors::Error
SugaredAPI::SetArmStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime data) {
    return process_error(dmx->SetArmStartTrigTrigWhen(task, data));
}

x::errors::Error
SugaredAPI::SetBufferAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) {
    return process_error(dmx->SetBufferAttributeUInt32(task, attribute, value));
}

x::errors::Error SugaredAPI::SetCalInfoAttributeBool(
    const char deviceName[],
    int32 attribute,
    bool32 value
) {
    return process_error(dmx->SetCalInfoAttributeBool(deviceName, attribute, value));
}

x::errors::Error SugaredAPI::SetCalInfoAttributeDouble(
    const char deviceName[],
    int32 attribute,
    float64 value
) {
    return process_error(dmx->SetCalInfoAttributeDouble(deviceName, attribute, value));
}

x::errors::Error SugaredAPI::SetCalInfoAttributeString(
    const char deviceName[],
    int32 attribute,
    const char value[]
) {
    return process_error(dmx->SetCalInfoAttributeString(deviceName, attribute, value));
}

x::errors::Error SugaredAPI::SetCalInfoAttributeUInt32(
    const char deviceName[],
    int32 attribute,
    uInt32 value
) {
    return process_error(dmx->SetCalInfoAttributeUInt32(deviceName, attribute, value));
}

x::errors::Error SugaredAPI::SetChanAttributeBool(
    TaskHandle task,
    const char channel[],
    int32 attribute,
    bool32 value
) {
    return process_error(dmx->SetChanAttributeBool(task, channel, attribute, value));
}

x::errors::Error SugaredAPI::SetChanAttributeDouble(
    TaskHandle task,
    const char channel[],
    int32 attribute,
    float64 value
) {
    return process_error(dmx->SetChanAttributeDouble(task, channel, attribute, value));
}

x::errors::Error SugaredAPI::SetChanAttributeDoubleArray(
    TaskHandle task,
    const char channel[],
    int32 attribute,
    const float64 value[],
    uInt32 size
) {
    return process_error(
        dmx->SetChanAttributeDoubleArray(task, channel, attribute, value, size)
    );
}

x::errors::Error SugaredAPI::SetChanAttributeInt32(
    TaskHandle task,
    const char channel[],
    int32 attribute,
    int32 value
) {
    return process_error(dmx->SetChanAttributeInt32(task, channel, attribute, value));
}

x::errors::Error SugaredAPI::SetChanAttributeString(
    TaskHandle task,
    const char channel[],
    int32 attribute,
    const char value[]
) {
    return process_error(dmx->SetChanAttributeString(task, channel, attribute, value));
}

x::errors::Error SugaredAPI::SetChanAttributeUInt32(
    TaskHandle task,
    const char channel[],
    int32 attribute,
    uInt32 value
) {
    return process_error(dmx->SetChanAttributeUInt32(task, channel, attribute, value));
}

x::errors::Error SugaredAPI::SetDigitalLogicFamilyPowerUpState(
    const char deviceName[],
    int32 logicFamily
) {
    return process_error(
        dmx->SetDigitalLogicFamilyPowerUpState(deviceName, logicFamily)
    );
}

x::errors::Error SugaredAPI::SetExportedSignalAttributeBool(
    TaskHandle task,
    int32 attribute,
    bool32 value
) {
    return process_error(dmx->SetExportedSignalAttributeBool(task, attribute, value));
}

x::errors::Error SugaredAPI::SetExportedSignalAttributeDouble(
    TaskHandle task,
    int32 attribute,
    float64 value
) {
    return process_error(dmx->SetExportedSignalAttributeDouble(task, attribute, value));
}

x::errors::Error SugaredAPI::SetExportedSignalAttributeInt32(
    TaskHandle task,
    int32 attribute,
    int32 value
) {
    return process_error(dmx->SetExportedSignalAttributeInt32(task, attribute, value));
}

x::errors::Error SugaredAPI::SetExportedSignalAttributeString(
    TaskHandle task,
    int32 attribute,
    const char value[]
) {
    return process_error(dmx->SetExportedSignalAttributeString(task, attribute, value));
}

x::errors::Error SugaredAPI::SetExportedSignalAttributeUInt32(
    TaskHandle task,
    int32 attribute,
    uInt32 value
) {
    return process_error(dmx->SetExportedSignalAttributeUInt32(task, attribute, value));
}

x::errors::Error
SugaredAPI::SetFirstSampClkWhen(TaskHandle task, CVIAbsoluteTime data) {
    return process_error(dmx->SetFirstSampClkWhen(task, data));
}

x::errors::Error
SugaredAPI::SetReadAttributeBool(TaskHandle task, int32 attribute, bool32 value) {
    return process_error(dmx->SetReadAttributeBool(task, attribute, value));
}

x::errors::Error
SugaredAPI::SetReadAttributeDouble(TaskHandle task, int32 attribute, float64 value) {
    return process_error(dmx->SetReadAttributeDouble(task, attribute, value));
}

x::errors::Error
SugaredAPI::SetReadAttributeInt32(TaskHandle task, int32 attribute, int32 value) {
    return process_error(dmx->SetReadAttributeInt32(task, attribute, value));
}

x::errors::Error SugaredAPI::SetReadAttributeString(
    TaskHandle task,
    int32 attribute,
    const char value[]
) {
    return process_error(dmx->SetReadAttributeString(task, attribute, value));
}

x::errors::Error
SugaredAPI::SetReadAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) {
    return process_error(dmx->SetReadAttributeUInt32(task, attribute, value));
}

x::errors::Error
SugaredAPI::SetReadAttributeUInt64(TaskHandle task, int32 attribute, uInt64 value) {
    return process_error(dmx->SetReadAttributeUInt64(task, attribute, value));
}

x::errors::Error
SugaredAPI::SetRealTimeAttributeBool(TaskHandle task, int32 attribute, bool32 value) {
    return process_error(dmx->SetRealTimeAttributeBool(task, attribute, value));
}

x::errors::Error
SugaredAPI::SetRealTimeAttributeInt32(TaskHandle task, int32 attribute, int32 value) {
    return process_error(dmx->SetRealTimeAttributeInt32(task, attribute, value));
}

x::errors::Error
SugaredAPI::SetRealTimeAttributeUInt32(TaskHandle task, int32 attribute, uInt32 value) {
    return process_error(dmx->SetRealTimeAttributeUInt32(task, attribute, value));
}

x::errors::Error SugaredAPI::SetRuntimeEnvironment(
    const char environment[],
    const char environmentVersion[],
    const char reserved1[],
    const char reserved2[]
) {
    return process_error(dmx->SetRuntimeEnvironment(
        environment,
        environmentVersion,
        reserved1,
        reserved2
    ));
}

x::errors::Error SugaredAPI::CreateCIAngEncoderChan(
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
) {
    return process_error(dmx->CreateCIAngEncoderChan(
        task,
        counter,
        nameToAssignToChannel,
        decodingType,
        zidxEnable,
        zidxVal,
        zidxPhase,
        units,
        pulsesPerRev,
        initialAngle,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateCIAngVelocityChan(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 decodingType,
    int32 units,
    uInt32 pulsesPerRev,
    const char customScaleName[]
) {
    return process_error(dmx->CreateCIAngVelocityChan(
        task,
        counter,
        nameToAssignToChannel,
        minVal,
        maxVal,
        decodingType,
        units,
        pulsesPerRev,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateCICountEdgesChan(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    int32 edge,
    uInt32 initialCount,
    int32 countDirection
) {
    return process_error(dmx->CreateCICountEdgesChan(
        task,
        counter,
        nameToAssignToChannel,
        edge,
        initialCount,
        countDirection
    ));
}

x::errors::Error SugaredAPI::CreateCIDutyCycleChan(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    float64 minFreq,
    float64 maxFreq,
    int32 edge,
    const char customScaleName[]
) {
    return process_error(dmx->CreateCIDutyCycleChan(
        task,
        counter,
        nameToAssignToChannel,
        minFreq,
        maxFreq,
        edge,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateCIFreqChan(
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
) {
    return process_error(dmx->CreateCIFreqChan(
        task,
        counter,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        edge,
        measMethod,
        measTime,
        divisor,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateCIGPSTimestampChan(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    int32 units,
    int32 syncMethod,
    const char customScaleName[]
) {
    return process_error(dmx->CreateCIGPSTimestampChan(
        task,
        counter,
        nameToAssignToChannel,
        units,
        syncMethod,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateCILinEncoderChan(
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
) {
    return process_error(dmx->CreateCILinEncoderChan(
        task,
        counter,
        nameToAssignToChannel,
        decodingType,
        zidxEnable,
        zidxVal,
        zidxPhase,
        units,
        distPerPulse,
        initialPos,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateCILinVelocityChan(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 decodingType,
    int32 units,
    float64 distPerPulse,
    const char customScaleName[]
) {
    return process_error(dmx->CreateCILinVelocityChan(
        task,
        counter,
        nameToAssignToChannel,
        minVal,
        maxVal,
        decodingType,
        units,
        distPerPulse,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateCIPeriodChan(
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
) {
    return process_error(dmx->CreateCIPeriodChan(
        task,
        counter,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        edge,
        measMethod,
        measTime,
        divisor,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateCIPulseChanFreq(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units
) {
    return process_error(dmx->CreateCIPulseChanFreq(
        task,
        counter,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units
    ));
}

x::errors::Error SugaredAPI::CreateCIPulseChanTicks(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    const char sourceTerminal[],
    float64 minVal,
    float64 maxVal
) {
    return process_error(dmx->CreateCIPulseChanTicks(
        task,
        counter,
        nameToAssignToChannel,
        sourceTerminal,
        minVal,
        maxVal
    ));
}

x::errors::Error SugaredAPI::CreateCIPulseChanTime(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units
) {
    return process_error(dmx->CreateCIPulseChanTime(
        task,
        counter,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units
    ));
}

x::errors::Error SugaredAPI::CreateCIPulseWidthChan(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 startingEdge,
    const char customScaleName[]
) {
    return process_error(dmx->CreateCIPulseWidthChan(
        task,
        counter,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        startingEdge,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateCISemiPeriodChan(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    const char customScaleName[]
) {
    return process_error(dmx->CreateCISemiPeriodChan(
        task,
        counter,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateCITwoEdgeSepChan(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 firstEdge,
    int32 secondEdge,
    const char customScaleName[]
) {
    return process_error(dmx->CreateCITwoEdgeSepChan(
        task,
        counter,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        firstEdge,
        secondEdge,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateCOPulseChanFreq(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    int32 units,
    int32 idleState,
    float64 initialDelay,
    float64 freq,
    float64 dutyCycle
) {
    return process_error(dmx->CreateCOPulseChanFreq(
        task,
        counter,
        nameToAssignToChannel,
        units,
        idleState,
        initialDelay,
        freq,
        dutyCycle
    ));
}

x::errors::Error SugaredAPI::CreateCOPulseChanTicks(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    const char sourceTerminal[],
    int32 idleState,
    int32 initialDelay,
    int32 lowTicks,
    int32 highTicks
) {
    return process_error(dmx->CreateCOPulseChanTicks(
        task,
        counter,
        nameToAssignToChannel,
        sourceTerminal,
        idleState,
        initialDelay,
        lowTicks,
        highTicks
    ));
}

x::errors::Error SugaredAPI::CreateCOPulseChanTime(
    TaskHandle task,
    const char counter[],
    const char nameToAssignToChannel[],
    int32 units,
    int32 idleState,
    float64 initialDelay,
    float64 lowTime,
    float64 highTime
) {
    return process_error(dmx->CreateCOPulseChanTime(
        task,
        counter,
        nameToAssignToChannel,
        units,
        idleState,
        initialDelay,
        lowTime,
        highTime
    ));
}

x::errors::Error SugaredAPI::CreateDIChan(
    TaskHandle task,
    const char lines[],
    const char nameToAssignToLines[],
    int32 lineGrouping
) {
    return process_error(
        dmx->CreateDIChan(task, lines, nameToAssignToLines, lineGrouping)
    );
}

x::errors::Error SugaredAPI::CreateDOChan(
    TaskHandle task,
    const char lines[],
    const char nameToAssignToLines[],
    int32 lineGrouping
) {
    return process_error(
        dmx->CreateDOChan(task, lines, nameToAssignToLines, lineGrouping)
    );
}

x::errors::Error SugaredAPI::CreateTEDSAIAccelChan(
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
) {
    return process_error(dmx->CreateTEDSAIAccelChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        currentExcitSource,
        currentExcitVal,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateTEDSAIBridgeChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 voltageExcitSource,
    float64 voltageExcitVal,
    const char customScaleName[]
) {
    return process_error(dmx->CreateTEDSAIBridgeChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        voltageExcitSource,
        voltageExcitVal,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateTEDSAICurrentChan(
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
) {
    return process_error(dmx->CreateTEDSAICurrentChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        shuntResistorLoc,
        extShuntResistorVal,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateTEDSAIForceBridgeChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 voltageExcitSource,
    float64 voltageExcitVal,
    const char customScaleName[]
) {
    return process_error(dmx->CreateTEDSAIForceBridgeChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        voltageExcitSource,
        voltageExcitVal,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateTEDSAIForceIEPEChan(
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
) {
    return process_error(dmx->CreateTEDSAIForceIEPEChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        currentExcitSource,
        currentExcitVal,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateTEDSAIMicrophoneChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 terminalConfig,
    int32 units,
    float64 maxSndPressLevel,
    int32 currentExcitSource,
    float64 currentExcitVal,
    const char customScaleName[]
) {
    return process_error(dmx->CreateTEDSAIMicrophoneChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        units,
        maxSndPressLevel,
        currentExcitSource,
        currentExcitVal,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateTEDSAIPosLVDTChan(
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
) {
    return process_error(dmx->CreateTEDSAIPosLVDTChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        voltageExcitSource,
        voltageExcitVal,
        voltageExcitFreq,
        acExcitWireMode,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateTEDSAIPosRVDTChan(
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
) {
    return process_error(dmx->CreateTEDSAIPosRVDTChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        voltageExcitSource,
        voltageExcitVal,
        voltageExcitFreq,
        acExcitWireMode,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateTEDSAIPressureBridgeChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 voltageExcitSource,
    float64 voltageExcitVal,
    const char customScaleName[]
) {
    return process_error(dmx->CreateTEDSAIPressureBridgeChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        voltageExcitSource,
        voltageExcitVal,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateTEDSAIRTDChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 resistanceConfig,
    int32 currentExcitSource,
    float64 currentExcitVal
) {
    return process_error(dmx->CreateTEDSAIRTDChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        resistanceConfig,
        currentExcitSource,
        currentExcitVal
    ));
}

x::errors::Error SugaredAPI::CreateTEDSAIResistanceChan(
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
) {
    return process_error(dmx->CreateTEDSAIResistanceChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        resistanceConfig,
        currentExcitSource,
        currentExcitVal,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateTEDSAIStrainGageChan(
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
) {
    return process_error(dmx->CreateTEDSAIStrainGageChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        voltageExcitSource,
        voltageExcitVal,
        initialBridgeVoltage,
        leadWireResistance,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateTEDSAIThrmcplChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 cjcSource,
    float64 cjcVal,
    const char cjcChannel[]
) {
    return process_error(dmx->CreateTEDSAIThrmcplChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        cjcSource,
        cjcVal,
        cjcChannel
    ));
}

x::errors::Error SugaredAPI::CreateTEDSAIThrmstrChanIex(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 resistanceConfig,
    int32 currentExcitSource,
    float64 currentExcitVal
) {
    return process_error(dmx->CreateTEDSAIThrmstrChanIex(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        resistanceConfig,
        currentExcitSource,
        currentExcitVal
    ));
}

x::errors::Error SugaredAPI::CreateTEDSAIThrmstrChanVex(
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
) {
    return process_error(dmx->CreateTEDSAIThrmstrChanVex(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        resistanceConfig,
        voltageExcitSource,
        voltageExcitVal,
        r1
    ));
}

x::errors::Error SugaredAPI::CreateTEDSAITorqueBridgeChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    float64 minVal,
    float64 maxVal,
    int32 units,
    int32 voltageExcitSource,
    float64 voltageExcitVal,
    const char customScaleName[]
) {
    return process_error(dmx->CreateTEDSAITorqueBridgeChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        minVal,
        maxVal,
        units,
        voltageExcitSource,
        voltageExcitVal,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateTEDSAIVoltageChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 terminalConfig,
    float64 minVal,
    float64 maxVal,
    int32 units,
    const char customScaleName[]
) {
    return process_error(dmx->CreateTEDSAIVoltageChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateTEDSAIVoltageChanWithExcit(
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
) {
    return process_error(dmx->CreateTEDSAIVoltageChanWithExcit(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        voltageExcitSource,
        voltageExcitVal,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAIVoltageChanWithExcit(
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
) {
    return process_error(dmx->CreateAIVoltageChanWithExcit(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        bridgeConfig,
        voltageExcitSource,
        voltageExcitVal,
        useExcitForScaling,
        customScaleName
    ));
}

x::errors::Error SugaredAPI::CreateAIVoltageRMSChan(
    TaskHandle task,
    const char physicalChannel[],
    const char nameToAssignToChannel[],
    int32 terminalConfig,
    float64 minVal,
    float64 maxVal,
    int32 units,
    const char customScaleName[]
) {
    return process_error(dmx->CreateAIVoltageRMSChan(
        task,
        physicalChannel,
        nameToAssignToChannel,
        terminalConfig,
        minVal,
        maxVal,
        units,
        customScaleName
    ));
}
}
