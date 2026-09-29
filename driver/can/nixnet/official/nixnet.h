/*!
   \file nixnet.h
   \brief NI-XNET C API Implementation
*/
/*
   Copyright (c) 2008 National Instruments Corporation.
   All rights reserved.
*/

#ifndef ___nixnet_h___
#define ___nixnet_h___

#if defined(__GNUC__)
#include <stdint.h>   // C99 typedef for GCC
#endif


/***********************************************************************
                     C O M P I L E R   D E F I N E S
***********************************************************************/

   /* Needed for C++ to C (DLL) calls  */
#ifdef __cplusplus
   extern "C" {
#endif


#if defined(_WIN32)
   #define  _NXFUNC __cdecl
#else
   #define  _NXFUNC
#endif

#ifdef _CVI_
   #pragma  EnableLibraryRuntimeChecking
#endif /* _CVI_ */

/***********************************************************************
                              S T A T U S
***********************************************************************/

#define NX_STATUS_QUALIFIER                             (nxStatus_t)(0x3FF63000)
#define NX_STATUS_WARNING                               (nxStatus_t)(0x00000000)
#define NX_STATUS_ERROR                                 (nxStatus_t)(0x80000000)

#define NX_WARNING_BASE                                 (NX_STATUS_QUALIFIER | NX_STATUS_WARNING)
#define NX_ERROR_BASE                                   (NX_STATUS_QUALIFIER | NX_STATUS_ERROR)

#define nxSuccess                                       (0)

   // For a description of each error/warning (including solutions), use
   // nxStatusToString

//! An internal error occurred in the NI-XNET driver. Solution: Contact National
//! Instruments and provide the log files found in %LOCALAPPDATA%\National
//! Instruments\NI-XNET\log. Note that this location might be hidden on your
//! computer. For LabVIEW RT systems, log files are located in /ni-rt/nixnet/log
//! or /var/lib/ni-xnet/log.
#define nxErrInternalError                              (nxStatus_t)(0xBFF63001) // -1074384895

//! Board self test failed(code 2). Solution: Try reinstalling the driver or
//! switching the slot(s) of the board(s). If the error persists, contact
//! National Instruments.
#define nxErrSelfTestError1                             (nxStatus_t)(0xBFF63002) // -1074384894

//! Board self test failed(code 3). Solution: Try reinstalling the driver or
//! switching the slot(s) of the board(s). If the error persists, contact
//! National Instruments.
#define nxErrSelfTestError2                             (nxStatus_t)(0xBFF63003) // -1074384893

//! Board self test failed(code 4). Solution: Try reinstalling the driver or
//! switching the slot(s) of the board(s). If the error persists, contact
//! National Instruments.
#define nxErrSelfTestError3                             (nxStatus_t)(0xBFF63004) // -1074384892

//! Board self test failed(code 5). Solution: Try reinstalling the driver or
//! switching the slot(s) of the board(s). If the error persists, contact
//! National Instruments.
#define nxErrSelfTestError4                             (nxStatus_t)(0xBFF63005) // -1074384891

//! Board self test failed(code 6). Solution: Try reinstalling the driver or
//! switching the slot(s) of the board(s). If the error persists, contact
//! National Instruments.
#define nxErrSelfTestError5                             (nxStatus_t)(0xBFF63006) // -1074384890

//! Computer went to hibernation mode and the board lost power. Solution:
//! Prevent the computer from going to hibernation mode in the control panel.
#define nxErrPowerSuspended                             (nxStatus_t)(0xBFF63007) // -1074384889

//! A write queue overflowed. Solution: Wait until queue space becomes available
//! and retry.
#define nxErrOutputQueueOverflow                        (nxStatus_t)(0xBFF63008) // -1074384888

//! The board's firmware did not answer a command. Solution: Stop your
//! application and execute a self test. Try deactivating/reactivating the
//! driver in the Device Manager. If the problem persists, contact National
//! Instruments.
#define nxErrFirmwareNoResponse                         (nxStatus_t)(0xBFF63009) // -1074384887

//! The operation timed out. Solution: Specify a timeout long enough to complete
//! the operation, or change the operation in a way that it can get completed in
//! less time (e.g., read less data).
#define nxErrEventTimeout                               (nxStatus_t)(0xBFF6300A) // -1074384886

//! A read queue overflowed. Solution: Reduce your data rate or call Read more
//! frequently.
#define nxErrInputQueueOverflow                         (nxStatus_t)(0xBFF6300B) // -1074384885

//! The Read buffer is too small to hold a single frame. Solution: Provide a
//! buffer large enough.
#define nxErrInputQueueReadSize                         (nxStatus_t)(0xBFF6300C) // -1074384884

//! You tried to open the same frame twice. This is not permitted. Solution:
//! Open each frame only once.
#define nxErrDuplicateFrameObject                       (nxStatus_t)(0xBFF6300D) // -1074384883

//! You tried to open the same stream object twice. This is not permitted.
//! Solution: Open each stream object only once.
#define nxErrDuplicateStreamObject                      (nxStatus_t)(0xBFF6300E) // -1074384882

//! Self test is not possible since the board is in use by an application.
//! Solution: Stop all NI-XNET applications before executing a self test.
#define nxErrSelfTestNotPossible                        (nxStatus_t)(0xBFF6300F) // -1074384881

//! Allocation of memory failed. You do not have sufficient memory in the
//! LabVIEW target. Solution: Add more RAM or try to use fewer resources in your
//! applications (arrays, XNET sessions, etc).
#define nxErrMemoryFull                                 (nxStatus_t)(0xBFF63010) // -1074384880

//! The maximum number of sessions was exceeded. Solution: Use fewer sessions.
#define nxErrMaxSessions                                (nxStatus_t)(0xBFF63011) // -1074384879

//! The maximum number of frames has been exceeded. Solution: Use fewer frames
//! in your sessions.
#define nxErrMaxFrames                                  (nxStatus_t)(0xBFF63012) // -1074384878

//! The maximum number of devices has been detected. Solution: Use fewer
//! devices.
#define nxErrMaxDevices                                 (nxStatus_t)(0xBFF63013) // -1074384877

//! A driver support file is missing. Solution: Try reinstalling the driver. If
//! the error persists, contact National Instruments.
#define nxErrMissingFile                                (nxStatus_t)(0xBFF63014) // -1074384876

//! This indicates that a NULL pointer or an empty string was passed to a
//! function. The user should verify that the parameters passed in make sense
//! for the given function.
#define nxErrParameterNullOrEmpty                       (nxStatus_t)(0xBFF63015) // -1074384875

//! The maximum number of schedules has been detected. Solution: Use fewer
//! schedules.
#define nxErrMaxSchedules                               (nxStatus_t)(0xBFF63016) // -1074384874

//! Board self test failed (code 17). Solution: Try reinstalling the driver or
//! switching the slot(s) of the board(s). If the error persists, contact
//! National Instruments.
#define nxErrSelfTestError6                             (nxStatus_t)(0xBFF63017) // -1074384873

//! You cannot start an NI-XNET application while a self test is in progress.
//! Solution: Complete the self test before starting any NI-XNET applications.
#define nxErrSelfTestInProgress                         (nxStatus_t)(0xBFF63018) // -1074384872

//! The session contains a PDU with an offset that is not byte-aligned. Only
//! byte-aligned PDUs are supported for PDU sessions.
#define nxErrNonByteAlignedPDU                          (nxStatus_t)(0xBFF63019) // -1074384871

//! An invalid reference has been passed to a NI-XNET session function.
//! Solution: Only pass reference retrieved from Create Session, or from an IO
//! name of a session in LabVIEW project.
#define nxErrInvalidSessionHandle                       (nxStatus_t)(0xBFF63020) // -1074384864

//! An invalid reference has been passed to a NI-XNET system function. Solution:
//! Only pass a valid system reference.
#define nxErrInvalidSystemHandle                        (nxStatus_t)(0xBFF63021) // -1074384863

//! A device reference was expected for a NI-XNET session function. Solution:
//! Only pass a device reference.
#define nxErrDeviceHandleExpected                       (nxStatus_t)(0xBFF63022) // -1074384862

//! An interface reference was expected for a NI-XNET session function.
//! Solution: Only pass an interface reference.
#define nxErrIntfHandleExpected                         (nxStatus_t)(0xBFF63023) // -1074384861

//! You have configured a property that conflicts with the current mode of the
//! session. For example, you have created a CAN output session with a frame
//! configured with a Timing Type = Cyclic and a Transmit Time of 0.
#define nxErrPropertyModeConflicting                    (nxStatus_t)(0xBFF63024) // -1074384860

//! The XNET Create Timing Source VI is supported only on LabVIEW Real-Time
//! targets currently running the Phar Lap ETS Real-Time Operating System.
#define nxErrTimingSourceNotSupported                   (nxStatus_t)(0xBFF63025) // -1074384859

//! You tried to create more than one LabVIEW timing source for a single
//! interface. Only one timing source per interface is supported. The timing
//! source remains until the top-level VI is idle (no longer running). Solution:
//! Call the XNET Create Timing Source VI only once per interface. You can use
//! the timing source with multiple timed structures (e.g., timed loops).
#define nxErrMultipleTimingSource                       (nxStatus_t)(0xBFF63026) // -1074384858

//! You invoked two or more VIs simultaneously for the same session, and those
//! VIs do not support overlap. For example, you attempted to invoke two Read
//! VIs at the same time for the same session. Solution: Wire the error cluster
//! from one VI to another, to enforce sequential execution for the session.
#define nxErrOverlappingIO                              (nxStatus_t)(0xBFF63027) // -1074384857

//! You are trying to start an interface that is missing bus power for the
//! transceiver. Some physical layers on NI-XNET hardware are internally
//! powered, but others require external power for the port to operate. This
//! error occurs when starting an interface on hardware that requires external
//! power when no power is detected. Solution: Supply proper voltage to your
//! transceiver. Refer to the NI-XNET Hardware Overview in the NI-XNET Hardware
//! and Software Manual for more information.
#define nxErrMissingBusPower                            (nxStatus_t)(0xBFF63028) // -1074384856

//! The connection with a CompactDAQ chassis was lost, and the host software and
//! modules are out of sync.  There is no direct recovery for this problem until
//! the chassis is reset. Solutions: Call DAQmx Reset Device as the first VI or
//! function in your application, prior to creating XNET sessions. Alternately,
//! you could reset the CompactDAQ chassis in Measurement and Automation
//! Explorer (MAX).
#define nxErrCdaqConnectionLost                         (nxStatus_t)(0xBFF63029) // -1074384855

//! The transceiver value set is invalid (for this port, e.g., LS on a HS port)
//! or you are trying to perform an operation that requires a different
//! transceiver (e.g., trying to change the state of a disconnected
//! transceiver). Solution: Set a valid value.
#define nxErrInvalidTransceiver                         (nxStatus_t)(0xBFF63071) // -1074384783

//! The baud rate value set is invalid. Solution: Set a valid value.
#define nxErrInvalidBaudRate                            (nxStatus_t)(0xBFF63072) // -1074384782

//! No baud rate value has been set. Solution: Set a valid value.
#define nxErrBaudRateNotConfigured                      (nxStatus_t)(0xBFF63073) // -1074384781

//! The bit timing values set are invalid. Solution: Set valid values.
#define nxErrInvalidBitTimings                          (nxStatus_t)(0xBFF63074) // -1074384780

//! The baud rate set does not match the transceiver's allowed range. Solution:
//! Change either the baud rate or the transceiver.
#define nxErrBaudRateXcvrMismatch                       (nxStatus_t)(0xBFF63075) // -1074384779

//! The configured terminal is not known for this interface. Solution: Make sure
//! that that you pass in a valid value to Connect Terminals or Disconnect
//! Terminals.
#define nxErrUnknownTimingSource                        (nxStatus_t)(0xBFF63076) // -1074384778

//! The configured terminal is inappropriate for the hardware. For example,
//! setting a source to FrontPanel0 on XNET hardware that does not have
//! front-panel trigger inputs, or selecting PXI_Clk10 for a non-PXI device.
//! Solution: Pick an appropriate terminal for the hardware.
#define nxErrUnknownSynchronizationSource               (nxStatus_t)(0xBFF63077) // -1074384777

//! The source that you connected to the Master Timebase destination is missing.
//! When the start trigger is received, the interface verifies that a signal is
//! present on the configured source. This check has determined that this signal
//! is missing. Solution: Verify that your cables are configured correctly and
//! that your timebase source is generating an appropriate waveform.
#define nxErrMissingTimebaseSource                      (nxStatus_t)(0xBFF63078) // -1074384776

//! The source that you connected to the Master Timebase destination is not
//! generating an appropriate signal. When the start trigger is received, the
//! interface verifies that a signal of a known frequency is present on the
//! configured source. This check has determined that this source is generating
//! a signal, but that the signal is not one of the supported frequencies for
//! this hardware. Solution: Verify that your source is generating a signal at a
//! supported frequency.
#define nxErrUnknownTimebaseFrequency                   (nxStatus_t)(0xBFF63079) // -1074384775

//! You are trying to disconnect a synchronization terminal that is not
//! currently connected. Solution: Only disconnect synchronization terminals
//! that have previously been connected.
#define nxErrUnconnectedSynchronizationSource           (nxStatus_t)(0xBFF6307A) // -1074384774

//! You are trying to connect a synchronization terminal that is already in use.
//! For example, you are trying to connect a trigger line to the Master Timebase
//! when a different trigger line is already connected to the Master Timebase.
//! Solution: Only connect to synchronization terminals that are not currently
//! in use.
#define nxErrConnectedSynchronizationTerminal           (nxStatus_t)(0xBFF6307B) // -1074384773

//! You are trying to connect an XNET terminal as a source terminal,  but the
//! desired XNET terminal is not valid as a source terminal. Solution: Only
//! connect valid source terminals to the source terminal in XNET Connect
//! Terminals.
#define nxErrInvalidSynchronizationSource               (nxStatus_t)(0xBFF6307C) // -1074384772

//! You are trying to connect an XNET terminal as a destination terminal, but
//! the desired XNET terminal is not valid as a destination terminal. Solution:
//! Only connect valid destination terminals to the destination terminal in XNET
//! Connect Terminals.
#define nxErrInvalidSynchronizationDestination          (nxStatus_t)(0xBFF6307D) // -1074384771

//! You are trying to connect two XNET terminals that are incompatible.
//! Solution: Only connect a source and destination terminals that are
//! compatible with each other.
#define nxErrInvalidSynchronizationCombination          (nxStatus_t)(0xBFF6307E) // -1074384770

//! The source that you connected to the Master Timebase destination has
//! disappeared. When the start trigger is received, the interface verifies that
//! a signal is present on the configured source. This check has determined that
//! this signal was present, but while the interface was running, the signal
//! disappeared, so all timebase configuration has reverted to using the onboard
//! (unsynchronized) oscillator. Solution: Verify that your cables are
//! configured correctly and that your timebase source is generating an
//! appropriate waveform the entire time your application is running.
#define nxErrTimebaseDisappeared                        (nxStatus_t)(0xBFF6307F) // -1074384769

//! You called Read (State : FlexRay : Cycle Macrotick), and the FlexRay
//! Macrotick is not connected as the master timebase of the interface.
//! Solution: Call Connect Terminals to connect source of FlexRay Macrotick to
//! destination of Master Timebase.
#define nxErrMacrotickDisconnected                      (nxStatus_t)(0xBFF63080) // -1074384768

//! The database specified could not be opened. Solution: Check that the alias
//! and/or the file exists and that it is a valid database. Also, make sure the
//! path does not contain Unicode characters, which cannot be utilized by
//! NI-XNET.
#define nxErrCannotOpenDatabaseFile                     (nxStatus_t)(0xBFF63081) // -1074384767

//! The cluster was not found in the database. Solution: Make sure you only
//! initialize a cluster in a session that is defined in the database.
#define nxErrClusterNotFound                            (nxStatus_t)(0xBFF63082) // -1074384766

//! The frame was not found in the database. Solution: Make sure you only
//! initialize frames in a session that are defined in the database.
#define nxErrFrameNotFound                              (nxStatus_t)(0xBFF63083) // -1074384765

//! The signal was not found in the database. Solution: Make sure you only
//! initialize signals in a session that are defined in the database.
#define nxErrSignalNotFound                             (nxStatus_t)(0xBFF63084) // -1074384764

//! A necessary property for a cluster was not found in the database. Solution:
//! Make sure you only initialize a cluster in a session that is completely
//! defined in the database.
#define nxErrUnconfiguredCluster                        (nxStatus_t)(0xBFF63085) // -1074384763

//! A necessary property for a frame was not found in the database. Solution:
//! Make sure you only initialize frames in a session that are completely
//! defined in the database.
#define nxErrUnconfiguredFrame                          (nxStatus_t)(0xBFF63086) // -1074384762

//! A necessary property for a signal was not found in the database. Solution:
//! Make sure you only initialize signals in a session that are completely
//! defined in the database.
#define nxErrUnconfiguredSignal                         (nxStatus_t)(0xBFF63087) // -1074384761

//! Multiple clusters have been specified in one session, either directly
//! (Stream I/O), or through the signals or frames specified. Solution: Make
//! sure that in one session, you open only one cluster, including frames or
//! signals that belong to the same cluster.
#define nxErrMultipleClusters                           (nxStatus_t)(0xBFF63088) // -1074384760

//! You specified a database of ':subordinate:' for a session mode other than
//! mode of Frame Input Stream. Solution: Either open a Frame Input Stream
//! session, or use a real or in-memory database.
#define nxErrSubordinateNotAllowed                      (nxStatus_t)(0xBFF63089) // -1074384759

//! The interface name given does not specify a valid and existing interface.
//! Solution: Use a valid and existing interface. On Windows, these may be found
//! in MAX. On Linux distributions, interface names can be queried with either
//! the lsni or nixnetconfig command line tools. Interface names can also be
//! obtained using XNET system properties, or the LabVIEW XNET Interface IO
//! name. If you are using CompactRIO, refer to the topic "Getting Started with
//! CompactRIO" in the NI-XNET Hardware and Software Help.
#define nxErrInvalidInterface                           (nxStatus_t)(0xBFF6308A) // -1074384758

//! The operation is invalid for this interface (e.g., you tried to open a set
//! of FlexRay frames on a CAN interface, or tried to request a CAN property
//! from a FlexRay interface). Solution: Run this operation on a suitable
//! interface.
#define nxErrInvalidProtocol                            (nxStatus_t)(0xBFF6308B) // -1074384757

//! You tried to set the AutoStart property to FALSE for an Input session. This
//! is not allowed. Solution: Do not set the AutoStart property (TRUE is
//! default).
#define nxErrInputSessionMustAutoStart                  (nxStatus_t)(0xBFF6308C) // -1074384756

//! The property ID you specified is not valid (or not valid for the current
//! session mode or form factor). Check to make sure the installed XNET driver
//! supports the specified property ID.
#define nxErrInvalidPropertyId                          (nxStatus_t)(0xBFF6308D) // -1074384755

//! The contents of the property is bigger than the size specified. Use the
//! nxGetPropertySize function to determine the size of the buffer needed.
#define nxErrInvalidPropertySize                        (nxStatus_t)(0xBFF6308E) // -1074384754

//! The function you called is not defined for the session mode (e.g., you
//! called a frame I/O function on a signal I/O session).
#define nxErrIncorrectMode                              (nxStatus_t)(0xBFF6308F) // -1074384753

//! The data that you passed to XNET Write or XNET Read is too small to hold all
//! the data specified for the session. Solution: Determine the number of
//! elements (frames or signals) that you configured for the session, and pass
//! that number of elements to XNET Write or XNET Read.
#define nxErrBufferTooSmall                             (nxStatus_t)(0xBFF63090) // -1074384752

//! For Signal Output sessions, the multiplexer signals used in the session must
//! be specified explicitly in the signal list.
#define nxErrMustSpecifyMultiplexers                    (nxStatus_t)(0xBFF63091) // -1074384751

//! You used an XNET Session IO name, and that session was not found in your
//! LabVIEW project. Solution: Within LabVIEW project, right-click the target
//! (RT or My Computer), and select New > NI-XNET Session. Add the VI that uses
//! the session under the target. If you are using the session with a built
//! application (.EXE), ensure that you copy the built configuration file
//! nixnetSession.txt such that it resides in the same folder as the executable.
#define nxErrSessionNotFound                            (nxStatus_t)(0xBFF63092) // -1074384750

//! You used the same XNET session name in multiple top-level VIs, which is not
//! supported. Solution: Use each session in only one top-level VI (application)
//! at a time.
#define nxErrMultipleUseOfSession                       (nxStatus_t)(0xBFF63093) // -1074384749

//! To execute this function properly, the session's list must contain only one
//! frame. Solution: Break your session up into multiple, each of which contains
//! only one frame.
#define nxErrOnlyOneFrame                               (nxStatus_t)(0xBFF63094) // -1074384748

//! You used the same alias for different database files which is not allowed.
//! Solution: Use each alias only for a single database file.
#define nxErrDuplicateAlias                             (nxStatus_t)(0xBFF63095) // -1074384747

//! You try to deploy a database file while another deployment is in progress.
//! Solution: Wait until the other deployment has finished and try again.
#define nxErrDeploymentInProgress                       (nxStatus_t)(0xBFF63096) // -1074384746

//! A signal or frame session has been opened, but it does not contain signals
//! or frames. Solution: Specify at least one signal or frame.
#define nxErrNoFramesOrSignals                          (nxStatus_t)(0xBFF63097) // -1074384745

//! The session mode selected is not supported by the protocol of the specified
//! interface. Solution: Choose a valid session mode.
#define nxErrInvalidMode                                (nxStatus_t)(0xBFF63098) // -1074384744

//! A session was created by references, but no database references have been
//! specified. Solution: Specify at least one appropriate database reference
//! (i.e., signal or frame or cluster ref depending on the session mode).
#define nxErrNeedReference                              (nxStatus_t)(0xBFF63099) // -1074384743

//! The interface has already been opened with different cluster settings than
//! the ones specified for this session. Solution: Make sure that the cluster
//! settings agree for the interface, or use a different interface.
#define nxErrDifferentClusterOpen                       (nxStatus_t)(0xBFF6309A) // -1074384742

//! The cycle repetition of a frame in the database for the FlexRay protocol is
//! invalid. Solution: Make sure that the cycle repetition is a power of 2
//! between 1 and 64.
#define nxErrFlexRayInvalidCycleRep                     (nxStatus_t)(0xBFF6309B) // -1074384741

//! You called XNET Clear for the session, then tried to perform another
//! operation. Solution: Defer clear (session close) until you are done using
//! it. This error can also occur if you branch a wire after creating the
//! session. Solution: Do not branch a session to multiple flows in the diagram.
#define nxErrSessionCleared                             (nxStatus_t)(0xBFF6309C) // -1074384740

//! You called Create Session VI with a list of items that does not match the
//! mode. This includes using: 1) signal items for a Frame I/O mode 2) frame
//! items for a Signal I/O mode 3) cluster item for a mode other than Frame
//! Input Stream or Frame Output Stream
#define nxErrWrongModeForCreateSelection                (nxStatus_t)(0xBFF6309D) // -1074384739

//! You tried to create a new session while the interface is already running.
//! Solution: Create all sessions before starting any of them.
#define nxErrInterfaceRunning                           (nxStatus_t)(0xBFF6309E) // -1074384738

//! The length of the payload written is larger than the maximum payload size
//! allowed.
#define nxErrPayloadTooLarge                            (nxStatus_t)(0xBFF6309F) // -1074384737
#define nxErrFrameWriteTooLarge                         (nxErrPayloadTooLarge)

//! You called a Read function with a nonzero timeout, and you used a negative
//! numberToRead. Negative value for numberToRead requests all available data
//! from the Read, which is ambiguous when used with a timeout. Solutions: 1)
//! Pass timeout of 0 and numberToRead of -1, to request all available data. 2)
//! Pass timeout > 0, and numberToRead > 0, to wait for a specific number of
//! data elements.
#define nxErrTimeoutWithoutNumToRead                    (nxStatus_t)(0xBFF630A0) // -1074384736

//! Timestamps are not (yet) supported for Write Signal XY. Solution: Do not
//! provide a timestamp array for Write Signal XY.
#define nxErrTimestampsNotSupported                     (nxStatus_t)(0xBFF630A1) // -1074384735

//! The condition parameter passed to Wait is not known. Solution: Pass a valid
//! parameter.
#define nxErrUnknownCondition                           (nxStatus_t)(0xBFF630A2) // -1074384734

//! You attempted an I/O operation, but the session is not yet started (and the
//! AutoStart property is set to FALSE). Solution: Call Start before you use
//! this I/O operation.
#define nxErrSessionNotStarted                          (nxStatus_t)(0xBFF630A3) // -1074384733

//! The maximum number of Wait operations has been exceeded. Solution: If you
//! are waiting for multiple events on the interface, use fewer Wait operations
//! on this interface (even for multiple sessions). If you are waiting for
//! multiple events for a frame (e.g., transmit complete), use only one Wait at
//! a time for that frame.
#define nxErrMaxWaitsExceeded                           (nxStatus_t)(0xBFF630A4) // -1074384732

//! You used an invalid name for an XNET Device. Solution: Get valid XNET Device
//! names from the XNET System properties (only).
#define nxErrInvalidDevice                              (nxStatus_t)(0xBFF630A5) // -1074384731

//! A terminal name passed to ConnectTerminals or DisconnectTerminals is
//! unknown. Solution: Only pass valid names.
#define nxErrInvalidTerminalName                        (nxStatus_t)(0xBFF630A6) // -1074384730

//! You tried to blink the port LEDs but these are currently busy. Solution:
//! Stop all applications running on that port; do not access it from MAX or LV
//! Project.
#define nxErrPortLEDsBusy                               (nxStatus_t)(0xBFF630A7) // -1074384729

//! You tried to set a FlexRay keyslot ID that is not listed as valid in the
//! database. Solution: Only pass slot IDs of frames that have the startup or
//! sync property set in the database.
#define nxErrInvalidKeyslot                             (nxStatus_t)(0xBFF630A8) // -1074384728

//! You tried to set a queue size that is bigger than the maximum allowed.
//! Solution: Specify an in-range queue size.
#define nxErrMaxQueueSizeExceeded                       (nxStatus_t)(0xBFF630A9) // -1074384727

//! You wrote a frame whose payload length is different than the payload length
//! configured by the database. Solution: Never write a different payload length
//! for a frame that is different than the configured payload length.
#define nxErrFrameSizeMismatch                          (nxStatus_t)(0xBFF630AA) // -1074384726

//! The index to indicate an session list element is too large. Solution:
//! Specify an index in the range 0 ... NumInList-1.
#define nxErrIndexTooBig                                (nxStatus_t)(0xBFF630AB) // -1074384725

//! You have tried to create a session that is invalid for the mode of the
//! driver/firmware. For example, you are using the Replay Exclusive mode for
//! Stream Output and you have an output session open.
#define nxErrSessionModeIncompatibility                 (nxStatus_t)(0xBFF630AC) // -1074384724

//! The trigger signal for a frame is allowed only in Single Point Signal
//! sessions (Input or Output). For Output Single Point Signal sessions, only
//! one trigger signal is allowed per frame. Solution: Do not use the trigger
//! signal, or change to a single point I/O session.
#define nxErrTriggerSignalNotAllowed                    (nxStatus_t)(0xBFF630AD) // -1074384723

//! To execute this function properly, the session's list must contain only one
//! cluster. Solution: Use only one cluster in the session.
#define nxErrOnlyOneCluster                             (nxStatus_t)(0xBFF630AE) // -1074384722

//! You attempted to convert a CAN or LIN frame with a payload length greater
//! than 8. For example, you may be converting a frame that uses a higher layer
//! transport protocol, such as SAE-J1939. NI-XNET currently supports conversion
//! of CAN/LIN frames only (layer 2). Solutions: 1) Implement higher layer
//! protocols (including signal conversion) within your code. 2) Contact
//! National Instruments to request this feature in a future version.
#define nxErrConvertInvalidPayload                      (nxStatus_t)(0xBFF630AF) // -1074384721

//! Allocation of memory failed for the data returned from LabVIEW XNET Read.
//! Solutions: 1) Wire a smaller "number to read" to XNET Read (default -1 uses
//! queue size).  2) For Signal Input Waveform, use a smaller resample rate. 3)
//! Set smaller value for session's queue size property (default is large to
//! avoid loss of data).
#define nxErrMemoryFullReadData                         (nxStatus_t)(0xBFF630B0) // -1074384720

//! Allocation of memory failed in the firmware. Solutions: 1) Create less
//! firmware objects 2) Set smaller value for output session's queue size
//! property (default is large to avoid loss of data).
#define nxErrMemoryFullFirmware                         (nxStatus_t)(0xBFF630B1) // -1074384719

//! The NI-XNET driver no longer can communicate with the device. Solution: Make
//! sure the device has not been removed from the computer.
#define nxErrCommunicationLost                          (nxStatus_t)(0xBFF630B2) // -1074384718

//! A LIN schedule has an invalid priority. Solution: Use a valid priority (0 =
//! NULL schedule, 1..254 = Run once schedule, 255 = Continuous schedule).
#define nxErrInvalidPriority                            (nxStatus_t)(0xBFF630B3) // -1074384717

//! (Dis)ConnectTerminals is not allowed for XNET C Series modules. Solution: To
//! connect the module start trigger, use the Session property Interface Source
//! Terminal Start Trigger.
#define nxErrSynchronizationNotAllowed                  (nxStatus_t)(0xBFF630B4) // -1074384716

//! You requested a time (like Start or Communication Time) before the event has
//! happened. Solution: Request the time only after it occurred.
#define nxErrTimeNotReached                             (nxStatus_t)(0xBFF630B5) // -1074384715

//! An internal input queue overflowed. Solution: Attempt to pull data from the
//! hardware faster. If you are connected by an external bus (for example, USB
//! or Ethernet), you can try to use a faster connection.
#define nxErrInternalInputQueueOverflow                 (nxStatus_t)(0xBFF630B6) // -1074384714

//! A bad firmware image file can not be loaded to the hardware. Solution:
//! Uninstall and reinstall the NI-XNET software as the default firmware file
//! may be corrupt. If you are using a custom firmware file, try rebuilding it.
#define nxErrBadImageFile                               (nxStatus_t)(0xBFF630B7) // -1074384713

//! The encoding of embedded network data (CAN, FlexRay, LIN, etc.) within the
//! TDMS file is invalid. Solutions: 1) In the application that wrote (created)
//! the logfile, and the application in which you are reading it, confirm that
//! both use the same major version for frame data encoding
//! (NI_network_frame_version property of the TDMS channel). 2) Ensure that your
//! file was not corrupted.
#define nxErrInvalidLogfile                             (nxStatus_t)(0xBFF630B8) // -1074384712

//! The NI-XNET hardware no longer can communicate with the transceiver cable.
//! This may be due to the cable being removed, a power loss event, an over
//! voltage condition on the power input, or a general communication error.
//! Solution: Make sure the dongle is properly latched and, for some hardware,
//! external power is properly applied. To detect other errors, stop your
//! application and execute a self test.
#define nxErrDongleCommunicationLost                    (nxStatus_t)(0xBFF630B9) // -1074384711

//! The NI-XNET driver timed out when performing low-level communication with
//! the device. This could occur if the device was removed, communication with
//! its parent chassis failed, or a general hardware failure occurred.
#define nxErrLowLevelCommunicationTimeout               (nxStatus_t)(0xBFF630BA) // -1074384710

//! A transfer to the C Series module was aborted. This could occur if the cDAQ
//! chassis was disconnected, reset, power cycled, or another issue occurred
//! that interfered with communication. Solution: Verify that the cDAQ chassis
//! and NI-XNET module are both present, powered, and operational in Measurement
//! and Automation Explorer (MAX). A reset of the cDAQ chassis may be necessary
//! to recover from the error.
#define nxErrCdaqTransferAborted                        (nxStatus_t)(0xBFF630BB) // -1074384709

//! An invalid value has been specified for the CAN FD ISO Mode. Solution:
//! Specify a valid value.
#define nxErrInvalidCanFdIsoMode                        (nxStatus_t)(0xBFF630BC) // -1074384708

//! A property value was out of range or incorrect. Solution: Specify a correct
//! value.
#define nxErrInvalidPropertyValue                       (nxStatus_t)(0xBFF630C0) // -1074384704

//! Integration of the interface into the FlexRay cluster failed, so
//! communication did not start for the interface. Solution: Check the cluster
//! and/or interface parameters and verify that there are startup frames
//! defined.
#define nxErrFlexRayIntegrationFailed                   (nxStatus_t)(0xBFF630C1) // -1074384703

//! The data passed to XNET Write is not formatted correctly for this session
//! type. For example, frame data cannot be passed to a PDU session. Solution:
//! Use the data type that matches the session mode to which you are writing.
#define nxErrInvalidBuffer                              (nxStatus_t)(0xBFF630C2) // -1074384702

//! The PDU was not found in the database. Solution: Make sure you initialize
//! only PDUs in a session that are defined in the database.
#define nxErrPduNotFound                                (nxStatus_t)(0xBFF630D0) // -1074384688

//! A necessary property for a PDU was not found in the database. Solution: Make
//! sure you initialize only PDUs in a session that are completely defined in
//! the database.
#define nxErrUnconfiguredPdu                            (nxStatus_t)(0xBFF630D1) // -1074384687

//! You tried to open the same PDU twice. This is not permitted. Solution: Open
//! each PDU only once.
#define nxErrDuplicatePduObject                         (nxStatus_t)(0xBFF630D2) // -1074384686

//! You can access this database object only by PDU, not by frame. Solution: For
//! CAN and LIN, this is not supported by the current version of NI-XNET; for
//! FlexRay, make sure the database is set to use PDUs.
#define nxErrNeedPdu                                    (nxStatus_t)(0xBFF630D3) // -1074384685

//! The requested PDU could not be mapped to a frame. The current version of
//! NI-XNET might not support this PDU type.
#define nxErrUnmappedPdu                                (nxStatus_t)(0xBFF630D4) // -1074384684

//! The requested PDU is mapped to multiple locations. With the selected
//! hardware, the current version of NI-XNET only supports PDUs mapped to a
//! single location.
#define nxErrMultiplePduMappings                        (nxStatus_t)(0xBFF630D5) // -1074384683

//! Remote communication with the LabVIEW RT target failed. Solution: Check if
//! NI-XNET has been installed on the RT target and check if the NI-XNET RPC
//! server has been started.
#define nxErrRPCCommunication                           (nxStatus_t)(0xBFF63100) // -1074384640

//! File transfer communication with the LabVIEW Real-Time (RT) target failed.
//! Solution: Check if the RT target has been powered on, the RT target has been
//! connected to the network, and if the IP address settings are correct. If the
//! RT target is password protected, make sure you provided the correct user
//! credentials.
#define nxErrFileTransferCommunication                  (nxStatus_t)(0xBFF63101) // -1074384639

//! File transfer to the LabVIEW Real-Time (RT) target failed, because the
//! required files could not be accessed. Solution: You may have executed a VI
//! that opened the database, but did not close. If that is the case, you should
//! change the VI to call Database Close, then reboot the RT controller to
//! continue.
#define nxErrFileTransferAccess                         (nxStatus_t)(0xBFF63102) // -1074384638
#define nxErrFTPFileAccess                              (nxErrFileTransferAccess)

//! The database file you want to use is already assigned to another alias.
//! Solution: Each database file can only be assigned to a single alias. Use the
//! alias that is already assigned to the database instead.
#define nxErrDatabaseAlreadyInUse                       (nxStatus_t)(0xBFF63103) // -1074384637

//! An internal file used by NI-XNET could not be accessed. Solution: Make sure
//! that the internal NI-XNET files are not write protected and that the
//! directories for these files exist.
#define nxErrInternalFileAccess                         (nxStatus_t)(0xBFF63104) // -1074384636

//! The file cannot be deployed because another file deployment is already
//! active. Solution: Wait until the other file deployment has finished, and try
//! again.
#define nxErrFileTransferActive                         (nxStatus_t)(0xBFF63105) // -1074384635

//! The nixnet.dll or one of its components could not be loaded. Solution: Try
//! reinstalling NI-XNET. If the error persists, contact National Instruments.
#define nxErrDllLoad                                    (nxStatus_t)(0xBFF63117) // -1074384617

//! You attempted to perform an action on a session or interface that is
//! started, and the action that requires the session/interface to be stopped.
//! Solution: Stop the object before performing this action.
#define nxErrObjectStarted                              (nxStatus_t)(0xBFF6311E) // -1074384610

//! The length of the default payload does not match the payload length
//! configured in the database. Solution: Write a default payload with a length
//! that matches the Payload Length property configured in the database.
#define nxErrDefaultPayloadNumBytes                     (nxStatus_t)(0xBFF6311F) // -1074384609

//! You attempted to set a CAN arbitration ID with an invalid value. For
//! example, a CAN standard arbitration ID supports only 11 bits. If you attempt
//! to set a standard arbitration ID that uses more than 11 bits, this error is
//! returned. Solution: Use a valid arbitration ID.
#define nxErrInvalidArbitrationId                       (nxStatus_t)(0xBFF63123) // -1074384605

//! You attempted to set a LIN ID with an invalid value. For example, a LIN ID
//! supports only 6 bits. If you attempt to set an ID that uses more than 6
//! bits, this error is returned. Solution: Use a valid LIN ID.
#define nxErrInvalidLinId                               (nxStatus_t)(0xBFF63124) // -1074384604

//! Too many database files are open. NI-XNET allows up to 63 database files to
//! be opened simultaneously. Solution: Open fewer files.
#define nxErrTooManyOpenFiles                           (nxStatus_t)(0xBFF63130) // -1074384592

//! Bad reference has been passed to a database function, e.g., a session
//! reference, or frame reference to retrieve properties from a signal.
#define nxErrDatabaseBadReference                       (nxStatus_t)(0xBFF63131) // -1074384591

//! Creating a database file failed. Solution: Verify access rights to the
//! destination directory or check if overwritten file has read only permission.
#define nxErrCreateDatabaseFile                         (nxStatus_t)(0xBFF63132) // -1074384590

//! A cluster with the same name already exists in the database. Solution: Use
//! another name for this cluster.
#define nxErrDuplicateClusterName                       (nxStatus_t)(0xBFF63133) // -1074384589

//! A frame with the same name already exists in the cluster. Solution: Use
//! another name for this frame.
#define nxErrDuplicateFrameName                         (nxStatus_t)(0xBFF63134) // -1074384588

//! A signal with the same name already exists in the frame. Solution: Use
//! another name for this signal.
#define nxErrDuplicateSignalName                        (nxStatus_t)(0xBFF63135) // -1074384587

//! An ECU with the same name already exists in the cluster. Solution: Use
//! another name for this ECU.
#define nxErrDuplicateECUName                           (nxStatus_t)(0xBFF63136) // -1074384586

//! A subframe with the same name already exists in the frame. Solution: Use
//! another name for this subframe.
#define nxErrDuplicateSubframeName                      (nxStatus_t)(0xBFF63137) // -1074384585

//! The operation is improper for the protocol in use, e.g., you cannot assign
//! FlexRay channels to a CAN frame.
#define nxErrImproperProtocol                           (nxStatus_t)(0xBFF63138) // -1074384584

//! Wrong parent relationship for a child that you are creating with XNET
//! Database Create.
#define nxErrObjectRelation                             (nxStatus_t)(0xBFF63139) // -1074384583

//! The retrieved required property is not defined on the specified object.
//! Solution: Make sure that your database file has this property defined or
//! that you set it in the objects created in memory.
#define nxErrUnconfiguredRequiredProperty               (nxStatus_t)(0xBFF6313B) // -1074384581

//! The feature is not supported under LabVIEW RT, e.g.Save Database
#define nxErrNotSupportedOnRT                           (nxStatus_t)(0xBFF6313C) // -1074384580

//! The object name contains unsupported characters. The name must contain just
//! alphanumeric characters and the underscore, but cannot begin with a digit.
//! The maximum size is 128.
#define nxErrNameSyntax                                 (nxStatus_t)(0xBFF6313D) // -1074384579

//! Unsupported database format. For reading a database, the extension must be
//! .xml, .dbc, .ncd, or .ldf. For saving, the extension must be .xml or .ldf
#define nxErrFileExtension                              (nxStatus_t)(0xBFF6313E) // -1074384578

//! Database object not found (e.g., an object with given name does not exist).
#define nxErrDatabaseObjectNotFound                     (nxStatus_t)(0xBFF6313F) // -1074384577

//! Database cache file cannot be removed or replaced on the disc (e.g., file is
//! write-protected).
#define nxErrRemoveDatabaseCacheFile                    (nxStatus_t)(0xBFF63140) // -1074384576

//! You are trying to write a read-only property, e.g., the mux value on a
//! signal is a read only property (can be changed on the subframe).
#define nxErrReadOnlyProperty                           (nxStatus_t)(0xBFF63141) // -1074384575

//! You are trying to change a signal to be a mux signal, but a mux is already
//! defined in this frame
#define nxErrFrameMuxExists                             (nxStatus_t)(0xBFF63142) // -1074384574

//! You are trying to define FlexRay in-cycle-repetition slots before defining
//! the first slot. Define the first slot (frame ID) before defining
//! in-cycle-repetition slots.
#define nxErrUndefinedFirstSlot                         (nxStatus_t)(0xBFF63144) // -1074384572

//! You are trying to define FlexRay in-cycle-repetition channels before
//! defining the first channels. Define the Channel Assignment on a frame before
//! defining in-cycle-repetition channels.
#define nxErrUndefinedFirstChannels                     (nxStatus_t)(0xBFF63145) // -1074384571

//! You must define the protocol before setting this property (e.g., the frame
//! ID has a different meaning in a CAN or FlexRay cluster).
#define nxErrUndefinedProtocol                          (nxStatus_t)(0xBFF63146) // -1074384570

//! The database information on the real-time system has been created with an
//! older NI-XNET version. This version is no longer supported. To correct this
//! error, re-deploy your database to the real-time system.
#define nxErrOldDatabaseCacheFile                       (nxStatus_t)(0xBFF63147) // -1074384569

//! Frame ConfigStatus: A signal within the frame exceeds the frame boundaries
//! (Payload Length).
#define nxErrDbConfigSigOutOfFrame                      (nxStatus_t)(0xBFF63148) // -1074384568

//! Frame ConfigStatus: A signal within the frame overlaps another signal.
#define nxErrDbConfigSigOverlapped                      (nxStatus_t)(0xBFF63149) // -1074384567

//! Frame ConfigStatus: A integer signal within the frame is defined with more
//! than 52 bits. Not supported.
#define nxErrDbConfigSig52BitInteger                    (nxStatus_t)(0xBFF6314A) // -1074384566

//! Frame ConfigStatus: Frame is defined with an invalid payload size. Allowed
//! sizes vary by bus type and application protocol.
#define nxErrDbConfigFrameNumBytes                      (nxStatus_t)(0xBFF6314B) // -1074384565

//! You are trying to add transmitted FlexRay frames to an ECU, with at least
//! two of them having Startup or Sync property on. Only one Sync or Startup
//! frame is allowed to be sent by an ECU.
#define nxErrMultSyncStartup                            (nxStatus_t)(0xBFF6314C) // -1074384564

//! You are trying to add TX/RX frames to an ECU which are defined in a
//! different cluster than the ECU.
#define nxErrInvalidCluster                             (nxStatus_t)(0xBFF6314D) // -1074384563

//! Database name parameter is incorrect. Solution: Use a valid name for the
//! database, e.g., ":memory:" for in-memory database.
#define nxErrDatabaseName                               (nxStatus_t)(0xBFF6314E) // -1074384562

//! Database object is locked because it is used in a session. Solution:
//! Configure the database before using it in a session.
#define nxErrDatabaseObjectLocked                       (nxStatus_t)(0xBFF6314F) // -1074384561

//! Alias name passed to a function is not defined. Solution: Define the alias
//! before calling the function.
#define nxErrAliasNotFound                              (nxStatus_t)(0xBFF63150) // -1074384560

//! Database file cannot be saved because frames are assigned to FlexRay
//! channels not defined in the cluster. Solution: Verify that all frames in the
//! FlexRay cluster are assigned to an existing cluster channel.
#define nxErrClusterFrameChannelRelation                (nxStatus_t)(0xBFF63151) // -1074384559

//! Frame ConfigStatus: This FlexRay frame transmitted in a dynamic segment uses
//! both channels A and B. This is not allowed. Solution: Use either channel A
//! or B.
#define nxErrDynFlexRayFrameChanAandB                   (nxStatus_t)(0xBFF63152) // -1074384558

//! Database is locked because it is being modified by an another instance of
//! the same application. Solution: Close the database in the other application
//! instance.
#define nxErrDatabaseLockedInUse                        (nxStatus_t)(0xBFF63153) // -1074384557

//! A frame name is ambiguous (e.g., a frame with the same name exists in
//! another cluster). Solution: Specify the cluster name for the frame using the
//! required syntax.
#define nxErrAmbiguousFrameName                         (nxStatus_t)(0xBFF63154) // -1074384556

//! A signal name is ambiguous (e.g., a signal with the same name exists in
//! another frame). Solution: Use <frame>.<signal> syntax for the signal.
#define nxErrAmbiguousSignalName                        (nxStatus_t)(0xBFF63155) // -1074384555

//! An ECU name is ambiguous (e.g., an ECU with the same name exists in another
//! cluster). Solution: Specify the cluster name for the ECU using the required
//! syntax.
#define nxErrAmbiguousECUName                           (nxStatus_t)(0xBFF63156) // -1074384554

//! A subframe name is ambiguous (e.g., a subframe with the same name exists in
//! another cluster). Solution: Specify the cluster name for the subframe using
//! the required syntax.
#define nxErrAmbiguousSubframeName                      (nxStatus_t)(0xBFF63157) // -1074384553

//! A LIN schedule name is ambiguous (e.g., a schedule with the same name exists
//! in another cluster). Solution: Specify the cluster name for the schedule
//! using the required syntax.
#define nxErrAmbiguousScheduleName                      (nxStatus_t)(0xBFF63158) // -1074384552

//! A LIN schedule with the same name already exists in the database. Solution:
//! Use another name for this schedule.
#define nxErrDuplicateScheduleName                      (nxStatus_t)(0xBFF63159) // -1074384551

//! Multiplexers (mode-dependent signals) are not supported when the given
//! protocol is used. Solution: Contact National Instruments to see whether
//! there is a newer NI-XNET version that supports multiplexers for the given
//! protocol.
#define nxErrProtocolMuxNotSupported                    (nxStatus_t)(0xBFF6315A) // -1074384550

//! Saving a FIBEX file containing a LIN cluster is not supported in this
//! NI-XNET version. Solution: Contact National Instruments to see whether there
//! is a newer NI-XNET version that supports saving a FIBEX file that contains a
//! LIN cluster.
#define nxErrSaveLINnotSupported                        (nxStatus_t)(0xBFF6315B) // -1074384549

//! This property requires an ECU configured as LIN master to be present in this
//! cluster. Solution: Create a LIN master ECU in this cluster.
#define nxErrLINmasterNotDefined                        (nxStatus_t)(0xBFF6315C) // -1074384548

//! You have tried to create a session using a frame that is incompatible with
//! the selected session type. For example, you are using a LIN diagnostic frame
//! with a single point output session.
#define nxErrSessionTypeFrameIncompatibility            (nxStatus_t)(0xBFF6315D) // -1074384547

//! You cannot mix open of NI-XNET database objects as both manual and
//! automatic. You open manually by calling the Database Open VI. You open
//! automatically when you 1) wire the IO name directly to a property node or
//! VI, 2) branch a wire to multiple data flows on the diagram, 3) use the IO
//! name with a VI or property node after closing it with the Database Close VI.
//! Solution: Change your diagram to use the manual technique in all locations
//! (always call Open and Close VIs), or to use the automatic technique in all
//! locations (never call Open or Close VIs).
#define nxErrMixAutoManualOpen                          (nxStatus_t)(0xBFF6315E) // -1074384546

//! Due to problems in LabVIEW versions 8.5 through 8.6.1, automatic open of
//! NI-XNET database objects is not supported. You open automatically when you
//! 1) wire the IO name directly to a property node or VI, 2) branch a wire to
//! multiple data flows on the diagram, 3) use the IO name with a VI or property
//! node after closing it with the Database Close VI. Solution: Change your
//! diagram to call the Database Open VI prior to any use (VI or property node)
//! in a data flow (including a new wire branch). Change your diagram to call
//! the Database Close VI when you are finished using the database in your
//! application.
#define nxErrAutoOpenNotSupported                       (nxStatus_t)(0xBFF6315F) // -1074384545

//! A write function was called with a number of array elements (frames, PDUs,
//! or signals) different from the number of elements configured in the session
//! (such as the "list" parameter of the Create Session function). Solution:
//! Ensure the write function calls the same number of elements as configured in
//! the session.
#define nxErrWrongNumElementsWritten                    (nxStatus_t)(0xBFF63160) // -1074384544
#define nxErrWrongNumSignalsWritten                     (nxErrWrongNumElementsWritten)

//! You used XNET session from multiple LabVIEW projects (or multiple
//! executables), which NI-XNET does not support. Solution: Run XNET sessions in
//! only one LabVIEW project at a time.
#define nxErrMultipleLvProject                          (nxStatus_t)(0xBFF63161) // -1074384543

//! When an XNET session is used at runtime, all sessions in the same scope are
//! created on the interface. The same scope is defined as all sessions within
//! the same LabVIEW project which use the same cluster and interface (same
//! physical cable configuration). If you attempt to use a session in the same
//! scope after running the VI, this error occurs. The most likely cause is that
//! you added a new session, and tried to use that new session in a running VI.
//! Solution: Configure all session in LabVIEW project, then run the VI(s) that
//! use those sessions.
#define nxErrSessionConflictLvProject                   (nxStatus_t)(0xBFF63162) // -1074384542

//! You used an empty name for an XNET database object (database, cluster, ECU,
//! frame, or signal). Empty name is not supported. Solution: Refer to NI-XNET
//! help for IO names to review the required syntax for the name, and change
//! your code to use that syntax.
#define nxErrDbObjectNameEmpty                          (nxStatus_t)(0xBFF63163) // -1074384541

//! You used a name for an XNET database object (such as frame or signal) that
//! did not include a valid cluster selection. Solution: Refer to the NI-XNET
//! help for the IO name that you are using, and use the syntax specified for
//! that class, which includes the cluster selection.
#define nxErrMissingAliasInDbObjectName                 (nxStatus_t)(0xBFF63164) // -1074384540

//! Unsupported database file version. Solution: Use only database versions that
//! are supported by this version of NI-XNET. Please see the NI-XNET
//! documentation for information on which database versions are currently
//! supported.
#define nxErrDatabaseImportVersion                      (nxStatus_t)(0xBFF63165) // -1074384539
#define nxErrFibexImportVersion                         (nxErrDatabaseImportVersion)

//! You used an empty name for the XNET Session. Empty name is not supported.
//! Solution: Use a valid XNET session name from your LabVIEW project.
#define nxErrEmptySessionName                           (nxStatus_t)(0xBFF63166) // -1074384538

//! There is not enough message RAM on the FlexRay hardware to configure the
//! data partition for the object(s). Solution: Please refer to the manual for
//! limitations on the number of objects that can be created at any given time
//! based on the payload length.
#define nxErrNotEnoughMessageRAMForObject               (nxStatus_t)(0xBFF63167) // -1074384537

//! The FlexRay keyslot ID has been configured and a startup session has been
//! created. Either the keyslot ID needs to be configured OR the startup session
//! needs to be created. Both cannot exist at the same time. Solution: Choose a
//! single method to configure startup sessions in your application.
#define nxErrKeySlotIDConfig                            (nxStatus_t)(0xBFF63168) // -1074384536

//! An unsupported session was created. For example, stream output is not
//! supported on FlexRay hardware. Solution: Only use supported sessions in your
//! application.
#define nxErrUnsupportedSession                         (nxStatus_t)(0xBFF63169) // -1074384535

//! An XNET session was created after starting the Interface. Only the Stream
//! Input session in the subordinate mode can be created after the Interface has
//! started. Solution: Create sessions prior to starting the XNET Interface in
//! your application.
#define nxErrObjectCreatedAfterStart                    (nxStatus_t)(0xBFF63170) // -1074384528

//! The Single Slot property was enabled on the XNET FlexRay Interface after the
//! interface had started. Solution: Enable the Single Slot property prior to
//! starting the XNET FlexRay Interface.
#define nxErrSingleSlotEnabledAfterStart                (nxStatus_t)(0xBFF63171) // -1074384527

//! The FlexRay macrotick offset specified for XNET Create Timing Source is
//! unsupported. Example: Specifying a macrotick offset greater than
//! MacroPerCycle will result in this error. Solution: Specify a macrotick
//! offset within the supported range for the cluster.
#define nxErrUnsupportedNumMacroticks                   (nxStatus_t)(0xBFF63172) // -1074384526

//! You used invalid syntax in the name of a database object (signal, frame, or
//! ECU). For example, you may have specified a frame's name as
//! <cluster>.<frame>, which is allowed in NI-XNET for C/C++, but not NI-XNET
//! for LabVIEW. Solution: Use the string syntax specified in the help topic for
//! the XNET I/O name class you are using.
#define nxErrBadSyntaxInDatabaseObjectName              (nxStatus_t)(0xBFF63173) // -1074384525

//! A LIN schedule entry name is ambiguous, e.g., a schedule entry with the same
//! name exists in another schedule. Solution: Specify the schedule name for the
//! schedule entry using the required syntax.
#define nxErrAmbiguousScheduleEntryName                 (nxStatus_t)(0xBFF63174) // -1074384524

//! A LIN schedule entry with the same name already exists in the schedule.
//! Solution: Use another name for this schedule entry.
#define nxErrDuplicateScheduleEntryName                 (nxStatus_t)(0xBFF63175) // -1074384523

//! At least one of the frames in the session has an undefined identifier.
//! Solution: Set the frame's "Identifier (Slot)" property before creating the
//! session.
#define nxErrUndefinedFrameId                           (nxStatus_t)(0xBFF63176) // -1074384522

//! At least one of the frames in the session has an undefined payload length.
//! Solution: Set the frame's "Payload Length (in bytes)" property before
//! creating the session.
#define nxErrUndefinedFramePayloadLength                (nxStatus_t)(0xBFF63177) // -1074384521

//! At least one of the signals in the session has an undefined start bit.
//! Solution: Set the "Start Bit" property of the signal before creating the
//! session.
#define nxErrUndefinedSignalStartBit                    (nxStatus_t)(0xBFF63178) // -1074384520

//! At least one of the signals in the session has an undefined number of bits.
//! Solution: Set the "Number of Bits" property of the signal before creating
//! the session.
#define nxErrUndefinedSignalNumBits                     (nxStatus_t)(0xBFF63179) // -1074384519

//! At least one of the signals in the session has an undefined byte order.
//! Solution: Set the "Byte Order" property of the signal before creating the
//! session.
#define nxErrUndefinedSignalByteOrder                   (nxStatus_t)(0xBFF6317A) // -1074384518

//! At least one of the signals in the session has an undefined data type.
//! Solution: Set the "Data Type" property of the signal before creating the
//! session.
#define nxErrUndefinedSignalDataType                    (nxStatus_t)(0xBFF6317B) // -1074384517

//! At least one of the subframes in the session has an undefined multiplexer
//! value. Solution: Set the "Multiplexer Value" property of the subframe before
//! creating the session.
#define nxErrUndefinedSubfMuxValue                      (nxStatus_t)(0xBFF6317C) // -1074384516

//! You provided an invalid index to Write (State LIN Schedule Change).
//! Solution: Use a number from 0 to N-1, where N is the number of LIN schedules
//! returned from the cluster property LIN Schedules. If you are using LabVIEW,
//! the string for the number must be decimal (not hexadecimal).
#define nxErrInvalidLinSchedIndex                       (nxStatus_t)(0xBFF6317D) // -1074384515

//! You provided an invalid name to Write (State LIN Schedule Change). Solution:
//! Use a valid LIN schedule name returned from the cluster property LIN
//! Schedules, or the session property Interface LIN Schedules. You can use the
//! short name (schedule only) or long name (schedule plus database and
//! cluster).
#define nxErrInvalidLinSchedName                        (nxStatus_t)(0xBFF6317E) // -1074384514

//! You provided an invalid active index for the session property.
#define nxErrInvalidActiveFrameIndex                    (nxStatus_t)(0xBFF6317F) // -1074384513

//! You provided an invalid name for Frame:Active of the session property node.
//! Solution: Use a valid item name from the session's List property. You can
//! use the short name (frame or signal only) or long name (frame/signal plus
//! database and cluster).
#define nxErrInvalidActiveFrameName                     (nxStatus_t)(0xBFF63180) // -1074384512

//! The database you are using requires using PDUs, and the operation is
//! ambiguous with respect to PDUs. Example: You are trying to get the frame
//! parent of the signal, but the PDU in which the signal is contained is
//! referenced in multiple frames.
#define nxErrAmbiguousPDU                               (nxStatus_t)(0xBFF63181) // -1074384511

//! A PDU with the same name already exists in the cluster. Solution: Use
//! another name for this PDU.
#define nxErrDuplicatePDU                               (nxStatus_t)(0xBFF63182) // -1074384510

//! You are trying to assign start bits or update bits to PDUs referenced in a
//! frame, but the number of elements in this array is different than the number
//! of referenced PDUs. Solution: Use the same number of elements in the array
//! as in the PDU references array.
#define nxErrNumberOfPDUs                               (nxStatus_t)(0xBFF63183) // -1074384509

//! The configuration of this object requires using advanced PDUs, which the
//! given protocol does not support. Solution: You cannot use this object in the
//! given protocol.
#define nxErrPDUsRequired                               (nxStatus_t)(0xBFF63184) // -1074384508

//! The maximum number of PDUs has been exceeded. Solution: Use fewer PDUs in
//! your sessions.
#define nxErrMaxPDUs                                    (nxStatus_t)(0xBFF63185) // -1074384507

//! This mode value is not currently supported. Solution: Use a valid value.
#define nxErrUnsupportedMode                            (nxStatus_t)(0xBFF63186) // -1074384506

//! The firmware image on your XNET hardware is corrupted. Solution: Update the
//! firmware of this XNET hardware in MAX.
#define nxErrBadFpgaSignature                           (nxStatus_t)(0xBFF63187) // -1074384505
#define nxErrBadcSeriesFpgaSignature                    (nxErrBadFpgaSignature)

//! The firmware version of your XNET hardware is not in sync with your host
//! computer. Solution: Update the firmware of this XNET hardware in MAX.
#define nxErrBadFpgaRevision                            (nxStatus_t)(0xBFF63188) // -1074384504
#define nxErrBadcSeriesFpgaRevision                     (nxErrBadFpgaRevision)

//! The firmware version of your XNET C Series module is not in sync with the
//! NI-XNET software on your remote target. Solution: Update the NI-XNET
//! software on the remote target.
#define nxErrBadFpgaRevisionOnTarget                    (nxStatus_t)(0xBFF63189) // -1074384503

//! The terminal you are trying to use is already in use. Only one connection
//! per terminal is allowed.  Solution: Disconnect the terminal that is already
//! in use.
#define nxErrRouteInUse                                 (nxStatus_t)(0xBFF6318A) // -1074384502

//! You need to install a supported version of NI-DAQmx for your XNET C Series
//! module to work correctly with your Compact DAQ system. Solution: Check the
//! NI-XNET readme file for supported versions of the NI-DAQmx driver software.
#define nxErrDAQmxIncorrectVersion                      (nxStatus_t)(0xBFF6318B) // -1074384501

//! Unable to create the requested route. This may be caused by a routing
//! conflict or an invalid terminal name. Solution: Fix invalid terminal names,
//! such as a blank string. Since NI-XNET relies on the NI-DAQmx driver software
//! to create routes on Compact DAQ chassis, use DAQmx to resolve routing
//! conflicts.
#define nxErrAddRoute                                   (nxStatus_t)(0xBFF6318C) // -1074384500

//! You attempted to transmit a go to sleep frame (by setting the LIN Sleep mode
//! to Remote Sleep) on a LIN interface configured as slave.  In conformance
//! with the LIN protocol standard, only an interface configured as master may
//! transmit a go to sleep frame.
#define nxErrRemoteSleepOnLinSlave                      (nxStatus_t)(0xBFF6318D) // -1074384499

//! You attempted to set properties related to Sleep and Wakeup when the FlexRay
//! cluster defined in the Fibex file does not support it. Solution: Edit the
//! Fibex file used in your application to include all relevant cluster wakeup
//! attributes.
#define nxErrSleepWakeupNotSupported                    (nxStatus_t)(0xBFF6318E) // -1074384498

//! A LIN diagnostic schedule change requires the diagnostic schedule to be
//! defined in the database. Solution: Define the diagnostic schedule in the
//! database.
#define nxErrDiagnosticScheduleNotDefined               (nxStatus_t)(0xBFF6318F) // -1074384497

//! The data payload written for a diagnostic frame for transmit does not
//! conform to the LIN transport layer specification.  Solution: Ensure the data
//! payload for a diagnostic frame conforms to the transport layer
//! specification.
#define nxErrLINTransportLayer                          (nxStatus_t)(0xBFF63192) // -1074384494

//! An error occurred within the NI-XNET example code for logfile access (TDMS).
//! Solution: For LabVIEW, the subVI with the error is shown as the source, and
//! you can open that subVI to determine the cause of the problem. For other
//! programming languages, review the source code for the logfile example to
//! determine the cause of the problem.
#define nxErrLogfile                                    (nxStatus_t)(0xBFF63193) // -1074384493

//! You attempted to write a LIN schedule and use a stream output replay timing
//! mode concurrently. You can only use the stream output immediate timing mode
//! cuncurrently with the LIN scheduler.
#define nxErrStrmOutTmgLinSchedulerConflict             (nxStatus_t)(0xBFF63200) // -1074384384

//! You attempted to create a session that is incompatible with the LIN
//! interface personality (master or slave), or set the LIN interface
//! personality to one that is incompatible with a session already created for
//! it. For example, setting the LIN interface to slave after creating a stream
//! output session will report this error, because only LIN interface as master
//! supports stream output.
#define nxErrSessnTypeLinIntfPrsIncompatible            (nxStatus_t)(0xBFF63201) // -1074384383

//! You attempted to save an LDF or DBC database, but the passed reference is
//! not a database cluster. Solution: A cluster reference must be used to
//! specify the cluster you want to export.
#define nxErrSaveClusterOnly                            (nxStatus_t)(0xBFF63202) // -1074384382
//! Need to define for compatibility with older versions
#define nxErrSaveLdfClusterOnly                         (nxErrSaveClusterOnly)

//! You tried to assign the same interface name twice. This is not permitted.
//! Solution: Assign a unique name to an interface.
#define nxErrDuplicateInterfaceName                     (nxStatus_t)(0xBFF63203) // -1074384381

//! Transceiver cable hardware revision is too new. The current driver does not
//! support this transceiver cable. Solution: Upgrade the NI-XNET driver.
#define nxErrIncompatibleTransceiverRevision            (nxStatus_t)(0xBFF63204) // -1074384380
#define nxErrIncompatiableTransceiverRevision           (nxErrIncompatibleTransceiverRevision)

//! Transceiver cable image revision is too new. The current driver does not
//! support this transceiver cable. Solution: Upgrade the NI-XNET driver or
//! downgrade the image on the transceiver cable.
#define nxErrIncompatibleTransceiverImage               (nxStatus_t)(0xBFF63205) // -1074384379
#define nxErrIncompatiableTransceiverImage              (nxErrIncompatibleTransceiverImage)

//! The property does not apply to this type of hardware. For example, a dongle
//! revision property cannot be accessed on an XNET device that does not support
//! Transceiver Cables. Solution: Do not apply the property to this type of
//! hardware.
#define nxErrPropertyNotsupported                       (nxStatus_t)(0xBFF63206) // -1074384378

//! Exporting cluster into the specified database type failed. Solution: Ensure
//! the database configuration is complete. Refer to the standard documentation
//! for the related file format.
#define nxErrExport_Semantic                            (nxStatus_t)(0xBFF63207) // -1074384377

//! A J1939 input queue overflowed. Reading large J1939 frames can make  the
//! queue overflow, and the Read function delivers fewer frames then specified.
//! Solution: Call the Read function again to read the remaining frames.
#define nxErrJ1939QueueOverflow                         (nxStatus_t)(0xBFF63208) // -1074384376

//! You are trying to transmit a non-J1939 frame with more than 8 bytes. Only
//! J1939 frames can use the J1939 transport protocol.  Solution: Verify the
//! transport protocol property on the frame in  the database.
#define nxErrNonJ1939FrameSize                          (nxStatus_t)(0xBFF63209) // -1074384375

//! You are trying to transmit a J1939 frame, but no J1939 address is assigned
//! to the session. Solution: Set the address using the J1939 address property.
#define nxErrJ1939MissingAddress                        (nxStatus_t)(0xBFF6320A) // -1074384374

//! The received J1939 TP.CM_CTS message has the wrong total size.
#define nxErrJ1939AddressLost                           (nxStatus_t)(0xBFF6320B) // -1074384373

//! The next packet value of the received J1939 TP.CM_CTS message is larger than
//! the total number of packets.
#define nxErrJ1939CtsNextPckLargerTotalPckNum           (nxStatus_t)(0xBFF6320C) // -1074384372

//! The received J1939 TP.CM_CTS message has a number of packets of 0, but the
//! next packet number is not 255.
#define nxErrJ1939CtsNextPck                            (nxStatus_t)(0xBFF6320D) // -1074384371

//! The received J1939 TP.CM_CTS message has a next packet value of 0.
#define nxErrJ1939CtsNextPckNull                        (nxStatus_t)(0xBFF6320E) // -1074384370

//! The received J1939 TP.CM_CTS message does not have the same PGN as in the
//! TP.CM_RTS message.
#define nxErrJ1939CtsPgn                                (nxStatus_t)(0xBFF6320F) // -1074384369

//! Received unexpected sequence number in the J1939 TP.DT message.
#define nxErrJ1939UnexpectedSeqNum                      (nxStatus_t)(0xBFF63210) // -1074384368

//! More Packets are requested than allowed in the J1939 TP.CM_CTS message.
#define nxErrJ1939MorePckReqThanAllowed                 (nxStatus_t)(0xBFF63211) // -1074384367

//! J1939 Timeout T1 while waiting for data.
#define nxErrJ1939TimeoutT1                             (nxStatus_t)(0xBFF63212) // -1074384366

//! J1939 Timeout T2 while waiting for data.
#define nxErrJ1939TimeoutT2                             (nxStatus_t)(0xBFF63213) // -1074384365

//! J1939 Timeout T3 while waiting for TP.CM_CTS or TP.CM_EndOfMsgAck.
#define nxErrJ1939TimeoutT3                             (nxStatus_t)(0xBFF63214) // -1074384364

//! J1939 Timeout T4 while waiting for next CTS MSG.
#define nxErrJ1939TimeoutT4                             (nxStatus_t)(0xBFF63215) // -1074384363

//! Received wrong DLC in the J1939 TP.CM_RTS message. DLC must be 8.
#define nxErrJ1939RtsDlc                                (nxStatus_t)(0xBFF63216) // -1074384362

//! Received wrong DLC in the J1939 TP.CM_CTS message. DLC must be 8.
#define nxErrJ1939CtsDlc                                (nxStatus_t)(0xBFF63217) // -1074384361

//! Received wrong DLC in the J1939 TP.CM_BAM message. DLC must be 8.
#define nxErrJ1939BamDlc                                (nxStatus_t)(0xBFF63218) // -1074384360

//! Received wrong DLC in the J1939 TP.DT message. DLC must be 8.
#define nxErrJ1939DtDlc                                 (nxStatus_t)(0xBFF63219) // -1074384359

//! Received wrong DLC in the J1939 TP.CM_Abort message. DLC must be 8.
#define nxErrJ1939AbortDlc                              (nxStatus_t)(0xBFF6321A) // -1074384358

//! Received wrong DLC in the J1939 TP.CM_EndOfMsgAck message. DLC must be 8.
#define nxErrJ1939EomaDlc                               (nxStatus_t)(0xBFF6321B) // -1074384357

//! Received wrong PGN in the J1939 TP.CM_Abort message.
#define nxErrJ1939AbortPgn                              (nxStatus_t)(0xBFF6321C) // -1074384356

//! Internal error occurred for send TP.CM_CTS Hold Message.
#define nxErrJ1939CtsHoldMsg                            (nxStatus_t)(0xBFF6321D) // -1074384355

//! Invalid total message size in J1939 TP.CM_RTS message. Expect 9..1785.
#define nxErrJ1939InvalidTotalSize                      (nxStatus_t)(0xBFF6321E) // -1074384354

//! Total number of packets in received J1939 TP.CM_RTS message must be greater
//! than 1.
#define nxErrJ1939TotalPckNum                           (nxStatus_t)(0xBFF6321F) // -1074384353

//! Reserved data bytes in J1939 received message are not BFF63FF.
#define nxErrJ1939ReservedData                          (nxStatus_t)(0xBFF63220) // -1074384352

//! Not enough system resources for the J1939 Transport Protocol.
#define nxErrJ1939NotEnoughSysRes                       (nxStatus_t)(0xBFF63221) // -1074384351

//! Received J1939 TP.CM_Abort message with reason ActiveConnection: Already in
//! one or more connection managed sessions and cannot support another.
#define nxErrJ1939AbortMsgActiveConnection              (nxStatus_t)(0xBFF63222) // -1074384350

//! Received J1939 TP.CM_Abort message with reason NotEnoughSystemResources:
//! System resources were needed for another task, so this connection managed
//! session was terminated.
#define nxErrJ1939AbortMsgNotEnoughSysRes               (nxStatus_t)(0xBFF63223) // -1074384349

//! Received J1939 TP.CM_Abort message with reason Timeout: A timeout occurred,
//! and this is the connection abort to close the session.
#define nxErrJ1939AbortMsgTimeout                       (nxStatus_t)(0xBFF63224) // -1074384348

//! Received J1939 TP.CM_Abort message with reason CtsReceived: CTS messages
//! received when data transfer is in progress.
#define nxErrJ1939AbortMsgCtsRec                        (nxStatus_t)(0xBFF63225) // -1074384347

//! Received J1939 TP.CM_Abort message with reason MaxRetransmit: Maximum
//! retransmit request limit reached.
#define nxErrJ1939AbortMsgMaxRetransmit                 (nxStatus_t)(0xBFF63226) // -1074384346

//! Remote communication with the LabVIEW RT target failed because the host and
//! target versions of NI-XNET are different. Solution: On the target, install
//! the same NI-XNET version that is installed on the host.
#define nxErrRPCVersion                                 (nxStatus_t)(0xBFF63227) // -1074384345

//! The CAN frame I/O mode is higher than the CAN cluster  I/O mode. This frame
//! cannot be transmitted on the network. Solution: Change the frame or cluster
//! I/O mode.
#define nxErrFrameCanIoMode                             (nxStatus_t)(0xBFF63228) // -1074384344

//! The current driver cannot update the firmware on your hardware. Solution:
//! Ask National Instruments for compatible driver software.
#define nxErrIncompatibleFlash                          (nxStatus_t)(0xBFF63229) // -1074384343

//! You are trying to use the CAN Transmit I/O Mode (TxIoMode) property in an
//! unsupported interface mode. Solution: You can use this property in only
//! non-ISO or ISO Legacy mode.
#define nxErrTxIoMode                                   (nxStatus_t)(0xBFF6322A) // -1074384342

//! You are trying to use the XS Transceiver Cable on unsupported hardware. This
//! currently requires a PXIe-8510 board.
#define nxErrXsDongleUnsupportedBoard                   (nxStatus_t)(0xBFF6322B) // -1074384341

//! You are trying to use a database alias name that contains an invalid
//! character (for example, a comma).
#define nxErrInvalidCharInDatabaseAlias                 (nxStatus_t)(0xBFF6322C) // -1074384340

//! You are trying to use a database filepath that contains an invalid character
//! (for example, a comma).
#define nxErrInvalidCharInDatabaseFilepath              (nxStatus_t)(0xBFF6322D) // -1074384339

//! You are trying to use CAN FD with a non-HS/FD port. CAN FD is supported with
//! High Speed CAN only.
#define nxErrInvalidCanFdPortType                       (nxStatus_t)(0xBFF6322E) // -1074384338

//! An unconditional LIN schedule entry is incorrectly configured. To fix this
//! issue, reference only one frame in this entry.
#define nxErrInvUnconditionalEntry                      (nxStatus_t)(0xBFF6322F) // -1074384337

//! An event LIN schedule entry has no collision resolving schedule assigned. To
//! fix this issue, assign a schedule to the schedule entry.
#define nxErrEventEntryNoSchedule                       (nxStatus_t)(0xBFF63230) // -1074384336

//! You have connected your USB device to a port that supports only Full Speed
//! (USB 1.1). NI-XNET USB devices require at least High Speed (USB 2.0+)
//! support for correct operation.
#define nxErrUnsupportedUsbSpeed                        (nxStatus_t)(0xBFF63231) // -1074384335

//! You added two or more frames with different payload sizes to the same LIN
//! event-triggered schedule entry. Use an equal payload length for all
//! event-triggered frames in an event-triggered schedule entry.
#define nxErrEventUnequalPayloadLength                  (nxStatus_t)(0xBFF63232) // -1074384334

//! You added two or more frames with different checksum types to the same
//! event-triggered schedule entry. The LIN checksum type is determined by the
//! LIN protocol version of the ECU transmitting the frame. Solution: Use the
//! same checksum model for all event-triggered frames in an event-triggered
//! schedule entry.
#define nxErrEventUnequalCheckSumType                   (nxStatus_t)(0xBFF63233) // -1074384333

//! The versions of the NI-XNET device driver and Compact RIO driver are
//! incompatible. Solution: Install drivers listed in the Recommended Software
//! Set.
#define nxErrCrioBadDriverVersions                      (nxStatus_t)(0xBFF63234) // -1074384332

//! The module support for this slot is missing features required for the
//! NI-XNET driver to be able to communicate with the module. Solution: Load a
//! bitfile to the chassis that contains support for NI-XNET modules for the
//! occupied slots.
#define nxErrCrioMissingSlotSupport                     (nxStatus_t)(0xBFF63235) // -1074384331

//! The interface you have attempted to use with NI-XNET Compatibility Library
//! is already in use by NI-XNET.  Solution: Stop all applications running on
//! the interface, or select one not in use by NI-XNET sessions.
#define nxErrIntfAlreadyInUseByNiXnet                   (nxStatus_t)(0xBFF63236) // -1074384330

//! The interface you have attempted to use with NI-XNET is already in use by
//! NI-XNET Compatibility Library.  Solution: Stop all applications running on
//! the interface, or select one not in use by the NI-XNET Compatibility
//! Library.
#define nxErrIntfAlreadyInUseByNiXcl                    (nxStatus_t)(0xBFF63237) // -1074384329

//! Byte array signals are not allowed for this session type or function.
//! Solution: Select only numeric signals for the session.
#define nxErrByteArrayNotAllowed                        (nxStatus_t)(0xBFF63238) // -1074384328

//! Mixing of byte array signals and numeric signals is not allowed for this
//! session type. Solution: Select only numeric signals or byte array signals
//! for the session.
#define nxErrNoByteArrayMix                             (nxStatus_t)(0xBFF63239) // -1074384327

//! Only one byte array signal is allowed for this session type or function.
//! Solution: Select only one byte array signal for the session.
#define nxErrOnlyOneByteArray                           (nxStatus_t)(0xBFF6323A) // -1074384326

//! The feature you have attempted to use is not supported. Solution: Use an
//! alternate feature that is supported.
#define nxErrNotSupported                               (nxStatus_t)(0xBFF6323B) // -1074384325

//! The specified property is write-only and is therefore not readable.
//! Solution: Do not read write-only properties.
#define nxErrWriteOnlyProperty                          (nxStatus_t)(0xBFF6323C) // -1074384324

//! FPGA download is not possible because the board is in use by an application.
//! Solution: Stop all NI-XNET applications before executing FPGA download.
#define nxErrFpgaDownloadNotPossible                    (nxStatus_t)(0xBFF6323D) // -1074384323

//! The trigger timestamp cannot be read, because the trigger source is not
//! connected. Solution: Connect the TimeTrigger to a valid source before
//! reading the timestamp.
#define nxErrTimeTriggerSourceNotConnected              (nxStatus_t)(0xBFF6323E) // -1074384322

//! The trigger timestamp cannot be read, because another operation is reading
//! the timestamp. Solution: Perform only one trigger timestamp read at a time.
#define nxErrTimeTriggerReadInProgress                  (nxStatus_t)(0xBFF6323F) // -1074384321

//! The operation did not complete because it was aborted. This typically occurs
//! when the XNET session is closed while an operation is in progress. Solution:
//! Allow the operation to finish before closing the XNET session.
#define nxErrOperationAborted                           (nxStatus_t)(0xBFF63240) // -1074384320
#define nxErrWaitAborted                                (nxErrOperationAborted)

//! The future time trigger cannot be generated, because the trigger destination
//! is not connected. Solution: Connect the TimeTrigger to a valid destination
//! before generating the future time trigger.
#define nxErrFutureTimeTriggerNotConnected              (nxStatus_t)(0xBFF63241) // -1074384319

//! The log file could not be created because it already exists. Solution:
//! Specify a different file name or choose a mode that allows overwriting the
//! file.
#define nxErrLogFileAlreadyExists                       (nxStatus_t)(0xBFF63242) // -1074384318

//! Could not create or replace the log file. This can happen if there are
//! insufficient file permissions at the specified location or invalid file path
//! syntax.
#define nxErrLogFileCreate                              (nxStatus_t)(0xBFF63243) // -1074384317

//! You cannot read data using XNET Read after a session has been started with
//! logging enabled.
#define nxErrLogModeRead                                (nxStatus_t)(0xBFF63244) // -1074384316

//! A problem has been detected with the firmware on the device. Solution: Stop
//! your application and try disabling/re-enabling the driver in Device Manager.
//! If the problem persists, contact National Instruments.
#define nxErrFirmwareFault                              (nxStatus_t)(0xBFF63245) // -1074384315

//! A write queue overflowed for a session that has not been started. Solution:
//! Write less data before starting the session, or write data after the session
//! has already been started.
#define nxErrOutputQueueOverflowNotStarted              (nxStatus_t)(0xBFF63246) // -1074384314

//! The total size of the data to be transmitted is larger than the internal
//! write queue. Solution: Write less data at a time.
#define nxErrWriteTooLarge                              (nxStatus_t)(0xBFF63247) // -1074384313

//! The requested number of packets do not fit in the current queue. Solution:
//! Increase the queue size or decrease the number of packets to read.
#define nxErrReadExceedsQueue                           (nxStatus_t)(0xBFF63248) // -1074384312

//! The length of the payload written is smaller than the minimum payload size
//! allowed.
#define nxErrPayloadTooSmall                            (nxStatus_t)(0xBFF63249) // -1074384311
#define nxErrFrameWriteTooSmall                         (nxErrPayloadTooSmall)

//! The XNET endpoint on this port cannot start while the Port Mode is set to
//! Tap. The endpoint has no access to data while in this mode. Solution: Set
//! the Port Mode to Direct to have full use of the XNET endpoint.
#define nxErrEndpointCannotStartWhileInTapMode          (nxStatus_t)(0xBFF6324A) // -1074384310

//! The future time trigger cannot be generated because a previously scheduled
//! future time trigger is still pending. Solution: Wait until the previously
//! scheduled future time trigger occurs, or cancel the pre-existing trigger by
//! disconnecting the time trigger resource.
#define nxErrFutureTimeTriggerPending                   (nxStatus_t)(0xBFF6324B) // -1074384309

//! The desired time to generate the future time trigger has already elapsed.
//! Solution: Schedule the future time trigger for a point in time further into
//! the future.
#define nxErrFutureTimeTriggerAlreadyElapsed            (nxStatus_t)(0xBFF6324C) // -1074384308

//! The network time cannot be adjusted while this clock is a slave (i.e., not
//! grandmaster). Solution: Wait until the port state is no longer set to slave
//! or configure this clock as a grandmaster.
#define nxErrNetworkTimeNotAdjustableWhileSlave         (nxStatus_t)(0xBFF6324D) // -1074384307

//! Output sessions must be created using an interface's endpoint path.
//! Solution: Change the interface to use the endpoint path (e.g., use the
//! interface name that does not contain "/monitor").
#define nxErrOutputStreamMustUseEndpointPath            (nxStatus_t)(0xBFF6324E) // -1074384306

//! The time protocol cannot be used while the Port Mode is set to Tap.
//! Solution: If Tap mode is desired, disable the time protocol on both ports of
//! the Tap pair before switching to Tap mode. If usage of the time protocol is
//! desired, set the Port Mode to Direct before enabling the time protocol.
#define nxErrCannotUseTimeProtocolWhileInTapMode        (nxStatus_t)(0xBFF6324F) // -1074384305

//! You cannot write data while the XNET session is started and the link is
//! down. Solution: Ensure the link is up prior to writing data. You can use the
//! Operational Status property to query the current link state.
#define nxErrCannotWriteWhileStartedAndLinkDown         (nxStatus_t)(0xBFF63250) // -1074384304

//! Device reset is not possible since the board is in use by an application.
//! Solution: Stop all NI-XNET applications before executing a device reset.
#define nxErrDeviceResetNotPossible                     (nxStatus_t)(0xBFF63251) // -1074384303

//! You cannot start an NI-XNET application while a device reset is in progress.
//! Solution: Complete the device reset before starting any NI-XNET
//! applications.
#define nxErrDeviceResetInProgress                      (nxStatus_t)(0xBFF63252) // -1074384302

//! You wrote a frame whose frame type value is not valid. Solution: Encode the
//! frame with frame type value that is supported by NI-XNET.
#define nxErrFrameTypeInvalid                           (nxStatus_t)(0xBFF63253) // -1074384301

//! The data path you have attempted to use is already in use by another
//! session. Solution: Stop the other application using the desired data path,
//! or select a data path not in use by NI-XNET sessions.
#define nxErrDataPathAlreadyInUseBySession              (nxStatus_t)(0xBFF63254) // -1074384300

//! The timescale value is invalid. Solution: Use a valid value.
#define nxErrInvalidTimescale                           (nxStatus_t)(0xBFF63255) // -1074384299

//! A hardware error has occurred while reading the Port MAC address. Solution:
//! Reset this XNET hardware in MAX.
#define nxErrReadMacAddressFailed                       (nxStatus_t)(0xBFF63256) // -1074384298

//! You cannot write to the Time Sync Protocol property while the Port Mode is
//! set to Tap. Solution: Do not write to this property while ports are
//! configured to use Tap mode. To configure the Time Sync Protocol property,
//! set the Port Mode to Direct before writing to this property.
#define nxErrTimeProtocolPropertyReadOnlyWhileInTapMode (nxStatus_t)(0xBFF63257) // -1074384297

//! The buffer is not large enough to contain the requested path.
#define nxErrPathBufferTooSmall                         (nxStatus_t)(0xBFF63258) // -1074384296

//! A NULL pointer or an empty string was passed when requesting a path.
#define nxErrPathBufferNull                             (nxStatus_t)(0xBFF63259) // -1074384295

//! The device hardware revision is too new for the currently installed driver.
//! Solution: Upgrade the NI-XNET driver.
#define nxErrHardwareNeedsNewerDriver                   (nxStatus_t)(0xBFF6325A) // -1074384294

//! An error occurred compiling the Ethernet Frame Filter due to incorrect
//! syntax. Refer to NI-XNET documentation for information about the supported
//! syntax.
#define nxErrBadEnetFrameFilterSyntax                   (nxStatus_t)(0xBFF6325B) // -1074384293

//! This function requires that the session's list contain only one PDU.
//! Solution: Create multiple sessions, each of which contains only one PDU.
#define nxErrSessionContainsMultiplePdus                (nxStatus_t)(0xBFF6325C) // -1074384292

//! A PDU session has been opened, but it does not contain any PDUs. Solution:
//! Specify at least one PDU.
#define nxErrNoPDUs                                     (nxStatus_t)(0xBFF6325D) // -1074384291

//! You are attempting to create a session with a PDU or frame that has an
//! invalid configuration. Review the frame or PDU properties in the NI-XNET
//! Database Editor for a detailed description of the error. Refer to NI-XNET
//! help for details about PDU and frame configurations.
#define nxErrInvalidMapping                             (nxStatus_t)(0xBFF6325E) // -1074384290

//! A file input/output error occurred while writing to the log file.
#define nxErrLogFileIO                                  (nxStatus_t)(0xBFF6325F) // -1074384289

//! Sleep handshake was not successful. Solution: Ensure the connected link
//! partner is configured to respond to sleep requests.
#define nxErrSleepFail                                  (nxStatus_t)(0xBFF63260) // -1074384288

//! Sleep request did not complete within a reasonable amount of time. Solution:
//! Ensure that the link is established before sending a sleep request.
#define nxErrSleepTimeout                               (nxStatus_t)(0xBFF63261) // -1074384287

//! Local port did not wake up within a reasonable amount of time. Solution:
//! Ensure that the link is established before attempting to wake.
#define nxErrWakeTimeout                                (nxStatus_t)(0xBFF63262) // -1074384286

//! Local port configuration does not support sleep. Solution: Configure the
//! port in a way that is sleep-compatible. Refer to NI-XNET help for details.
#define nxErrSleepUnsupportedWithPortConfig             (nxStatus_t)(0xBFF63263) // -1074384285

//! NI-XNET does not support consecutive non-data frame types. Solution: Write
//! only one non-data frame type between data frames. Refer to NI-XNET help for
//! details.
#define nxErrContiguousNonDataFrames                    (nxStatus_t)(0xBFF63264) // -1074384284

//! The transmitted raw frames are not formatted correctly. For more
//! information, refer to "Raw Frame Format" in the NI-XNET help.
#define nxErrInvalidRawFrame                            (nxStatus_t)(0xBFF63265) // -1074384283

//! The environment variable `NIXNET_USE_SYSTEM_LOCALE` must be set to 0 or 1.
//! Refer to NI-XNET help for details.
#define nxErrInvalidLocaleEnvironmentVariableValue      (nxStatus_t)(0xBFF63266) // -1074384282

   // Warning Section

//! The specified CAN FD baud rate exceeds the CiA-certified baud rate for the
//! transceiver. Higher speeds are possible experimentally, but bus errors may
//! be detected or generated during communication. For more information, refer
//! to NI-XNET CAN hardware details in the NI-XNET help.
#define nxWarnFdBaudExceedsCapability                   (nxStatus_t)(0x3FF63040) // 1073098816

//! A warning occurred when importing this database. The NI-XNET Database Editor
//! may show additional information.
#define nxWarnDatabaseImport                            (nxStatus_t)(0x3FF63085) // 1073098885

//! The database file has been imported, but it was not created by the XNET
//! Editor or using the XNET API. Saving the database file with the XNET API or
//! XNET Editor may lose information from the original file.
#define nxWarnDatabaseImportFIBEXNoXNETFile             (nxStatus_t)(0x3FF63086) // 1073098886

//! A warning occurred when importing this database. This file was not created
//! by NI-XNET, so some features may not be supported. The NI-XNET Database
//! Editor may show additional information.
#define nxWarnDatabaseImportFIBEXNoXNETFilePlusWarning  (nxStatus_t)(0x3FF63087) // 1073098887

//! Close Database returns a warning instead of an error when an invalid
//! reference is passed to the function.
#define nxWarnDatabaseBadReference                      (nxStatus_t)(0x3FF63131) // 1073099057

//! You are retrieving signals from a frame that uses advanced PDU
//! configuration. The signal start bit is given relative to the PDU, and it may
//! be different than the start bit relative to the frame.
#define nxWarnAdvancedPDU                               (nxStatus_t)(0x3FF63132) // 1073099058

//! The multiplexer size exceeds 16 bits. This is not supported for Single Point
//! sessions.
#define nxWarnMuxExceeds16Bit                           (nxStatus_t)(0x3FF63133) // 1073099059

/***********************************************************************
                        P R O P E R T Y   I D S
***********************************************************************/

// Class IDs used for encoding of property IDs (nxProp*)
// Also class parameter of function nxdbCreateObject, nxdbDeleteObject, and
// nxdbFindObject
#define nxClass_Database                  (u32)0x00000000   // Database
#define nxClass_Cluster                   (u32)0x00010000   // Cluster
#define nxClass_Frame                     (u32)0x00020000   // Frame
#define nxClass_Signal                    (u32)0x00030000   // Signal
#define nxClass_Subframe                  (u32)0x00040000   // Subframe
#define nxClass_ECU                       (u32)0x00050000   // ECU
#define nxClass_LINSched                  (u32)0x00060000   // LIN Schedule
#define nxClass_LINSchedEntry             (u32)0x00070000   // LIN Schedule Entry
#define nxClass_PDU                       (u32)0x00080000   // PDU
#define nxClass_Session                   (u32)0x00100000   // Session
#define nxClass_System                    (u32)0x00110000   // System
#define nxClass_Device                    (u32)0x00120000   // Device
#define nxClass_Interface                 (u32)0x00130000   // Interface
#define nxClass_Alias                     (u32)0x00140000   // Alias
#define nxClass_IPStack                   (u32)0x00AB0000   // IP Stack
#define nxClass_TcpSocket                 (u32)0x00AC0000   // TCP Socket
#define nxClass_UdpSocket                 (u32)0x00AD0000   // UDP Socket

#define nxClass_Mask                      (u32)0x00FF0000   // mask for object class

//! Datatype IDs used in encoding of property IDs (nxProp*)
#define nxPrptype_u32                     (u32)0x00000000
#define nxPrptype_f64                     (u32)0x01000000
#define nxPrptype_bool                    (u32)0x02000000   // use u8 as datatype (semantic only)
#define nxPrptype_string                  (u32)0x03000000
#define nxPrptype_1Dstring                (u32)0x04000000   // comma-separated list
#define nxPrptype_ref                     (u32)0x05000000   // u32 reference (handle)
#define nxPrptype_1Dref                   (u32)0x06000000   // array of u32 reference
#define nxPrptype_time                    (u32)0x07000000   // nxTimestamp_t
#define nxPrptype_time100ns               (u32)0x07000000   // nxTimestamp100ns_t
#define nxPrptype_1Du32                   (u32)0x08000000   // array of u32 values
#define nxPrptype_u64                     (u32)0x09000000
#define nxPrptype_1Du8                    (u32)0x0A000000   // array of u8 values
#define nxPrptype_1Du16                   (u32)0x0B000000   // array of u16 values
#define nxPrptype_struct                  (u32)0x0C000000   // struct or array-of-struct
#define nxPrptype_time1ns                 (u32)0x0D000000   // nxTimestamp1ns_t
#define nxPrptype_i32                     (u32)0x0E000000

#define nxPrptype_Mask                    (u32)0xFF000000   // mask for nxPrptype

/* PropertyId parameter of nxGetProperty, nxGetPropertySize, nxSetProperty functions. */

// Session:Application Protocol
#define nxPropSession_ApplicationProtocol        ((u32)0x00100091) // u32 --r
// Session:Auto Start?
#define nxPropSession_AutoStart                  ((u32)0x02100001) // bool --rw
// Session:Cluster Name
#define nxPropSession_ClusterName                ((u32)0x0310000A) // string --r
// Session:Database
#define nxPropSession_DatabaseName               ((u32)0x03100002) // string --r
// Session:Ethernet:Filtering:Frame Filter
#define nxPropSession_EnetFrameFilter            ((u32)0x031000F4) // string --rw
// Session:Ethernet:Logging:Error?
#define nxPropSession_EnetLogError               ((u32)0x021000BD) // bool --r
// Session:Ethernet:Logging:Filepath
#define nxPropSession_EnetLogFile                ((u32)0x031000BA) // string --rw
// Session:Ethernet:Logging:Mode
#define nxPropSession_EnetLogMode                ((u32)0x001000B9) // u32 --rw
// Session:Ethernet:Logging:MultipleFiles:EnableMultipleFiles
#define nxPropSession_EnetLogEnableMultipleFiles ((u32)0x021000F9) // bool --rw
// Session:Ethernet:Logging:MultipleFiles:FileSizeThreshold
#define nxPropSession_EnetLogFileSizeThreshold   ((u32)0x091000FA) // u64 --rw
// Session:Ethernet:Logging:Operation
#define nxPropSession_EnetLogOperation           ((u32)0x001000BB) // u32 --rw
// Session:Ethernet:Number of Frames Received
#define nxPropSession_EnetNumFramesReceived      ((u32)0x091000EF) // u64 --r
// Session:Ethernet:Source MAC Address Auto?
#define nxPropSession_EnetSourceMacAddressAuto   ((u32)0x021000A9) // bool --rw
// Session:Interface:64bit Baud Rate
#define nxPropSession_IntfBaudRate64             ((u32)0x09100016) // u64 --rw
// Session:Interface:Adjust Local Time
#define nxPropSession_IntfAdjustLocalTime        ((u32)0x091000AB) // u64 --w
#define nxPropSession_IntfEnetAdjustLocalTime    (nxPropSession_IntfAdjustLocalTime)
// Session:Interface:Baud Rate (32-bit) Superseded by nxPropSession_IntfBaudRate64
#define nxPropSession_IntfBaudRate               ((u32)0x00100016) // u32 --rw
// Session:Interface:Bus Error Frames to Input Stream?
#define nxPropSession_IntfBusErrToInStrm         ((u32)0x02100015) // bool --rw
// Session:Interface:CAN:64bit FD Baud Rate
#define nxPropSession_IntfCanFdBaudRate64        ((u32)0x09100027) // u64 --rw
// Session:Interface:CAN:Disable Protocol Exception Handling
#define nxPropSession_IntfCanDisableProtExceptionHandling ((u32)0x021000A4) // bool --rw
// Session:Interface:CAN:Enable Edge Filter
#define nxPropSession_IntfCanEdgeFilter          ((u32)0x021000A2) // bool --rw
// Session:Interface:CAN:External Transceiver Config
#define nxPropSession_IntfCANExtTcvrConfig       ((u32)0x00100023) // u32 --w
// Session:Interface:CAN:CAN FD Baudrate (32-bit) Superseded by nxPropSession_IntfCanFdBaudRate64
#define nxPropSession_IntfCanFdBaudRate          ((u32)0x00100027) // u32 --rw
// Session:Interface:CAN:FD ISO Mode
#define nxPropSession_IntfCanFdIsoMode           ((u32)0x0010003E) // u32 --rw
// Session:Interface:CAN:I/O Mode
#define nxPropSession_IntfCanIoMode              ((u32)0x00100026) // u32 --r
// Session:Interface:CAN:Listen Only?
#define nxPropSession_IntfCANLstnOnly            ((u32)0x02100022) // bool --rw
// Session:Interface:CAN:Pending Transmit Order
#define nxPropSession_IntfCANPendTxOrder         ((u32)0x00100020) // u32 --w
// Session:Interface:CAN:Single Shot Transmit?
#define nxPropSession_IntfCANSingShot            ((u32)0x02100024) // bool --rw
// Session:Interface:CAN:Termination
#define nxPropSession_IntfCANTerm                ((u32)0x00100025) // u32 --rw
// Session:Interface:CAN:Transceiver State
#define nxPropSession_IntfCANTcvrState           ((u32)0x00100028) // u32 --rw
// Session:Interface:CAN:Transceiver Type
#define nxPropSession_IntfCANTcvrType            ((u32)0x00100029) // u32 --rw
// Session:Interface:CAN:Transmit I/O Mode
#define nxPropSession_IntfCanTxIoMode            ((u32)0x00100039) // u32 --rw
// Session:Interface:CAN:Transmit Pause
#define nxPropSession_IntfCanTransmitPause       ((u32)0x021000A3) // bool --rw
// Session:Interface:Echo Transmit?
#define nxPropSession_IntfEchoTx                 ((u32)0x02100010) // bool --rw
// Session:Interface:Ethernet:Endpoint:Receive Filter
#define nxPropSession_IntfEnetEptReceiveFilter   ((u32)0x0C1000BC) // struct --rw
// Session:Interface:Ethernet:Endpoint:Transmit Bandwidth
#define nxPropSession_IntfEnetEptTransmitBandwidth ((u32)0x091000BF) // u64 --rw
// Session:Interface:Ethernet:IPv4 Address
#define nxPropSession_IntfEnetIpV4Address        ((u32)0x031000EB) // string --r
// Session:Interface:Ethernet:Jumbo Frames
#define nxPropSession_IntfEnetJumboFrames        ((u32)0x001000F3) // u32 --r
// Session:Interface:Ethernet:Link Speed
#define nxPropSession_IntfEnetLinkSpeed          ((u32)0x001000F2) // u32 --r
// Session:Interface:Ethernet:Link Speed Configured
#define nxPropSession_IntfEnetLinkSpeedConfigured ((u32)0x001000F1) // u32 --r
// Session:Interface:Ethernet:MAC Address
#define nxPropSession_IntfEnetMacAddress         ((u32)0x031000A8) // string --r
// Session:Interface:Ethernet:OS Network Adapter Description
#define nxPropSession_IntfEnetOsNetworkAdapterDescription ((u32)0x031000ED) // string --r
// Session:Interface:Ethernet:OS Network Adapter Name
#define nxPropSession_IntfEnetOsNetworkAdapterName ((u32)0x031000EC) // string --r
// Session:Interface:Ethernet:Operational Status
#define nxPropSession_IntfEnetOperationalStatus  ((u32)0x001000AA) // u32 --r
// Session:Interface:Ethernet:Output Stream Timescale
#define nxPropSession_IntfEnetOutStrmTimescale   ((u32)0x001000F7) // u32 --rw
// Session:Interface:Ethernet:PHY Power Mode
#define nxPropSession_IntfEnetPhyPowerMode       ((u32)0x001000F9) // u32 --r
// Session:Interface:Ethernet:PHY State
#define nxPropSession_IntfEnetPhyState           ((u32)0x001000A7) // u32 --r
// Session:Interface:Ethernet:Port Mode
#define nxPropSession_IntfEnetPortMode           ((u32)0x001000A6) // u32 --r
// Session:Interface:Ethernet:Signal Quality
#define nxPropSession_IntfEnetSignalQuality      ((u32)0x0E1000F6) // i32 --r
// Session:Interface:Ethernet:Sleep Capability
#define nxPropSession_IntfEnetSleepCapability    ((u32)0x001000F8) // u32 --r
// Session:Interface:Ethernet:Statistics:MAC:Counter Names
#define nxPropSession_IntfEnetStatsCounterNames  ((u32)0x041000AC) // 1Dstring --r
// Session:Interface:Ethernet:Statistics:MAC:Counter Values
#define nxPropSession_IntfEnetStatsCounterValues ((u32)0x041000AD) // 1Dstring --r
// Session:Interface:Ethernet:Statistics:MAC:Rx Bad Frames Count
#define nxPropSession_IntfEnetStatsRxBadFrames   ((u32)0x091000B6) // u64 --r
// Session:Interface:Ethernet:Statistics:MAC:Rx Bytes Count
#define nxPropSession_IntfEnetStatsRxBytes       ((u32)0x091000AE) // u64 --r
// Session:Interface:Ethernet:Statistics:MAC:Rx Good Frames Count
#define nxPropSession_IntfEnetStatsRxGoodFrames  ((u32)0x091000AF) // u64 --r
// Session:Interface:Ethernet:Statistics:MAC:Tx Bytes Count
#define nxPropSession_IntfEnetStatsTxBytes       ((u32)0x091000B7) // u64 --r
// Session:Interface:Ethernet:Statistics:MAC:Tx Good Frames Count
#define nxPropSession_IntfEnetStatsTxGoodFrames  ((u32)0x091000B8) // u64 --r
// Session:Interface:Ethernet:Statistics:PHY:Counter Names
#define nxPropSession_IntfEnetStatsPhyCounterNames ((u32)0x041000FD) // 1Dstring --r
// Session:Interface:Ethernet:Statistics:PHY:Counter Values
#define nxPropSession_IntfEnetStatsPhyCounterValues ((u32)0x041000FE) // 1Dstring --r
// Session:Interface:Ethernet:Statistics:PHY:Low Power Sleep Count
#define nxPropSession_IntfEnetStatsPhyLowPowerSleep ((u32)0x09100101) // u64 --r
// Session:Interface:Ethernet:Statistics:PHY:Sleep Failure Count
#define nxPropSession_IntfEnetStatsPhySleepFailure ((u32)0x09100103) // u64 --r
// Session:Interface:Ethernet:Statistics:PHY:Wake Up Failure Count
#define nxPropSession_IntfEnetStatsPhyWakeupFailure ((u32)0x09100102) // u64 --r
// Session:Interface:Ethernet:Statistics:PHY:Wake Up Pulse Count
#define nxPropSession_IntfEnetStatsPhyWakeupPulse ((u32)0x091000FF) // u64 --r
// Session:Interface:Ethernet:Statistics:PHY:Wake Up Request Count
#define nxPropSession_IntfEnetStatsPhyWakeupRequest ((u32)0x09100100) // u64 --r
// Session:Interface:Ethernet:Time Sync:Adjust Network Time
#define nxPropSession_IntfEnetTimeAdjustNetworkTime ((u32)0x091000D1) // u64 --w
// Session:Interface:Ethernet:Time Sync:BMCA Enabled?
#define nxPropSession_IntfEnetTimeBMCAEnabled    ((u32)0x021000C2) // bool --rw
// Session:Interface:Ethernet:Time Sync:Clock Accuracy
#define nxPropSession_IntfEnetTimeClkAccuracy    ((u32)0x001000C6) // u32 --r
// Session:Interface:Ethernet:Time Sync:Clock Class
#define nxPropSession_IntfEnetTimeClkClass       ((u32)0x001000C5) // u32 --r
// Session:Interface:Ethernet:Time Sync:Clock ID
#define nxPropSession_IntfEnetTimeClkID          ((u32)0x031000C4) // string --r
// Session:Interface:Ethernet:Time Sync:Clock Offset Scaled Log Variance
#define nxPropSession_IntfEnetTimeClkOffsetVar   ((u32)0x001000C7) // u32 --r
// Session:Interface:Ethernet:Time Sync:Grandmaster Clock Accuracy
#define nxPropSession_IntfEnetTimeGMClkAccuracy  ((u32)0x001000CD) // u32 --r
// Session:Interface:Ethernet:Time Sync:Grandmaster Clock Class
#define nxPropSession_IntfEnetTimeGMClkClass     ((u32)0x001000CC) // u32 --r
// Session:Interface:Ethernet:Time Sync:Grandmaster Clock ID
#define nxPropSession_IntfEnetTimeGMClkID        ((u32)0x031000CB) // string --r
// Session:Interface:Ethernet:Time Sync:Grandmaster Clock Offset Scaled Log Variance
#define nxPropSession_IntfEnetTimeGMClkOffsetVar ((u32)0x001000CE) // u32 --r
// Session:Interface:Ethernet:Time Sync:Grandmaster Priority1
#define nxPropSession_IntfEnetTimeGMPriority1    ((u32)0x001000CF) // u32 --r
// Session:Interface:Ethernet:Time Sync:Grandmaster Priority2
#define nxPropSession_IntfEnetTimeGMPriority2    ((u32)0x001000D0) // u32 --r
// Session:Interface:Ethernet:Time Sync:Offset From Master
#define nxPropSession_IntfEnetTimeOffsetFromMaster ((u32)0x091000C3) // u64 --r
// Session:Interface:Ethernet:Time Sync:Port:AS Capable?
#define nxPropSession_IntfEnetTimePortASCapable  ((u32)0x021000E1) // bool --r
// Session:Interface:Ethernet:Time Sync:Port:Announce Receipt Timeout
#define nxPropSession_IntfEnetTimePortAnnounceReceiptTimeout ((u32)0x001000E0) // u32 --rw
// Session:Interface:Ethernet:Time Sync:Port:Announce Transmit Enabled?
#define nxPropSession_IntfEnetTimePortAnnounceTransmitEnabled ((u32)0x021000DF) // bool --rw
// Session:Interface:Ethernet:Time Sync:Port:Log Announce Interval
#define nxPropSession_IntfEnetTimePortLogAnnounceInterval ((u32)0x001000DE) // u32 --r
// Session:Interface:Ethernet:Time Sync:Port:Log Announce Interval Configured
#define nxPropSession_IntfEnetTimePortLogAnnounceIntervalConfigured ((u32)0x001000DD) // u32 --rw
// Session:Interface:Ethernet:Time Sync:Port:Log Pdelay_Req Interval
#define nxPropSession_IntfEnetTimePortLogPdelayInterval ((u32)0x001000D9) // u32 --r
// Session:Interface:Ethernet:Time Sync:Port:Log Pdelay_Req Interval Configured
#define nxPropSession_IntfEnetTimePortLogPdelayIntervalConfigured ((u32)0x001000D8) // u32 --rw
// Session:Interface:Ethernet:Time Sync:Port:Log Sync Interval
#define nxPropSession_IntfEnetTimePortLogSyncInterval ((u32)0x001000DB) // u32 --r
// Session:Interface:Ethernet:Time Sync:Port:Log Sync Interval Configured
#define nxPropSession_IntfEnetTimePortLogSyncIntervalConfigured ((u32)0x001000DA) // u32 --rw
// Session:Interface:Ethernet:Time Sync:Port:Pdelay Enabled?
#define nxPropSession_IntfEnetTimePortPdelayEnabled ((u32)0x021000D7) // bool --rw
// Session:Interface:Ethernet:Time Sync:Port:Port State
#define nxPropSession_IntfEnetTimePortState      ((u32)0x001000D3) // u32 --r
// Session:Interface:Ethernet:Time Sync:Port:Port State Configured
#define nxPropSession_IntfEnetTimePortStateConfigured ((u32)0x001000D2) // u32 --rw
// Session:Interface:Ethernet:Time Sync:Port:Propagation Delay
#define nxPropSession_IntfEnetTimePortPropDelay  ((u32)0x011000D4) // f64 --r
// Session:Interface:Ethernet:Time Sync:Port:Propagation Delay Configured
#define nxPropSession_IntfEnetTimePortPropDelayConfigured ((u32)0x011000D5) // f64 --rw
// Session:Interface:Ethernet:Time Sync:Port:Propagation Delay Threshold
#define nxPropSession_IntfEnetTimePortPropDelayThreshold ((u32)0x011000D6) // f64 --rw
// Session:Interface:Ethernet:Time Sync:Port:Statistics:Counter Names
#define nxPropSession_IntfEnetTimePortStatsCounterNames ((u32)0x041000E3) // 1Dstring --r
// Session:Interface:Ethernet:Time Sync:Port:Statistics:Counter Values
#define nxPropSession_IntfEnetTimePortStatsCounterValues ((u32)0x041000E4) // 1Dstring --r
// Session:Interface:Ethernet:Time Sync:Port:Statistics:Rx Announce Count
#define nxPropSession_IntfEnetTimePortStatsRxAnnounce ((u32)0x091000E6) // u64 --r
// Session:Interface:Ethernet:Time Sync:Port:Statistics:Rx Pdelay Request Count
#define nxPropSession_IntfEnetTimePortStatsRxPdelayRequest ((u32)0x091000E7) // u64 --r
// Session:Interface:Ethernet:Time Sync:Port:Statistics:Rx Sync Count
#define nxPropSession_IntfEnetTimePortStatsRxSync ((u32)0x091000E5) // u64 --r
// Session:Interface:Ethernet:Time Sync:Port:Statistics:Tx Announce Count
#define nxPropSession_IntfEnetTimePortStatsTxAnnounce ((u32)0x091000E9) // u64 --r
// Session:Interface:Ethernet:Time Sync:Port:Statistics:Tx Pdelay Request Count
#define nxPropSession_IntfEnetTimePortStatsTxPdelayRequest ((u32)0x091000EA) // u64 --r
// Session:Interface:Ethernet:Time Sync:Port:Statistics:Tx Sync Count
#define nxPropSession_IntfEnetTimePortStatsTxSync ((u32)0x091000E8) // u64 --r
// Session:Interface:Ethernet:Time Sync:Port:Sync Receipt Timeout
#define nxPropSession_IntfEnetTimePortSyncReceiptTimeout ((u32)0x001000DC) // u32 --rw
// Session:Interface:Ethernet:Time Sync:Port:Sync Status
#define nxPropSession_IntfEnetTimePortSyncStatus ((u32)0x001000F0) // u32 --r
// Session:Interface:Ethernet:Time Sync:Port:Synced?
#define nxPropSession_IntfEnetTimePortSynced     ((u32)0x021000E2) // bool --r
// Session:Interface:Ethernet:Time Sync:Priority1
#define nxPropSession_IntfEnetTimePriority1      ((u32)0x001000C8) // u32 --rw
// Session:Interface:Ethernet:Time Sync:Priority2
#define nxPropSession_IntfEnetTimePriority2      ((u32)0x001000C9) // u32 --rw
// Session:Interface:Ethernet:Time Sync:Protocol
#define nxPropSession_IntfEnetTimeProtocol       ((u32)0x001000C0) // u32 --rw
// Session:Interface:Ethernet:Time Sync:Protocol Enabled?
#define nxPropSession_IntfEnetTimeProtocolEnabled ((u32)0x021000C1) // bool --rw
// Session:Interface:Ethernet:Time Sync:Steps to Grandmaster
#define nxPropSession_IntfEnetTimeStepsToGM      ((u32)0x001000CA) // u32 --r
// Session:Interface:Ethernet:Trigger PPS Synced?
#define nxPropSession_IntfEnetTriggerPpsSynced   ((u32)0x02100104) // bool --r
// Session:Interface:FlexRay:Accepted Startup Range
#define nxPropSession_IntfFlexRayAccStartRng     ((u32)0x00100030) // u32 --rw
// Session:Interface:FlexRay:Allow Halt Due To Clock?
#define nxPropSession_IntfFlexRayAlwHltClk       ((u32)0x02100031) // bool --rw
// Session:Interface:FlexRay:Allow Passive to Active
#define nxPropSession_IntfFlexRayAlwPassAct      ((u32)0x00100032) // u32 --rw
// Session:Interface:FlexRay:Auto Asleep When Stopped?
#define nxPropSession_IntfFlexRayAutoAslpWhnStp  ((u32)0x0210003A) // bool --rw
// Session:Interface:FlexRay:Cluster Drift Damping
#define nxPropSession_IntfFlexRayClstDriftDmp    ((u32)0x00100033) // u32 --rw
// Session:Interface:FlexRay:Coldstart?
#define nxPropSession_IntfFlexRayColdstart       ((u32)0x02100034) // bool --r
// Session:Interface:FlexRay:Connected Channels
#define nxPropSession_IntfFlexRayConnectedChs    ((u32)0x0010003C) // u32 --rw
// Session:Interface:FlexRay:Decoding Correction
#define nxPropSession_IntfFlexRayDecCorr         ((u32)0x00100035) // u32 --rw
// Session:Interface:FlexRay:Delay Compensation Ch A
#define nxPropSession_IntfFlexRayDelayCompA      ((u32)0x00100036) // u32 --rw
// Session:Interface:FlexRay:Delay Compensation Ch B
#define nxPropSession_IntfFlexRayDelayCompB      ((u32)0x00100037) // u32 --rw
// Session:Interface:FlexRay:Key Slot Identifier
#define nxPropSession_IntfFlexRayKeySlotID       ((u32)0x00100038) // u32 --rw
// Session:Interface:FlexRay:Latest Tx
#define nxPropSession_IntfFlexRayLatestTx        ((u32)0x00100041) // u32 --r
// Session:Interface:FlexRay:Listen Timeout
#define nxPropSession_IntfFlexRayListTimo        ((u32)0x00100042) // u32 --rw
// Session:Interface:FlexRay:Macro Initial Offset Ch A
#define nxPropSession_IntfFlexRayMacInitOffA     ((u32)0x00100043) // u32 --rw
// Session:Interface:FlexRay:Macro Initial Offset Ch B
#define nxPropSession_IntfFlexRayMacInitOffB     ((u32)0x00100044) // u32 --rw
// Session:Interface:FlexRay:Max Drift
#define nxPropSession_IntfFlexRayMaxDrift        ((u32)0x00100047) // u32 --rw
// Session:Interface:FlexRay:Micro Initial Offset Ch A
#define nxPropSession_IntfFlexRayMicInitOffA     ((u32)0x00100045) // u32 --rw
// Session:Interface:FlexRay:Micro Initial Offset Ch B
#define nxPropSession_IntfFlexRayMicInitOffB     ((u32)0x00100046) // u32 --rw
// Session:Interface:FlexRay:Microtick
#define nxPropSession_IntfFlexRayMicrotick       ((u32)0x00100048) // u32 --r
// Session:Interface:FlexRay:Null Frames To Input Stream?
#define nxPropSession_IntfFlexRayNullToInStrm    ((u32)0x02100049) // bool --rw
// Session:Interface:FlexRay:Offset Correction
#define nxPropSession_IntfFlexRayOffCorr         ((u32)0x00100058) // u32 --r
// Session:Interface:FlexRay:Offset Correction Out
#define nxPropSession_IntfFlexRayOffCorrOut      ((u32)0x00100050) // u32 --rw
// Session:Interface:FlexRay:Rate Correction
#define nxPropSession_IntfFlexRayRateCorr        ((u32)0x00100059) // u32 --r
// Session:Interface:FlexRay:Rate Correction Out
#define nxPropSession_IntfFlexRayRateCorrOut     ((u32)0x00100052) // u32 --rw
// Session:Interface:FlexRay:Samples Per Microtick
#define nxPropSession_IntfFlexRaySampPerMicro    ((u32)0x00100053) // u32 --rw
// Session:Interface:FlexRay:Single Slot Enabled?
#define nxPropSession_IntfFlexRaySingSlotEn      ((u32)0x02100054) // bool --rw
// Session:Interface:FlexRay:Sleep
#define nxPropSession_IntfFlexRaySleep           ((u32)0x0010003B) // u32 --w
// Session:Interface:FlexRay:Statistics Enabled?
#define nxPropSession_IntfFlexRayStatisticsEn    ((u32)0x0210005A) // bool --rw
// Session:Interface:FlexRay:Symbol Frames To Input Stream?
#define nxPropSession_IntfFlexRaySymToInStrm     ((u32)0x0210003D) // bool --rw
// Session:Interface:FlexRay:Sync Frame Status
#define nxPropSession_IntfFlexRaySyncStatus      ((u32)0x0010005F) // u32 --r
// Session:Interface:FlexRay:Sync Frames Channel A Even
#define nxPropSession_IntfFlexRaySyncChAEven     ((u32)0x0810005B) // 1Du32 --r
// Session:Interface:FlexRay:Sync Frames Channel A Odd
#define nxPropSession_IntfFlexRaySyncChAOdd      ((u32)0x0810005C) // 1Du32 --r
// Session:Interface:FlexRay:Sync Frames Channel B Even
#define nxPropSession_IntfFlexRaySyncChBEven     ((u32)0x0810005D) // 1Du32 --r
// Session:Interface:FlexRay:Sync Frames Channel B Odd
#define nxPropSession_IntfFlexRaySyncChBOdd      ((u32)0x0810005E) // 1Du32 --r
// Session:Interface:FlexRay:Termination
#define nxPropSession_IntfFlexRayTerm            ((u32)0x00100057) // u32 --rw
// Session:Interface:FlexRay:Wakeup Channel
#define nxPropSession_IntfFlexRayWakeupCh        ((u32)0x00100055) // u32 --rw
// Session:Interface:FlexRay:Wakeup Pattern
#define nxPropSession_IntfFlexRayWakeupPtrn      ((u32)0x00100056) // u32 --rw
// Session:Interface:I/O Name
#define nxPropSession_IntfName                   ((u32)0x03100013) // string --r
// Session:Interface:LIN:Break Delimiter Length
#define nxPropSession_IntfLINBreakDelimiterLength ((u32)0x001000F5) // u32 --rw
// Session:Interface:LIN:Break Length
#define nxPropSession_IntfLINBreakLength         ((u32)0x00100070) // u32 --rw
// Session:Interface:LIN:Checksum to Input Stream?
#define nxPropSession_IntfLINChecksumToInStrm    ((u32)0x02100081) // bool --rw
// Session:Interface:LIN:Diagnostics P2min
#define nxPropSession_IntfLINDiagP2min           ((u32)0x01100077) // f64 --rw
// Session:Interface:LIN:Diagnostics STmin
#define nxPropSession_IntfLINDiagSTmin           ((u32)0x01100076) // f64 --rw
// Session:Interface:LIN:Master?
#define nxPropSession_IntfLINMaster              ((u32)0x02100072) // bool --rw
// Session:Interface:LIN:No Response Frames to Input Stream?
#define nxPropSession_IntfLINNoResponseToInStrm  ((u32)0x02100080) // bool --rw
// Session:Interface:LIN:Output Stream Slave Response List By NAD
#define nxPropSession_IntfLINOStrSlvRspLstByNAD  ((u32)0x08100079) // 1Du32 --rw
// Session:Interface:LIN:Schedule Names
#define nxPropSession_IntfLINSchedNames          ((u32)0x04100075) // 1Dstring --r
// Session:Interface:LIN:Sleep
#define nxPropSession_IntfLINSleep               ((u32)0x00100073) // u32 --w
// Session:Interface:LIN:Start Allowed without Bus Power?
#define nxPropSession_IntfLINAlwStartWoBusPwr    ((u32)0x02100078) // bool --rw
// Session:Interface:LIN:Termination
#define nxPropSession_IntfLINTerm                ((u32)0x00100074) // u32 --rw
// Session:Interface:Output Stream List
#define nxPropSession_IntfOutStrmList            ((u32)0x06100011) // 1Dref --rw
// Session:Interface:Output Stream List By ID
#define nxPropSession_IntfOutStrmListById        ((u32)0x08100021) // 1Du32 --rw
// Session:Interface:Output Stream Timing
#define nxPropSession_IntfOutStrmTimng           ((u32)0x00100012) // u32 --rw
// Session:Interface:Source Terminal:Start Trigger
#define nxPropSession_IntfSrcTermStartTrigger    ((u32)0x03100090) // string --rw
// Session:Interface:Start Trigger Frames to Input Stream?
#define nxPropSession_IntfStartTrigToInStrm      ((u32)0x02100014) // bool --rw
// Session:List
#define nxPropSession_List                       ((u32)0x04100003) // 1Dstring --r
// Session:Mode
#define nxPropSession_Mode                       ((u32)0x00100004) // u32 --r
// Session:Number in List
#define nxPropSession_NumInList                  ((u32)0x00100005) // u32 --r
// Session:Number of Frames
#define nxPropSession_NumFrames                  ((u32)0x0010000D) // u32 --r
// Session:Number of Values Pending
#define nxPropSession_NumPend                    ((u32)0x00100006) // u32 --r
// Session:Number of Values Unused
#define nxPropSession_NumUnused                  ((u32)0x0010000B) // u32 --r
// Session:Payload Length Maximum
#define nxPropSession_PayldLenMax                ((u32)0x00100009) // u32 --r
// Session:Protocol
#define nxPropSession_Protocol                   ((u32)0x00100008) // u32 --r
// Session:Queue Size
#define nxPropSession_QueueSize                  ((u32)0x0010000C) // u32 --rw
// Session:Resample Rate
#define nxPropSession_ResampRate                 ((u32)0x01100007) // f64 --rw
// Session:SAE J1939:ECU
#define nxPropSession_J1939ECU                   ((u32)0x05100093) // ref --w
// Session:SAE J1939:ECU Busy
#define nxPropSession_J1939ECUBusy               ((u32)0x021000A1) // bool --rw
// Session:SAE J1939:Fill Byte
#define nxPropSession_J1939FillByte              ((u32)0x0010009F) // u32 --rw
// Session:SAE J1939:Include Destination Address in PGN
#define nxPropSession_J1939IncludeDestAddrInPGN  ((u32)0x021000A5) // bool --rw
// Session:SAE J1939:Maximum Repeat CTS
#define nxPropSession_J1939MaxRepeatCTS          ((u32)0x0010009E) // u32 --rw
// Session:SAE J1939:Node Address
#define nxPropSession_J1939Address               ((u32)0x00100092) // u32 --rw
// Session:SAE J1939:Node Name
#define nxPropSession_J1939Name                  ((u32)0x09100094) // u64 --rw
// Session:SAE J1939:Number of Packets Received
#define nxPropSession_J1939NumPacketsRecv        ((u32)0x0010009C) // u32 --rw
// Session:SAE J1939:Number of Packets Response
#define nxPropSession_J1939NumPacketsResp        ((u32)0x0010009D) // u32 --rw
// Session:SAE J1939:Timing:Hold Time Th
#define nxPropSession_J1939HoldTimeTh            ((u32)0x0110009B) // f64 --rw
// Session:SAE J1939:Timing:Response Time Tr_GD
#define nxPropSession_J1939ResponseTimeTrGD      ((u32)0x0110009A) // f64 --rw
// Session:SAE J1939:Timing:Response Time Tr_SD
#define nxPropSession_J1939ResponseTimeTrSD      ((u32)0x01100099) // f64 --rw
// Session:SAE J1939:Timing:Timeout T1
#define nxPropSession_J1939TimeoutT1             ((u32)0x01100095) // f64 --rw
// Session:SAE J1939:Timing:Timeout T2
#define nxPropSession_J1939TimeoutT2             ((u32)0x01100096) // f64 --rw
// Session:SAE J1939:Timing:Timeout T3
#define nxPropSession_J1939TimeoutT3             ((u32)0x01100097) // f64 --rw
// Session:SAE J1939:Timing:Timeout T4
#define nxPropSession_J1939TimeoutT4             ((u32)0x01100098) // f64 --rw
// Session:SAE J1939:Write Queue Size
#define nxPropSession_J1939WriteQueueSize        ((u32)0x001000A0) // u32 --rw
// System:CompactDAQ Packet Time
#define nxPropSys_CDAQPktTime                    ((u32)0x0111000C) // f64 --rw
// System:Devices
#define nxPropSys_DevRefs                        ((u32)0x06110002) // 1Dref --r
// System:Interfaces
#define nxPropSys_IntfRefs                       ((u32)0x06110003) // 1Dref --r
// System:Interfaces (All)
#define nxPropSys_IntfRefsAll                    ((u32)0x0611000D) // 1Dref --r
// System:Interfaces (CAN)
#define nxPropSys_IntfRefsCAN                    ((u32)0x06110004) // 1Dref --r
// System:Interfaces (Ethernet)
#define nxPropSys_IntfRefsEthernet               ((u32)0x06110001) // 1Dref --r
// System:Interfaces (FlexRay)
#define nxPropSys_IntfRefsFlexRay                ((u32)0x06110005) // 1Dref --r
// System:Interfaces (LIN)
#define nxPropSys_IntfRefsLIN                    ((u32)0x06110007) // 1Dref --r
// System:Version:Build
#define nxPropSys_VerBuild                       ((u32)0x00110006) // u32 --r
// System:Version:Major
#define nxPropSys_VerMajor                       ((u32)0x00110008) // u32 --r
// System:Version:Minor
#define nxPropSys_VerMinor                       ((u32)0x00110009) // u32 --r
// System:Version:Phase
#define nxPropSys_VerPhase                       ((u32)0x0011000A) // u32 --r
// System:Version:Update
#define nxPropSys_VerUpdate                      ((u32)0x0011000B) // u32 --r
// Device:Form Factor
#define nxPropDev_FormFac                        ((u32)0x00120001) // u32 --r
// Device:Interfaces
#define nxPropDev_IntfRefs                       ((u32)0x06120002) // 1Dref --r
// Device:Interfaces (All)
#define nxPropDev_IntfRefsAll                    ((u32)0x06120008) // 1Dref --r
// Device:Number of Ports
#define nxPropDev_NumPorts                       ((u32)0x00120004) // u32 --r
// Device:Number of Ports (All)
#define nxPropDev_NumPortsAll                    ((u32)0x00120007) // u32 --r
// Device:Product Name
#define nxPropDev_Name                           ((u32)0x03120003) // string --r
// Device:Product Number
#define nxPropDev_ProductNum                     ((u32)0x00120008) // u32 --r
// Device:Serial Number
#define nxPropDev_SerNum                         ((u32)0x00120005) // u32 --r
// Device:Slot Number
#define nxPropDev_SlotNum                        ((u32)0x00120006) // u32 --r
// Interface:CAN:Termination Capability
#define nxPropIntf_CANTermCap                    ((u32)0x00130008) // u32 --r
// Interface:CAN:Transceiver Capability
#define nxPropIntf_CANTcvrCap                    ((u32)0x00130007) // u32 --r
// Interface:Device
#define nxPropIntf_DevRef                        ((u32)0x05130001) // ref --r
// Interface:Dongle Compatible Firmware Version
#define nxPropIntf_DongleCompatibleFirmwareVersion ((u32)0x0013000F) // u32 --r
// Interface:Dongle Compatible Revision
#define nxPropIntf_DongleCompatibleRevision      ((u32)0x0013000E) // u32 --r
// Interface:Dongle Firmware Version
#define nxPropIntf_DongleFirmwareVersion         ((u32)0x0013000D) // u32 --r
// Interface:Dongle ID
#define nxPropIntf_DongleID                      ((u32)0x0013000A) // u32 --r
// Interface:Dongle Revision
#define nxPropIntf_DongleRevision                ((u32)0x0013000C) // u32 --r
// Interface:Dongle State
#define nxPropIntf_DongleState                   ((u32)0x00130009) // u32 --r
// Interface:Name
#define nxPropIntf_Name                          ((u32)0x03130002) // string --r
// Interface:Number
#define nxPropIntf_Num                           ((u32)0x00130003) // u32 --r
// Interface:Port Number
#define nxPropIntf_PortNum                       ((u32)0x00130004) // u32 --r
// Interface:Protocol
#define nxPropIntf_Protocol                      ((u32)0x00130005) // u32 --r

/* PropertyId parameter of nxGetSubProperty, nxGetSubPropertySize, nxSetSubProperty functions. */

// Session:Frame:CAN:Start Time Offset
#define nxPropSessionSub_CANStartTimeOff         ((u32)0x01100081) // f64 --w
// Session:Frame:CAN:Transmit Time
#define nxPropSessionSub_CANTxTime               ((u32)0x01100082) // f64 --w
// Session:Frame:LIN:Transmit N Corrupted Checksums
#define nxPropSessionSub_LINTxNCorruptedChksums  ((u32)0x00100085) // u32 --w
// Session:Frame:Output Queue Update Frequency
#define nxPropSessionSub_OutputQueueUpdateFreq   ((u32)0x00100084) // u32 --w
// Session:Frame:SAE J1939:Address Filter
#define nxPropSessionSub_J1939_AddrFilter        ((u32)0x03100086) // string --w
// Session:Frame:Skip N Cyclic Frames
#define nxPropSessionSub_SkipNCyclicFrames       ((u32)0x00100083) // u32 --w

/* PropertyId parameter of nxdbGetProperty, nxdbGetPropertySize, nxdbSetProperty functions. */

// Database:Clusters
#define nxPropDatabase_ClstRefs                  ((u32)0x06000002) // 1Dref --r
// Database:Name
#define nxPropDatabase_Name                      ((u32)0x03000001) // string --r
// Database:Show Invalid From Open?
#define nxPropDatabase_ShowInvalidFromOpen       ((u32)0x02000003) // bool --rw
// Cluster:64bit Baud Rate
#define nxPropClst_BaudRate64                    ((u32)0x09010001) // u64 --rw
// Cluster:Application Protocol
#define nxPropClst_ApplicationProtocol           ((u32)0x00010073) // u32 --rw
// Cluster:Baud Rate (32-bit) Superseded by nxPropClst_BaudRate64
#define nxPropClst_BaudRate                      ((u32)0x00010001) // u32 --rw
// Cluster:CAN:64bit FD Baud Rate
#define nxPropClst_CanFdBaudRate64               ((u32)0x09010011) // u64 --rw
// Cluster:CAN:FD Baudrate (32-bit) Superseded by nxPropClst_CanFdBaudRate64
#define nxPropClst_CanFdBaudRate                 ((u32)0x00010011) // u32 --rw
// Cluster:CAN:FD ISO Mode
#define nxPropClst_CanFdIsoMode                  ((u32)0x00010074) // u32 --r
// Cluster:CAN:I/O Mode
#define nxPropClst_CanIoMode                     ((u32)0x00010010) // u32 --rw
// Cluster:Comment
#define nxPropClst_Comment                       ((u32)0x03010008) // string --rw
// Cluster:Configuration Status
#define nxPropClst_ConfigStatus                  ((u32)0x00010009) // u32 --r
// Cluster:Database
#define nxPropClst_DatabaseRef                   ((u32)0x05010002) // ref --r
// Cluster:ECUs
#define nxPropClst_ECURefs                       ((u32)0x06010003) // 1Dref --r
// Cluster:FlexRay:Action Point Offset
#define nxPropClst_FlexRayActPtOff               ((u32)0x00010020) // u32 --rw
// Cluster:FlexRay:Allow Passive to Active
#define nxPropClst_FlexRayAlwPassAct             ((u32)0x00010072) // u32 --rw
// Cluster:FlexRay:CAS Rx Low Max
#define nxPropClst_FlexRayCASRxLMax              ((u32)0x00010021) // u32 --rw
// Cluster:FlexRay:Channels
#define nxPropClst_FlexRayChannels               ((u32)0x00010022) // u32 --rw
// Cluster:FlexRay:Cluster Drift Damping
#define nxPropClst_FlexRayClstDriftDmp           ((u32)0x00010023) // u32 --rw
// Cluster:FlexRay:Cold Start Attempts
#define nxPropClst_FlexRayColdStAts              ((u32)0x00010024) // u32 --rw
// Cluster:FlexRay:Cycle
#define nxPropClst_FlexRayCycle                  ((u32)0x00010025) // u32 --rw
// Cluster:FlexRay:Dynamic Segment Start
#define nxPropClst_FlexRayDynSegStart            ((u32)0x00010026) // u32 --r
// Cluster:FlexRay:Dynamic Slot Idle Phase
#define nxPropClst_FlexRayDynSlotIdlPh           ((u32)0x00010027) // u32 --rw
// Cluster:FlexRay:Latest Guaranteed Dynamic Slot
#define nxPropClst_FlexRayLatestGuarDyn          ((u32)0x0001002B) // u32 --r
// Cluster:FlexRay:Latest Usable Dynamic Slot
#define nxPropClst_FlexRayLatestUsableDyn        ((u32)0x0001002A) // u32 --r
// Cluster:FlexRay:Listen Noise
#define nxPropClst_FlexRayLisNoise               ((u32)0x00010028) // u32 --rw
// Cluster:FlexRay:Macro Per Cycle
#define nxPropClst_FlexRayMacroPerCycle          ((u32)0x00010029) // u32 --rw
// Cluster:FlexRay:Macrotick
#define nxPropClst_FlexRayMacrotick              ((u32)0x01010030) // f64 --r
// Cluster:FlexRay:Max Without Clock Correction Fatal
#define nxPropClst_FlexRayMaxWoClkCorFat         ((u32)0x00010031) // u32 --rw
// Cluster:FlexRay:Max Without Clock Correction Passive
#define nxPropClst_FlexRayMaxWoClkCorPas         ((u32)0x00010032) // u32 --rw
// Cluster:FlexRay:Minislot
#define nxPropClst_FlexRayMinislot               ((u32)0x00010034) // u32 --rw
// Cluster:FlexRay:Minislot Action Point Offset
#define nxPropClst_FlexRayMinislotActPt          ((u32)0x00010033) // u32 --rw
// Cluster:FlexRay:NIT
#define nxPropClst_FlexRayNIT                    ((u32)0x00010036) // u32 --rw
// Cluster:FlexRay:NIT Start
#define nxPropClst_FlexRayNITStart               ((u32)0x00010037) // u32 --r
// Cluster:FlexRay:Network Management Vector Length
#define nxPropClst_FlexRayNMVecLen               ((u32)0x00010035) // u32 --rw
// Cluster:FlexRay:Number Of Minislots
#define nxPropClst_FlexRayNumMinislt             ((u32)0x00010038) // u32 --rw
// Cluster:FlexRay:Number Of Static Slots
#define nxPropClst_FlexRayNumStatSlt             ((u32)0x00010039) // u32 --rw
// Cluster:FlexRay:Offset Correction Start
#define nxPropClst_FlexRayOffCorSt               ((u32)0x00010040) // u32 --rw
// Cluster:FlexRay:Payload Length Dynamic Maximum
#define nxPropClst_FlexRayPayldLenDynMax         ((u32)0x00010041) // u32 --rw
// Cluster:FlexRay:Payload Length Maximum
#define nxPropClst_FlexRayPayldLenMax            ((u32)0x00010042) // u32 --r
// Cluster:FlexRay:Payload Length Static
#define nxPropClst_FlexRayPayldLenSt             ((u32)0x00010043) // u32 --rw
// Cluster:FlexRay:Static Slot
#define nxPropClst_FlexRayStatSlot               ((u32)0x00010045) // u32 --rw
// Cluster:FlexRay:Symbol Window
#define nxPropClst_FlexRaySymWin                 ((u32)0x00010046) // u32 --rw
// Cluster:FlexRay:Symbol Window Start
#define nxPropClst_FlexRaySymWinStart            ((u32)0x00010047) // u32 --r
// Cluster:FlexRay:Sync Node Max
#define nxPropClst_FlexRaySyncNodeMax            ((u32)0x00010048) // u32 --rw
// Cluster:FlexRay:TSS Transmitter
#define nxPropClst_FlexRayTSSTx                  ((u32)0x00010049) // u32 --rw
// Cluster:FlexRay:Use Wakeup?
#define nxPropClst_FlexRayUseWakeup              ((u32)0x02010055) // bool --rw
// Cluster:FlexRay:Wakeup Symbol Rx Idle
#define nxPropClst_FlexRayWakeSymRxIdl           ((u32)0x00010050) // u32 --rw
// Cluster:FlexRay:Wakeup Symbol Rx Low
#define nxPropClst_FlexRayWakeSymRxLow           ((u32)0x00010051) // u32 --rw
// Cluster:FlexRay:Wakeup Symbol Rx Window
#define nxPropClst_FlexRayWakeSymRxWin           ((u32)0x00010052) // u32 --rw
// Cluster:FlexRay:Wakeup Symbol Tx Idle
#define nxPropClst_FlexRayWakeSymTxIdl           ((u32)0x00010053) // u32 --rw
// Cluster:FlexRay:Wakeup Symbol Tx Low
#define nxPropClst_FlexRayWakeSymTxLow           ((u32)0x00010054) // u32 --rw
// Cluster:Frames
#define nxPropClst_FrmRefs                       ((u32)0x06010004) // 1Dref --r
// Cluster:LIN:Schedules
#define nxPropClst_LINSchedules                  ((u32)0x06010070) // 1Dref --r
// Cluster:LIN:Tick
#define nxPropClst_LINTick                       ((u32)0x01010071) // f64 --rw
// Cluster:Name (Short)
#define nxPropClst_Name                          ((u32)0x03010005) // string --rw
// Cluster:PDUs
#define nxPropClst_PDURefs                       ((u32)0x06010008) // 1Dref --r
// Cluster:PDUs Required?
#define nxPropClst_PDUsReqd                      ((u32)0x0201000A) // bool --r
// Cluster:Protocol
#define nxPropClst_Protocol                      ((u32)0x00010006) // u32 --rw
// Cluster:Signals
#define nxPropClst_SigRefs                       ((u32)0x06010007) // 1Dref --r
// Frame:Application Protocol
#define nxPropFrm_ApplicationProtocol            ((u32)0x00020064) // u32 --rw
// Frame:CAN:Extended Identifier?
#define nxPropFrm_CANExtID                       ((u32)0x02020010) // bool --rw
// Frame:CAN:I/O Mode
#define nxPropFrm_CANioMode                      ((u32)0x00020066) // u32 --rw
// Frame:CAN:Timing Type
#define nxPropFrm_CANTimingType                  ((u32)0x00020011) // u32 --rw
// Frame:CAN:Transmit Time
#define nxPropFrm_CANTxTime                      ((u32)0x01020012) // f64 --rw
// Frame:Cluster
#define nxPropFrm_ClusterRef                     ((u32)0x05020001) // ref --r
// Frame:Comment
#define nxPropFrm_Comment                        ((u32)0x03020002) // string --rw
// Frame:Configuration Status
#define nxPropFrm_ConfigStatus                   ((u32)0x00020009) // u32 --r
// Frame:Default Payload
#define nxPropFrm_DefaultPayload                 ((u32)0x0A020005) // 1Du8 --rw
// Frame:FlexRay:Base Cycle
#define nxPropFrm_FlexRayBaseCycle               ((u32)0x00020020) // u32 --rw
// Frame:FlexRay:Channel Assignment
#define nxPropFrm_FlexRayChAssign                ((u32)0x00020021) // u32 --rw
// Frame:FlexRay:Cycle Repetition
#define nxPropFrm_FlexRayCycleRep                ((u32)0x00020022) // u32 --rw
// Frame:FlexRay:In Cycle Repetitions:Channel Assignments
#define nxPropFrm_FlexRayInCycRepChAssigns       ((u32)0x08020032) // 1Du32 --rw
// Frame:FlexRay:In Cycle Repetitions:Enabled?
#define nxPropFrm_FlexRayInCycRepEnabled         ((u32)0x02020030) // bool --r
// Frame:FlexRay:In Cycle Repetitions:Identifiers
#define nxPropFrm_FlexRayInCycRepIDs             ((u32)0x08020031) // 1Du32 --rw
// Frame:FlexRay:Payload Preamble?
#define nxPropFrm_FlexRayPreamble                ((u32)0x02020023) // bool --rw
// Frame:FlexRay:Startup?
#define nxPropFrm_FlexRayStartup                 ((u32)0x02020024) // bool --rw
// Frame:FlexRay:Sync?
#define nxPropFrm_FlexRaySync                    ((u32)0x02020025) // bool --rw
// Frame:FlexRay:Timing Type
#define nxPropFrm_FlexRayTimingType              ((u32)0x00020026) // u32 --rw
// Frame:Identifier
#define nxPropFrm_ID                             ((u32)0x00020003) // u32 --rw
// Frame:LIN:Checksum
#define nxPropFrm_LINChecksum                    ((u32)0x00020050) // u32 --r
// Frame:Mux:Data Multiplexer Signal
#define nxPropFrm_MuxDataMuxSigRef               ((u32)0x05020041) // ref --r
// Frame:Mux:Is Data Multiplexed?
#define nxPropFrm_MuxIsMuxed                     ((u32)0x02020040) // bool --r
// Frame:Mux:Static Signals
#define nxPropFrm_MuxStaticSigRefs               ((u32)0x06020042) // 1Dref --r
// Frame:Mux:Subframes
#define nxPropFrm_MuxSubframeRefs                ((u32)0x06020043) // 1Dref --r
// Frame:Name (Short)
#define nxPropFrm_Name                           ((u32)0x03020004) // string --rw
// Frame:PDU Start Bits
#define nxPropFrm_PDUStartBits                   ((u32)0x08020061) // 1Du32 --rw
// Frame:PDU Update Bits
#define nxPropFrm_PDUUpdateBits                  ((u32)0x08020063) // 1Du32 --rw
// Frame:PDUs
#define nxPropFrm_PDURefs                        ((u32)0x06020060) // 1Dref --rw
// Frame:Payload Length
#define nxPropFrm_PayloadLen                     ((u32)0x00020007) // u32 --rw
// Frame:Signals
#define nxPropFrm_SigRefs                        ((u32)0x06020008) // 1Dref --r
// Frame:Variable Payload
#define nxPropFrm_VariablePayload                ((u32)0x02020065) // bool --rw
// Signal:Byte Order
#define nxPropSig_ByteOrdr                       ((u32)0x00030001) // u32 --rw
// Signal:Comment
#define nxPropSig_Comment                        ((u32)0x03030002) // string --rw
// Signal:Configuration Status
#define nxPropSig_ConfigStatus                   ((u32)0x00030009) // u32 --r
// Signal:Data Type
#define nxPropSig_DataType                       ((u32)0x00030003) // u32 --rw
// Signal:Default Value
#define nxPropSig_Default                        ((u32)0x01030004) // f64 --rw
// Signal:Frame
#define nxPropSig_FrameRef                       ((u32)0x05030005) // ref --r
// Signal:Maximum Value
#define nxPropSig_Max                            ((u32)0x01030006) // f64 --rw
// Signal:Minimum Value
#define nxPropSig_Min                            ((u32)0x01030007) // f64 --rw
// Signal:Mux:Data Multiplexer?
#define nxPropSig_MuxIsDataMux                   ((u32)0x02030030) // bool --rw
// Signal:Mux:Dynamic?
#define nxPropSig_MuxIsDynamic                   ((u32)0x02030031) // bool --r
// Signal:Mux:Multiplexer Value
#define nxPropSig_MuxValue                       ((u32)0x00030032) // u32 --r
// Signal:Mux:Subframe
#define nxPropSig_MuxSubfrmRef                   ((u32)0x05030033) // ref --r
// Signal:Name (Short)
#define nxPropSig_Name                           ((u32)0x03030008) // string --rw
// Signal:Name (Unique to Cluster)
#define nxPropSig_NameUniqueToCluster            ((u32)0x03030010) // string --r
// Signal:Number of Bits
#define nxPropSig_NumBits                        ((u32)0x00030012) // u32 --rw
// Signal:PDU
#define nxPropSig_PDURef                         ((u32)0x05030011) // ref --r
// Signal:Scaling Factor
#define nxPropSig_ScaleFac                       ((u32)0x01030013) // f64 --rw
// Signal:Scaling Offset
#define nxPropSig_ScaleOff                       ((u32)0x01030014) // f64 --rw
// Signal:Start Bit
#define nxPropSig_StartBit                       ((u32)0x00030015) // u32 --rw
// Signal:Unit
#define nxPropSig_Unit                           ((u32)0x03030016) // string --rw
// Subframe:Configuration Status
#define nxPropSubfrm_ConfigStatus                ((u32)0x00040009) // u32 --r
// Subframe:Dynamic Signals
#define nxPropSubfrm_DynSigRefs                  ((u32)0x06040001) // 1Dref --r
// Subframe:Frame
#define nxPropSubfrm_FrmRef                      ((u32)0x05040002) // ref --r
// Subframe:Multiplexer Value
#define nxPropSubfrm_MuxValue                    ((u32)0x00040003) // u32 --rw
// Subframe:Name (Short)
#define nxPropSubfrm_Name                        ((u32)0x03040004) // string --rw
// Subframe:Name (Unique to Cluster)
#define nxPropSubfrm_NameUniqueToCluster         ((u32)0x03040007) // string --r
// Subframe:PDU
#define nxPropSubfrm_PDURef                      ((u32)0x05040005) // ref --r
// ECU:Cluster
#define nxPropECU_ClstRef                        ((u32)0x05050001) // ref --r
// ECU:Comment
#define nxPropECU_Comment                        ((u32)0x03050005) // string --rw
// ECU:Configuration Status
#define nxPropECU_ConfigStatus                   ((u32)0x00050009) // u32 --r
// ECU:FlexRay:Coldstart?
#define nxPropECU_FlexRayIsColdstart             ((u32)0x02050010) // bool --r
// ECU:FlexRay:Connected Channels
#define nxPropECU_FlexRayConnectedChs            ((u32)0x00050014) // u32 --rw
// ECU:FlexRay:Startup Frame
#define nxPropECU_FlexRayStartupFrameRef         ((u32)0x05050011) // ref --r
// ECU:FlexRay:Wakeup Channels
#define nxPropECU_FlexRayWakeupChs               ((u32)0x00050013) // u32 --rw
// ECU:FlexRay:Wakeup Pattern
#define nxPropECU_FlexRayWakeupPtrn              ((u32)0x00050012) // u32 --rw
// ECU:Frames Received
#define nxPropECU_RxFrmRefs                      ((u32)0x06050003) // 1Dref --rw
// ECU:Frames Transmitted
#define nxPropECU_TxFrmRefs                      ((u32)0x06050004) // 1Dref --rw
// ECU:LIN:Configured NAD
#define nxPropECU_LINConfigNAD                   ((u32)0x00050023) // u32 --rw
// ECU:LIN:Function ID
#define nxPropECU_LINFunctionID                  ((u32)0x00050025) // u32 --rw
// ECU:LIN:Initial NAD
#define nxPropECU_LINInitialNAD                  ((u32)0x00050022) // u32 --rw
// ECU:LIN:Master?
#define nxPropECU_LINMaster                      ((u32)0x02050020) // bool --rw
// ECU:LIN:P2min
#define nxPropECU_LINP2min                       ((u32)0x01050026) // f64 --rw
// ECU:LIN:Protocol Version
#define nxPropECU_LINProtocolVer                 ((u32)0x00050021) // u32 --rw
// ECU:LIN:STmin
#define nxPropECU_LINSTmin                       ((u32)0x01050027) // f64 --rw
// ECU:LIN:Supplier ID
#define nxPropECU_LINSupplierID                  ((u32)0x00050024) // u32 --rw
// ECU:Name (Short)
#define nxPropECU_Name                           ((u32)0x03050002) // string --rw
// ECU:SAE J1939:Node Name
#define nxPropECU_J1939NodeName                  ((u32)0x09050029) // u64 --rw
// ECU:SAE J1939:Preferred Address
#define nxPropECU_J1939PreferredAddress          ((u32)0x00050028) // u32 --rw
// LIN Schedule:Cluster
#define nxPropLINSched_ClstRef                   ((u32)0x05060005) // ref --r
// LIN Schedule:Comment
#define nxPropLINSched_Comment                   ((u32)0x03060006) // string --rw
// LIN Schedule:Configuration Status
#define nxPropLINSched_ConfigStatus              ((u32)0x00060007) // u32 --r
// LIN Schedule:Entries
#define nxPropLINSched_Entries                   ((u32)0x06060001) // 1Dref --r
// LIN Schedule:Name
#define nxPropLINSched_Name                      ((u32)0x03060002) // string --rw
// LIN Schedule:Priority
#define nxPropLINSched_Priority                  ((u32)0x00060003) // u32 --rw
// LIN Schedule:Run Mode
#define nxPropLINSched_RunMode                   ((u32)0x00060004) // u32 --rw
// LIN Schedule Entry:Collision Resolving Schedule
#define nxPropLINSchedEntry_CollisionResSched    ((u32)0x05070001) // ref --rw
// LIN Schedule Entry:Delay
#define nxPropLINSchedEntry_Delay                ((u32)0x01070002) // f64 --rw
// LIN Schedule Entry:Event Identifier
#define nxPropLINSchedEntry_EventID              ((u32)0x00070003) // u32 --rw
// LIN Schedule Entry:Frames
#define nxPropLINSchedEntry_Frames               ((u32)0x06070004) // 1Dref --rw
// LIN Schedule Entry:Name
#define nxPropLINSchedEntry_Name                 ((u32)0x03070006) // string --rw
// LIN Schedule Entry:Name (Unique to Cluster)
#define nxPropLINSchedEntry_NameUniqueToCluster  ((u32)0x03070008) // string --r
// LIN Schedule Entry:Node Configuration:Free Format:Data Bytes
#define nxPropLINSchedEntry_NC_FF_DataBytes      ((u32)0x0A070009) // 1Du8 --rw
// LIN Schedule Entry:Schedule
#define nxPropLINSchedEntry_Sched                ((u32)0x05070007) // ref --r
// LIN Schedule Entry:Type
#define nxPropLINSchedEntry_Type                 ((u32)0x00070005) // u32 --rw
// PDU:Cluster
#define nxPropPDU_ClusterRef                     ((u32)0x05080004) // ref --r
// PDU:Comment
#define nxPropPDU_Comment                        ((u32)0x03080002) // string --rw
// PDU:Configuration Status
#define nxPropPDU_ConfigStatus                   ((u32)0x00080007) // u32 --r
// PDU:Default Payload
#define nxPropPDU_DefaultPayload                 ((u32)0x0A080005) // 1Du8 --rw
// PDU:Frames
#define nxPropPDU_FrmRefs                        ((u32)0x06080006) // 1Dref --r
// PDU:Mux:Data Multiplexer Signal
#define nxPropPDU_MuxDataMuxSigRef               ((u32)0x05080009) // ref --r
// PDU:Mux:Is Data Multiplexed?
#define nxPropPDU_MuxIsMuxed                     ((u32)0x02080008) // bool --r
// PDU:Mux:Static Signals
#define nxPropPDU_MuxStaticSigRefs               ((u32)0x0608000A) // 1Dref --r
// PDU:Mux:Subframes
#define nxPropPDU_MuxSubframeRefs                ((u32)0x0608000B) // 1Dref --r
// PDU:Name (Short)
#define nxPropPDU_Name                           ((u32)0x03080001) // string --rw
// PDU:Payload Length
#define nxPropPDU_PayloadLen                     ((u32)0x00080003) // u32 --rw
// PDU:Signals
#define nxPropPDU_SigRefs                        ((u32)0x06080005) // 1Dref --r


#define nxPropSession_IntfCANOutStrmListById nxPropSession_IntfOutStrmListById

/***********************************************************************
   C O N S T A N T S   F O R   F U N C T I O N   P A R A M E T E R S
***********************************************************************/

   // Parameter Mode of function nxCreateSession
#define nxMode_SignalInSinglePoint           0  // SignalInSinglePoint
#define nxMode_SignalInWaveform              1  // SignalInWaveform
#define nxMode_SignalInXY                    2  // SignalInXY
#define nxMode_SignalOutSinglePoint          3  // SignalOutSinglePoint
#define nxMode_SignalOutWaveform             4  // SignalOutWaveform
#define nxMode_SignalOutXY                   5  // SignalOutXY
#define nxMode_FrameInStream                 6  // FrameInStream
#define nxMode_FrameInQueued                 7  // FrameInQueued
#define nxMode_FrameInSinglePoint            8  // FrameInSinglePoint
#define nxMode_FrameOutStream                9  // FrameOutStream
#define nxMode_FrameOutQueued                10 // FrameOutQueued
#define nxMode_FrameOutSinglePoint           11 // FrameOutSinglePoint
#define nxMode_SignalConversionSinglePoint   12 // SignalConversionSinglePoint

   // Parameter Scope of functions nxStart, nxStop
#define nxStartStop_Normal                   0  // StartStop_Normal
#define nxStartStop_SessionOnly              1  // StartStop_SessionOnly
#define nxStartStop_InterfaceOnly            2  // StartStop_InterfaceOnly
#define nxStartStop_SessionOnlyBlocking      3  // StartStop_SessionOnlyBlocking

   // Parameter Modifier of nxBlink
#define nxBlink_Disable                      0  // Blink_Disable
#define nxBlink_Enable                       1  // Blink_Enable

   // Terminal names for nxConnectTerminals and nxDisconnectTerminals (source or destination)
#define nxTerm_PXI_Trig0                     "PXI_Trig0"             // PXI_Trig0 same as RTSI0
#define nxTerm_PXI_Trig1                     "PXI_Trig1"
#define nxTerm_PXI_Trig2                     "PXI_Trig2"
#define nxTerm_PXI_Trig3                     "PXI_Trig3"
#define nxTerm_PXI_Trig4                     "PXI_Trig4"
#define nxTerm_PXI_Trig5                     "PXI_Trig5"
#define nxTerm_PXI_Trig6                     "PXI_Trig6"
#define nxTerm_PXI_Trig7                     "PXI_Trig7"
#define nxTerm_FrontPanel0                   "FrontPanel0"
#define nxTerm_FrontPanel1                   "FrontPanel1"
#define nxTerm_PXI_Star                      "PXI_Star"
#define nxTerm_PXI_Clk10                     "PXI_Clk10"
#define nxTerm_10MHzTimebase                 "10MHzTimebase"
#define nxTerm_1MHzTimebase                  "1MHzTimebase"
#define nxTerm_MasterTimebase                "MasterTimebase"
#define nxTerm_CommTrigger                   "CommTrigger"
#define nxTerm_StartTrigger                  "StartTrigger"
#define nxTerm_FlexRayStartCycle             "FlexRayStartCycle"
#define nxTerm_FlexRayMacrotick              "FlexRayMacrotick"
#define nxTerm_LogTrigger                    "LogTrigger"
#define nxTerm_TimeTrigger                   "TimeTrigger"
#define nxTerm_NetworkTimePPS                "NetworkTimePPS"
#define nxTerm_NetworkTime1MHz               "NetworkTime1MHz"

   /* StateID for nxReadState
   These constants use an encoding similar to property ID (nxProp_ prefix).
   */
      // Current time of the interface (using nxTimestamp100ns_t)
#define nxState_TimeCurrent                  ((u32)0x00000001 | nxClass_Interface | nxPrptype_time100ns) // TimeCurrent
      // Time when communication began on the interface (protocol operational / integrated)
#define nxState_TimeCommunicating            ((u32)0x00000002 | nxClass_Interface | nxPrptype_time100ns) // TimeCommunicating
      // Start time of the interface, when the attempt to communicate began (startup protocol)
#define nxState_TimeStart                    ((u32)0x00000003 | nxClass_Interface | nxPrptype_time100ns) // TimeStart
      // Session information: Use macros with prefix nxSessionInfo_Get_ to get fields of the u32
#define nxState_SessionInfo                  ((u32)0x00000004 | nxClass_Interface | nxPrptype_u32)    // SessionInfo
      // Current time of the interface, returned with two timestamps
#define nxState_TimeCurrent2                 ((u32)0x00000005 | nxClass_Interface | nxPrptype_struct) // uses nxTimeLocalNetwork_t
      // Time when communication began on the interface, returned with two timestamps
#define nxState_TimeCommunicating2           ((u32)0x00000006 | nxClass_Interface | nxPrptype_struct) // uses nxTimeLocalNetwork_t
      // Start time of the interface, returned with two timestamps
#define nxState_TimeStart2                   ((u32)0x00000007 | nxClass_Interface | nxPrptype_struct) // uses nxTimeLocalNetwork_t
      // CAN communication: Use macros with prefix nxCANComm_Get_ to get fields of the u32
#define nxState_CANComm                      ((u32)0x00000010 | nxClass_Interface | nxPrptype_u32)    // CANComm
      // FlexRay communication: Use macros with prefix nxFlexRayComm_Get_ to get fields of the u32
#define nxState_FlexRayComm                  ((u32)0x00000020 | nxClass_Interface | nxPrptype_u32)    // FlexRayComm
      // FlexRay statistics: Use typedef nxFlexRayStats_t to read these statistics using a struct of multiple u32
#define nxState_FlexRayStats                 ((u32)0x00000021 | nxClass_Interface | nxPrptype_1Du32)  // FlexRayStats
      // LIN communication: Use macros with prefix nxLINComm_Get_ to get fields of 2 u32's
#define nxState_LINComm                      ((u32)0x00000030 | nxClass_Interface | nxPrptype_u32)    // LINComm
      // J1939 communication:
#define nxState_J1939Comm                    ((u32)0x00000040 | nxClass_Interface | nxPrptype_u32)    // J1939Comm

   /* StateID for nxWriteState
   These constants use an encoding similar to property ID (nxProp_ prefix).
   */
#define nxState_LINScheduleChange            ((u32)0x00000081 | nxClass_Interface | nxPrptype_u32)    // LINScheduleChange
#define nxState_LINDiagnosticScheduleChange  ((u32)0x00000083 | nxClass_Interface | nxPrptype_u32)    // LINDiagnosticScheduleChange
#define nxState_FlexRaySymbol                ((u32)0x00000082 | nxClass_Interface | nxPrptype_u32)    // FlexRaySymbol
#define nxState_EthernetSleep                ((u32)0x00000084 | nxClass_Interface | nxPrptype_u32)    // EthernetSleep
#define nxState_EthernetWake                 ((u32)0x00000085 | nxClass_Interface | nxPrptype_u32)    // EthernetWake

#define  nxCanFdMode_ISO         0  // ISO mode, default
#define  nxCanFdMode_NonISO      1  // Non-ISO mode (first CAN FD definition)
#define  nxCanFdMode_ISO_Legacy  2  // ISO mode, but behaviour like non-ISO

   // Macros to get fields of u32 returned by nxReadState of nxState_SessionInfo
      // Get state of frames in the session; uses constants with prefix nxSessionInfoState_
#define nxSessionInfo_Get_State(StateValue)  ((u8)( (u32)StateValue         & 0x00000003))

   // State of frames in the session, from nxState_SessionInfo (nxSessionInfo_Get_State)
#define nxSessionInfoState_Stopped           0     // all frames stopped
#define nxSessionInfoState_Started           1     // all frames started
#define nxSessionInfoState_Mix               2     // one or more frames started, and one or more frames stopped

   // Macros to get fields of u32 returned by nxReadState of nxState_CANComm
      // Get CAN communication state; uses constants with prefix nxCANCommState_
#define nxCANComm_Get_CommState(StateValue)  ((u8)( (u32)StateValue         & 0x0000000F))
      // Get CAN transceiver error (!NERR); 1 = error, 0 = no error
#define nxCANComm_Get_TcvrErr(StateValue)    ((u8)( ((u32)StateValue >> 4)  & 0x00000001))
      // Get indication of CAN controller/transceiver sleep; 1 = asleep, 0 = awake
#define nxCANComm_Get_Sleep(StateValue)      ((u8)( ((u32)StateValue >> 5)  & 0x00000001))
      // Get last bus error that incremented counters; uses constants with prefix nxCANLastErr_
#define nxCANComm_Get_LastErr(StateValue)    ((u8)( ((u32)StateValue >> 8)  & 0x0000000F))
      // Get Transmit Error Counter as defined by the CAN protocol specification
#define nxCANComm_Get_TxErrCount(StateValue) ((u8)( ((u32)StateValue >> 16) & 0x000000FF))
      // Get Receive Error Counter as defined by the CAN protocol specification
#define nxCANComm_Get_RxErrCount(StateValue) ((u8)( ((u32)StateValue >> 24) & 0x000000FF))

   // Communication state from nxState_CANComm (nxCANComm_Get_CommState)
#define nxCANCommState_ErrorActive           0
#define nxCANCommState_ErrorPassive          1
#define nxCANCommState_BusOff                2
#define nxCANCommState_Init                  3

   // Last bus error from nxState_CANComm (nxCANComm_Get_LastErr)
#define nxCANLastErr_None                    0
#define nxCANLastErr_Stuff                   1
#define nxCANLastErr_Form                    2
#define nxCANLastErr_Ack                     3
#define nxCANLastErr_Bit1                    4
#define nxCANLastErr_Bit0                    5
#define nxCANLastErr_CRC                     6

   // constants for nxPropClst_CanIoMode, nxPropSession_IntfCanIoMode, nxPropSession_IntfCanTxIoMode, nxPropAlias_CanIoMode
#define nxCANioMode_CAN                      0
#define nxCANioMode_CAN_FD                   1
#define nxCANioMode_CAN_FD_BRS               2

   // Macros to get fields of u32 returned by nxReadState of nxState_FlexRayComm
      /* Get FlexRay Protocol Operation Control (POC) state,
      which uses constants with prefix nxFlexRayPOCState_ */
#define nxFlexRayComm_Get_POCState(StateValue)        ((u8)((u32)StateValue & 0x0000000F))
      /* From FlexRay spec 9.3.1.3.4: "the number of consecutive even/odd cycle pairs
      (vClockCorrectionFailed) that have passed without clock synchronization having performed an offset or a rate
      correction due to lack of synchronization frames (as maintained by the POC process)."
      This value is used for comparison to the cluster thresholds MaxWithoutClockCorrectFatal and
      MaxWithoutClockCorrectionPassive (XNET properties nxPropClst_FlexRayMaxWoClkCorFat
      and nxPropClst_FlexRayMaxWoClkCorPas). */
#define nxFlexRayComm_Get_ClockCorrFailed(StateValue) ((u8)( ((u32)StateValue >> 4) & 0x0000000F))
      /* From FlexRay spec 9.3.1.3.1: "the number of consecutive even/odd cycle pairs (vAllowPassiveToActive)
      that have passed with valid rate and offset correction terms, but the node still in POC:normal passive
      state due to a host configured delay to POC:normal active state (as maintained by the POC process).
      This value is used for comparison to the interface threshold AllowPassiveToActive
      (XNET property nxPropSession_IntfFlexRayAlwPassAct). */
#define nxFlexRayComm_Get_PassiveToActiveCount(StateValue)  \
                                                      ((u8)( ((u32)StateValue >> 8) & 0x0000001F))
#define nxFlexRayComm_Get_ChannelASleep(StateValue)  \
                                                      ((u8)( ((u32)StateValue >> 13) & 0x00000001))
#define nxFlexRayComm_Get_ChannelBSleep(StateValue)  \
                                                      ((u8)( ((u32)StateValue >> 14) & 0x00000001))

   // POC state (Protocol Operation Control state) from nxFlexRayPOC_Get_State
#define nxFlexRayPOCState_DefaultConfig      0
#define nxFlexRayPOCState_Ready              1
#define nxFlexRayPOCState_NormalActive       2
#define nxFlexRayPOCState_NormalPassive      3
#define nxFlexRayPOCState_Halt               4
#define nxFlexRayPOCState_Monitor            5
#define nxFlexRayPOCState_Config             15

   // Macros to get fields of 1st u32 returned by nxReadState of nxState_LINComm
      // Get indication of LIN interface sleep state; 1 = asleep, 0 = awake
#define nxLINComm_Get_Sleep(StateValue)      ((u8)( ((u32)StateValue >> 1) & 0x00000001))
      // Get LIN communication state; uses constants with prefix nxLINCommState_
#define nxLINComm_Get_CommState(StateValue)  ((u8)( ((u32)StateValue >> 2) & 0x00000003))
      // Get last bus error; this code uses constants with prefix nxLINLastErrCode_
#define nxLINComm_Get_LastErrCode(StateValue) \
                                             ((u8)( ((u32)StateValue >> 4) & 0x0000000F))
      // Get received data for last bus error; this value applies only to specific codes
#define nxLINComm_Get_LastErrReceived(StateValue) \
                                             ((u8)( ((u32)StateValue >> 8) & 0x000000FF))
      // Get expected data for last bus error; this value applies only to specific codes
#define nxLINComm_Get_LastErrExpected(StateValue) \
                                             ((u8)( ((u32)StateValue >> 16) & 0x000000FF))
      // Get ID of last bus error; this value applies only to specific codes
#define nxLINComm_Get_LastErrID(StateValue) \
                                             ((u8)( ((u32)StateValue >> 24) & 0x0000003F))
      // Get indication of LIN transceiver ready (powered); 1 = powered, 0 = not powered
#define nxLINComm_Get_TcvrRdy(StateValue)    ((u8)( ((u32)StateValue >> 31) & 0x00000001))

   // Macros to get fields of 2nd u32 returned by nxReadState of nxState_LINComm
      // Get index of the currently running schedule (0xFF if Null-schedule).
#define nxLINComm_Get2_ScheduleIndex(State2Value) \
                                             ((u8)( ((u32)State2Value) & 0x000000FF))

   // Communication state from nxState_LINComm (nxLINComm_Get_CommState macro)
#define nxLINCommState_Idle                  0
#define nxLINCommState_Active                1
#define nxLINCommState_Inactive              2

   // Diagnostic schedule state for nxState_LINDiagnosticScheduleChange
#define nxLINDiagnosticSchedule_NULL         0x0000
#define nxLINDiagnosticSchedule_MasterReq    0x0001
#define nxLINDiagnosticSchedule_SlaveResp    0x0002

   // Last error code from nxState_LINComm (nxLINComm_Get_LastErrCode macro)
#define nxLINLastErrCode_None                0
#define nxLINLastErrCode_UnknownId           1
#define nxLINLastErrCode_Form                2
#define nxLINLastErrCode_Framing             3
#define nxLINLastErrCode_Readback            4
#define nxLINLastErrCode_Timeout             5
#define nxLINLastErrCode_CRC                 6

   // Condition of nxWait
#define nxCondition_TransmitComplete         0x8001  // TransmitComplete
#define nxCondition_IntfCommunicating        0x8002  // IntfCommunicating
#define nxCondition_IntfRemoteWakeup         0x8003  // IntfRemoteWakeup
#define nxCondition_EthernetSynced           0x8004  // EthernetSynced

   // Constants for use with Timeout parameter of read and write functions
#define nxTimeout_None                       (0)
#define nxTimeout_Infinite                   (-1)

   // Parameter Mode of function nxdbGetDBCAttribute and nxdbGetDBCAttributeSize
#define nxGetDBCMode_Attribute               0  // Attribute
#define nxGetDBCMode_EnumerationList         1  // Enumeration List
#define nxGetDBCMode_AttributeList           2  // List of available attributes of given type
#define nxGetDBCMode_ValueTableList          3  // Value table for a signal

   // Copy mode of function nxdbMerge
#define nxdbMerge_CopyUseSource     0  // CopyUseSource
#define nxdbMerge_CopyUseTarget     1  // CopyUseTarget
#define nxdbMerge_MergeUseSource    2  // MergeUseSource
#define nxdbMerge_MergeUseTarget    3  // MergeUseTarget

   // For nxReadState or nxReadStateTimeTrigger using nxTimeLocalNetwork_t,
   // the following bitmasks are used with the Flags field.
#define nxTimeFlags_NetworkSynced               0x00000001

   // For UseFlags field of nxEptRxFilter_Element_t, which is
   // a single element of the array that is get/set using
   // property nxPropSession_IntfEnetEptReceiveFilter.
#define nxEptRxFilter_UseFlags_VID              0x00000001
#define nxEptRxFilter_UseFlags_Priority         0x00000002
#define nxEptRxFilter_UseFlags_DestinationMAC   0x00000004

   // Timescale parameter of nxFutureTimeTrigger
#define nxTimescale_LocalTime                0
#define nxTimescale_NetworkTime              1

/***********************************************************************
   C O N S T A N T S   F O R   H A R D W A R E   P R O P E R T I E S
***********************************************************************/

// System/Device/Interface properties (hardware info)

   // Property ID nxPropSys_VerPhase
#define nxPhase_Development                  0
#define nxPhase_Alpha                        1
#define nxPhase_Beta                         2
#define nxPhase_Release                      3

   // Property ID nxPropDev_FormFac
#define nxDevForm_PXI                        0
#define nxDevForm_PCI                        1
#define nxDevForm_cSeries                    2
#define nxDevForm_PXIe                       3
#define nxDevForm_USB                        4
#define nxDevForm_PCIe                       5

   // Property ID nxPropIntf_CANTermCap
#define nxCANTermCap_No                      0
#define nxCANTermCap_Yes                     1

   // Property ID nxPropIntf_CANTcvrCap
#define nxCANTcvrCap_HS                      0
#define nxCANTcvrCap_LS                      1
#define nxCANTcvrCap_XS                      3
#define nxCANTcvrCap_XS_HS_LS                4
#define nxCANTcvrCap_Unknown                 0xFFFFFFFF

   // Property ID nxPropIntf_Protocol and nxPropClst_Protocol
#define nxProtocol_Unknown                   0xFFFFFFFE   //The value will be reported on a port with unplugged transceiver cable
#define nxProtocol_CAN                       0
#define nxProtocol_FlexRay                   1
#define nxProtocol_LIN                       2
#define nxProtocol_Ethernet                  3

   // Property ID nxPropClst_ApplicationProtocol and nxPropSession_ApplicationProtocol
#define nxAppProtocol_None                   0
#define nxAppProtocol_J1939                  1

   // State of the dongle (nxPropIntf_DongleState)
#define nxDongleState_NoDongle_NoExtPower    1
#define nxDongleState_NoDongle_ExtPower      2
#define nxDongleState_Dongle_NoExtPower      3
#define nxDongleState_Ready                  4
#define nxDongleState_Busy                   5
#define nxDongleState_CommError              13
#define nxDongleState_OverCurrent            14

   // Dongle ID (nxPropIntf_DongleID)
#define nxDongleID_LS_CAN           1
#define nxDongleID_HS_CAN           2
#define nxDongleID_SW_CAN           3
#define nxDongleID_XS_CAN           4
#define nxDongleID_LIN              6
#define nxDongleID_DongleLess       13
#define nxDongleID_Unknown          14

/***********************************************************************
   C O N S T A N T S   F O R   S E S S I O N   P R O P E R T I E S
***********************************************************************/

// Session properties (including runtime interface properties)

   // Macro to set nxPropSession_IntfBaudRate for an advanced CAN baud rate (bit timings)
   // If you pass a basic baud rate like 125000 or 500000, NI-XNET calculates bit timings for you
#define nxAdvCANBaudRate_Set(TimeQuantum, TimeSeg0, TimeSeg1, SyncJumpWidth) ( \
            (((u32)TimeQuantum) & 0x0000FFFF) | \
            (((u32)TimeSeg0 << 16) & 0x000F0000) | \
            (((u32)TimeSeg1 << 20) & 0x00700000) | \
            (((u32)SyncJumpWidth << 24) & 0x03000000) | \
            ((u32)0x80000000) )

   // Macros to get fields of nxPropSession_IntfBaudRate for an advanced CAN baud rate
#define nxAdvCANBaudRate_Get_TimeQuantum(AdvBdRt)     ((u16)( ((u32)AdvBdRt) & 0x0000FFFF))
#define nxAdvCANBaudRate_Get_TimeSeg0(AdvBdRt)        ((u8)( ((u32)AdvBdRt >> 16) & 0x0000000F))
#define nxAdvCANBaudRate_Get_TimeSeg1(AdvBdRt)        ((u8)( ((u32)AdvBdRt >> 20) & 0x00000007))
#define nxAdvCANBaudRate_Get_SyncJumpWidth(AdvBdRt)   ((u8)( ((u32)AdvBdRt >> 24) & 0x00000003))
#define nxAdvCANBaudRate_Get_NumSamples(AdvBdRt)      ((u8)( ((u32)AdvBdRt >> 26) & 0x00000001))

   // Property ID nxPropSession_IntfCANTerm
#define nxCANTerm_Off                        0
#define nxCANTerm_On                         1

    // Property ID nxPropSession_IntfCANTcvrState
#define nxCANTcvrState_Normal                0
#define nxCANTcvrState_Sleep                 1
#define nxCANTcvrState_SWWakeup              2
#define nxCANTcvrState_SWHighSpeed           3

   // Property ID nxPropSession_IntfCANTcvrType
#define nxCANTcvrType_HS                     0
#define nxCANTcvrType_LS                     1
#define nxCANTcvrType_SW                     2
#define nxCANTcvrType_Ext                    3
#define nxCANTcvrType_Disc                   4

   // Property ID nxPropSession_IntfFlexRaySampPerMicro
#define nxFlexRaySampPerMicro_1              0
#define nxFlexRaySampPerMicro_2              1
#define nxFlexRaySampPerMicro_4              2

   // Property ID nxPropSession_IntfFlexRayTerm
#define nxFlexRayTerm_Off                    0
#define nxFlexRayTerm_On                     1

   // Property ID nxPropSession_IntfLINSleep
#define nxLINSleep_RemoteSleep               0
#define nxLINSleep_RemoteWake                1
#define nxLINSleep_LocalSleep                2
#define nxLINSleep_LocalWake                 3

   // Property ID nxPropSession_IntfLINTerm
#define nxLINTerm_Off                        0
#define nxLINTerm_On                         1

   // Property ID nxPropSession_IntfOutStrmTimng
#define nxOutStrmTimng_Immediate             0
#define nxOutStrmTimng_ReplayExclusive       1
#define nxOutStrmTimng_ReplayInclusive       2

   // Property ID nxPropSession_IntfCANPendTxOrder
#define nxCANPendTxOrder_AsSubmitted         0
#define nxCANPendTxOrder_ByIdentifier        1

   // Property ID nxPropSession_IntfFlexRaySleep
#define nxFlexRaySleep_LocalSleep            0
#define nxFlexRaySleep_LocalWake             1
#define nxFlexRaySleep_RemoteWake            2

   // Property ID nxPropSession_IntfCANExtTcvrConfig
   // These bits can be combined to define the capabilities of the connected
   // external transceiver.
#define nxCANExtTcvrConfig_NormalSupported              (1 << 2)
#define nxCANExtTcvrConfig_SleepSupported               (1 << 5)
#define nxCANExtTcvrConfig_SWWakeupSupported            (1 << 8)
#define nxCANExtTcvrConfig_SWHighSpeedSupported         (1 << 11)
#define nxCANExtTcvrConfig_PowerOnSupported             (1 << 14)
#define nxCANExtTcvrConfig_NormalOutput0Set             (1 << 0)
#define nxCANExtTcvrConfig_SleepOutput0Set              (1 << 3)
#define nxCANExtTcvrConfig_SWWakeupOutput0Set           (1 << 6)
#define nxCANExtTcvrConfig_SWHighSpeedOutput0Set        (1 << 9)
#define nxCANExtTcvrConfig_PowerOnOutput0Set            (1 << 12)
#define nxCANExtTcvrConfig_NormalOutput1Set             (1 << 1)
#define nxCANExtTcvrConfig_SleepOutput1Set              (1 << 4)
#define nxCANExtTcvrConfig_SWWakeupOutput1Set           (1 << 7)
#define nxCANExtTcvrConfig_SWHighSpeedOutput1Set        (1 << 10)
#define nxCANExtTcvrConfig_PowerOnOutput1Set            (1 << 13)
#define nxCANExtTcvrConfig_nErrConnected                (1 << 31)

   // Ethernet port modes (nxPropSession_IntfEnetPortMode)
#define nxEnetPortMode_Direct                0
#define nxEnetPortMode_Tap                   1

   // Ethernet PHY states (nxPropSession_IntfEnetPhyState)
#define nxEnetPhyState_Slave                 0
#define nxEnetPhyState_Master                1
#define nxEnetPhyState_Auto                  2

   // Ethernet Link speeds (nxPropSession_IntfEnetLinkSpeedConfigured and nxPropSession_IntfEnetLinkSpeed)
   // Note: nxEnetLinkSpeed_LinkDown is valid only for nxPropSession_IntfEnetLinkSpeed
#define nxEnetLinkSpeed_LinkDown      0
#define nxEnetLinkSpeed_100Mbps       1
#define nxEnetLinkSpeed_1000Mbps      2

   // Ethernet Jumbo Frames enum values (nxPropSession_IntfEnetJumboFrames)
#define nxEnetJumboFrames_Disabled           0
#define nxEnetJumboFrames_9018Bytes          1

   // Ethernet operational status values (nxPropSession_IntfEnetOperationalStatus)
#define nxEnetOperationalStatus_Down         0
#define nxEnetOperationalStatus_Up           1

   // Ethernet logging mode values (nxPropSession_IntfEnetLogMode)
#define nxEnetLogMode_Off                    0
#define nxEnetLogMode_Log                    1

   // Ethernet logging operation values (nxPropSession_IntfEnetLogOperation)
#define nxEnetLogOperation_CreateOrReplace   0
#define nxEnetLogOperation_Create            1

   // Ethernet Time Sync Protocol (nxPropSession_IntfEnetTimeProtocol)
#define nxEnetTimeProtocol_IEEE8021as        0

   // Ethernet Time Sync Clock Accuracy (nxPropSession_IntfEnetTimeClkAccuracy)
#define nxEnetTimeClkAccuracy_Within25nsec      32
#define nxEnetTimeClkAccuracy_Within100nsec     33
#define nxEnetTimeClkAccuracy_Within250nsec     34
#define nxEnetTimeClkAccuracy_Within1usec       35
#define nxEnetTimeClkAccuracy_Within2500nsec    36
#define nxEnetTimeClkAccuracy_Within10usec      37
#define nxEnetTimeClkAccuracy_Within25usec      38
#define nxEnetTimeClkAccuracy_Within100usec     39
#define nxEnetTimeClkAccuracy_Within250usec     40
#define nxEnetTimeClkAccuracy_Within1msec       41
#define nxEnetTimeClkAccuracy_Within2500usec    42
#define nxEnetTimeClkAccuracy_Within10msec      43
#define nxEnetTimeClkAccuracy_Within25msec      44
#define nxEnetTimeClkAccuracy_Within100msec     45
#define nxEnetTimeClkAccuracy_Within250msec     46
#define nxEnetTimeClkAccuracy_Within1sec        47
#define nxEnetTimeClkAccuracy_Within10sec       48
#define nxEnetTimeClkAccuracy_GreatherThan10sec 49
#define nxEnetTimeClkAccuracy_Unknown           254


   // Ethernet Time Sync Port Sync Status (nxPropSession_IntfEnetTimePortSyncStatus)
#define nxEnetTimePortSyncStatus_Synced                   0
#define nxEnetTimePortSyncStatus_EnetLinkDown             1
#define nxEnetTimePortSyncStatus_ProtocolDisabled         2
#define nxEnetTimePortSyncStatus_MeasuringPropDelay       3
#define nxEnetTimePortSyncStatus_MasterPendingAnnounce    4
#define nxEnetTimePortSyncStatus_WaitingForMaster         5
#define nxEnetTimePortSyncStatus_SyncingToMaster          6
#define nxEnetTimePortSyncStatus_PeerNotProtoCapable      7
#define nxEnetTimePortSyncStatus_PropDelayExceedsTreshold 8
#define nxEnetTimePortSyncStatus_SyncReceiptTimeout       9
#define nxEnetTimePortSyncStatus_FrequencyOutOfRange      10
#define nxEnetTimePortSyncStatus_SyncIntervalOutOfRange   11
#define nxEnetTimePortSyncStatus_MultipleMastersDetected  12


   // Ethernet Time Sync Port State Configured (nxPropSession_IntfEnetTimePortStateConfigured)
#define nxEnetTimePortState_Disabled         3
#define nxEnetTimePortState_Master           6
#define nxEnetTimePortState_Passive          7
#define nxEnetTimePortState_Slave            9

   // Ethernet Time Sync PDelay Req Interval (nxPropSession_IntfEnetTimePortLogPdelayIntervalConfigured)
#define nxEnetTimePDelayReqInterval_125ms   -3
#define nxEnetTimePDelayReqInterval_250ms   -2
#define nxEnetTimePDelayReqInterval_500ms   -1
#define nxEnetTimePDelayReqInterval_1s       0
#define nxEnetTimePDelayReqInterval_2s       1

   // Ethernet Time Sync Sync Interval (nxPropSession_IntfEnetTimePortLogSyncIntervalConfigured)
#define nxEnetTimeSyncInterval_125ms        -3
#define nxEnetTimeSyncInterval_250ms        -2
#define nxEnetTimeSyncInterval_500ms        -1
#define nxEnetTimeSyncInterval_1s            0
#define nxEnetTimeSyncInterval_2s            1

   // Ethernet Time Sync Announce Interval (nxPropSession_IntfEnetTimePortLogAnnounceIntervalConfigured)
#define nxEnetTimeAnnounceInterval_125ms    -3
#define nxEnetTimeAnnounceInterval_250ms    -2
#define nxEnetTimeAnnounceInterval_500ms    -1
#define nxEnetTimeAnnounceInterval_1s        0
#define nxEnetTimeAnnounceInterval_2s        1

   // Ethernet Sleep Capability (nxPropSession_IntfEnetSleepCapability)
#define nxEnetSleepCapability_DisabledOrNotAvailable 0
#define nxEnetSleepCapability_Enabled  1

   // Ethernet PHY Power Mode (nxPropSession_IntfEnetPhyPowerMode)
#define nxEnetPhyPowerMode_Normal 0
#define nxEnetPhyPowerMode_Sleep 1

/***********************************************************************
   C O N S T A N T S   F O R   D A T A B A S E   P R O P E R T I E S
***********************************************************************/

// Database properties (Database/Cluster/ECU/Frame/Subframe/Signal)

   // Property ID nxPropClst_FlexRayChannels, nxPropFrm_FlexRayChAssign
   // nxPropClst_FlexRayConnectedChannels and nxPropClst_FlexRayWakeChannels
#define nxFrmFlexRayChAssign_A                  1
#define nxFrmFlexRayChAssign_B                  2
#define nxFrmFlexRayChAssign_AandB              3
#define nxFrmFlexRayChAssign_None               4

   // Property ID nxPropClst_FlexRaySampClkPer
#define nxClstFlexRaySampClkPer_p0125us         0
#define nxClstFlexRaySampClkPer_p025us          1
#define nxClstFlexRaySampClkPer_p05us           2

   // Property ID nxPropFrm_FlexRayTimingType
#define nxFrmFlexRayTiming_Cyclic               0
#define nxFrmFlexRayTiming_Event                1

   // Property ID nxPropFrm_CANTimingType
#define nxFrmCANTiming_CyclicData               0
#define nxFrmCANTiming_EventData                1
#define nxFrmCANTiming_CyclicRemote             2
#define nxFrmCANTiming_EventRemote              3
#define nxFrmCANTiming_CyclicEvent              4

   // Property ID nxPropSig_ByteOrdr
#define nxSigByteOrdr_LittleEndian              0  // Intel
#define nxSigByteOrdr_BigEndian                 1  // Motorola

   // Property ID nxPropSig_DataType
#define nxSigDataType_Signed                    0
#define nxSigDataType_Unsigned                  1
#define nxSigDataType_IEEEFloat                 2
#define nxSigDataType_ByteArray                 3

   // Property ID nxPropECU_LINProtocolVer
#define nxLINProtocolVer_1_2                    2
#define nxLINProtocolVer_1_3                    3
#define nxLINProtocolVer_2_0                    4
#define nxLINProtocolVer_2_1                    5
#define nxLINProtocolVer_2_2                    6

   // Property ID nxPropLINSched_RunMode
#define nxLINSchedRunMode_Continuous            0
#define nxLINSchedRunMode_Once                  1
#define nxLINSchedRunMode_Null                  2

   // Property ID nxPropLINSchedEntry_Type
#define nxLINSchedEntryType_Unconditional       0
#define nxLINSchedEntryType_Sporadic            1
#define nxLINSchedEntryType_EventTriggered      2
#define nxLINSchedEntryType_NodeConfigService   3

   // Property ID nxPropFrm_LINChecksum
#define nxFrmLINChecksum_Classic                0
#define nxFrmLINChecksum_Enhanced               1

/***********************************************************************
                            D A T A   T Y P E S
***********************************************************************/

   /* The ANSI C99 standard defines simple numeric types of a specific size,
   such as int32_t for a signed 32-bit integer.
   Many C/C++ compilers are not ANSI C99 by default, such as Microsoft Visual C/C++.
   Therefore, NI-XNET does not require use of ANSI C99.
   Since NI-XNET does not attempt to override ANSI C99 types (as defined in stdint.h),
   it uses legacy National Instruments numeric types such as i32. If desired, you can use
   ANSI C99 numeric types instead of the analogous NI-XNET numeric type
   (i.e. int32_t instead of i32). */

#ifndef _NIDAQ_Header_      // Traditional NI-DAQ header defines numeric types same as below.
#ifndef nNISS100_kCPP       // Same for NI SS layer.

#ifndef _NI_i8_DEFINED_
#define _NI_i8_DEFINED_
typedef signed char        i8;
#endif
#ifndef _NI_i16_DEFINED_
#define _NI_i16_DEFINED_
typedef signed short       i16;
#endif

#ifndef _NI_i32_DEFINED_
#define _NI_i32_DEFINED_
#if (defined(_MSC_VER) || defined(_CVI_) || defined(__BORLANDC__))
   typedef signed int     i32;
#elif defined(__GNUC__)
   typedef int32_t i32;
#endif
#endif

#ifndef _NI_i64_DEFINED_
#define _NI_i64_DEFINED_
#if (defined(_MSC_VER) || defined(_CVI_) || defined(__BORLANDC__))
   typedef __int64     i64;
#elif defined(__GNUC__)
   typedef long long int   i64;
#endif
#endif

#ifndef _NI_u8_DEFINED_
#define _NI_u8_DEFINED_
typedef unsigned char      u8;
#endif

#ifndef _NI_u16_DEFINED_
#define _NI_u16_DEFINED_
typedef unsigned short     u16;
#endif

#ifndef _NI_u32_DEFINED_
#define _NI_u32_DEFINED_
#if (defined(_MSC_VER) || defined(_CVI_) || defined(__BORLANDC__))
   typedef unsigned int    u32;
#elif defined(__GNUC__)
   typedef uint32_t u32;
#endif
#endif

#ifndef _NI_u64_DEFINED_
#define _NI_u64_DEFINED_
#if (defined(_MSC_VER) || defined(_CVI_) || defined(__BORLANDC__))
   typedef unsigned __int64     u64;
#elif defined(__GNUC__)
   typedef unsigned long long int   u64;
#endif
#endif

#ifndef _NI_f32_DEFINED_
#define _NI_f32_DEFINED_
typedef float              f32;
#endif

#ifndef _NI_f64_DEFINED_
#define _NI_f64_DEFINED_
typedef double             f64;
#endif

typedef void*              nxVoidPtr;
typedef u32*               nxU32Ptr;
#endif // nNISS100_kCPP
#endif // _NIDAQ_Header_

   // Session Reference (handle).
typedef u32 nxSessionRef_t;

   // Database Reference (handle).
typedef u32 nxDatabaseRef_t;

typedef i32 nxStatus_t;       // Return value

   // Absolute timestamp in 100 nanosecond increments.
   // This 64-bit type contains the number of 100 ns intervals that have
   // elapsed since 1 January 1601 00:00:00 Coordinated Universal Time (UTC).
typedef u64 nxTimestamp100ns_t;
   // In previous releases this name was used for nxTimestamp100ns_t.
typedef nxTimestamp100ns_t nxTimestamp_t;

   // Absolute timestamp in 1 nanosecond increments.
   // This 64-bit type contains the number of 1 ns intervals that have
   // elapsed since 1 January 1970 00:00:00 International Atomic Time (TAI).
typedef u64 nxTimestamp1ns_t;

typedef struct _nxFlexRayStats_t {
      u32 NumSyntaxErrorChA;
      u32 NumSyntaxErrorChB;
      u32 NumContentErrorChA;
      u32 NumContentErrorChB;
      u32 NumSlotBoundaryViolationChA;
      u32 NumSlotBoundaryViolationChB;
   } nxFlexRayStats_t;

   // nxReadState nxState_J1939Comm StateValue
typedef struct _nxJ1939CommState_t {
      u32 PGN;
      u8  SourceAddress;
      u8  DestinationAddress;
      u8  TransmitError;
      u8  ReceiveError;
      u32 Reserved1;
      u32 Reserved2;
   } nxJ1939CommState_t;

   // Use with nxReadState nxState_TimeCurrent2, nxState_TimeCommunicating2,
   //    or nxState_TimeStart2;
   // Use with nxReadStateTimeTrigger;
   // Flags field uses constants with prefix nxTimeFlags_

#if defined(_WIN32) || defined(__linux__)
#pragma pack( push, 8 )
#endif

typedef struct _nxTimeLocalNetwork_t {
      nxTimestamp1ns_t LocalTime;
      nxTimestamp1ns_t NetworkTime;
      u32 Flags;
   } nxTimeLocalNetwork_t;

#if defined(_WIN32) || defined(__linux__)
#pragma pack( pop )
#endif

   // Destination MAC address uses null-terminated string of
   // syntax AA:BB:CC:DD:EE:FF
typedef char nxMACAddress_t[18];

   // The property nxPropSession_IntfEnetEptReceiveFilter uses an
   // array of this element type.
   // UseFlags field uses constants with prefix nxEptRxFilter_UseFlags_

#if defined(_WIN32) || defined(__linux__)
#pragma pack( push, 8 )
#endif

typedef struct _nxEptRxFilter_Element_t {
      u32 UseFlags;
      u16 VID;
      u8 Priority;
      nxMACAddress_t DestinationMAC;
   } nxEptRxFilter_Element_t;

#if defined(_WIN32) || defined(__linux__)
#pragma pack( pop )
#endif

/***********************************************************************
                                F R A M E
***********************************************************************/

   // Type
#define nxFrameType_CAN_Data                 0x00
#define nxFrameType_CAN_Remote               0x01
#define nxFrameType_CAN_BusError             0x02
#define nxFrameType_CAN20_Data               0x08
#define nxFrameType_CANFD_Data               0x10
#define nxFrameType_CANFDBRS_Data            0x18
#define nxFrameType_FlexRay_Data             0x20
#define nxFrameType_FlexRay_Null             0x21
#define nxFrameType_FlexRay_Symbol           0x22
#define nxFrameType_LIN_Data                 0x40
#define nxFrameType_LIN_BusError             0x41
#define nxFrameType_LIN_NoResponse           0x42
#define nxFrameType_J1939_Data               0xC0
#define nxFrameType_Special_Delay            0xE0
#define nxFrameType_Special_LogTrigger       0xE1
#define nxFrameType_Special_StartTrigger     0xE2


   /* For Data frames, your application may not be concerned with specifics for
   CAN, FlexRay, or LIN. For example, you can use fields of the frame to determine
   the contents of Payload, and write general-purpose code to map signal
   values in/out of the Payload data bytes.
   This macro can be used with the frame's Type to determine if the frame is a
   data frame. The macro is used in boolean conditionals. */
#define nxFrameType_IsData(frametype) \
            ((u8)(frametype) & 0x7) == 0)

#define nxFrameId_CAN_IsExtended             0x20000000

   // Macros to get fields of frame Identifier for FlexRay input
#define nxFrameId_FlexRay_Get_Slot(FrameId)        (u16)( ((u32)FrameId) & 0x0000FFFF)

   /* When Type is nxFrameType_FlexRay_Data,
   the following bitmasks are used with the Flags field.
   */
#define nxFrameFlags_FlexRay_Startup         0x01     // Startup frame
#define nxFrameFlags_FlexRay_Sync            0x02     // Sync frame
#define nxFrameFlags_FlexRay_Preamble        0x04     // Preamble bit
#define nxFrameFlags_FlexRay_ChA             0x10     // Transfer on Channel A
#define nxFrameFlags_FlexRay_ChB             0x20     // Transfer on Channel B

   /* When Type is nxFrameType_LIN_Data,
   the following bitmasks are used with the Flags field.
   */
#define nxFrameFlags_LIN_EventSlot           0x01     // Unconditional frame in event-triggered slot

   /* When Type is nxFrameType_CAN_Data, nxFrameType_CAN_Remote,
   nxFrameType_FlexRay_Data, or nxFrameType_LIN_Data,
   the following bitmasks are used with the Flags field.
   */
#define nxFrameFlags_TransmitEcho            0x80

   /* When Type is nxFrameType_FlexRay_Symbol,
   the following values are used with the first Payload byte (offset 0).
   */
#define nxFlexRaySymbol_MTS                  0x00
#define nxFlexRaySymbol_Wakeup               0x01

#define  nxInternal_PadPayload(paylod) \
            ( (u16)(paylod) ? (( (u16)(paylod) + 7) & 0x07F8) : 8)

#define  nxFrameFixed_t(payld) \
            struct { \
               nxTimestamp100ns_t  Timestamp; \
               u32                 Identifier; \
               u8                  Type; \
               u8                  Flags; \
               u8                  Info; \
               u8                  PayloadLength; \
               u8                  Payload[ nxInternal_PadPayload(payld) ]; \
            }

// -----------------------------------------------------------------------------
// If you are using CVI version 2009 or earlier, you may see a compile error for
// this line. Upgrade to CVI version 2010 or later for the fix, disable the
// "Build with C99 extensions" compiler option in the "Build Options" dialog of
// CVI or edit your copy of the header file to resolve the error.
// -----------------------------------------------------------------------------
typedef nxFrameFixed_t(8) nxFrameCAN_t;
typedef nxFrameFixed_t(8) nxFrameLIN_t;
typedef nxFrameFixed_t(1) nxFrameVar_t;

#define nxFramePayldLenHigh_Mask_J1939       (7)

#define nxSizeofFrameHeader                  (16)

#define nxFrameSize(payload) \
            (  nxSizeofFrameHeader + nxInternal_PadPayload(payload) )

   /* Get Payload Length. For J1939, the lowest three bits of the Info field
   are the high bits of the PayloadLength (valid up to 1785). */
#define nxFrameGetPayloadLength(frameptr) \
            (  (u32) (frameptr)->PayloadLength | \
               ((frameptr)->Type == nxFrameType_J1939_Data ? \
                  (u32) ((frameptr)->Info & nxFramePayldLenHigh_Mask_J1939) << 8 : \
                  0) \
            )



   /* Set Payload Length. For J1939, the lowest three bits of the Info field
      are the high bits of the PayloadLength (valid up to 1785 bytes). Before using this
      macro with J1939, the frame type should be set to nxFrameType_J1939_Data */
#define nxFrameSetPayloadLength(frameptr,PayloadLen) \
   (frameptr)->PayloadLength = (u8)((PayloadLen)&0xFF); \
   if (nxFrameType_J1939_Data == (frameptr)->Type ) { \
         (frameptr)->Info = (u8)( ( (frameptr)->Info & ( ~ nxFramePayldLenHigh_Mask_J1939 ) ) | \
         (( (PayloadLen) >> 8 ) & nxFramePayldLenHigh_Mask_J1939 ));}


   /* Use this macro to iterate through variable-length frames.
   You call this macro as a function, as if it used the following prototype:
      nxFrameVar_t * nxFrameIterate(nxFrameVar_t * frameptr);
   The input parameter must be initialized to point to the header of a valid frame.
   The macro returns a pointer to the header of the next frame in the buffer.
   In other words, the macro will iterate from one variable-length frame to
   the next variable-length frame.
   */
#define nxFrameIterate(frameptr) \
            (nxFrameVar_t *) ( (u8 *)(frameptr) + nxFrameSize( nxFrameGetPayloadLength(frameptr) ) )

// ******** Ethernet uses a distinct frame format ********

   // Values for Type field.
#define nxEnetFrameType_Data                 0x00
#define nxEnetFrameType_Delay                0xE1
#define nxEnetFrameType_FutureTimeWait       0xE2

   /* When Type is nxEnetFrameType_Data,
   the following bitmasks are used with the Flags field.
   */
#define nxEnetFlags_Transmit                 0x80000000
#define nxEnetFlags_Receive                  0x40000000
#define nxEnetFlags_NetworkSynced            0x00800000
#define nxEnetFlags_Error                    0x00010000

   /* Header that can be used for iteration of Ethernet frame data.
   A 32-bit Frame Check Sequence (FCS) is located after the FrameData.
   The number of bytes of FrameData varies for each Ethernet frame.
   The Length field is the length of FrameData plus 28 (for fields in
   nxFrameEnet_t plus the FCS).
   */

#if defined(_WIN32) || defined(__linux__)
#pragma pack( push, 1 )
#endif

   /* Macro to convert frame Length parameter in the Ethernet struct from network byte order to host order for nxFrameIterateEthernetWrite
   The length is typically in Big Endian byte order
   This macro only works for 2 byte / 16 bit values */

#define BigToHostOrder16(val)      ( (((val) >> 8) & 0x00FF) | (((val) << 8) & 0xFF00) )

   /* Ethernet-specific macro to iterate frames: 

   Use this macro to iterate through variable-length Ethernet frames.
   This macro functions similarly to nxFrameIterate (see description above).
   Note : Use nxFrameIterateEthernetWrite only for Ethernet frames during a frame write session */

#define nxFrameIterateEthernetWrite(frameptr)      (nxFrameEnet_t *) ( (u8 *)(frameptr) + BigToHostOrder16((frameptr->Length)))

   /* This macro is for variable Length Ethernet frames during read sessions, frame's Length 
   parameter during read session is already in host order not requiring conversion to network byte order, 
   Therefore the need for a separate macro
   Note : Use nxFrameIterateEthernetRead only for Ethernet frames during a frame read session */

#define nxFrameIterateEthernetRead(frameptr)      (nxFrameEnet_t *) ( (u8 *)(frameptr) + (frameptr->Length))

typedef struct _nxFrameEnet_t {
      u16 Length;
      u16 Type;
      nxTimestamp1ns_t DeviceTimestamp;
      nxTimestamp1ns_t NetworkTimestamp;
      u32 Flags;
      u8 FrameData[1];
   } nxFrameEnet_t;

#if defined(_WIN32) || defined(__linux__)
#pragma pack( pop )
#endif


#ifndef _NX_FUNCTION_EXPORT

/***********************************************************************
       F U N C T I O N   P R O T O T Y P E S  :  S E S S I O N
***********************************************************************/

nxStatus_t _NXFUNC nxCreateSession (
                           const char * DatabaseName,
                           const char * ClusterName,
                           const char * List,
                           const char * Interface,
                           u32 Mode,
                           nxSessionRef_t * SessionRef);

nxStatus_t _NXFUNC nxCreateSessionByRef (
                           u32 NumberOfDatabaseRef,
                           nxDatabaseRef_t * ArrayOfDatabaseRef,
                           const char * Interface,
                           u32 Mode,
                           nxSessionRef_t * SessionRef);

nxStatus_t _NXFUNC nxGetProperty (
                           nxSessionRef_t SessionRef,
                           u32 PropertyID,
                           u32 PropertySize,
                           void * PropertyValue);

nxStatus_t _NXFUNC nxGetPropertySize (
                           nxSessionRef_t SessionRef,
                           u32 PropertyID,
                           u32 * PropertySize);

nxStatus_t _NXFUNC nxSetProperty (
                           nxSessionRef_t SessionRef,
                           u32 PropertyID,
                           u32 PropertySize,
                           void * PropertyValue);

nxStatus_t _NXFUNC nxGetSubProperty (
                           nxSessionRef_t SessionRef,
                           u32 ActiveIndex,
                           u32 PropertyID,
                           u32 PropertySize,
                           void * PropertyValue);

nxStatus_t _NXFUNC nxGetSubPropertySize (
                           nxSessionRef_t SessionRef,
                           u32 ActiveIndex,
                           u32 PropertyID,
                           u32 * PropertySize);

nxStatus_t _NXFUNC nxSetSubProperty (
                           nxSessionRef_t SessionRef,
                           u32 ActiveIndex,
                           u32 PropertyID,
                           u32 PropertySize,
                           void * PropertyValue);

nxStatus_t _NXFUNC nxReadFrame (
                           nxSessionRef_t SessionRef,
                           void * Buffer,
                           u32 SizeOfBuffer,
                           f64 Timeout,
                           u32 * NumberOfBytesReturned);

nxStatus_t _NXFUNC nxReadSignalSinglePoint (
                           nxSessionRef_t SessionRef,
                           f64 * ValueBuffer,
                           u32 SizeOfValueBuffer,
                           nxTimestamp100ns_t * TimestampBuffer,
                           u32 SizeOfTimestampBuffer);

nxStatus_t _NXFUNC nxReadSignalWaveform (
                           nxSessionRef_t SessionRef,
                           f64 Timeout,
                           nxTimestamp100ns_t * StartTime,
                           f64 * DeltaTime,
                           f64 * ValueBuffer,
                           u32 SizeOfValueBuffer,
                           u32 * NumberOfValuesReturned);

nxStatus_t _NXFUNC nxReadSignalXY (
                           nxSessionRef_t SessionRef,
                           nxTimestamp100ns_t * TimeLimit,
                           f64 * ValueBuffer,
                           u32 SizeOfValueBuffer,
                           nxTimestamp100ns_t * TimestampBuffer,
                           u32 SizeOfTimestampBuffer,
                           u32 * NumPairsBuffer,
                           u32 SizeOfNumPairsBuffer);

nxStatus_t _NXFUNC nxReadState (
                           nxSessionRef_t SessionRef,
                           u32 StateID,
                           u32 StateSize,
                           void * StateValue,
                           nxStatus_t * Fault);

nxStatus_t _NXFUNC nxReadStateTimeTrigger (
                           nxSessionRef_t SessionRef,
                           f64 Timeout,
                           u32 StateSize,
                           void * StateValue);

nxStatus_t _NXFUNC nxWriteFrame (
                           nxSessionRef_t SessionRef,
                           void * Buffer,
                           u32 NumberOfBytesForFrames,
                           f64 Timeout);

nxStatus_t _NXFUNC nxWriteSignalSinglePoint (
                           nxSessionRef_t SessionRef,
                           f64 * ValueBuffer,
                           u32 SizeOfValueBuffer);

nxStatus_t _NXFUNC nxWriteState (
                           nxSessionRef_t SessionRef,
                           u32 StateID,
                           u32 StateSize,
                           void * StateValue);

nxStatus_t _NXFUNC nxWriteSignalWaveform (
                           nxSessionRef_t SessionRef,
                           f64 Timeout,
                           f64 * ValueBuffer,
                           u32 SizeOfValueBuffer);

nxStatus_t _NXFUNC nxWriteSignalXY (
                           nxSessionRef_t SessionRef,
                           f64 Timeout,
                           f64 * ValueBuffer,
                           u32 SizeOfValueBuffer,
                           nxTimestamp100ns_t * TimestampBuffer,
                           u32 SizeOfTimestampBuffer,
                           u32 * NumPairsBuffer,
                           u32 SizeOfNumPairsBuffer);

nxStatus_t _NXFUNC nxConvertFramesToSignalsSinglePoint (
                           nxSessionRef_t SessionRef,
                           void * FrameBuffer,
                           u32 NumberOfBytesForFrames,
                           f64 * ValueBuffer,
                           u32 SizeOfValueBuffer,
                           nxTimestamp100ns_t * TimestampBuffer,
                           u32 SizeOfTimestampBuffer);

nxStatus_t _NXFUNC nxConvertSignalsToFramesSinglePoint (
                           nxSessionRef_t SessionRef,
                           f64 * ValueBuffer,
                           u32 SizeOfValueBuffer,
                           void * Buffer,
                           u32 SizeOfBuffer,
                           u32 * NumberOfBytesReturned);

nxStatus_t _NXFUNC nxConvertFramesToByteArraySinglePoint (
                           nxSessionRef_t SessionRef,
                           void * FrameBuffer,
                           u32 NumberOfBytesForFrames,
                           u8 * ValueBuffer,
                           u32 SizeOfValueBuffer);

nxStatus_t _NXFUNC nxConvertByteArrayToFramesSinglePoint (
                           nxSessionRef_t SessionRef,
                           u8 * ValueBuffer,
                           u32 SizeOfValueBuffer,
                           void * Buffer,
                           u32 SizeOfBuffer,
                           u32 * NumberOfBytesReturned);

nxStatus_t _NXFUNC nxConvertTimestamp100nsTo1ns (
                           nxTimestamp100ns_t From,
                           nxTimestamp1ns_t * To);

nxStatus_t _NXFUNC nxConvertTimestamp1nsTo100ns (
                           nxTimestamp1ns_t From,
                           nxTimestamp100ns_t * To);


nxStatus_t _NXFUNC nxBlink (
                           nxSessionRef_t InterfaceRef,
                           u32 Modifier);

nxStatus_t _NXFUNC nxClear (
                           nxSessionRef_t SessionRef);

nxStatus_t _NXFUNC nxConnectTerminals (
                           nxSessionRef_t SessionRef,
                           const char * source,
                           const char * destination);


nxStatus_t _NXFUNC nxDisconnectTerminals (
                           nxSessionRef_t SessionRef,
                           const char * source,
                           const char * destination);

nxStatus_t _NXFUNC nxFlush (
                           nxSessionRef_t SessionRef);

nxStatus_t _NXFUNC nxStart (
                           nxSessionRef_t SessionRef,
                           u32 Scope);

nxStatus_t _NXFUNC nxStop (
                           nxSessionRef_t SessionRef,
                           u32 Scope);

void _NXFUNC nxStatusToString (
                           nxStatus_t Status,
                           u32 SizeofString,
                           char * StatusDescription);

nxStatus_t _NXFUNC nxSystemOpen (
                           nxSessionRef_t * SystemRef);

nxStatus_t _NXFUNC nxSystemClose (
                           nxSessionRef_t SystemRef);

nxStatus_t _NXFUNC nxWait (
                           nxSessionRef_t SessionRef,
                           u32 Condition,
                           u32 ParamIn,
                           f64 Timeout,
                           u32 * ParamOut);

nxStatus_t _NXFUNC nxFutureTimeTrigger (
                           nxSessionRef_t SessionRef,
                           nxTimestamp1ns_t When,
                           u32 Timescale);


/***********************************************************************
       F U N C T I O N   P R O T O T Y P E S  :  D A T A B A S E
***********************************************************************/

nxStatus_t _NXFUNC nxdbOpenDatabase (
                           const char * DatabaseName,
                           nxDatabaseRef_t * DatabaseRef);

nxStatus_t _NXFUNC nxdbCloseDatabase (
                           nxDatabaseRef_t DatabaseRef,
                           u32 CloseAllRefs);

nxStatus_t _NXFUNC nxdbCreateObject (
                           nxDatabaseRef_t ParentObjectRef,
                           u32 ObjectClass,
                           const char * ObjectName,
                           nxDatabaseRef_t * DbObjectRef);

nxStatus_t _NXFUNC nxdbFindObject (
                           nxDatabaseRef_t ParentObjectRef,
                           u32 ObjectClass,
                           const char * ObjectName,
                           nxDatabaseRef_t * DbObjectRef);

nxStatus_t _NXFUNC nxdbDeleteObject (
                           nxDatabaseRef_t DbObjectRef);

nxStatus_t _NXFUNC nxdbSaveDatabase (
                           nxDatabaseRef_t DatabaseRef,
                           const char * DbFilepath);

nxStatus_t _NXFUNC nxdbGetProperty (
                           nxDatabaseRef_t DbObjectRef,
                           u32 PropertyID,
                           u32 PropertySize,
                           void * PropertyValue);

nxStatus_t _NXFUNC nxdbGetPropertySize (
                           nxDatabaseRef_t DbObjectRef,
                           u32 PropertyID,
                           u32 * PropertySize);

nxStatus_t _NXFUNC nxdbSetProperty (
                           nxDatabaseRef_t DbObjectRef,
                           u32 PropertyID,
                           u32 PropertySize,
                           void * PropertyValue);

/* The NI-XNET documentation does not describe the Mode parameter.
   The Mode parameter was added near release. It specifies the type of DBC attribute
   that you want to search for. In the v1.2 release there is only one value:
      nxGetDBCMode_Attribute (0)
   Other values will be supported in subsequent releases. */
nxStatus_t _NXFUNC nxdbGetDBCAttributeSize (
                           nxDatabaseRef_t DbObjectRef,
                           u32 Mode,
                           const char* AttributeName,
                           u32 *AttributeTextSize);

/* The NI-XNET documentation does not describe the Mode parameter.
   The Mode parameter was added near release. It specifies the type of DBC attribute
   that you want to search for. In the v1.2 release there is only one value:
      nxGetDBCMode_Attribute (0)
   Other values will be supported in subsequent releases. */
nxStatus_t _NXFUNC nxdbGetDBCAttribute (
                           nxDatabaseRef_t DbObjectRef,
                           u32 Mode,
                           const char* AttributeName,
                           u32 AttributeTextSize,
                           char* AttributeText,
                           u32 * IsDefault);

nxStatus_t _NXFUNC nxdbMerge (
                           nxDatabaseRef_t TargetClusterRef,
                           nxDatabaseRef_t SourceObjRef,
                           u32 CopyMode,
                           char *Prefix,
                           u32 WaitForComplete,
                           u32 *PercentComplete);


nxStatus_t _NXFUNC nxdbAddAlias (
                           const char * DatabaseAlias,
                           const char * DatabaseFilepath,
                           u32          DefaultBaudRate);

nxStatus_t _NXFUNC nxdbAddAlias64 (
                           const char * DatabaseAlias,
                           const char * DatabaseFilepath,
                           u64          DefaultBaudRate);

nxStatus_t _NXFUNC nxdbRemoveAlias (
                           const char * DatabaseAlias);

nxStatus_t _NXFUNC nxdbDeploy (
                           const char * IPAddress,
                           const char * DatabaseAlias,
                           u32 WaitForComplete,
                           u32 * PercentComplete);

nxStatus_t _NXFUNC nxdbUndeploy (
                           const char * IPAddress,
                           const char * DatabaseAlias);

nxStatus_t _NXFUNC nxdbGetDatabaseList (
                           const char * IPAddress,
                           u32 SizeofAliasBuffer,
                           char * AliasBuffer,
                           u32 SizeofFilepathBuffer,
                           char * FilepathBuffer,
                           u32 * NumberOfDatabases);

nxStatus_t _NXFUNC nxdbGetDatabaseListSizes (
                           const char * IPAddress,
                           u32 * SizeofAliasBuffer,
                           u32 * SizeofFilepathBuffer);

#endif // _NX_FUNCTION_EXPORT

#ifdef __cplusplus
   /* See top of header file.  */
   }
#endif // __cplusplus

#endif // ___nixnet_h___
