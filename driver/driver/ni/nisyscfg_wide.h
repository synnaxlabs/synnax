/*============================================================================*/
/*                       NI System Configuration API                          */
/*----------------------------------------------------------------------------*/
/*    Copyright (c) National Instruments 2010-2023.  All Rights Reserved.     */
/*----------------------------------------------------------------------------*/
/*                                                                            */
/* Title:       nisyscfg_wide.h                                               */
/* Purpose:     Wide entry points for NI System Configuration API functions   */
/*                                                                            */
/*============================================================================*/

#ifndef ___nisysconfig_wide_h___
#define ___nisysconfig_wide_h___

#include "nisyscfg.h"

#if defined(__cplusplus) || defined(__cplusplus__)
extern "C" {
#endif

   /**************************************************************************/
   /* Core System Configuration functions                                    */
   /**************************************************************************/

   NISYSCFGCFUNC NISysCfgInitializeSessionW(
      const wchar_t *                       targetName,
      const wchar_t *                       username,
      const wchar_t *                       password,
      NISysCfgLocale                        language,
      NISysCfgBool                          forcePropertyRefresh,
      unsigned int                          connectTimeoutMsec,
      NISysCfgEnumExpertHandle *            expertsEnumHandle,
      NISysCfgSessionHandle *               sessionHandle
      );

   NISYSCFGCFUNC NISysCfgGetSystemExpertsW(
      NISysCfgSessionHandle                 sessionHandle,
      const wchar_t *                       expertNames,
      NISysCfgEnumExpertHandle *            expertEnumHandle
      );

   NISYSCFGCFUNC NISysCfgFindHardwareW(
      NISysCfgSessionHandle                 sessionHandle,
      NISysCfgFilterMode                    filterMode,
      NISysCfgFilterHandle                  filterHandle,
      const wchar_t *                       expertNames,
      NISysCfgEnumResourceHandle *          resourceEnumHandle
      );

   NISYSCFGCFUNC NISysCfgFindSystemsW(
      NISysCfgSessionHandle                 sessionHandle,
      const wchar_t *                       deviceClass,
      NISysCfgBool                          detectOnlineSystems,
      NISysCfgIncludeCachedResults          cacheMode,
      NISysCfgSystemNameFormat              findOutputMode,
      unsigned int                          timeoutMsec,
      NISysCfgBool                          onlyInstallableSystems,
      NISysCfgEnumSystemHandle *            systemEnumHandle
      );

   // Caller should free result using NISysCfgFreeDetailedStringW
   NISYSCFGCFUNC NISysCfgSelfTestHardwareW(
      NISysCfgResourceHandle                resourceHandle,
      unsigned int                          mode,
      wchar_t **                            detailedResult
      );

   // Caller should free result using NISysCfgFreeDetailedStringW
   NISYSCFGCFUNC NISysCfgSelfCalibrateHardwareW(
      NISysCfgResourceHandle                resourceHandle,
      wchar_t **                            detailedResult
      );

   NISYSCFGCFUNC NISysCfgRenameResourceW(
      NISysCfgResourceHandle                resourceHandle,
      const wchar_t *                       newName,
      NISysCfgBool                          overwriteIfExists,
      NISysCfgBool                          updateDependencies,
      NISysCfgBool *                        nameAlreadyExisted,
      NISysCfgResourceHandle *              overwrittenResourceHandle
      );

   NISYSCFGCFUNC NISysCfgDeleteResourceW(
      NISysCfgResourceHandle                 resourceHandle,
      NISysCfgDeleteValidationMode           mode,
      NISysCfgBool *                         dependentItemsDeleted,
      wchar_t **                             detailedResult
      );

   NISYSCFGCFUNC NISysCfgGetResourcePropertyW(
      NISysCfgResourceHandle                resourceHandle,
      NISysCfgResourceProperty              propertyID,
      void *                                value
      );

   NISYSCFGCDECL NISysCfgSetResourcePropertyW(
      NISysCfgResourceHandle                resourceHandle,
      NISysCfgResourceProperty              propertyID,
      ...
      );

   NISYSCFGCDECL NISysCfgSetResourcePropertyWithTypeW(
      NISysCfgResourceHandle                resourceHandle,
      NISysCfgResourceProperty              propertyID,
      NISysCfgPropertyType                  propertyType,
      ...
      );

   NISYSCFGCFUNC NISysCfgGetResourceIndexedPropertyW(
      NISysCfgResourceHandle                resourceHandle,
      NISysCfgIndexedProperty               propertyID,
      unsigned int                          index,
      void *                                value
      );

   typedef void (NISYSCFGCONV NISysCfgResourcePropertyUserCallback)
      (NISysCfgIndexedProperty propertyID, NISysCfgPropertyType propertyType, void* value);

   NISYSCFGCFUNC NISysCfgGetResourcePropertiesW(
      NISysCfgResourceHandle                resourceHandle,
      NISysCfgResourcePropertyUserCallback  userCallback,
      int                                   numProperties,
      const NISysCfgResourceProperty        propertyIDs[]
      );

   // Caller should free result using NISysCfgFreeDetailedStringW
   NISYSCFGCFUNC NISysCfgSaveResourceChangesW(
      NISysCfgResourceHandle                resourceHandle,
      NISysCfgBool *                        changesRequireRestart,
      wchar_t **                            detailedResult
      );

   NISYSCFGCFUNC NISysCfgGetSystemPropertyW(
      NISysCfgSessionHandle                 sessionHandle,
      NISysCfgSystemProperty                propertyID,
      void *                                value
      );

   NISYSCFGCDECL NISysCfgSetSystemPropertyW(
      NISysCfgSessionHandle                 sessionHandle,
      NISysCfgSystemProperty                propertyID,
      ...
      );

   // Caller should free result using NISysCfgFreeDetailedStringW
   NISYSCFGCFUNC NISysCfgSaveSystemChangesW(
      NISysCfgSessionHandle                 sessionHandle,
      NISysCfgBool *                        changesRequireRestart,
      wchar_t **                            detailedResult
      );

   NISYSCFGCDECL NISysCfgSetFilterPropertyW(
      NISysCfgFilterHandle                  filterHandle,
      NISysCfgFilterProperty                propertyID,
      ...
      );

   NISYSCFGCDECL NISysCfgSetFilterPropertyWithTypeW(
      NISysCfgFilterHandle                  filterHandle,
      NISysCfgFilterProperty                propertyID,
      NISysCfgPropertyType                  propertyType,
      ...
      );

   NISYSCFGCFUNC NISysCfgRestartW(
      NISysCfgSessionHandle                 sessionHandle,
      NISysCfgBool                          waitForRestartToFinish,
      NISysCfgBool                          installMode,
      NISysCfgBool                          flushDNS,
      unsigned int                          timeoutMsec,
      wchar_t                               newIpAddress[]
      );

   NISYSCFGCFUNC NISysCfgGetSystemImageAsFolderW(
      NISysCfgSessionHandle                  sessionHandle,
      NISysCfgBool                           autoRestart,
      const wchar_t *                        destinationFolder,
      const wchar_t *                        encryptionPassphrase,
      unsigned int                           numBlacklistEntries,
      const wchar_t **                       blacklistFilesDirectories,
      NISysCfgBool                           overwriteDestination,
      NISysCfgBool                           installedSoftwareOnly
      );

   NISYSCFGCFUNC NISysCfgCreateSystemImageAsFolderW(
      NISysCfgSessionHandle                  sessionHandle,
      const wchar_t *                        imageTitle,
      const wchar_t *                        imageID,
      const wchar_t *                        imageVersion,
      const wchar_t *                        imageDescription,
      NISysCfgBool                           autoRestart,
      const wchar_t *                        destinationFolder,
      const wchar_t *                        encryptionPassphrase,
      unsigned int                           numBlacklistEntries,
      const wchar_t **                       blacklistFilesDirectories,
      NISysCfgBool                           overwriteIfExists
      );

   NISYSCFGCFUNC NISysCfgSetSystemImageFromFolderW(
      NISysCfgSessionHandle                  sessionHandle,
      NISysCfgBool                           autoRestart,
      const wchar_t *                        sourceFolder,
      const wchar_t *                        encryptionPassphrase,
      unsigned int                           numBlacklistEntries,
      const wchar_t **                       blacklistFilesDirectories,
      NISysCfgBool                           originalSystemOnly,
      NISysCfgNetworkInterfaceSettings       networkSettings
      );

   NISYSCFGCFUNC NISysCfgGetFilteredSoftwareComponentsW(
      const wchar_t *                        repositoryPath,
      const wchar_t *                        deviceClass,
      const wchar_t *                        operatingSystem,
      unsigned int                           productID,
      NISysCfgIncludeComponentTypes          itemTypes,
      NISysCfgEnumSoftwareComponentHandle *  componentEnumHandle
      );

   NISYSCFGCFUNC NISysCfgGetFilteredSoftwareSetsW(
      const wchar_t *                        repositoryPath,
      const wchar_t *                        deviceClass,
      const wchar_t *                        operatingSystem,
      unsigned int                           productID,
      NISysCfgEnumSoftwareSetHandle *        setEnumHandle
      );

   NISYSCFGCFUNC NISysCfgGetFilteredBaseSystemImagesW(
      const wchar_t *                        repositoryPath,
      const wchar_t *                        deviceClass,
      const wchar_t *                        operatingSystem,
      unsigned int                           productID,
      NISysCfgEnumSoftwareComponentHandle *  systemImageEnumHandle
      );

   NISYSCFGCFUNC NISysCfgInstallUninstallComponentsW(
      NISysCfgSessionHandle                 sessionHandle,
      NISysCfgBool                          autoRestart,
      NISysCfgBool                          autoSelectDependencies,
      NISysCfgEnumSoftwareComponentHandle   componentsToInstallHandle,
      unsigned int                          numComponentsToUninstall,
      const wchar_t **                      componentIDsToUninstall,
      NISysCfgEnumDependencyHandle*         brokenDependencyEnumHandle
      );

   NISYSCFGCFUNC NISysCfgInstallUninstallComponents2W(
      NISysCfgSessionHandle                 sessionHandle,
      NISysCfgBool                          autoRestart,
      NISysCfgBool                          autoSelectDependencies,
      NISysCfgBool                          autoSelectRecommends,
      NISysCfgEnumSoftwareComponentHandle   componentsToInstallHandle,
      unsigned int                          numComponentsToUninstall,
      const wchar_t **                      componentIDsToUninstall,
      NISysCfgEnumDependencyHandle*         brokenDependencyEnumHandle
      );

   NISYSCFGCFUNC NISysCfgInstallSoftwareSetW(
      NISysCfgSessionHandle                 sessionHandle,
      NISysCfgBool                          autoRestart,
      const wchar_t *                       softwareSetID,
      const wchar_t *                       version,
      NISysCfgEnumSoftwareComponentHandle   addonsToInstallHandle,
      NISysCfgEnumDependencyHandle*         brokenDependencyEnumHandle
      );

   NISYSCFGCFUNC NISysCfgFormatWithBaseSystemImageW(
      NISysCfgSessionHandle                 sessionHandle,
      NISysCfgBool                          autoRestart,
      NISysCfgFileSystemMode                fileSystem,
      NISysCfgNetworkInterfaceSettings      networkSettings,
      const wchar_t *                       systemImageID,
      const wchar_t *                       systemImageVersion,
      unsigned int                          timeoutMsec
      );

   NISYSCFGCFUNC NISysCfgExportConfigurationW(
      NISysCfgSessionHandle                 sessionHandle,
      const wchar_t *                       destinationFile,
      const wchar_t *                       expertNames,
      NISysCfgBool                          overwriteIfExists
      );

   NISYSCFGCFUNC NISysCfgImportConfigurationW(
      NISysCfgSessionHandle                 sessionHandle,
      const wchar_t *                       sourceFile,
      const wchar_t *                       expertNames,
      NISysCfgImportMode                    importMode,
      wchar_t **                            detailedResult
      );

   NISYSCFGCFUNC NISysCfgGenerateMAXReportW(
      NISysCfgSessionHandle                 sessionHandle,
      const wchar_t *                       outputFilename,
      NISysCfgReportType                    reportType,
      NISysCfgBool                          overwrite
      );

   NISYSCFGCFUNC NISysCfgAddSoftwareFeedW(
      NISysCfgSessionHandle                 sessionHandle,
      const wchar_t *                       feedName,
      const wchar_t *                       uri,
      NISysCfgBool                          enabled,
      NISysCfgBool                          trusted
      );

   NISYSCFGCFUNC NISysCfgModifySoftwareFeedW(
      NISysCfgSessionHandle                 sessionHandle,
      const wchar_t *                       feedName,
      const wchar_t *                       newFeedName,
      const wchar_t *                       uri,
      NISysCfgBool                          enabled,
      NISysCfgBool                          trusted
      );

   NISYSCFGCFUNC NISysCfgRemoveSoftwareFeedW(
      NISysCfgSessionHandle                 sessionHandle,
      const wchar_t *                       feedName
      );

   /**************************************************************************/
   /* System Configuration firmware modification functions                   */
   /**************************************************************************/

   // Caller should free result using NISysCfgFreeDetailedStringW
   NISYSCFGCFUNC NISysCfgUpgradeFirmwareFromFileW(
      NISysCfgResourceHandle                 resourceHandle,
      const wchar_t *                        firmwareFile,
      NISysCfgBool                           autoStopTasks,
      NISysCfgBool                           alwaysOverwrite,
      NISysCfgBool                           waitForOperationToFinish,
      NISysCfgFirmwareStatus *               firmwareStatus,
      wchar_t **                             detailedResult
      );

   // Caller should free result using NISysCfgFreeDetailedStringW
   NISYSCFGCFUNC NISysCfgUpgradeFirmwareVersionW(
      NISysCfgResourceHandle                 resourceHandle,
      const wchar_t *                        firmwareVersion,
      NISysCfgBool                           autoStopTasks,
      NISysCfgBool                           alwaysOverwrite,
      NISysCfgBool                           waitForOperationToFinish,
      NISysCfgFirmwareStatus *               firmwareStatus,
      wchar_t **                             detailedResult
      );

   // Caller should free result using NISysCfgFreeDetailedStringW
   NISYSCFGCFUNC NISysCfgEraseFirmwareW(
      NISysCfgResourceHandle                 resourceHandle,
      NISysCfgBool                           autoStopTasks,
      NISysCfgFirmwareStatus *               firmwareStatus,
      wchar_t **                             detailedResult
      );

   // Caller should free result using NISysCfgFreeDetailedStringW
   NISYSCFGCFUNC NISysCfgCheckFirmwareStatusW(
      NISysCfgResourceHandle                 resourceHandle,
      int *                                  percentComplete,
      NISysCfgFirmwareStatus *               firmwareStatus,
      wchar_t **                             detailedResult
      );

   // Caller should free detailedResult using NISysCfgFreeDetailedString.
   NISYSCFGCFUNC NISysCfgActivateFeatureW(
      NISysCfgResourceHandle                 resourceHandle,
      unsigned int                           featureID,
      const wchar_t *                        activationCode,
      wchar_t **                             detailedResult
      );

   /**************************************************************************/
   /* System Configuration enumerators and utility functions                 */
   /**************************************************************************/

   NISYSCFGCFUNC NISysCfgChangeAdministratorPasswordW(
      NISysCfgSessionHandle                 sessionHandle,
      const wchar_t *                       newPassword
      );

   NISYSCFGCFUNC NISysCfgAddComponentToEnumW(
      NISysCfgEnumSoftwareComponentHandle   componentsHandle,
      const wchar_t *                       ID,
      const wchar_t *                       version,
      NISysCfgVersionSelectionMode          mode
      );

   NISYSCFGCFUNC NISysCfgFreeDetailedStringW(
      wchar_t                               str[]
      );

   NISYSCFGCFUNC NISysCfgNextSystemInfoW(
      NISysCfgEnumSystemHandle              systemEnumHandle,
      wchar_t                               system[]
      );

   NISYSCFGCFUNC NISysCfgNextExpertInfoW(
      NISysCfgEnumExpertHandle              expertEnumHandle,
      wchar_t                               expertName[],
      wchar_t                               displayName[],
      wchar_t                               version[]
      );

   // Caller should free detailedDescription using NISysCfgFreeDetailedStringW
   NISYSCFGCFUNC NISysCfgNextComponentInfoW(
      NISysCfgEnumSoftwareComponentHandle   componentsEnumHandle,
      wchar_t                               ID[],
      wchar_t                               version[],
      wchar_t                               title[],
      NISysCfgComponentType *               itemType,
      wchar_t **                            detailedDescription
      );

   // Caller should free detailedDescription using NISysCfgFreeDetailedStringW
   NISYSCFGCFUNC NISysCfgGetSoftwareSetInfoW(
      NISysCfgSoftwareSetHandle             setHandle,
      NISysCfgIncludeComponentTypes         itemTypes,
      NISysCfgBool                          includeAddOnDeps,
      wchar_t                               ID[],
      wchar_t                               version[],
      wchar_t                               title[],
      NISysCfgComponentType *               setType,
      wchar_t **                            detailedDescription,
      NISysCfgEnumSoftwareComponentHandle * addOnsHandle,
      NISysCfgEnumSoftwareComponentHandle * itemsHandle
      );

   // Caller should free detailedDescription using NISysCfgFreeDetailedStringW
   NISYSCFGCFUNC NISysCfgNextDependencyInfoW(
      NISysCfgEnumDependencyHandle          dependencyEnumHandle,
      wchar_t                               dependerID[],
      wchar_t                               dependerVersion[],
      wchar_t                               dependerTitle[],
      wchar_t **                            dependerDetailedDescription,
      wchar_t                               dependeeID[],
      wchar_t                               dependeeVersion[],
      wchar_t                               dependeeTitle[],
      wchar_t **                            dependeeDetailedDescription
      );

   NISYSCFGCFUNC NISysCfgNextSoftwareFeedW(
      NISysCfgEnumSoftwareFeedHandle        feedEnumHandle,
      wchar_t                               feedName[],
      wchar_t                               uri[],
      NISysCfgBool *                        enabled,
      NISysCfgBool *                        trusted
      );

   NISYSCFGCFUNC NISysCfgGetStatusDescriptionW(
      NISysCfgSessionHandle                 sessionHandle,
      NISysCfgStatus                        status,
      wchar_t **                            detailedDescription
      );

#ifdef __cplusplus
}
#endif

#endif
