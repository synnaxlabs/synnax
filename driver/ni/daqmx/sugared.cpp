// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

#include "driver/ni/daqmx/sugared.h"

#include "driver/ni/errors/errors.h"

xerrors::Error SugaredDAQmx::process_error(const int32 status) const {
    return parse_error(dmx, status);
}

xerrors::Error SugaredDAQmx::AddCDAQSyncConnection(const char portList[]) {
    return process_error(dmx->AddCDAQSyncConnection(portList));
}

xerrors::Error SugaredDAQmx::AddGlobalChansToTask(TaskHandle task,
                                                  const char channelNames[]) {
    return process_error(dmx->AddGlobalChansToTask(task, channelNames));
}

xerrors::Error SugaredDAQmx::AddNetworkDevice(const char ipAddress[],
                                              const char deviceName[],
                                              bool32 attemptReservation,
                                              float64 timeout, char deviceNameOut[],
                                              uInt32 deviceNameOutBufferSize) {
    return process_error(dmx->AddNetworkDevice(ipAddress, deviceName,
                                               attemptReservation, timeout,
                                               deviceNameOut, deviceNameOutBufferSize));
}

xerrors::Error SugaredDAQmx::AreConfiguredCDAQSyncPortsDisconnected(
    const char chassisDevicesPorts[], float64 timeout, bool32 *disconnectedPortsExist) {
    return process_error(
        dmx->AreConfiguredCDAQSyncPortsDisconnected(chassisDevicesPorts, timeout,
                                                    disconnectedPortsExist));
}

xerrors::Error SugaredDAQmx::AutoConfigureCDAQSyncConnections(
    const char chassisDevicesPorts[], float64 timeout) {
    return process_error(
        dmx->AutoConfigureCDAQSyncConnections(chassisDevicesPorts, timeout));
}

xerrors::Error SugaredDAQmx::CalculateReversePolyCoeff(
    const float64 forwardCoeffs[], uInt32 numForwardCoeffsIn, float64 minValX,
    float64 maxValX, int32 numPointsToCompute, int32 reversePolyOrder,
    float64 reverseCoeffs[]) {
    return process_error(dmx->CalculateReversePolyCoeff(
        forwardCoeffs, numForwardCoeffsIn, minValX, maxValX, numPointsToCompute,
        reversePolyOrder, reverseCoeffs));
}

xerrors::Error SugaredDAQmx::CfgAnlgEdgeRefTrig(TaskHandle task,
                                                const char triggerSource[],
                                                int32 triggerSlope,
                                                float64 triggerLevel,
                                                uInt32 pretriggerSamples) {
    return process_error(dmx->CfgAnlgEdgeRefTrig(task, triggerSource, triggerSlope,
                                                 triggerLevel, pretriggerSamples));
}

xerrors::Error SugaredDAQmx::CfgAnlgEdgeStartTrig(TaskHandle task,
                                                  const char triggerSource[],
                                                  int32 triggerSlope,
                                                  float64 triggerLevel) {
    return process_error(
        dmx->CfgAnlgEdgeStartTrig(task, triggerSource, triggerSlope, triggerLevel));
}

xerrors::Error SugaredDAQmx::CfgAnlgMultiEdgeRefTrig(
    TaskHandle task, const char triggerSources[], const int32 triggerSlopeArray[],
    const float64 triggerLevelArray[], uInt32 pretriggerSamples, uInt32 arraySize) {
    return process_error(dmx->CfgAnlgMultiEdgeRefTrig(
        task, triggerSources, triggerSlopeArray, triggerLevelArray, pretriggerSamples,
        arraySize));
}

xerrors::Error SugaredDAQmx::CfgAnlgMultiEdgeStartTrig(
    TaskHandle task, const char triggerSources[], const int32 triggerSlopeArray[],
    const float64 triggerLevelArray[], uInt32 arraySize) {
    return process_error(dmx->CfgAnlgMultiEdgeStartTrig(
        task, triggerSources, triggerSlopeArray, triggerLevelArray, arraySize));
}

xerrors::Error SugaredDAQmx::CfgAnlgWindowRefTrig(TaskHandle task,
                                                  const char triggerSource[],
                                                  int32 triggerWhen, float64 windowTop,
                                                  float64 windowBottom,
                                                  uInt32 pretriggerSamples) {
    return process_error(dmx->CfgAnlgWindowRefTrig(task, triggerSource, triggerWhen,
                                                   windowTop, windowBottom,
                                                   pretriggerSamples));
}

xerrors::Error SugaredDAQmx::CfgAnlgWindowStartTrig(TaskHandle task,
                                                    const char triggerSource[],
                                                    int32 triggerWhen,
                                                    float64 windowTop,
                                                    float64 windowBottom) {
    return process_error(
        dmx->CfgAnlgWindowStartTrig(task, triggerSource, triggerWhen, windowTop,
                                    windowBottom));
}

xerrors::Error SugaredDAQmx::CfgBurstHandshakingTimingExportClock(
    TaskHandle task, int32 sampleMode, uInt64 sampsPerChan, float64 sampleClkRate,
    const char sampleClkOutpTerm[], int32 sampleClkPulsePolarity, int32 pauseWhen,
    int32 readyEventActiveLevel) {
    return process_error(dmx->CfgBurstHandshakingTimingExportClock(
        task, sampleMode, sampsPerChan, sampleClkRate, sampleClkOutpTerm,
        sampleClkPulsePolarity, pauseWhen, readyEventActiveLevel));
}

xerrors::Error SugaredDAQmx::CfgBurstHandshakingTimingImportClock(
    TaskHandle task, int32 sampleMode, uInt64 sampsPerChan, float64 sampleClkRate,
    const char sampleClkSrc[], int32 sampleClkActiveEdge, int32 pauseWhen,
    int32 readyEventActiveLevel) {
    return process_error(dmx->CfgBurstHandshakingTimingImportClock(
        task, sampleMode, sampsPerChan, sampleClkRate, sampleClkSrc,
        sampleClkActiveEdge, pauseWhen, readyEventActiveLevel));
}

xerrors::Error SugaredDAQmx::CfgChangeDetectionTiming(
    TaskHandle task, const char risingEdgeChan[], const char fallingEdgeChan[],
    int32 sampleMode, uInt64 sampsPerChan) {
    return process_error(dmx->CfgChangeDetectionTiming(
        task, risingEdgeChan, fallingEdgeChan, sampleMode, sampsPerChan));
}

xerrors::Error SugaredDAQmx::CfgDigEdgeRefTrig(TaskHandle task,
                                               const char triggerSource[],
                                               int32 triggerEdge,
                                               uInt32 pretriggerSamples) {
    return process_error(
        dmx->CfgDigEdgeRefTrig(task, triggerSource, triggerEdge, pretriggerSamples));
}

xerrors::Error SugaredDAQmx::CfgDigEdgeStartTrig(TaskHandle task,
                                                 const char triggerSource[],
                                                 int32 triggerEdge) {
    return process_error(dmx->CfgDigEdgeStartTrig(task, triggerSource, triggerEdge));
}

xerrors::Error SugaredDAQmx::CfgDigPatternRefTrig(TaskHandle task,
                                                  const char triggerSource[],
                                                  const char triggerPattern[],
                                                  int32 triggerWhen,
                                                  uInt32 pretriggerSamples) {
    return process_error(dmx->CfgDigPatternRefTrig(task, triggerSource, triggerPattern,
                                                   triggerWhen, pretriggerSamples));
}

xerrors::Error SugaredDAQmx::CfgDigPatternStartTrig(TaskHandle task,
                                                    const char triggerSource[],
                                                    const char triggerPattern[],
                                                    int32 triggerWhen) {
    return process_error(
        dmx->CfgDigPatternStartTrig(task, triggerSource, triggerPattern, triggerWhen));
}

xerrors::Error SugaredDAQmx::CfgHandshakingTiming(TaskHandle task, int32 sampleMode,
                                                  uInt64 sampsPerChan) {
    return process_error(dmx->CfgHandshakingTiming(task, sampleMode, sampsPerChan));
}

xerrors::Error SugaredDAQmx::CfgImplicitTiming(TaskHandle task, int32 sampleMode,
                                               uInt64 sampsPerChan) {
    return process_error(dmx->CfgImplicitTiming(task, sampleMode, sampsPerChan));
}

xerrors::Error SugaredDAQmx::CfgInputBuffer(TaskHandle task, uInt32 numSampsPerChan) {
    return process_error(dmx->CfgInputBuffer(task, numSampsPerChan));
}

xerrors::Error SugaredDAQmx::CfgOutputBuffer(TaskHandle task, uInt32 numSampsPerChan) {
    return process_error(dmx->CfgOutputBuffer(task, numSampsPerChan));
}

xerrors::Error SugaredDAQmx::CfgPipelinedSampClkTiming(
    TaskHandle task, const char source[], float64 rate, int32 activeEdge,
    int32 sampleMode, uInt64 sampsPerChan) {
    return process_error(
        dmx->CfgPipelinedSampClkTiming(task, source, rate, activeEdge, sampleMode,
                                       sampsPerChan));
}

xerrors::Error SugaredDAQmx::CfgSampClkTiming(TaskHandle task, const char source[],
                                              float64 rate, int32 activeEdge,
                                              int32 sampleMode, uInt64 sampsPerChan) {
    return process_error(
        dmx->CfgSampClkTiming(task, source, rate, activeEdge, sampleMode,
                              sampsPerChan));
}

xerrors::Error SugaredDAQmx::CfgTimeStartTrig(TaskHandle task, CVIAbsoluteTime when,
                                              int32 timescale) {
    return process_error(dmx->CfgTimeStartTrig(task, when, timescale));
}

xerrors::Error SugaredDAQmx::CfgWatchdogAOExpirStates(
    TaskHandle task, const char channelNames[], const float64 expirStateArray[],
    const int32 outputTypeArray[], uInt32 arraySize) {
    return process_error(dmx->CfgWatchdogAOExpirStates(
        task, channelNames, expirStateArray, outputTypeArray, arraySize));
}

xerrors::Error SugaredDAQmx::CfgWatchdogCOExpirStates(
    TaskHandle task, const char channelNames[], const int32 expirStateArray[],
    uInt32 arraySize) {
    return process_error(
        dmx->CfgWatchdogCOExpirStates(task, channelNames, expirStateArray, arraySize));
}

xerrors::Error SugaredDAQmx::CfgWatchdogDOExpirStates(
    TaskHandle task, const char channelNames[], const int32 expirStateArray[],
    uInt32 arraySize) {
    return process_error(
        dmx->CfgWatchdogDOExpirStates(task, channelNames, expirStateArray, arraySize));
}

xerrors::Error SugaredDAQmx::ClearTEDS(const char physicalChannel[]) {
    return process_error(dmx->ClearTEDS(physicalChannel));
}

xerrors::Error SugaredDAQmx::ClearTask(TaskHandle task) {
    return process_error(dmx->ClearTask(task));
}

xerrors::Error SugaredDAQmx::ConfigureLogging(TaskHandle task, const char filePath[],
                                              int32 loggingMode, const char groupName[],
                                              int32 operation) {
    return process_error(
        dmx->ConfigureLogging(task, filePath, loggingMode, groupName, operation));
}

xerrors::Error SugaredDAQmx::ConfigureTEDS(const char physicalChannel[],
                                           const char filePath[]) {
    return process_error(dmx->ConfigureTEDS(physicalChannel, filePath));
}

xerrors::Error SugaredDAQmx::ConnectTerms(const char sourceTerminal[],
                                          const char destinationTerminal[],
                                          int32 signalModifiers) {
    return process_error(
        dmx->ConnectTerms(sourceTerminal, destinationTerminal, signalModifiers));
}

xerrors::Error SugaredDAQmx::ControlWatchdogTask(TaskHandle task, int32 action) {
    return process_error(dmx->ControlWatchdogTask(task, action));
}

xerrors::Error SugaredDAQmx::CreateAIAccel4WireDCVoltageChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    int32 terminalConfig, float64 minVal, float64 maxVal, int32 units,
    float64 sensitivity, int32 sensitivityUnits, int32 voltageExcitSource,
    float64 voltageExcitVal, bool32 useExcitForScaling, const char customScaleName[]) {
    return process_error(dmx->CreateAIAccel4WireDCVoltageChan(
        task, physicalChannel, nameToAssignToChannel, terminalConfig, minVal, maxVal,
        units, sensitivity, sensitivityUnits, voltageExcitSource, voltageExcitVal,
        useExcitForScaling, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIAccelChan(TaskHandle task,
                                               const char physicalChannel[],
                                               const char nameToAssignToChannel[],
                                               int32 terminalConfig, float64 minVal,
                                               float64 maxVal, int32 units,
                                               float64 sensitivity,
                                               int32 sensitivityUnits,
                                               int32 currentExcitSource,
                                               float64 currentExcitVal,
                                               const char customScaleName[]) {
    return process_error(dmx->CreateAIAccelChan(task, physicalChannel,
                                                nameToAssignToChannel, terminalConfig,
                                                minVal, maxVal, units, sensitivity,
                                                sensitivityUnits, currentExcitSource,
                                                currentExcitVal, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIAccelChargeChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    int32 terminalConfig, float64 minVal, float64 maxVal, int32 units,
    float64 sensitivity, int32 sensitivityUnits, const char customScaleName[]) {
    return process_error(dmx->CreateAIAccelChargeChan(
        task, physicalChannel, nameToAssignToChannel, terminalConfig, minVal, maxVal,
        units, sensitivity, sensitivityUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIBridgeChan(TaskHandle task,
                                                const char physicalChannel[],
                                                const char nameToAssignToChannel[],
                                                float64 minVal, float64 maxVal,
                                                int32 units, int32 bridgeConfig,
                                                int32 voltageExcitSource,
                                                float64 voltageExcitVal,
                                                float64 nominalBridgeResistance,
                                                const char customScaleName[]) {
    return process_error(dmx->CreateAIBridgeChan(task, physicalChannel,
                                                 nameToAssignToChannel, minVal, maxVal,
                                                 units, bridgeConfig,
                                                 voltageExcitSource, voltageExcitVal,
                                                 nominalBridgeResistance,
                                                 customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIChargeChan(TaskHandle task,
                                                const char physicalChannel[],
                                                const char nameToAssignToChannel[],
                                                int32 terminalConfig, float64 minVal,
                                                float64 maxVal, int32 units,
                                                const char customScaleName[]) {
    return process_error(dmx->CreateAIChargeChan(task, physicalChannel,
                                                 nameToAssignToChannel, terminalConfig,
                                                 minVal, maxVal, units,
                                                 customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAICurrentChan(TaskHandle task,
                                                 const char physicalChannel[],
                                                 const char nameToAssignToChannel[],
                                                 int32 terminalConfig, float64 minVal,
                                                 float64 maxVal, int32 units,
                                                 int32 shuntResistorLoc,
                                                 float64 extShuntResistorVal,
                                                 const char customScaleName[]) {
    return process_error(dmx->CreateAICurrentChan(task, physicalChannel,
                                                  nameToAssignToChannel, terminalConfig,
                                                  minVal, maxVal, units,
                                                  shuntResistorLoc, extShuntResistorVal,
                                                  customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAICurrentRMSChan(TaskHandle task,
                                                    const char physicalChannel[],
                                                    const char nameToAssignToChannel[],
                                                    int32 terminalConfig,
                                                    float64 minVal, float64 maxVal,
                                                    int32 units, int32 shuntResistorLoc,
                                                    float64 extShuntResistorVal,
                                                    const char customScaleName[]) {
    return process_error(dmx->CreateAICurrentRMSChan(
        task, physicalChannel, nameToAssignToChannel, terminalConfig, minVal, maxVal,
        units, shuntResistorLoc, extShuntResistorVal, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIForceBridgePolynomialChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    float64 minVal, float64 maxVal, int32 units, int32 bridgeConfig,
    int32 voltageExcitSource, float64 voltageExcitVal, float64 nominalBridgeResistance,
    const float64 forwardCoeffs[], uInt32 numForwardCoeffs,
    const float64 reverseCoeffs[], uInt32 numReverseCoeffs, int32 electricalUnits,
    int32 physicalUnits, const char customScaleName[]) {
    return process_error(dmx->CreateAIForceBridgePolynomialChan(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units,
        bridgeConfig, voltageExcitSource, voltageExcitVal, nominalBridgeResistance,
        forwardCoeffs, numForwardCoeffs, reverseCoeffs, numReverseCoeffs,
        electricalUnits, physicalUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIForceBridgeTableChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    float64 minVal, float64 maxVal, int32 units, int32 bridgeConfig,
    int32 voltageExcitSource, float64 voltageExcitVal, float64 nominalBridgeResistance,
    const float64 electricalVals[], uInt32 numElectricalVals, int32 electricalUnits,
    const float64 physicalVals[], uInt32 numPhysicalVals, int32 physicalUnits,
    const char customScaleName[]) {
    return process_error(dmx->CreateAIForceBridgeTableChan(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units,
        bridgeConfig, voltageExcitSource, voltageExcitVal, nominalBridgeResistance,
        electricalVals, numElectricalVals, electricalUnits, physicalVals,
        numPhysicalVals, physicalUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIForceBridgeTwoPointLinChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    float64 minVal, float64 maxVal, int32 units, int32 bridgeConfig,
    int32 voltageExcitSource, float64 voltageExcitVal, float64 nominalBridgeResistance,
    float64 firstElectricalVal, float64 secondElectricalVal, int32 electricalUnits,
    float64 firstPhysicalVal, float64 secondPhysicalVal, int32 physicalUnits,
    const char customScaleName[]) {
    return process_error(dmx->CreateAIForceBridgeTwoPointLinChan(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units,
        bridgeConfig, voltageExcitSource, voltageExcitVal, nominalBridgeResistance,
        firstElectricalVal, secondElectricalVal, electricalUnits, firstPhysicalVal,
        secondPhysicalVal, physicalUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIForceIEPEChan(TaskHandle task,
                                                   const char physicalChannel[],
                                                   const char nameToAssignToChannel[],
                                                   int32 terminalConfig, float64 minVal,
                                                   float64 maxVal, int32 units,
                                                   float64 sensitivity,
                                                   int32 sensitivityUnits,
                                                   int32 currentExcitSource,
                                                   float64 currentExcitVal,
                                                   const char customScaleName[]) {
    return process_error(dmx->CreateAIForceIEPEChan(task, physicalChannel,
                                                    nameToAssignToChannel,
                                                    terminalConfig, minVal, maxVal,
                                                    units, sensitivity,
                                                    sensitivityUnits,
                                                    currentExcitSource, currentExcitVal,
                                                    customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIFreqVoltageChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    float64 minVal, float64 maxVal, int32 units, float64 thresholdLevel,
    float64 hysteresis, const char customScaleName[]) {
    return process_error(dmx->CreateAIFreqVoltageChan(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units,
        thresholdLevel, hysteresis, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIMicrophoneChan(TaskHandle task,
                                                    const char physicalChannel[],
                                                    const char nameToAssignToChannel[],
                                                    int32 terminalConfig, int32 units,
                                                    float64 micSensitivity,
                                                    float64 maxSndPressLevel,
                                                    int32 currentExcitSource,
                                                    float64 currentExcitVal,
                                                    const char customScaleName[]) {
    return process_error(dmx->CreateAIMicrophoneChan(
        task, physicalChannel, nameToAssignToChannel, terminalConfig, units,
        micSensitivity, maxSndPressLevel, currentExcitSource, currentExcitVal,
        customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIPosEddyCurrProxProbeChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    float64 minVal, float64 maxVal, int32 units, float64 sensitivity,
    int32 sensitivityUnits, const char customScaleName[]) {
    return process_error(dmx->CreateAIPosEddyCurrProxProbeChan(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units,
        sensitivity, sensitivityUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIPosLVDTChan(TaskHandle task,
                                                 const char physicalChannel[],
                                                 const char nameToAssignToChannel[],
                                                 float64 minVal, float64 maxVal,
                                                 int32 units, float64 sensitivity,
                                                 int32 sensitivityUnits,
                                                 int32 voltageExcitSource,
                                                 float64 voltageExcitVal,
                                                 float64 voltageExcitFreq,
                                                 int32 acExcitWireMode,
                                                 const char customScaleName[]) {
    return process_error(dmx->CreateAIPosLVDTChan(task, physicalChannel,
                                                  nameToAssignToChannel, minVal, maxVal,
                                                  units, sensitivity, sensitivityUnits,
                                                  voltageExcitSource, voltageExcitVal,
                                                  voltageExcitFreq, acExcitWireMode,
                                                  customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIPosRVDTChan(TaskHandle task,
                                                 const char physicalChannel[],
                                                 const char nameToAssignToChannel[],
                                                 float64 minVal, float64 maxVal,
                                                 int32 units, float64 sensitivity,
                                                 int32 sensitivityUnits,
                                                 int32 voltageExcitSource,
                                                 float64 voltageExcitVal,
                                                 float64 voltageExcitFreq,
                                                 int32 acExcitWireMode,
                                                 const char customScaleName[]) {
    return process_error(dmx->CreateAIPosRVDTChan(task, physicalChannel,
                                                  nameToAssignToChannel, minVal, maxVal,
                                                  units, sensitivity, sensitivityUnits,
                                                  voltageExcitSource, voltageExcitVal,
                                                  voltageExcitFreq, acExcitWireMode,
                                                  customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIPowerChan(TaskHandle task,
                                               const char physicalChannel[],
                                               const char nameToAssignToChannel[],
                                               float64 voltageSetpoint,
                                               float64 currentSetpoint,
                                               bool32 outputEnable) {
    return process_error(dmx->CreateAIPowerChan(task, physicalChannel,
                                                nameToAssignToChannel, voltageSetpoint,
                                                currentSetpoint, outputEnable));
}

xerrors::Error SugaredDAQmx::CreateAIPressureBridgePolynomialChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    float64 minVal, float64 maxVal, int32 units, int32 bridgeConfig,
    int32 voltageExcitSource, float64 voltageExcitVal, float64 nominalBridgeResistance,
    const float64 forwardCoeffs[], uInt32 numForwardCoeffs,
    const float64 reverseCoeffs[], uInt32 numReverseCoeffs, int32 electricalUnits,
    int32 physicalUnits, const char customScaleName[]) {
    return process_error(dmx->CreateAIPressureBridgePolynomialChan(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units,
        bridgeConfig, voltageExcitSource, voltageExcitVal, nominalBridgeResistance,
        forwardCoeffs, numForwardCoeffs, reverseCoeffs, numReverseCoeffs,
        electricalUnits, physicalUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIPressureBridgeTableChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    float64 minVal, float64 maxVal, int32 units, int32 bridgeConfig,
    int32 voltageExcitSource, float64 voltageExcitVal, float64 nominalBridgeResistance,
    const float64 electricalVals[], uInt32 numElectricalVals, int32 electricalUnits,
    const float64 physicalVals[], uInt32 numPhysicalVals, int32 physicalUnits,
    const char customScaleName[]) {
    return process_error(dmx->CreateAIPressureBridgeTableChan(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units,
        bridgeConfig, voltageExcitSource, voltageExcitVal, nominalBridgeResistance,
        electricalVals, numElectricalVals, electricalUnits, physicalVals,
        numPhysicalVals, physicalUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIPressureBridgeTwoPointLinChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    float64 minVal, float64 maxVal, int32 units, int32 bridgeConfig,
    int32 voltageExcitSource, float64 voltageExcitVal, float64 nominalBridgeResistance,
    float64 firstElectricalVal, float64 secondElectricalVal, int32 electricalUnits,
    float64 firstPhysicalVal, float64 secondPhysicalVal, int32 physicalUnits,
    const char customScaleName[]) {
    return process_error(dmx->CreateAIPressureBridgeTwoPointLinChan(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units,
        bridgeConfig, voltageExcitSource, voltageExcitVal, nominalBridgeResistance,
        firstElectricalVal, secondElectricalVal, electricalUnits, firstPhysicalVal,
        secondPhysicalVal, physicalUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIRTDChan(TaskHandle task,
                                             const char physicalChannel[],
                                             const char nameToAssignToChannel[],
                                             float64 minVal, float64 maxVal,
                                             int32 units, int32 rtdType,
                                             int32 resistanceConfig,
                                             int32 currentExcitSource,
                                             float64 currentExcitVal, float64 r0) {
    return process_error(dmx->CreateAIRTDChan(task, physicalChannel,
                                              nameToAssignToChannel, minVal, maxVal,
                                              units, rtdType, resistanceConfig,
                                              currentExcitSource, currentExcitVal, r0));
}

xerrors::Error SugaredDAQmx::CreateAIResistanceChan(TaskHandle task,
                                                    const char physicalChannel[],
                                                    const char nameToAssignToChannel[],
                                                    float64 minVal, float64 maxVal,
                                                    int32 units, int32 resistanceConfig,
                                                    int32 currentExcitSource,
                                                    float64 currentExcitVal,
                                                    const char customScaleName[]) {
    return process_error(dmx->CreateAIResistanceChan(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units,
        resistanceConfig, currentExcitSource, currentExcitVal, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIRosetteStrainGageChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    float64 minVal, float64 maxVal, int32 rosetteType, float64 gageOrientation,
    const int32 rosetteMeasTypes[], uInt32 numRosetteMeasTypes, int32 strainConfig,
    int32 voltageExcitSource, float64 voltageExcitVal, float64 gageFactor,
    float64 nominalGageResistance, float64 poissonRatio, float64 leadWireResistance) {
    return process_error(dmx->CreateAIRosetteStrainGageChan(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, rosetteType,
        gageOrientation, rosetteMeasTypes, numRosetteMeasTypes, strainConfig,
        voltageExcitSource, voltageExcitVal, gageFactor, nominalGageResistance,
        poissonRatio, leadWireResistance));
}

xerrors::Error SugaredDAQmx::CreateAIStrainGageChan(TaskHandle task,
                                                    const char physicalChannel[],
                                                    const char nameToAssignToChannel[],
                                                    float64 minVal, float64 maxVal,
                                                    int32 units, int32 strainConfig,
                                                    int32 voltageExcitSource,
                                                    float64 voltageExcitVal,
                                                    float64 gageFactor,
                                                    float64 initialBridgeVoltage,
                                                    float64 nominalGageResistance,
                                                    float64 poissonRatio,
                                                    float64 leadWireResistance,
                                                    const char customScaleName[]) {
    return process_error(dmx->CreateAIStrainGageChan(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units,
        strainConfig, voltageExcitSource, voltageExcitVal, gageFactor,
        initialBridgeVoltage, nominalGageResistance, poissonRatio, leadWireResistance,
        customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAITempBuiltInSensorChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    int32 units) {
    return process_error(
        dmx->CreateAITempBuiltInSensorChan(task, physicalChannel, nameToAssignToChannel,
                                           units));
}

xerrors::Error SugaredDAQmx::CreateAIThrmcplChan(TaskHandle task,
                                                 const char physicalChannel[],
                                                 const char nameToAssignToChannel[],
                                                 float64 minVal, float64 maxVal,
                                                 int32 units, int32 thermocoupleType,
                                                 int32 cjcSource, float64 cjcVal,
                                                 const char cjcChannel[]) {
    return process_error(dmx->CreateAIThrmcplChan(task, physicalChannel,
                                                  nameToAssignToChannel, minVal, maxVal,
                                                  units, thermocoupleType, cjcSource,
                                                  cjcVal, cjcChannel));
}

xerrors::Error SugaredDAQmx::CreateAIThrmstrChanIex(TaskHandle task,
                                                    const char physicalChannel[],
                                                    const char nameToAssignToChannel[],
                                                    float64 minVal, float64 maxVal,
                                                    int32 units, int32 resistanceConfig,
                                                    int32 currentExcitSource,
                                                    float64 currentExcitVal, float64 a,
                                                    float64 b, float64 c) {
    return process_error(dmx->CreateAIThrmstrChanIex(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units,
        resistanceConfig, currentExcitSource, currentExcitVal, a, b, c));
}

xerrors::Error SugaredDAQmx::SetScaleAttributeDouble(
    const char scaleName[], int32 attribute, float64 value) {
    return process_error(dmx->SetScaleAttributeDouble(scaleName, attribute, value));
}

xerrors::Error SugaredDAQmx::SetScaleAttributeDoubleArray(
    const char scaleName[], int32 attribute, const float64 value[], uInt32 size) {
    return process_error(
        dmx->SetScaleAttributeDoubleArray(scaleName, attribute, value, size));
}

xerrors::Error SugaredDAQmx::SetScaleAttributeInt32(const char scaleName[],
                                                    int32 attribute, int32 value) {
    return process_error(dmx->SetScaleAttributeInt32(scaleName, attribute, value));
}

xerrors::Error SugaredDAQmx::SetScaleAttributeString(
    const char scaleName[], int32 attribute, const char value[]) {
    return process_error(dmx->SetScaleAttributeString(scaleName, attribute, value));
}

xerrors::Error
SugaredDAQmx::SetStartTrigTrigWhen(TaskHandle task, CVIAbsoluteTime data) {
    return process_error(dmx->SetStartTrigTrigWhen(task, data));
}

xerrors::Error
SugaredDAQmx::SetSyncPulseTimeWhen(TaskHandle task, CVIAbsoluteTime data) {
    return process_error(dmx->SetSyncPulseTimeWhen(task, data));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeBool(TaskHandle task, int32 attribute,
                                                    bool32 value) {
    return process_error(dmx->SetTimingAttributeBool(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeDouble(
    TaskHandle task, int32 attribute, float64 value) {
    return process_error(dmx->SetTimingAttributeDouble(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeExBool(
    TaskHandle task, const char deviceNames[], int32 attribute, bool32 value) {
    return process_error(
        dmx->SetTimingAttributeExBool(task, deviceNames, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeExDouble(
    TaskHandle task, const char deviceNames[], int32 attribute, float64 value) {
    return process_error(
        dmx->SetTimingAttributeExDouble(task, deviceNames, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeExInt32(
    TaskHandle task, const char deviceNames[], int32 attribute, int32 value) {
    return process_error(
        dmx->SetTimingAttributeExInt32(task, deviceNames, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeExString(
    TaskHandle task, const char deviceNames[], int32 attribute, const char value[]) {
    return process_error(
        dmx->SetTimingAttributeExString(task, deviceNames, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeExTimestamp(
    TaskHandle task, const char deviceNames[], int32 attribute, CVIAbsoluteTime value) {
    return process_error(
        dmx->SetTimingAttributeExTimestamp(task, deviceNames, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeExUInt32(
    TaskHandle task, const char deviceNames[], int32 attribute, uInt32 value) {
    return process_error(
        dmx->SetTimingAttributeExUInt32(task, deviceNames, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeExUInt64(
    TaskHandle task, const char deviceNames[], int32 attribute, uInt64 value) {
    return process_error(
        dmx->SetTimingAttributeExUInt64(task, deviceNames, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeInt32(
    TaskHandle task, int32 attribute, int32 value) {
    return process_error(dmx->SetTimingAttributeInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeString(
    TaskHandle task, int32 attribute, const char value[]) {
    return process_error(dmx->SetTimingAttributeString(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeTimestamp(
    TaskHandle task, int32 attribute, CVIAbsoluteTime value) {
    return process_error(dmx->SetTimingAttributeTimestamp(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeUInt32(
    TaskHandle task, int32 attribute, uInt32 value) {
    return process_error(dmx->SetTimingAttributeUInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTimingAttributeUInt64(
    TaskHandle task, int32 attribute, uInt64 value) {
    return process_error(dmx->SetTimingAttributeUInt64(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTrigAttributeBool(TaskHandle task, int32 attribute,
                                                  bool32 value) {
    return process_error(dmx->SetTrigAttributeBool(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTrigAttributeDouble(TaskHandle task, int32 attribute,
                                                    float64 value) {
    return process_error(dmx->SetTrigAttributeDouble(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTrigAttributeDoubleArray(
    TaskHandle task, int32 attribute, const float64 value[], uInt32 size) {
    return process_error(
        dmx->SetTrigAttributeDoubleArray(task, attribute, value, size));
}

xerrors::Error SugaredDAQmx::SetTrigAttributeInt32(TaskHandle task, int32 attribute,
                                                   int32 value) {
    return process_error(dmx->SetTrigAttributeInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTrigAttributeInt32Array(
    TaskHandle task, int32 attribute, const int32 value[], uInt32 size) {
    return process_error(dmx->SetTrigAttributeInt32Array(task, attribute, value, size));
}

xerrors::Error SugaredDAQmx::SetTrigAttributeString(TaskHandle task, int32 attribute,
                                                    const char value[]) {
    return process_error(dmx->SetTrigAttributeString(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTrigAttributeTimestamp(
    TaskHandle task, int32 attribute, CVIAbsoluteTime value) {
    return process_error(dmx->SetTrigAttributeTimestamp(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetTrigAttributeUInt32(TaskHandle task, int32 attribute,
                                                    uInt32 value) {
    return process_error(dmx->SetTrigAttributeUInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetWatchdogAttributeBool(
    TaskHandle task, const char lines[], int32 attribute, bool32 value) {
    return process_error(dmx->SetWatchdogAttributeBool(task, lines, attribute, value));
}

xerrors::Error SugaredDAQmx::SetWatchdogAttributeDouble(
    TaskHandle task, const char lines[], int32 attribute, float64 value) {
    return process_error(
        dmx->SetWatchdogAttributeDouble(task, lines, attribute, value));
}

xerrors::Error SugaredDAQmx::SetWatchdogAttributeInt32(
    TaskHandle task, const char lines[], int32 attribute, int32 value) {
    return process_error(dmx->SetWatchdogAttributeInt32(task, lines, attribute, value));
}

xerrors::Error SugaredDAQmx::SetWatchdogAttributeString(
    TaskHandle task, const char lines[], int32 attribute, const char value[]) {
    return process_error(
        dmx->SetWatchdogAttributeString(task, lines, attribute, value));
}

xerrors::Error SugaredDAQmx::SetWriteAttributeBool(TaskHandle task, int32 attribute,
                                                   bool32 value) {
    return process_error(dmx->SetWriteAttributeBool(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetWriteAttributeDouble(
    TaskHandle task, int32 attribute, float64 value) {
    return process_error(dmx->SetWriteAttributeDouble(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetWriteAttributeInt32(TaskHandle task, int32 attribute,
                                                    int32 value) {
    return process_error(dmx->SetWriteAttributeInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetWriteAttributeString(
    TaskHandle task, int32 attribute, const char value[]) {
    return process_error(dmx->SetWriteAttributeString(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetWriteAttributeUInt32(
    TaskHandle task, int32 attribute, uInt32 value) {
    return process_error(dmx->SetWriteAttributeUInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetWriteAttributeUInt64(
    TaskHandle task, int32 attribute, uInt64 value) {
    return process_error(dmx->SetWriteAttributeUInt64(task, attribute, value));
}

xerrors::Error SugaredDAQmx::StartNewFile(TaskHandle task, const char filePath[]) {
    return process_error(dmx->StartNewFile(task, filePath));
}

xerrors::Error SugaredDAQmx::StartTask(TaskHandle task) {
    return process_error(dmx->StartTask(task));
}

xerrors::Error SugaredDAQmx::StopTask(TaskHandle task) {
    return process_error(dmx->StopTask(task));
}

xerrors::Error SugaredDAQmx::TaskControl(TaskHandle task, int32 action) {
    return process_error(dmx->TaskControl(task, action));
}

xerrors::Error SugaredDAQmx::TristateOutputTerm(const char outputTerminal[]) {
    return process_error(dmx->TristateOutputTerm(outputTerminal));
}

xerrors::Error SugaredDAQmx::UnregisterDoneEvent(TaskHandle task, uInt32 options,
                                                 DAQmxDoneEventCallbackPtr
                                                 callbackFunction, void *callbackData) {
    return process_error(
        dmx->UnregisterDoneEvent(task, options, callbackFunction, callbackData));
}

xerrors::Error SugaredDAQmx::UnregisterEveryNSamplesEvent(
    TaskHandle task, int32 everyNSamplesEventType, uInt32 nSamples, uInt32 options,
    DAQmxEveryNSamplesEventCallbackPtr callbackFunction, void *callbackData) {
    return process_error(dmx->UnregisterEveryNSamplesEvent(
        task, everyNSamplesEventType, nSamples, options, callbackFunction,
        callbackData));
}

xerrors::Error SugaredDAQmx::UnregisterSignalEvent(TaskHandle task, int32 signalID,
                                                   uInt32 options,
                                                   DAQmxSignalEventCallbackPtr
                                                   callbackFunction,
                                                   void *callbackData) {
    return process_error(
        dmx->UnregisterSignalEvent(task, signalID, options, callbackFunction,
                                   callbackData));
}

xerrors::Error SugaredDAQmx::UnreserveNetworkDevice(const char deviceName[]) {
    return process_error(dmx->UnreserveNetworkDevice(deviceName));
}

xerrors::Error SugaredDAQmx::WaitForNextSampleClock(TaskHandle task, float64 timeout,
                                                    bool32 *isLate) {
    return process_error(dmx->WaitForNextSampleClock(task, timeout, isLate));
}

xerrors::Error SugaredDAQmx::WaitForValidTimestamp(TaskHandle task,
                                                   int32 timestampEvent,
                                                   float64 timeout,
                                                   CVIAbsoluteTime *timestamp) {
    return process_error(
        dmx->WaitForValidTimestamp(task, timestampEvent, timeout, timestamp));
}

xerrors::Error SugaredDAQmx::WaitUntilTaskDone(TaskHandle task, float64 timeToWait) {
    return process_error(dmx->WaitUntilTaskDone(task, timeToWait));
}

xerrors::Error SugaredDAQmx::WriteAnalogF64(TaskHandle task, int32 numSampsPerChan,
                                            bool32 autoStart, float64 timeout,
                                            int32 dataLayout,
                                            const float64 writeArray[],
                                            int32 *sampsPerChanWritten,
                                            bool32 *reserved) {
    return process_error(dmx->WriteAnalogF64(task, numSampsPerChan, autoStart, timeout,
                                             dataLayout, writeArray,
                                             sampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteAnalogScalarF64(TaskHandle task, bool32 autoStart,
                                                  float64 timeout, float64 value,
                                                  bool32 *reserved) {
    return process_error(
        dmx->WriteAnalogScalarF64(task, autoStart, timeout, value, reserved));
}

xerrors::Error SugaredDAQmx::WriteBinaryI16(TaskHandle task, int32 numSampsPerChan,
                                            bool32 autoStart, float64 timeout,
                                            int32 dataLayout, const int16 writeArray[],
                                            int32 *sampsPerChanWritten,
                                            bool32 *reserved) {
    return process_error(dmx->WriteBinaryI16(task, numSampsPerChan, autoStart, timeout,
                                             dataLayout, writeArray,
                                             sampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteBinaryI32(TaskHandle task, int32 numSampsPerChan,
                                            bool32 autoStart, float64 timeout,
                                            int32 dataLayout, const int32 writeArray[],
                                            int32 *sampsPerChanWritten,
                                            bool32 *reserved) {
    return process_error(dmx->WriteBinaryI32(task, numSampsPerChan, autoStart, timeout,
                                             dataLayout, writeArray,
                                             sampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteBinaryU16(TaskHandle task, int32 numSampsPerChan,
                                            bool32 autoStart, float64 timeout,
                                            int32 dataLayout, const uInt16 writeArray[],
                                            int32 *sampsPerChanWritten,
                                            bool32 *reserved) {
    return process_error(dmx->WriteBinaryU16(task, numSampsPerChan, autoStart, timeout,
                                             dataLayout, writeArray,
                                             sampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteBinaryU32(TaskHandle task, int32 numSampsPerChan,
                                            bool32 autoStart, float64 timeout,
                                            int32 dataLayout, const uInt32 writeArray[],
                                            int32 *sampsPerChanWritten,
                                            bool32 *reserved) {
    return process_error(dmx->WriteBinaryU32(task, numSampsPerChan, autoStart, timeout,
                                             dataLayout, writeArray,
                                             sampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteCtrFreq(TaskHandle task, int32 numSampsPerChan,
                                          bool32 autoStart, float64 timeout,
                                          int32 dataLayout, const float64 frequency[],
                                          const float64 dutyCycle[],
                                          int32 *numSampsPerChanWritten,
                                          bool32 *reserved) {
    return process_error(dmx->WriteCtrFreq(task, numSampsPerChan, autoStart, timeout,
                                           dataLayout, frequency, dutyCycle,
                                           numSampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteCtrFreqScalar(TaskHandle task, bool32 autoStart,
                                                float64 timeout, float64 frequency,
                                                float64 dutyCycle, bool32 *reserved) {
    return process_error(
        dmx->WriteCtrFreqScalar(task, autoStart, timeout, frequency, dutyCycle,
                                reserved));
}

xerrors::Error SugaredDAQmx::WriteCtrTicks(TaskHandle task, int32 numSampsPerChan,
                                           bool32 autoStart, float64 timeout,
                                           int32 dataLayout, const uInt32 highTicks[],
                                           const uInt32 lowTicks[],
                                           int32 *numSampsPerChanWritten,
                                           bool32 *reserved) {
    return process_error(dmx->WriteCtrTicks(task, numSampsPerChan, autoStart, timeout,
                                            dataLayout, highTicks, lowTicks,
                                            numSampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteCtrTicksScalar(TaskHandle task, bool32 autoStart,
                                                 float64 timeout, uInt32 highTicks,
                                                 uInt32 lowTicks, bool32 *reserved) {
    return process_error(
        dmx->WriteCtrTicksScalar(task, autoStart, timeout, highTicks, lowTicks,
                                 reserved));
}

xerrors::Error SugaredDAQmx::WriteCtrTime(TaskHandle task, int32 numSampsPerChan,
                                          bool32 autoStart, float64 timeout,
                                          int32 dataLayout, const float64 highTime[],
                                          const float64 lowTime[],
                                          int32 *numSampsPerChanWritten,
                                          bool32 *reserved) {
    return process_error(dmx->WriteCtrTime(task, numSampsPerChan, autoStart, timeout,
                                           dataLayout, highTime, lowTime,
                                           numSampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteCtrTimeScalar(TaskHandle task, bool32 autoStart,
                                                float64 timeout, float64 highTime,
                                                float64 lowTime, bool32 *reserved) {
    return process_error(
        dmx->WriteCtrTimeScalar(task, autoStart, timeout, highTime, lowTime, reserved));
}

xerrors::Error SugaredDAQmx::WriteDigitalLines(TaskHandle task, int32 numSampsPerChan,
                                               bool32 autoStart, float64 timeout,
                                               int32 dataLayout,
                                               const uInt8 writeArray[],
                                               int32 *sampsPerChanWritten,
                                               bool32 *reserved) {
    return process_error(dmx->WriteDigitalLines(task, numSampsPerChan, autoStart,
                                                timeout, dataLayout, writeArray,
                                                sampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteDigitalScalarU32(TaskHandle task, bool32 autoStart,
                                                   float64 timeout, uInt32 value,
                                                   bool32 *reserved) {
    return process_error(
        dmx->WriteDigitalScalarU32(task, autoStart, timeout, value, reserved));
}

xerrors::Error SugaredDAQmx::WriteDigitalU16(TaskHandle task, int32 numSampsPerChan,
                                             bool32 autoStart, float64 timeout,
                                             int32 dataLayout,
                                             const uInt16 writeArray[],
                                             int32 *sampsPerChanWritten,
                                             bool32 *reserved) {
    return process_error(dmx->WriteDigitalU16(task, numSampsPerChan, autoStart, timeout,
                                              dataLayout, writeArray,
                                              sampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteDigitalU32(TaskHandle task, int32 numSampsPerChan,
                                             bool32 autoStart, float64 timeout,
                                             int32 dataLayout,
                                             const uInt32 writeArray[],
                                             int32 *sampsPerChanWritten,
                                             bool32 *reserved) {
    return process_error(dmx->WriteDigitalU32(task, numSampsPerChan, autoStart, timeout,
                                              dataLayout, writeArray,
                                              sampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteDigitalU8(TaskHandle task, int32 numSampsPerChan,
                                            bool32 autoStart, float64 timeout,
                                            int32 dataLayout, const uInt8 writeArray[],
                                            int32 *sampsPerChanWritten,
                                            bool32 *reserved) {
    return process_error(dmx->WriteDigitalU8(task, numSampsPerChan, autoStart, timeout,
                                             dataLayout, writeArray,
                                             sampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteRaw(TaskHandle task, int32 numSamps, bool32 autoStart,
                                      float64 timeout, const uInt8 writeArray[],
                                      int32 *sampsPerChanWritten, bool32 *reserved) {
    return process_error(dmx->WriteRaw(task, numSamps, autoStart, timeout, writeArray,
                                       sampsPerChanWritten, reserved));
}

xerrors::Error SugaredDAQmx::WriteToTEDSFromArray(const char physicalChannel[],
                                                  const uInt8 bitStream[],
                                                  uInt32 arraySize,
                                                  int32 basicTEDSOptions) {
    return process_error(
        dmx->WriteToTEDSFromArray(physicalChannel, bitStream, arraySize,
                                  basicTEDSOptions));
}

xerrors::Error SugaredDAQmx::WriteToTEDSFromFile(const char physicalChannel[],
                                                 const char filePath[],
                                                 int32 basicTEDSOptions) {
    return process_error(
        dmx->WriteToTEDSFromFile(physicalChannel, filePath, basicTEDSOptions));
}

xerrors::Error SugaredDAQmx::CreateLinScale(const char scaleName[], float64 slope,
                                            float64 yIntercept, int32 preScaledUnits,
                                            const char customScaleName[]) {
    return process_error(dmx->CreateLinScale(scaleName, slope, yIntercept,
                                             preScaledUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateMapScale(const char scaleName[],
                                            float64 prescaledMin, float64 prescaledMax,
                                            float64 scaledMin, float64 scaledMax,
                                            int32 preScaledUnits,
                                            const char customScaleName[]) {
    return process_error(dmx->CreateMapScale(scaleName, prescaledMin, prescaledMax,
                                             scaledMin, scaledMax, preScaledUnits,
                                             customScaleName));
}

xerrors::Error SugaredDAQmx::CreateTableScale(const char scaleName[],
                                              const float64 prescaledVals[],
                                              uInt32 numPrescaledVals,
                                              const float64 scaledVals[],
                                              uInt32 numScaledVals,
                                              int32 preScaledUnits,
                                              const char customScaleName[]) {
    return process_error(dmx->CreateTableScale(scaleName, prescaledVals,
                                               numPrescaledVals, scaledVals,
                                               numScaledVals, preScaledUnits,
                                               customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIVoltageChan(TaskHandle task,
                                                 const char physicalChannel[],
                                                 const char nameToAssignToChannel[],
                                                 int32 terminalConfig, float64 minVal,
                                                 float64 maxVal, int32 units,
                                                 const char customScaleName[]) {
    return process_error(dmx->CreateAIVoltageChan(task, physicalChannel,
                                                  nameToAssignToChannel, terminalConfig,
                                                  minVal, maxVal, units,
                                                  customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAOCurrentChan(TaskHandle task,
                                                 const char physicalChannel[],
                                                 const char nameToAssignToChannel[],
                                                 float64 minVal, float64 maxVal,
                                                 int32 units,
                                                 const char customScaleName[]) {
    return process_error(dmx->CreateAOCurrentChan(task, physicalChannel,
                                                  nameToAssignToChannel, minVal, maxVal,
                                                  units, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAOFuncGenChan(TaskHandle task,
                                                 const char physicalChannel[],
                                                 const char nameToAssignToChannel[],
                                                 int32 type, float64 freq,
                                                 float64 amplitude, float64 offset) {
    return process_error(dmx->CreateAOFuncGenChan(task, physicalChannel,
                                                  nameToAssignToChannel, type, freq,
                                                  amplitude, offset));
}

xerrors::Error SugaredDAQmx::CreateAOVoltageChan(TaskHandle task,
                                                 const char physicalChannel[],
                                                 const char nameToAssignToChannel[],
                                                 float64 minVal, float64 maxVal,
                                                 int32 units,
                                                 const char customScaleName[]) {
    return process_error(dmx->CreateAOVoltageChan(task, physicalChannel,
                                                  nameToAssignToChannel, minVal, maxVal,
                                                  units, customScaleName));
}

xerrors::Error SugaredDAQmx::CreatePolynomialScale(const char scaleName[],
                                                   const float64 forwardCoeffs[],
                                                   uInt32 numForwardCoeffs,
                                                   const float64 reverseCoeffs[],
                                                   uInt32 numReverseCoeffs,
                                                   int32 preScaledUnits,
                                                   const char customScaleName[]) {
    return process_error(dmx->CreatePolynomialScale(scaleName, forwardCoeffs,
                                                    numForwardCoeffs, reverseCoeffs,
                                                    numReverseCoeffs, preScaledUnits,
                                                    customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIVelocityIEPEChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    int32 terminalConfig, float64 minVal, float64 maxVal, int32 units,
    float64 sensitivity, int32 sensitivityUnits, int32 currentExcitSource,
    float64 currentExcitVal, const char customScaleName[]) {
    return process_error(dmx->CreateAIVelocityIEPEChan(
        task, physicalChannel, nameToAssignToChannel, terminalConfig, minVal, maxVal,
        units, sensitivity, sensitivityUnits, currentExcitSource, currentExcitVal,
        customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAITorqueBridgeTableChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    float64 minVal, float64 maxVal, int32 units, int32 bridgeConfig,
    int32 voltageExcitSource, float64 voltageExcitVal, float64 nominalBridgeResistance,
    const float64 electricalVals[], uInt32 numElectricalVals, int32 electricalUnits,
    const float64 physicalVals[], uInt32 numPhysicalVals, int32 physicalUnits,
    const char customScaleName[]) {
    return process_error(dmx->CreateAITorqueBridgeTableChan(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units,
        bridgeConfig, voltageExcitSource, voltageExcitVal, nominalBridgeResistance,
        electricalVals, numElectricalVals, electricalUnits, physicalVals,
        numPhysicalVals, physicalUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAITorqueBridgePolynomialChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    float64 minVal, float64 maxVal, int32 units, int32 bridgeConfig,
    int32 voltageExcitSource, float64 voltageExcitVal, float64 nominalBridgeResistance,
    const float64 forwardCoeffs[], uInt32 numForwardCoeffs,
    const float64 reverseCoeffs[], uInt32 numReverseCoeffs, int32 electricalUnits,
    int32 physicalUnits, const char customScaleName[]) {
    return process_error(dmx->CreateAITorqueBridgePolynomialChan(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units,
        bridgeConfig, voltageExcitSource, voltageExcitVal, nominalBridgeResistance,
        forwardCoeffs, numForwardCoeffs, reverseCoeffs, numReverseCoeffs,
        electricalUnits, physicalUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAITorqueBridgeTwoPointLinChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    float64 minVal, float64 maxVal, int32 units, int32 bridgeConfig,
    int32 voltageExcitSource, float64 voltageExcitVal, float64 nominalBridgeResistance,
    float64 firstElectricalVal, float64 secondElectricalVal, int32 electricalUnits,
    float64 firstPhysicalVal, float64 secondPhysicalVal, int32 physicalUnits,
    const char customScaleName[]) {
    return process_error(dmx->CreateAITorqueBridgeTwoPointLinChan(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units,
        bridgeConfig, voltageExcitSource, voltageExcitVal, nominalBridgeResistance,
        firstElectricalVal, secondElectricalVal, electricalUnits, firstPhysicalVal,
        secondPhysicalVal, physicalUnits, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateTask(const char sessionName[], TaskHandle *task) {
    return process_error(dmx->CreateTask(sessionName, task));
}

xerrors::Error SugaredDAQmx::CreateWatchdogTimerTaskEx(
    const char deviceName[], const char sessionName[], TaskHandle *task,
    float64 timeout) {
    return process_error(
        dmx->CreateWatchdogTimerTaskEx(deviceName, sessionName, task, timeout));
}

xerrors::Error SugaredDAQmx::DeleteNetworkDevice(const char deviceName[]) {
    return process_error(dmx->DeleteNetworkDevice(deviceName));
}

xerrors::Error SugaredDAQmx::DeleteSavedGlobalChan(const char channelName[]) {
    return process_error(dmx->DeleteSavedGlobalChan(channelName));
}

xerrors::Error SugaredDAQmx::DeleteSavedScale(const char scaleName[]) {
    return process_error(dmx->DeleteSavedScale(scaleName));
}

xerrors::Error SugaredDAQmx::DeleteSavedTask(const char taskName[]) {
    return process_error(dmx->DeleteSavedTask(taskName));
}

xerrors::Error SugaredDAQmx::DeviceSupportsCal(const char deviceName[],
                                               bool32 *calSupported) {
    return process_error(dmx->DeviceSupportsCal(deviceName, calSupported));
}

xerrors::Error SugaredDAQmx::DisableRefTrig(TaskHandle task) {
    return process_error(dmx->DisableRefTrig(task));
}

xerrors::Error SugaredDAQmx::DisableStartTrig(TaskHandle task) {
    return process_error(dmx->DisableStartTrig(task));
}

xerrors::Error SugaredDAQmx::DisconnectTerms(const char sourceTerminal[],
                                             const char destinationTerminal[]) {
    return process_error(dmx->DisconnectTerms(sourceTerminal, destinationTerminal));
}

xerrors::Error SugaredDAQmx::ExportSignal(TaskHandle task, int32 signalID,
                                          const char outputTerminal[]) {
    return process_error(dmx->ExportSignal(task, signalID, outputTerminal));
}

xerrors::Error SugaredDAQmx::GetAIChanCalCalDate(TaskHandle task,
                                                 const char channelName[], uInt32 *year,
                                                 uInt32 *month, uInt32 *day,
                                                 uInt32 *hour, uInt32 *minute) {
    return process_error(
        dmx->GetAIChanCalCalDate(task, channelName, year, month, day, hour, minute));
}

xerrors::Error SugaredDAQmx::GetAIChanCalExpDate(TaskHandle task,
                                                 const char channelName[], uInt32 *year,
                                                 uInt32 *month, uInt32 *day,
                                                 uInt32 *hour, uInt32 *minute) {
    return process_error(
        dmx->GetAIChanCalExpDate(task, channelName, year, month, day, hour, minute));
}

xerrors::Error SugaredDAQmx::GetAnalogPowerUpStatesWithOutputType(
    const char channelNames[], float64 stateArray[], int32 channelTypeArray[],
    uInt32 *arraySize) {
    return process_error(
        dmx->GetAnalogPowerUpStatesWithOutputType(channelNames, stateArray,
                                                  channelTypeArray, arraySize));
}

xerrors::Error SugaredDAQmx::GetArmStartTrigTimestampVal(
    TaskHandle task, CVIAbsoluteTime *data) {
    return process_error(dmx->GetArmStartTrigTimestampVal(task, data));
}

xerrors::Error SugaredDAQmx::GetArmStartTrigTrigWhen(
    TaskHandle task, CVIAbsoluteTime *data) {
    return process_error(dmx->GetArmStartTrigTrigWhen(task, data));
}

xerrors::Error SugaredDAQmx::GetAutoConfiguredCDAQSyncConnections(
    char portList[], uInt32 portListSize) {
    return process_error(
        dmx->GetAutoConfiguredCDAQSyncConnections(portList, portListSize));
}

xerrors::Error SugaredDAQmx::GetBufferAttributeUInt32(
    TaskHandle task, int32 attribute, uInt32 *value) {
    return process_error(dmx->GetBufferAttributeUInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetCalInfoAttributeBool(
    const char deviceName[], int32 attribute, bool32 *value) {
    return process_error(dmx->GetCalInfoAttributeBool(deviceName, attribute, value));
}

xerrors::Error SugaredDAQmx::GetCalInfoAttributeDouble(
    const char deviceName[], int32 attribute, float64 *value) {
    return process_error(dmx->GetCalInfoAttributeDouble(deviceName, attribute, value));
}

xerrors::Error SugaredDAQmx::GetCalInfoAttributeString(
    const char deviceName[], int32 attribute, char value[], uInt32 size) {
    return process_error(
        dmx->GetCalInfoAttributeString(deviceName, attribute, value, size));
}

xerrors::Error SugaredDAQmx::GetCalInfoAttributeUInt32(
    const char deviceName[], int32 attribute, uInt32 *value) {
    return process_error(dmx->GetCalInfoAttributeUInt32(deviceName, attribute, value));
}

xerrors::Error SugaredDAQmx::GetChanAttributeBool(TaskHandle task, const char channel[],
                                                  int32 attribute, bool32 *value) {
    return process_error(dmx->GetChanAttributeBool(task, channel, attribute, value));
}

xerrors::Error SugaredDAQmx::GetChanAttributeDouble(TaskHandle task,
                                                    const char channel[],
                                                    int32 attribute, float64 *value) {
    return process_error(dmx->GetChanAttributeDouble(task, channel, attribute, value));
}

xerrors::Error SugaredDAQmx::GetChanAttributeDoubleArray(
    TaskHandle task, const char channel[], int32 attribute, float64 value[],
    uInt32 size) {
    return process_error(
        dmx->GetChanAttributeDoubleArray(task, channel, attribute, value, size));
}

xerrors::Error SugaredDAQmx::GetChanAttributeInt32(TaskHandle task,
                                                   const char channel[],
                                                   int32 attribute, int32 *value) {
    return process_error(dmx->GetChanAttributeInt32(task, channel, attribute, value));
}

xerrors::Error SugaredDAQmx::GetChanAttributeString(TaskHandle task,
                                                    const char channel[],
                                                    int32 attribute, char value[],
                                                    uInt32 size) {
    return process_error(
        dmx->GetChanAttributeString(task, channel, attribute, value, size));
}

xerrors::Error SugaredDAQmx::GetChanAttributeUInt32(TaskHandle task,
                                                    const char channel[],
                                                    int32 attribute, uInt32 *value) {
    return process_error(dmx->GetChanAttributeUInt32(task, channel, attribute, value));
}

xerrors::Error SugaredDAQmx::GetDeviceAttributeBool(const char deviceName[],
                                                    int32 attribute, bool32 *value) {
    return process_error(dmx->GetDeviceAttributeBool(deviceName, attribute, value));
}

xerrors::Error SugaredDAQmx::GetDeviceAttributeDouble(
    const char deviceName[], int32 attribute, float64 *value) {
    return process_error(dmx->GetDeviceAttributeDouble(deviceName, attribute, value));
}

xerrors::Error SugaredDAQmx::GetDeviceAttributeDoubleArray(
    const char deviceName[], int32 attribute, float64 value[], uInt32 size) {
    return process_error(
        dmx->GetDeviceAttributeDoubleArray(deviceName, attribute, value, size));
}

xerrors::Error SugaredDAQmx::GetDeviceAttributeInt32(
    const char deviceName[], int32 attribute, int32 *value) {
    return process_error(dmx->GetDeviceAttributeInt32(deviceName, attribute, value));
}

xerrors::Error SugaredDAQmx::GetDeviceAttributeInt32Array(
    const char deviceName[], int32 attribute, int32 value[], uInt32 size) {
    return process_error(
        dmx->GetDeviceAttributeInt32Array(deviceName, attribute, value, size));
}

xerrors::Error SugaredDAQmx::GetDeviceAttributeString(
    const char deviceName[], int32 attribute, char value[], uInt32 size) {
    return process_error(
        dmx->GetDeviceAttributeString(deviceName, attribute, value, size));
}

xerrors::Error SugaredDAQmx::GetDeviceAttributeUInt32(
    const char deviceName[], int32 attribute, uInt32 *value) {
    return process_error(dmx->GetDeviceAttributeUInt32(deviceName, attribute, value));
}

xerrors::Error SugaredDAQmx::GetDeviceAttributeUInt32Array(
    const char deviceName[], int32 attribute, uInt32 value[], uInt32 size) {
    return process_error(
        dmx->GetDeviceAttributeUInt32Array(deviceName, attribute, value, size));
}

xerrors::Error SugaredDAQmx::GetDigitalLogicFamilyPowerUpState(
    const char deviceName[], int32 *logicFamily) {
    return process_error(
        dmx->GetDigitalLogicFamilyPowerUpState(deviceName, logicFamily));
}

xerrors::Error SugaredDAQmx::GetDisconnectedCDAQSyncPorts(
    char portList[], uInt32 portListSize) {
    return process_error(dmx->GetDisconnectedCDAQSyncPorts(portList, portListSize));
}

xerrors::Error SugaredDAQmx::GetExportedSignalAttributeBool(
    TaskHandle task, int32 attribute, bool32 *value) {
    return process_error(dmx->GetExportedSignalAttributeBool(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetExportedSignalAttributeDouble(
    TaskHandle task, int32 attribute, float64 *value) {
    return process_error(dmx->GetExportedSignalAttributeDouble(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetExportedSignalAttributeInt32(
    TaskHandle task, int32 attribute, int32 *value) {
    return process_error(dmx->GetExportedSignalAttributeInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetExportedSignalAttributeString(
    TaskHandle task, int32 attribute, char value[], uInt32 size) {
    return process_error(
        dmx->GetExportedSignalAttributeString(task, attribute, value, size));
}

xerrors::Error SugaredDAQmx::GetExportedSignalAttributeUInt32(
    TaskHandle task, int32 attribute, uInt32 *value) {
    return process_error(dmx->GetExportedSignalAttributeUInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetExtCalLastDateAndTime(
    const char deviceName[], uInt32 *year, uInt32 *month, uInt32 *day, uInt32 *hour,
    uInt32 *minute) {
    return process_error(
        dmx->GetExtCalLastDateAndTime(deviceName, year, month, day, hour, minute));
}

xerrors::Error
SugaredDAQmx::GetExtendedErrorInfo(char errorString[], uInt32 bufferSize) {
    return process_error(dmx->GetExtendedErrorInfo(errorString, bufferSize));
}

xerrors::Error
SugaredDAQmx::GetFirstSampClkWhen(TaskHandle task, CVIAbsoluteTime *data) {
    return process_error(dmx->GetFirstSampClkWhen(task, data));
}

xerrors::Error SugaredDAQmx::GetFirstSampTimestampVal(
    TaskHandle task, CVIAbsoluteTime *data) {
    return process_error(dmx->GetFirstSampTimestampVal(task, data));
}

xerrors::Error SugaredDAQmx::GetNthTaskChannel(TaskHandle task, uInt32 index,
                                               char buffer[], int32 bufferSize) {
    return process_error(dmx->GetNthTaskChannel(task, index, buffer, bufferSize));
}

xerrors::Error SugaredDAQmx::GetNthTaskDevice(TaskHandle task, uInt32 index,
                                              char buffer[], int32 bufferSize) {
    return process_error(dmx->GetNthTaskDevice(task, index, buffer, bufferSize));
}

xerrors::Error SugaredDAQmx::GetNthTaskReadChannel(TaskHandle task, uInt32 index,
                                                   char buffer[], int32 bufferSize) {
    return process_error(dmx->GetNthTaskReadChannel(task, index, buffer, bufferSize));
}

xerrors::Error SugaredDAQmx::GetPersistedChanAttributeBool(
    const char channel[], int32 attribute, bool32 *value) {
    return process_error(dmx->GetPersistedChanAttributeBool(channel, attribute, value));
}

xerrors::Error SugaredDAQmx::GetPersistedChanAttributeString(
    const char channel[], int32 attribute, char value[], uInt32 size) {
    return process_error(
        dmx->GetPersistedChanAttributeString(channel, attribute, value, size));
}

xerrors::Error SugaredDAQmx::GetPersistedScaleAttributeBool(
    const char scaleName[], int32 attribute, bool32 *value) {
    return process_error(
        dmx->GetPersistedScaleAttributeBool(scaleName, attribute, value));
}

xerrors::Error SugaredDAQmx::GetPersistedScaleAttributeString(
    const char scaleName[], int32 attribute, char value[], uInt32 size) {
    return process_error(
        dmx->GetPersistedScaleAttributeString(scaleName, attribute, value, size));
}

xerrors::Error SugaredDAQmx::GetPersistedTaskAttributeBool(
    const char taskName[], int32 attribute, bool32 *value) {
    return process_error(
        dmx->GetPersistedTaskAttributeBool(taskName, attribute, value));
}

xerrors::Error SugaredDAQmx::GetPersistedTaskAttributeString(
    const char taskName[], int32 attribute, char value[], uInt32 size) {
    return process_error(
        dmx->GetPersistedTaskAttributeString(taskName, attribute, value, size));
}

xerrors::Error SugaredDAQmx::GetPhysicalChanAttributeBool(
    const char physicalChannel[], int32 attribute, bool32 *value) {
    return process_error(
        dmx->GetPhysicalChanAttributeBool(physicalChannel, attribute, value));
}

xerrors::Error SugaredDAQmx::GetPhysicalChanAttributeBytes(
    const char physicalChannel[], int32 attribute, uInt8 value[], uInt32 size) {
    return process_error(
        dmx->GetPhysicalChanAttributeBytes(physicalChannel, attribute, value, size));
}

xerrors::Error SugaredDAQmx::GetPhysicalChanAttributeDouble(
    const char physicalChannel[], int32 attribute, float64 *value) {
    return process_error(
        dmx->GetPhysicalChanAttributeDouble(physicalChannel, attribute, value));
}

xerrors::Error SugaredDAQmx::GetPhysicalChanAttributeDoubleArray(
    const char physicalChannel[], int32 attribute, float64 value[], uInt32 size) {
    return process_error(
        dmx->GetPhysicalChanAttributeDoubleArray(physicalChannel, attribute, value,
                                                 size));
}

xerrors::Error SugaredDAQmx::GetPhysicalChanAttributeInt32(
    const char physicalChannel[], int32 attribute, int32 *value) {
    return process_error(
        dmx->GetPhysicalChanAttributeInt32(physicalChannel, attribute, value));
}

xerrors::Error SugaredDAQmx::GetPhysicalChanAttributeInt32Array(
    const char physicalChannel[], int32 attribute, int32 value[], uInt32 size) {
    return process_error(
        dmx->GetPhysicalChanAttributeInt32Array(physicalChannel, attribute, value,
                                                size));
}

xerrors::Error SugaredDAQmx::GetPhysicalChanAttributeString(
    const char physicalChannel[], int32 attribute, char value[], uInt32 size) {
    return process_error(
        dmx->GetPhysicalChanAttributeString(physicalChannel, attribute, value, size));
}

xerrors::Error SugaredDAQmx::GetPhysicalChanAttributeUInt32(
    const char physicalChannel[], int32 attribute, uInt32 *value) {
    return process_error(
        dmx->GetPhysicalChanAttributeUInt32(physicalChannel, attribute, value));
}

xerrors::Error SugaredDAQmx::GetPhysicalChanAttributeUInt32Array(
    const char physicalChannel[], int32 attribute, uInt32 value[], uInt32 size) {
    return process_error(
        dmx->GetPhysicalChanAttributeUInt32Array(physicalChannel, attribute, value,
                                                 size));
}

xerrors::Error SugaredDAQmx::GetReadAttributeBool(TaskHandle task, int32 attribute,
                                                  bool32 *value) {
    return process_error(dmx->GetReadAttributeBool(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetReadAttributeDouble(TaskHandle task, int32 attribute,
                                                    float64 *value) {
    return process_error(dmx->GetReadAttributeDouble(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetReadAttributeInt32(TaskHandle task, int32 attribute,
                                                   int32 *value) {
    return process_error(dmx->GetReadAttributeInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetReadAttributeString(TaskHandle task, int32 attribute,
                                                    char value[], uInt32 size) {
    return process_error(dmx->GetReadAttributeString(task, attribute, value, size));
}

xerrors::Error SugaredDAQmx::GetReadAttributeUInt32(TaskHandle task, int32 attribute,
                                                    uInt32 *value) {
    return process_error(dmx->GetReadAttributeUInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetReadAttributeUInt64(TaskHandle task, int32 attribute,
                                                    uInt64 *value) {
    return process_error(dmx->GetReadAttributeUInt64(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetRealTimeAttributeBool(
    TaskHandle task, int32 attribute, bool32 *value) {
    return process_error(dmx->GetRealTimeAttributeBool(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetRealTimeAttributeInt32(
    TaskHandle task, int32 attribute, int32 *value) {
    return process_error(dmx->GetRealTimeAttributeInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetRealTimeAttributeUInt32(
    TaskHandle task, int32 attribute, uInt32 *value) {
    return process_error(dmx->GetRealTimeAttributeUInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetRefTrigTimestampVal(TaskHandle task,
                                                    CVIAbsoluteTime *data) {
    return process_error(dmx->GetRefTrigTimestampVal(task, data));
}

xerrors::Error SugaredDAQmx::GetScaleAttributeDoubleArray(
    const char scaleName[], int32 attribute, float64 value[], uInt32 size) {
    return process_error(
        dmx->GetScaleAttributeDoubleArray(scaleName, attribute, value, size));
}

xerrors::Error SugaredDAQmx::GetScaleAttributeInt32(const char scaleName[],
                                                    int32 attribute, int32 *value) {
    return process_error(dmx->GetScaleAttributeInt32(scaleName, attribute, value));
}

xerrors::Error SugaredDAQmx::GetScaleAttributeString(
    const char scaleName[], int32 attribute, char value[], uInt32 size) {
    return process_error(
        dmx->GetScaleAttributeString(scaleName, attribute, value, size));
}

xerrors::Error SugaredDAQmx::GetSelfCalLastDateAndTime(
    const char deviceName[], uInt32 *year, uInt32 *month, uInt32 *day, uInt32 *hour,
    uInt32 *minute) {
    return process_error(
        dmx->GetSelfCalLastDateAndTime(deviceName, year, month, day, hour, minute));
}

xerrors::Error SugaredDAQmx::GetStartTrigTimestampVal(
    TaskHandle task, CVIAbsoluteTime *data) {
    return process_error(dmx->GetStartTrigTimestampVal(task, data));
}

xerrors::Error SugaredDAQmx::GetStartTrigTrigWhen(TaskHandle task,
                                                  CVIAbsoluteTime *data) {
    return process_error(dmx->GetStartTrigTrigWhen(task, data));
}

xerrors::Error SugaredDAQmx::GetSyncPulseTimeWhen(TaskHandle task,
                                                  CVIAbsoluteTime *data) {
    return process_error(dmx->GetSyncPulseTimeWhen(task, data));
}

xerrors::Error SugaredDAQmx::GetSystemInfoAttributeString(
    int32 attribute, char value[], uInt32 size) {
    return process_error(dmx->GetSystemInfoAttributeString(attribute, value, size));
}

xerrors::Error SugaredDAQmx::GetSystemInfoAttributeUInt32(
    int32 attribute, uInt32 *value) {
    return process_error(dmx->GetSystemInfoAttributeUInt32(attribute, value));
}

xerrors::Error SugaredDAQmx::GetTaskAttributeBool(TaskHandle task, int32 attribute,
                                                  bool32 *value) {
    return process_error(dmx->GetTaskAttributeBool(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetTaskAttributeString(TaskHandle task, int32 attribute,
                                                    char value[], uInt32 size) {
    return process_error(dmx->GetTaskAttributeString(task, attribute, value, size));
}

xerrors::Error SugaredDAQmx::GetTaskAttributeUInt32(TaskHandle task, int32 attribute,
                                                    uInt32 *value) {
    return process_error(dmx->GetTaskAttributeUInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetTimingAttributeBool(TaskHandle task, int32 attribute,
                                                    bool32 *value) {
    return process_error(dmx->GetTimingAttributeBool(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetTimingAttributeDouble(
    TaskHandle task, int32 attribute, float64 *value) {
    return process_error(dmx->GetTimingAttributeDouble(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetTimingAttributeExBool(
    TaskHandle task, const char deviceNames[], int32 attribute, bool32 *value) {
    return process_error(
        dmx->GetTimingAttributeExBool(task, deviceNames, attribute, value));
}

xerrors::Error SugaredDAQmx::GetTimingAttributeExDouble(
    TaskHandle task, const char deviceNames[], int32 attribute, float64 *value) {
    return process_error(
        dmx->GetTimingAttributeExDouble(task, deviceNames, attribute, value));
}

xerrors::Error SugaredDAQmx::GetTimingAttributeExInt32(
    TaskHandle task, const char deviceNames[], int32 attribute, int32 *value) {
    return process_error(
        dmx->GetTimingAttributeExInt32(task, deviceNames, attribute, value));
}

xerrors::Error SugaredDAQmx::GetTimingAttributeExString(
    TaskHandle task, const char deviceNames[], int32 attribute, char value[],
    uInt32 size) {
    return process_error(
        dmx->GetTimingAttributeExString(task, deviceNames, attribute, value, size));
}

xerrors::Error SugaredDAQmx::GetTimingAttributeExTimestamp(
    TaskHandle task, const char deviceNames[], int32 attribute,
    CVIAbsoluteTime *value) {
    return process_error(
        dmx->GetTimingAttributeExTimestamp(task, deviceNames, attribute, value));
}

xerrors::Error SugaredDAQmx::GetTimingAttributeExUInt32(
    TaskHandle task, const char deviceNames[], int32 attribute, uInt32 *value) {
    return process_error(
        dmx->GetTimingAttributeExUInt32(task, deviceNames, attribute, value));
}

xerrors::Error SugaredDAQmx::GetTimingAttributeExUInt64(
    TaskHandle task, const char deviceNames[], int32 attribute, uInt64 *value) {
    return process_error(
        dmx->GetTimingAttributeExUInt64(task, deviceNames, attribute, value));
}

xerrors::Error SugaredDAQmx::GetTimingAttributeInt32(
    TaskHandle task, int32 attribute, int32 *value) {
    return process_error(dmx->GetTimingAttributeInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetTimingAttributeString(
    TaskHandle task, int32 attribute, char value[], uInt32 size) {
    return process_error(dmx->GetTimingAttributeString(task, attribute, value, size));
}

xerrors::Error SugaredDAQmx::GetTimingAttributeTimestamp(
    TaskHandle task, int32 attribute, CVIAbsoluteTime *value) {
    return process_error(dmx->GetTimingAttributeTimestamp(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetTimingAttributeUInt32(
    TaskHandle task, int32 attribute, uInt32 *value) {
    return process_error(dmx->GetTimingAttributeUInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetTimingAttributeUInt64(
    TaskHandle task, int32 attribute, uInt64 *value) {
    return process_error(dmx->GetTimingAttributeUInt64(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetTrigAttributeBool(TaskHandle task, int32 attribute,
                                                  bool32 *value) {
    return process_error(dmx->GetTrigAttributeBool(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetTrigAttributeDouble(TaskHandle task, int32 attribute,
                                                    float64 *value) {
    return process_error(dmx->GetTrigAttributeDouble(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetTrigAttributeDoubleArray(
    TaskHandle task, int32 attribute, float64 value[], uInt32 size) {
    return process_error(
        dmx->GetTrigAttributeDoubleArray(task, attribute, value, size));
}

xerrors::Error SugaredDAQmx::GetTrigAttributeInt32(TaskHandle task, int32 attribute,
                                                   int32 *value) {
    return process_error(dmx->GetTrigAttributeInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetTrigAttributeInt32Array(
    TaskHandle task, int32 attribute, int32 value[], uInt32 size) {
    return process_error(dmx->GetTrigAttributeInt32Array(task, attribute, value, size));
}

xerrors::Error SugaredDAQmx::GetTrigAttributeString(TaskHandle task, int32 attribute,
                                                    char value[], uInt32 size) {
    return process_error(dmx->GetTrigAttributeString(task, attribute, value, size));
}

xerrors::Error SugaredDAQmx::GetTrigAttributeTimestamp(
    TaskHandle task, int32 attribute, CVIAbsoluteTime *value) {
    return process_error(dmx->GetTrigAttributeTimestamp(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetTrigAttributeUInt32(TaskHandle task, int32 attribute,
                                                    uInt32 *value) {
    return process_error(dmx->GetTrigAttributeUInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetWatchdogAttributeBool(
    TaskHandle task, const char lines[], int32 attribute, bool32 *value) {
    return process_error(dmx->GetWatchdogAttributeBool(task, lines, attribute, value));
}

xerrors::Error SugaredDAQmx::GetWatchdogAttributeDouble(
    TaskHandle task, const char lines[], int32 attribute, float64 *value) {
    return process_error(
        dmx->GetWatchdogAttributeDouble(task, lines, attribute, value));
}

xerrors::Error SugaredDAQmx::GetWatchdogAttributeInt32(
    TaskHandle task, const char lines[], int32 attribute, int32 *value) {
    return process_error(dmx->GetWatchdogAttributeInt32(task, lines, attribute, value));
}

xerrors::Error SugaredDAQmx::GetWatchdogAttributeString(
    TaskHandle task, const char lines[], int32 attribute, char value[], uInt32 size) {
    return process_error(
        dmx->GetWatchdogAttributeString(task, lines, attribute, value, size));
}

xerrors::Error SugaredDAQmx::GetWriteAttributeBool(TaskHandle task, int32 attribute,
                                                   bool32 *value) {
    return process_error(dmx->GetWriteAttributeBool(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetWriteAttributeDouble(
    TaskHandle task, int32 attribute, float64 *value) {
    return process_error(dmx->GetWriteAttributeDouble(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetWriteAttributeInt32(TaskHandle task, int32 attribute,
                                                    int32 *value) {
    return process_error(dmx->GetWriteAttributeInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetWriteAttributeString(
    TaskHandle task, int32 attribute, char value[], uInt32 size) {
    return process_error(dmx->GetWriteAttributeString(task, attribute, value, size));
}

xerrors::Error SugaredDAQmx::GetWriteAttributeUInt32(
    TaskHandle task, int32 attribute, uInt32 *value) {
    return process_error(dmx->GetWriteAttributeUInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::GetWriteAttributeUInt64(
    TaskHandle task, int32 attribute, uInt64 *value) {
    return process_error(dmx->GetWriteAttributeUInt64(task, attribute, value));
}

xerrors::Error SugaredDAQmx::IsTaskDone(TaskHandle task, bool32 *isTaskDone) {
    return process_error(dmx->IsTaskDone(task, isTaskDone));
}

xerrors::Error SugaredDAQmx::LoadTask(const char sessionName[], TaskHandle *task) {
    return process_error(dmx->LoadTask(sessionName, task));
}

xerrors::Error SugaredDAQmx::PerformBridgeOffsetNullingCalEx(
    TaskHandle task, const char channel[], bool32 skipUnsupportedChannels) {
    return process_error(
        dmx->PerformBridgeOffsetNullingCalEx(task, channel, skipUnsupportedChannels));
}

xerrors::Error SugaredDAQmx::PerformBridgeShuntCalEx(
    TaskHandle task, const char channel[], float64 shuntResistorValue,
    int32 shuntResistorLocation, int32 shuntResistorSelect, int32 shuntResistorSource,
    float64 bridgeResistance, bool32 skipUnsupportedChannels) {
    return process_error(dmx->PerformBridgeShuntCalEx(
        task, channel, shuntResistorValue, shuntResistorLocation, shuntResistorSelect,
        shuntResistorSource, bridgeResistance, skipUnsupportedChannels));
}

xerrors::Error SugaredDAQmx::PerformStrainShuntCalEx(
    TaskHandle task, const char channel[], float64 shuntResistorValue,
    int32 shuntResistorLocation, int32 shuntResistorSelect, int32 shuntResistorSource,
    bool32 skipUnsupportedChannels) {
    return process_error(dmx->PerformStrainShuntCalEx(
        task, channel, shuntResistorValue, shuntResistorLocation, shuntResistorSelect,
        shuntResistorSource, skipUnsupportedChannels));
}

xerrors::Error SugaredDAQmx::PerformThrmcplLeadOffsetNullingCal(
    TaskHandle task, const char channel[], bool32 skipUnsupportedChannels) {
    return process_error(
        dmx->PerformThrmcplLeadOffsetNullingCal(task, channel,
                                                skipUnsupportedChannels));
}

xerrors::Error SugaredDAQmx::ReadAnalogF64(TaskHandle task, int32 numSampsPerChan,
                                           float64 timeout, int32 fillMode,
                                           float64 readArray[], uInt32 arraySizeInSamps,
                                           int32 *sampsPerChanRead, bool32 *reserved) {
    return process_error(dmx->ReadAnalogF64(task, numSampsPerChan, timeout, fillMode,
                                            readArray, arraySizeInSamps,
                                            sampsPerChanRead, reserved));
}

xerrors::Error SugaredDAQmx::ReadAnalogScalarF64(TaskHandle task, float64 timeout,
                                                 float64 *value, bool32 *reserved) {
    return process_error(dmx->ReadAnalogScalarF64(task, timeout, value, reserved));
}

xerrors::Error SugaredDAQmx::ReadBinaryI16(TaskHandle task, int32 numSampsPerChan,
                                           float64 timeout, int32 fillMode,
                                           int16 readArray[], uInt32 arraySizeInSamps,
                                           int32 *sampsPerChanRead, bool32 *reserved) {
    return process_error(dmx->ReadBinaryI16(task, numSampsPerChan, timeout, fillMode,
                                            readArray, arraySizeInSamps,
                                            sampsPerChanRead, reserved));
}

xerrors::Error SugaredDAQmx::ReadBinaryI32(TaskHandle task, int32 numSampsPerChan,
                                           float64 timeout, int32 fillMode,
                                           int32 readArray[], uInt32 arraySizeInSamps,
                                           int32 *sampsPerChanRead, bool32 *reserved) {
    return process_error(dmx->ReadBinaryI32(task, numSampsPerChan, timeout, fillMode,
                                            readArray, arraySizeInSamps,
                                            sampsPerChanRead, reserved));
}

xerrors::Error SugaredDAQmx::ReadBinaryU16(TaskHandle task, int32 numSampsPerChan,
                                           float64 timeout, int32 fillMode,
                                           uInt16 readArray[], uInt32 arraySizeInSamps,
                                           int32 *sampsPerChanRead, bool32 *reserved) {
    return process_error(dmx->ReadBinaryU16(task, numSampsPerChan, timeout, fillMode,
                                            readArray, arraySizeInSamps,
                                            sampsPerChanRead, reserved));
}

xerrors::Error SugaredDAQmx::ReadBinaryU32(TaskHandle task, int32 numSampsPerChan,
                                           float64 timeout, int32 fillMode,
                                           uInt32 readArray[], uInt32 arraySizeInSamps,
                                           int32 *sampsPerChanRead, bool32 *reserved) {
    return process_error(dmx->ReadBinaryU32(task, numSampsPerChan, timeout, fillMode,
                                            readArray, arraySizeInSamps,
                                            sampsPerChanRead, reserved));
}

xerrors::Error SugaredDAQmx::ReadCounterF64(TaskHandle task, int32 numSampsPerChan,
                                            float64 timeout, float64 readArray[],
                                            uInt32 arraySizeInSamps,
                                            int32 *sampsPerChanRead, bool32 *reserved) {
    return process_error(dmx->ReadCounterF64(task, numSampsPerChan, timeout, readArray,
                                             arraySizeInSamps, sampsPerChanRead,
                                             reserved));
}

xerrors::Error SugaredDAQmx::ReadCounterF64Ex(TaskHandle task, int32 numSampsPerChan,
                                              float64 timeout, int32 fillMode,
                                              float64 readArray[],
                                              uInt32 arraySizeInSamps,
                                              int32 *sampsPerChanRead,
                                              bool32 *reserved) {
    return process_error(dmx->ReadCounterF64Ex(task, numSampsPerChan, timeout, fillMode,
                                               readArray, arraySizeInSamps,
                                               sampsPerChanRead, reserved));
}

xerrors::Error SugaredDAQmx::ReadCounterScalarF64(TaskHandle task, float64 timeout,
                                                  float64 *value, bool32 *reserved) {
    return process_error(dmx->ReadCounterScalarF64(task, timeout, value, reserved));
}

xerrors::Error SugaredDAQmx::ReadCounterScalarU32(TaskHandle task, float64 timeout,
                                                  uInt32 *value, bool32 *reserved) {
    return process_error(dmx->ReadCounterScalarU32(task, timeout, value, reserved));
}

xerrors::Error SugaredDAQmx::ReadCounterU32(TaskHandle task, int32 numSampsPerChan,
                                            float64 timeout, uInt32 readArray[],
                                            uInt32 arraySizeInSamps,
                                            int32 *sampsPerChanRead, bool32 *reserved) {
    return process_error(dmx->ReadCounterU32(task, numSampsPerChan, timeout, readArray,
                                             arraySizeInSamps, sampsPerChanRead,
                                             reserved));
}

xerrors::Error SugaredDAQmx::ReadCounterU32Ex(TaskHandle task, int32 numSampsPerChan,
                                              float64 timeout, int32 fillMode,
                                              uInt32 readArray[],
                                              uInt32 arraySizeInSamps,
                                              int32 *sampsPerChanRead,
                                              bool32 *reserved) {
    return process_error(dmx->ReadCounterU32Ex(task, numSampsPerChan, timeout, fillMode,
                                               readArray, arraySizeInSamps,
                                               sampsPerChanRead, reserved));
}

xerrors::Error SugaredDAQmx::ReadCtrFreq(TaskHandle task, int32 numSampsPerChan,
                                         float64 timeout, int32 interleaved,
                                         float64 readArrayFrequency[],
                                         float64 readArrayDutyCycle[],
                                         uInt32 arraySizeInSamps,
                                         int32 *sampsPerChanRead, bool32 *reserved) {
    return process_error(dmx->ReadCtrFreq(task, numSampsPerChan, timeout, interleaved,
                                          readArrayFrequency, readArrayDutyCycle,
                                          arraySizeInSamps, sampsPerChanRead,
                                          reserved));
}

xerrors::Error SugaredDAQmx::ReadCtrFreqScalar(TaskHandle task, float64 timeout,
                                               float64 *frequency, float64 *dutyCycle,
                                               bool32 *reserved) {
    return process_error(
        dmx->ReadCtrFreqScalar(task, timeout, frequency, dutyCycle, reserved));
}

xerrors::Error SugaredDAQmx::ReadCtrTicks(TaskHandle task, int32 numSampsPerChan,
                                          float64 timeout, int32 interleaved,
                                          uInt32 readArrayHighTicks[],
                                          uInt32 readArrayLowTicks[],
                                          uInt32 arraySizeInSamps,
                                          int32 *sampsPerChanRead, bool32 *reserved) {
    return process_error(dmx->ReadCtrTicks(task, numSampsPerChan, timeout, interleaved,
                                           readArrayHighTicks, readArrayLowTicks,
                                           arraySizeInSamps, sampsPerChanRead,
                                           reserved));
}

xerrors::Error SugaredDAQmx::ReadCtrTicksScalar(TaskHandle task, float64 timeout,
                                                uInt32 *highTicks, uInt32 *lowTicks,
                                                bool32 *reserved) {
    return process_error(
        dmx->ReadCtrTicksScalar(task, timeout, highTicks, lowTicks, reserved));
}

xerrors::Error SugaredDAQmx::ReadCtrTime(TaskHandle task, int32 numSampsPerChan,
                                         float64 timeout, int32 interleaved,
                                         float64 readArrayHighTime[],
                                         float64 readArrayLowTime[],
                                         uInt32 arraySizeInSamps,
                                         int32 *sampsPerChanRead, bool32 *reserved) {
    return process_error(dmx->ReadCtrTime(task, numSampsPerChan, timeout, interleaved,
                                          readArrayHighTime, readArrayLowTime,
                                          arraySizeInSamps, sampsPerChanRead,
                                          reserved));
}

xerrors::Error SugaredDAQmx::ReadCtrTimeScalar(TaskHandle task, float64 timeout,
                                               float64 *highTime, float64 *lowTime,
                                               bool32 *reserved) {
    return process_error(
        dmx->ReadCtrTimeScalar(task, timeout, highTime, lowTime, reserved));
}

xerrors::Error SugaredDAQmx::ReadDigitalLines(TaskHandle task, int32 numSampsPerChan,
                                              float64 timeout, int32 fillMode,
                                              uInt8 readArray[],
                                              uInt32 arraySizeInBytes,
                                              int32 *sampsPerChanRead,
                                              int32 *numBytesPerSamp,
                                              bool32 *reserved) {
    return process_error(dmx->ReadDigitalLines(task, numSampsPerChan, timeout, fillMode,
                                               readArray, arraySizeInBytes,
                                               sampsPerChanRead, numBytesPerSamp,
                                               reserved));
}

xerrors::Error SugaredDAQmx::ReadDigitalScalarU32(TaskHandle task, float64 timeout,
                                                  uInt32 *value, bool32 *reserved) {
    return process_error(dmx->ReadDigitalScalarU32(task, timeout, value, reserved));
}

xerrors::Error SugaredDAQmx::ReadDigitalU16(TaskHandle task, int32 numSampsPerChan,
                                            float64 timeout, int32 fillMode,
                                            uInt16 readArray[], uInt32 arraySizeInSamps,
                                            int32 *sampsPerChanRead, bool32 *reserved) {
    return process_error(dmx->ReadDigitalU16(task, numSampsPerChan, timeout, fillMode,
                                             readArray, arraySizeInSamps,
                                             sampsPerChanRead, reserved));
}

xerrors::Error SugaredDAQmx::ReadDigitalU32(TaskHandle task, int32 numSampsPerChan,
                                            float64 timeout, int32 fillMode,
                                            uInt32 readArray[], uInt32 arraySizeInSamps,
                                            int32 *sampsPerChanRead, bool32 *reserved) {
    return process_error(dmx->ReadDigitalU32(task, numSampsPerChan, timeout, fillMode,
                                             readArray, arraySizeInSamps,
                                             sampsPerChanRead, reserved));
}

xerrors::Error SugaredDAQmx::ReadDigitalU8(TaskHandle task, int32 numSampsPerChan,
                                           float64 timeout, int32 fillMode,
                                           uInt8 readArray[], uInt32 arraySizeInSamps,
                                           int32 *sampsPerChanRead, bool32 *reserved) {
    return process_error(dmx->ReadDigitalU8(task, numSampsPerChan, timeout, fillMode,
                                            readArray, arraySizeInSamps,
                                            sampsPerChanRead, reserved));
}

xerrors::Error SugaredDAQmx::ReadPowerBinaryI16(TaskHandle task, int32 numSampsPerChan,
                                                float64 timeout, int32 fillMode,
                                                int16 readArrayVoltage[],
                                                int16 readArrayCurrent[],
                                                uInt32 arraySizeInSamps,
                                                int32 *sampsPerChanRead,
                                                bool32 *reserved) {
    return process_error(dmx->ReadPowerBinaryI16(task, numSampsPerChan, timeout,
                                                 fillMode, readArrayVoltage,
                                                 readArrayCurrent, arraySizeInSamps,
                                                 sampsPerChanRead, reserved));
}

xerrors::Error SugaredDAQmx::ReadPowerF64(TaskHandle task, int32 numSampsPerChan,
                                          float64 timeout, int32 fillMode,
                                          float64 readArrayVoltage[],
                                          float64 readArrayCurrent[],
                                          uInt32 arraySizeInSamps,
                                          int32 *sampsPerChanRead, bool32 *reserved) {
    return process_error(dmx->ReadPowerF64(task, numSampsPerChan, timeout, fillMode,
                                           readArrayVoltage, readArrayCurrent,
                                           arraySizeInSamps, sampsPerChanRead,
                                           reserved));
}

xerrors::Error SugaredDAQmx::ReadPowerScalarF64(TaskHandle task, float64 timeout,
                                                float64 *voltage, float64 *current,
                                                bool32 *reserved) {
    return process_error(
        dmx->ReadPowerScalarF64(task, timeout, voltage, current, reserved));
}

xerrors::Error SugaredDAQmx::ReadRaw(TaskHandle task, int32 numSampsPerChan,
                                     float64 timeout, uInt8 readArray[],
                                     uInt32 arraySizeInBytes, int32 *sampsRead,
                                     int32 *numBytesPerSamp, bool32 *reserved) {
    return process_error(dmx->ReadRaw(task, numSampsPerChan, timeout, readArray,
                                      arraySizeInBytes, sampsRead, numBytesPerSamp,
                                      reserved));
}

xerrors::Error SugaredDAQmx::RegisterDoneEvent(TaskHandle task, uInt32 options,
                                               DAQmxDoneEventCallbackPtr
                                               callbackFunction, void *callbackData) {
    return process_error(
        dmx->RegisterDoneEvent(task, options, callbackFunction, callbackData));
}

xerrors::Error SugaredDAQmx::RegisterEveryNSamplesEvent(
    TaskHandle task, int32 everyNSamplesEventType, uInt32 nSamples, uInt32 options,
    DAQmxEveryNSamplesEventCallbackPtr callbackFunction, void *callbackData) {
    return process_error(dmx->RegisterEveryNSamplesEvent(
        task, everyNSamplesEventType, nSamples, options, callbackFunction,
        callbackData));
}

xerrors::Error SugaredDAQmx::RegisterSignalEvent(TaskHandle task, int32 signalID,
                                                 uInt32 options,
                                                 DAQmxSignalEventCallbackPtr
                                                 callbackFunction, void *callbackData) {
    return process_error(
        dmx->RegisterSignalEvent(task, signalID, options, callbackFunction,
                                 callbackData));
}

xerrors::Error SugaredDAQmx::RemoveCDAQSyncConnection(const char portList[]) {
    return process_error(dmx->RemoveCDAQSyncConnection(portList));
}

xerrors::Error SugaredDAQmx::ReserveNetworkDevice(const char deviceName[],
                                                  bool32 overrideReservation) {
    return process_error(dmx->ReserveNetworkDevice(deviceName, overrideReservation));
}

xerrors::Error SugaredDAQmx::ResetBufferAttribute(TaskHandle task, int32 attribute) {
    return process_error(dmx->ResetBufferAttribute(task, attribute));
}

xerrors::Error SugaredDAQmx::ResetChanAttribute(TaskHandle task, const char channel[],
                                                int32 attribute) {
    return process_error(dmx->ResetChanAttribute(task, channel, attribute));
}

xerrors::Error SugaredDAQmx::ResetDevice(const char deviceName[]) {
    return process_error(dmx->ResetDevice(deviceName));
}

xerrors::Error SugaredDAQmx::ResetRealTimeAttribute(TaskHandle task, int32 attribute) {
    return process_error(dmx->ResetRealTimeAttribute(task, attribute));
}

xerrors::Error SugaredDAQmx::ResetTimingAttribute(TaskHandle task, int32 attribute) {
    return process_error(dmx->ResetTimingAttribute(task, attribute));
}

xerrors::Error SugaredDAQmx::ResetTimingAttributeEx(TaskHandle task,
                                                    const char deviceNames[],
                                                    int32 attribute) {
    return process_error(dmx->ResetTimingAttributeEx(task, deviceNames, attribute));
}

xerrors::Error SugaredDAQmx::ResetTrigAttribute(TaskHandle task, int32 attribute) {
    return process_error(dmx->ResetTrigAttribute(task, attribute));
}

xerrors::Error SugaredDAQmx::ResetWriteAttribute(TaskHandle task, int32 attribute) {
    return process_error(dmx->ResetWriteAttribute(task, attribute));
}

xerrors::Error SugaredDAQmx::RestoreLastExtCalConst(const char deviceName[]) {
    return process_error(dmx->RestoreLastExtCalConst(deviceName));
}

xerrors::Error SugaredDAQmx::SaveGlobalChan(TaskHandle task, const char channelName[],
                                            const char saveAs[], const char author[],
                                            uInt32 options) {
    return process_error(
        dmx->SaveGlobalChan(task, channelName, saveAs, author, options));
}

xerrors::Error SugaredDAQmx::SaveTask(TaskHandle task, const char saveAs[],
                                      const char author[], uInt32 options) {
    return process_error(dmx->SaveTask(task, saveAs, author, options));
}

xerrors::Error SugaredDAQmx::SelfCal(const char deviceName[]) {
    return process_error(dmx->SelfCal(deviceName));
}

xerrors::Error SugaredDAQmx::SelfTestDevice(const char deviceName[]) {
    return process_error(dmx->SelfTestDevice(deviceName));
}

xerrors::Error SugaredDAQmx::SetAIChanCalCalDate(TaskHandle task,
                                                 const char channelName[], uInt32 year,
                                                 uInt32 month, uInt32 day, uInt32 hour,
                                                 uInt32 minute) {
    return process_error(
        dmx->SetAIChanCalCalDate(task, channelName, year, month, day, hour, minute));
}

xerrors::Error SugaredDAQmx::SetAIChanCalExpDate(TaskHandle task,
                                                 const char channelName[], uInt32 year,
                                                 uInt32 month, uInt32 day, uInt32 hour,
                                                 uInt32 minute) {
    return process_error(
        dmx->SetAIChanCalExpDate(task, channelName, year, month, day, hour, minute));
}

xerrors::Error SugaredDAQmx::SetAnalogPowerUpStatesWithOutputType(
    const char channelNames[], const float64 stateArray[],
    const int32 channelTypeArray[], uInt32 arraySize) {
    return process_error(
        dmx->SetAnalogPowerUpStatesWithOutputType(channelNames, stateArray,
                                                  channelTypeArray, arraySize));
}

xerrors::Error SugaredDAQmx::SetArmStartTrigTrigWhen(
    TaskHandle task, CVIAbsoluteTime data) {
    return process_error(dmx->SetArmStartTrigTrigWhen(task, data));
}

xerrors::Error SugaredDAQmx::SetBufferAttributeUInt32(
    TaskHandle task, int32 attribute, uInt32 value) {
    return process_error(dmx->SetBufferAttributeUInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetCalInfoAttributeBool(
    const char deviceName[], int32 attribute, bool32 value) {
    return process_error(dmx->SetCalInfoAttributeBool(deviceName, attribute, value));
}

xerrors::Error SugaredDAQmx::SetCalInfoAttributeDouble(
    const char deviceName[], int32 attribute, float64 value) {
    return process_error(dmx->SetCalInfoAttributeDouble(deviceName, attribute, value));
}

xerrors::Error SugaredDAQmx::SetCalInfoAttributeString(
    const char deviceName[], int32 attribute, const char value[]) {
    return process_error(dmx->SetCalInfoAttributeString(deviceName, attribute, value));
}

xerrors::Error SugaredDAQmx::SetCalInfoAttributeUInt32(
    const char deviceName[], int32 attribute, uInt32 value) {
    return process_error(dmx->SetCalInfoAttributeUInt32(deviceName, attribute, value));
}

xerrors::Error SugaredDAQmx::SetChanAttributeBool(TaskHandle task, const char channel[],
                                                  int32 attribute, bool32 value) {
    return process_error(dmx->SetChanAttributeBool(task, channel, attribute, value));
}

xerrors::Error SugaredDAQmx::SetChanAttributeDouble(TaskHandle task,
                                                    const char channel[],
                                                    int32 attribute, float64 value) {
    return process_error(dmx->SetChanAttributeDouble(task, channel, attribute, value));
}

xerrors::Error SugaredDAQmx::SetChanAttributeDoubleArray(
    TaskHandle task, const char channel[], int32 attribute, const float64 value[],
    uInt32 size) {
    return process_error(
        dmx->SetChanAttributeDoubleArray(task, channel, attribute, value, size));
}

xerrors::Error SugaredDAQmx::SetChanAttributeInt32(TaskHandle task,
                                                   const char channel[],
                                                   int32 attribute, int32 value) {
    return process_error(dmx->SetChanAttributeInt32(task, channel, attribute, value));
}

xerrors::Error SugaredDAQmx::SetChanAttributeString(TaskHandle task,
                                                    const char channel[],
                                                    int32 attribute,
                                                    const char value[]) {
    return process_error(dmx->SetChanAttributeString(task, channel, attribute, value));
}

xerrors::Error SugaredDAQmx::SetChanAttributeUInt32(TaskHandle task,
                                                    const char channel[],
                                                    int32 attribute, uInt32 value) {
    return process_error(dmx->SetChanAttributeUInt32(task, channel, attribute, value));
}

xerrors::Error SugaredDAQmx::SetDigitalLogicFamilyPowerUpState(
    const char deviceName[], int32 logicFamily) {
    return process_error(
        dmx->SetDigitalLogicFamilyPowerUpState(deviceName, logicFamily));
}

xerrors::Error SugaredDAQmx::SetExportedSignalAttributeBool(
    TaskHandle task, int32 attribute, bool32 value) {
    return process_error(dmx->SetExportedSignalAttributeBool(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetExportedSignalAttributeDouble(
    TaskHandle task, int32 attribute, float64 value) {
    return process_error(dmx->SetExportedSignalAttributeDouble(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetExportedSignalAttributeInt32(
    TaskHandle task, int32 attribute, int32 value) {
    return process_error(dmx->SetExportedSignalAttributeInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetExportedSignalAttributeString(
    TaskHandle task, int32 attribute, const char value[]) {
    return process_error(dmx->SetExportedSignalAttributeString(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetExportedSignalAttributeUInt32(
    TaskHandle task, int32 attribute, uInt32 value) {
    return process_error(dmx->SetExportedSignalAttributeUInt32(task, attribute, value));
}

xerrors::Error
SugaredDAQmx::SetFirstSampClkWhen(TaskHandle task, CVIAbsoluteTime data) {
    return process_error(dmx->SetFirstSampClkWhen(task, data));
}

xerrors::Error SugaredDAQmx::SetReadAttributeBool(TaskHandle task, int32 attribute,
                                                  bool32 value) {
    return process_error(dmx->SetReadAttributeBool(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetReadAttributeDouble(TaskHandle task, int32 attribute,
                                                    float64 value) {
    return process_error(dmx->SetReadAttributeDouble(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetReadAttributeInt32(TaskHandle task, int32 attribute,
                                                   int32 value) {
    return process_error(dmx->SetReadAttributeInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetReadAttributeString(TaskHandle task, int32 attribute,
                                                    const char value[]) {
    return process_error(dmx->SetReadAttributeString(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetReadAttributeUInt32(TaskHandle task, int32 attribute,
                                                    uInt32 value) {
    return process_error(dmx->SetReadAttributeUInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetReadAttributeUInt64(TaskHandle task, int32 attribute,
                                                    uInt64 value) {
    return process_error(dmx->SetReadAttributeUInt64(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetRealTimeAttributeBool(
    TaskHandle task, int32 attribute, bool32 value) {
    return process_error(dmx->SetRealTimeAttributeBool(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetRealTimeAttributeInt32(
    TaskHandle task, int32 attribute, int32 value) {
    return process_error(dmx->SetRealTimeAttributeInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetRealTimeAttributeUInt32(
    TaskHandle task, int32 attribute, uInt32 value) {
    return process_error(dmx->SetRealTimeAttributeUInt32(task, attribute, value));
}

xerrors::Error SugaredDAQmx::SetRuntimeEnvironment(const char environment[],
                                                   const char environmentVersion[],
                                                   const char reserved1[],
                                                   const char reserved2[]) {
    return process_error(
        dmx->SetRuntimeEnvironment(environment, environmentVersion, reserved1,
                                   reserved2));
}


xerrors::Error SugaredDAQmx::CreateCIAngEncoderChan(TaskHandle task,
                                                    const char counter[],
                                                    const char nameToAssignToChannel[],
                                                    int32 decodingType,
                                                    bool32 zidxEnable, float64 zidxVal,
                                                    int32 zidxPhase, int32 units,
                                                    uInt32 pulsesPerRev,
                                                    float64 initialAngle,
                                                    const char customScaleName[]) {
    return process_error(dmx->CreateCIAngEncoderChan(
        task, counter, nameToAssignToChannel, decodingType, zidxEnable, zidxVal,
        zidxPhase, units, pulsesPerRev, initialAngle, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateCIAngVelocityChan(
    TaskHandle task, const char counter[], const char nameToAssignToChannel[],
    float64 minVal, float64 maxVal, int32 decodingType, int32 units,
    uInt32 pulsesPerRev, const char customScaleName[]) {
    return process_error(dmx->CreateCIAngVelocityChan(
        task, counter, nameToAssignToChannel, minVal, maxVal, decodingType, units,
        pulsesPerRev, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateCICountEdgesChan(TaskHandle task,
                                                    const char counter[],
                                                    const char nameToAssignToChannel[],
                                                    int32 edge, uInt32 initialCount,
                                                    int32 countDirection) {
    return process_error(dmx->CreateCICountEdgesChan(
        task, counter, nameToAssignToChannel, edge, initialCount, countDirection));
}

xerrors::Error SugaredDAQmx::CreateCIDutyCycleChan(TaskHandle task,
                                                   const char counter[],
                                                   const char nameToAssignToChannel[],
                                                   float64 minFreq, float64 maxFreq,
                                                   int32 edge,
                                                   const char customScaleName[]) {
    return process_error(dmx->CreateCIDutyCycleChan(task, counter,
                                                    nameToAssignToChannel, minFreq,
                                                    maxFreq, edge, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateCIFreqChan(TaskHandle task, const char counter[],
                                              const char nameToAssignToChannel[],
                                              float64 minVal, float64 maxVal,
                                              int32 units, int32 edge, int32 measMethod,
                                              float64 measTime, uInt32 divisor,
                                              const char customScaleName[]) {
    return process_error(dmx->CreateCIFreqChan(task, counter, nameToAssignToChannel,
                                               minVal, maxVal, units, edge, measMethod,
                                               measTime, divisor, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateCIGPSTimestampChan(
    TaskHandle task, const char counter[], const char nameToAssignToChannel[],
    int32 units, int32 syncMethod, const char customScaleName[]) {
    return process_error(dmx->CreateCIGPSTimestampChan(
        task, counter, nameToAssignToChannel, units, syncMethod, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateCILinEncoderChan(TaskHandle task,
                                                    const char counter[],
                                                    const char nameToAssignToChannel[],
                                                    int32 decodingType,
                                                    bool32 zidxEnable, float64 zidxVal,
                                                    int32 zidxPhase, int32 units,
                                                    float64 distPerPulse,
                                                    float64 initialPos,
                                                    const char customScaleName[]) {
    return process_error(dmx->CreateCILinEncoderChan(
        task, counter, nameToAssignToChannel, decodingType, zidxEnable, zidxVal,
        zidxPhase, units, distPerPulse, initialPos, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateCILinVelocityChan(
    TaskHandle task, const char counter[], const char nameToAssignToChannel[],
    float64 minVal, float64 maxVal, int32 decodingType, int32 units,
    float64 distPerPulse, const char customScaleName[]) {
    return process_error(dmx->CreateCILinVelocityChan(
        task, counter, nameToAssignToChannel, minVal, maxVal, decodingType, units,
        distPerPulse, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateCIPeriodChan(TaskHandle task, const char counter[],
                                                const char nameToAssignToChannel[],
                                                float64 minVal, float64 maxVal,
                                                int32 units, int32 edge,
                                                int32 measMethod, float64 measTime,
                                                uInt32 divisor,
                                                const char customScaleName[]) {
    return process_error(dmx->CreateCIPeriodChan(task, counter, nameToAssignToChannel,
                                                 minVal, maxVal, units, edge,
                                                 measMethod, measTime, divisor,
                                                 customScaleName));
}

xerrors::Error SugaredDAQmx::CreateCIPulseChanFreq(TaskHandle task,
                                                   const char counter[],
                                                   const char nameToAssignToChannel[],
                                                   float64 minVal, float64 maxVal,
                                                   int32 units) {
    return process_error(
        dmx->CreateCIPulseChanFreq(task, counter, nameToAssignToChannel, minVal, maxVal,
                                   units));
}

xerrors::Error SugaredDAQmx::CreateCIPulseChanTicks(TaskHandle task,
                                                    const char counter[],
                                                    const char nameToAssignToChannel[],
                                                    const char sourceTerminal[],
                                                    float64 minVal, float64 maxVal) {
    return process_error(dmx->CreateCIPulseChanTicks(
        task, counter, nameToAssignToChannel, sourceTerminal, minVal, maxVal));
}

xerrors::Error SugaredDAQmx::CreateCIPulseChanTime(TaskHandle task,
                                                   const char counter[],
                                                   const char nameToAssignToChannel[],
                                                   float64 minVal, float64 maxVal,
                                                   int32 units) {
    return process_error(
        dmx->CreateCIPulseChanTime(task, counter, nameToAssignToChannel, minVal, maxVal,
                                   units));
}

xerrors::Error SugaredDAQmx::CreateCIPulseWidthChan(TaskHandle task,
                                                    const char counter[],
                                                    const char nameToAssignToChannel[],
                                                    float64 minVal, float64 maxVal,
                                                    int32 units, int32 startingEdge,
                                                    const char customScaleName[]) {
    return process_error(dmx->CreateCIPulseWidthChan(
        task, counter, nameToAssignToChannel, minVal, maxVal, units, startingEdge,
        customScaleName));
}

xerrors::Error SugaredDAQmx::CreateCISemiPeriodChan(TaskHandle task,
                                                    const char counter[],
                                                    const char nameToAssignToChannel[],
                                                    float64 minVal, float64 maxVal,
                                                    int32 units,
                                                    const char customScaleName[]) {
    return process_error(dmx->CreateCISemiPeriodChan(
        task, counter, nameToAssignToChannel, minVal, maxVal, units, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateCITwoEdgeSepChan(TaskHandle task,
                                                    const char counter[],
                                                    const char nameToAssignToChannel[],
                                                    float64 minVal, float64 maxVal,
                                                    int32 units, int32 firstEdge,
                                                    int32 secondEdge,
                                                    const char customScaleName[]) {
    return process_error(dmx->CreateCITwoEdgeSepChan(
        task, counter, nameToAssignToChannel, minVal, maxVal, units, firstEdge,
        secondEdge, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateCOPulseChanFreq(TaskHandle task,
                                                   const char counter[],
                                                   const char nameToAssignToChannel[],
                                                   int32 units, int32 idleState,
                                                   float64 initialDelay, float64 freq,
                                                   float64 dutyCycle) {
    return process_error(dmx->CreateCOPulseChanFreq(task, counter,
                                                    nameToAssignToChannel, units,
                                                    idleState, initialDelay, freq,
                                                    dutyCycle));
}

xerrors::Error SugaredDAQmx::CreateCOPulseChanTicks(TaskHandle task,
                                                    const char counter[],
                                                    const char nameToAssignToChannel[],
                                                    const char sourceTerminal[],
                                                    int32 idleState, int32 initialDelay,
                                                    int32 lowTicks, int32 highTicks) {
    return process_error(dmx->CreateCOPulseChanTicks(
        task, counter, nameToAssignToChannel, sourceTerminal, idleState, initialDelay,
        lowTicks, highTicks));
}

xerrors::Error SugaredDAQmx::CreateCOPulseChanTime(TaskHandle task,
                                                   const char counter[],
                                                   const char nameToAssignToChannel[],
                                                   int32 units, int32 idleState,
                                                   float64 initialDelay,
                                                   float64 lowTime, float64 highTime) {
    return process_error(dmx->CreateCOPulseChanTime(task, counter,
                                                    nameToAssignToChannel, units,
                                                    idleState, initialDelay, lowTime,
                                                    highTime));
}

xerrors::Error SugaredDAQmx::CreateDIChan(TaskHandle task, const char lines[],
                                          const char nameToAssignToLines[],
                                          int32 lineGrouping) {
    return process_error(
        dmx->CreateDIChan(task, lines, nameToAssignToLines, lineGrouping));
}

xerrors::Error SugaredDAQmx::CreateDOChan(TaskHandle task, const char lines[],
                                          const char nameToAssignToLines[],
                                          int32 lineGrouping) {
    return process_error(
        dmx->CreateDOChan(task, lines, nameToAssignToLines, lineGrouping));
}

xerrors::Error SugaredDAQmx::CreateTEDSAIAccelChan(TaskHandle task,
                                                   const char physicalChannel[],
                                                   const char nameToAssignToChannel[],
                                                   int32 terminalConfig, float64 minVal,
                                                   float64 maxVal, int32 units,
                                                   int32 currentExcitSource,
                                                   float64 currentExcitVal,
                                                   const char customScaleName[]) {
    return process_error(dmx->CreateTEDSAIAccelChan(task, physicalChannel,
                                                    nameToAssignToChannel,
                                                    terminalConfig, minVal, maxVal,
                                                    units, currentExcitSource,
                                                    currentExcitVal, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateTEDSAIBridgeChan(TaskHandle task,
                                                    const char physicalChannel[],
                                                    const char nameToAssignToChannel[],
                                                    float64 minVal, float64 maxVal,
                                                    int32 units,
                                                    int32 voltageExcitSource,
                                                    float64 voltageExcitVal,
                                                    const char customScaleName[]) {
    return process_error(dmx->CreateTEDSAIBridgeChan(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units,
        voltageExcitSource, voltageExcitVal, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateTEDSAICurrentChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    int32 terminalConfig, float64 minVal, float64 maxVal, int32 units,
    int32 shuntResistorLoc, float64 extShuntResistorVal, const char customScaleName[]) {
    return process_error(dmx->CreateTEDSAICurrentChan(
        task, physicalChannel, nameToAssignToChannel, terminalConfig, minVal, maxVal,
        units, shuntResistorLoc, extShuntResistorVal, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateTEDSAIForceBridgeChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    float64 minVal, float64 maxVal, int32 units, int32 voltageExcitSource,
    float64 voltageExcitVal, const char customScaleName[]) {
    return process_error(dmx->CreateTEDSAIForceBridgeChan(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units,
        voltageExcitSource, voltageExcitVal, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateTEDSAIForceIEPEChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    int32 terminalConfig, float64 minVal, float64 maxVal, int32 units,
    int32 currentExcitSource, float64 currentExcitVal, const char customScaleName[]) {
    return process_error(dmx->CreateTEDSAIForceIEPEChan(
        task, physicalChannel, nameToAssignToChannel, terminalConfig, minVal, maxVal,
        units, currentExcitSource, currentExcitVal, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateTEDSAIMicrophoneChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    int32 terminalConfig, int32 units, float64 maxSndPressLevel,
    int32 currentExcitSource, float64 currentExcitVal, const char customScaleName[]) {
    return process_error(dmx->CreateTEDSAIMicrophoneChan(
        task, physicalChannel, nameToAssignToChannel, terminalConfig, units,
        maxSndPressLevel, currentExcitSource, currentExcitVal, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateTEDSAIPosLVDTChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    float64 minVal, float64 maxVal, int32 units, int32 voltageExcitSource,
    float64 voltageExcitVal, float64 voltageExcitFreq, int32 acExcitWireMode,
    const char customScaleName[]) {
    return process_error(dmx->CreateTEDSAIPosLVDTChan(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units,
        voltageExcitSource, voltageExcitVal, voltageExcitFreq, acExcitWireMode,
        customScaleName));
}

xerrors::Error SugaredDAQmx::CreateTEDSAIPosRVDTChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    float64 minVal, float64 maxVal, int32 units, int32 voltageExcitSource,
    float64 voltageExcitVal, float64 voltageExcitFreq, int32 acExcitWireMode,
    const char customScaleName[]) {
    return process_error(dmx->CreateTEDSAIPosRVDTChan(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units,
        voltageExcitSource, voltageExcitVal, voltageExcitFreq, acExcitWireMode,
        customScaleName));
}

xerrors::Error SugaredDAQmx::CreateTEDSAIPressureBridgeChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    float64 minVal, float64 maxVal, int32 units, int32 voltageExcitSource,
    float64 voltageExcitVal, const char customScaleName[]) {
    return process_error(dmx->CreateTEDSAIPressureBridgeChan(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units,
        voltageExcitSource, voltageExcitVal, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateTEDSAIRTDChan(TaskHandle task,
                                                 const char physicalChannel[],
                                                 const char nameToAssignToChannel[],
                                                 float64 minVal, float64 maxVal,
                                                 int32 units, int32 resistanceConfig,
                                                 int32 currentExcitSource,
                                                 float64 currentExcitVal) {
    return process_error(dmx->CreateTEDSAIRTDChan(task, physicalChannel,
                                                  nameToAssignToChannel, minVal, maxVal,
                                                  units, resistanceConfig,
                                                  currentExcitSource, currentExcitVal));
}

xerrors::Error SugaredDAQmx::CreateTEDSAIResistanceChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    float64 minVal, float64 maxVal, int32 units, int32 resistanceConfig,
    int32 currentExcitSource, float64 currentExcitVal, const char customScaleName[]) {
    return process_error(dmx->CreateTEDSAIResistanceChan(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units,
        resistanceConfig, currentExcitSource, currentExcitVal, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateTEDSAIStrainGageChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    float64 minVal, float64 maxVal, int32 units, int32 voltageExcitSource,
    float64 voltageExcitVal, float64 initialBridgeVoltage, float64 leadWireResistance,
    const char customScaleName[]) {
    return process_error(dmx->CreateTEDSAIStrainGageChan(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units,
        voltageExcitSource, voltageExcitVal, initialBridgeVoltage, leadWireResistance,
        customScaleName));
}

xerrors::Error SugaredDAQmx::CreateTEDSAIThrmcplChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    float64 minVal, float64 maxVal, int32 units, int32 cjcSource, float64 cjcVal,
    const char cjcChannel[]) {
    return process_error(dmx->CreateTEDSAIThrmcplChan(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units, cjcSource,
        cjcVal, cjcChannel));
}

xerrors::Error SugaredDAQmx::CreateTEDSAIThrmstrChanIex(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    float64 minVal, float64 maxVal, int32 units, int32 resistanceConfig,
    int32 currentExcitSource, float64 currentExcitVal) {
    return process_error(dmx->CreateTEDSAIThrmstrChanIex(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units,
        resistanceConfig, currentExcitSource, currentExcitVal));
}

xerrors::Error SugaredDAQmx::CreateTEDSAIThrmstrChanVex(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    float64 minVal, float64 maxVal, int32 units, int32 resistanceConfig,
    int32 voltageExcitSource, float64 voltageExcitVal, float64 r1) {
    return process_error(dmx->CreateTEDSAIThrmstrChanVex(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units,
        resistanceConfig, voltageExcitSource, voltageExcitVal, r1));
}

xerrors::Error SugaredDAQmx::CreateTEDSAITorqueBridgeChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    float64 minVal, float64 maxVal, int32 units, int32 voltageExcitSource,
    float64 voltageExcitVal, const char customScaleName[]) {
    return process_error(dmx->CreateTEDSAITorqueBridgeChan(
        task, physicalChannel, nameToAssignToChannel, minVal, maxVal, units,
        voltageExcitSource, voltageExcitVal, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateTEDSAIVoltageChan(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    int32 terminalConfig, float64 minVal, float64 maxVal, int32 units,
    const char customScaleName[]) {
    return process_error(dmx->CreateTEDSAIVoltageChan(
        task, physicalChannel, nameToAssignToChannel, terminalConfig, minVal, maxVal,
        units, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateTEDSAIVoltageChanWithExcit(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    int32 terminalConfig, float64 minVal, float64 maxVal, int32 units,
    int32 voltageExcitSource, float64 voltageExcitVal, const char customScaleName[]) {
    return process_error(dmx->CreateTEDSAIVoltageChanWithExcit(
        task, physicalChannel, nameToAssignToChannel, terminalConfig, minVal, maxVal,
        units, voltageExcitSource, voltageExcitVal, customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIVoltageChanWithExcit(
    TaskHandle task, const char physicalChannel[], const char nameToAssignToChannel[],
    int32 terminalConfig, float64 minVal, float64 maxVal, int32 units,
    int32 bridgeConfig, int32 voltageExcitSource, float64 voltageExcitVal,
    bool32 useExcitForScaling, const char customScaleName[]) {
    return process_error(dmx->CreateAIVoltageChanWithExcit(
        task, physicalChannel, nameToAssignToChannel, terminalConfig, minVal, maxVal,
        units, bridgeConfig, voltageExcitSource, voltageExcitVal, useExcitForScaling,
        customScaleName));
}

xerrors::Error SugaredDAQmx::CreateAIVoltageRMSChan(TaskHandle task,
                                                    const char physicalChannel[],
                                                    const char nameToAssignToChannel[],
                                                    int32 terminalConfig,
                                                    float64 minVal, float64 maxVal,
                                                    int32 units,
                                                    const char customScaleName[]) {
    return process_error(dmx->CreateAIVoltageRMSChan(
        task, physicalChannel, nameToAssignToChannel, terminalConfig, minVal, maxVal,
        units, customScaleName));
}
