Unicode True

!include "MUI2.nsh"
!include "nsDialogs.nsh"
!include "LogicLib.nsh"

; Variables for service installation options
Var InstallService
Var ServiceAutoStart
Var DataDir
Var Dialog
Var ServiceCheckbox
Var AutoStartCheckbox
Var DataDirText
Var DataDirLabel

!insertmacro MUI_PAGE_WELCOME
Page custom ServiceOptionsPage ServiceOptionsPageLeave
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Name "Synnax"
OutFile "synnax-setup-v${VERSION}.exe"
RequestExecutionLevel admin
InstallDir "$PROGRAMFILES64\Synnax\Core"
ShowInstDetails show

; Custom page for service installation options
Function ServiceOptionsPage
    !insertmacro MUI_HEADER_TEXT "Service Options" "Configure Synnax as a Windows Service"

    nsDialogs::Create 1018
    Pop $Dialog
    ${If} $Dialog == error
        Abort
    ${EndIf}

    ${NSD_CreateCheckbox} 0 0 100% 12u "Install Synnax as a Windows Service"
    Pop $ServiceCheckbox
    ${NSD_SetState} $ServiceCheckbox ${BST_CHECKED}

    ${NSD_CreateCheckbox} 0 20u 100% 12u "Start service automatically on boot"
    Pop $AutoStartCheckbox
    ${NSD_SetState} $AutoStartCheckbox ${BST_CHECKED}

    ${NSD_CreateLabel} 0 45u 100% 12u "Data directory:"
    Pop $DataDirLabel

    ${NSD_CreateText} 0 58u 100% 12u "$LOCALAPPDATA\Synnax\data"
    Pop $DataDirText

    ${NSD_CreateLabel} 0 78u 100% 36u "If checked, Synnax will run in the background as a Windows Service. The service can be managed via 'synnax service' commands or the Windows Services console (services.msc)."

    nsDialogs::Show
FunctionEnd

Function ServiceOptionsPageLeave
    ${NSD_GetState} $ServiceCheckbox $InstallService
    ${NSD_GetState} $AutoStartCheckbox $ServiceAutoStart
    ${NSD_GetText} $DataDirText $DataDir
FunctionEnd

Section "MainSection" SEC01
    DetailPrint "Resolved INSTALL DIR: $INSTDIR"

    CreateDirectory "$INSTDIR"
    DetailPrint "Installing to: $INSTDIR"

    ; Stop and remove existing service if present
    DetailPrint "Checking for existing service..."
    nsExec::ExecToStack '"$INSTDIR\synnax-server.exe" service stop'
    Pop $0
    ; Wait for graceful shutdown
    Sleep 6000
    nsExec::ExecToStack '"$INSTDIR\synnax-server.exe" service uninstall'
    Pop $0

    Delete "$INSTDIR\synnax-server.exe"
    Delete "$INSTDIR\synnax.bat"

    SetOutPath "$INSTDIR"
    File /oname=synnax-server.exe "synnax-server.exe"

    # Create batch file alias
    FileOpen $0 "$INSTDIR\synnax.bat" w
    FileWrite $0 "@echo off$\r$\n"
    FileWrite $0 "synnax-server.exe %*$\r$\n"
    FileClose $0

    # Create data directory
    CreateDirectory "$DataDir"

    # Add to system PATH using EnVar (requires EnVar.dll in x86-unicode folder)
    DetailPrint "Adding $INSTDIR to system PATH..."
    EnVar::SetHKLM
    EnVar::AddValue "PATH" "$INSTDIR"
    Pop $0
    DetailPrint "EnVar::AddValue (system PATH) returned: $0"

    # Install as service if selected
    ${If} $InstallService == ${BST_CHECKED}
        DetailPrint "Installing Synnax as Windows Service..."

        ${If} $ServiceAutoStart == ${BST_CHECKED}
            nsExec::ExecToLog '"$INSTDIR\synnax-server.exe" service install --data "$DataDir" --auto-start --insecure'
        ${Else}
            nsExec::ExecToLog '"$INSTDIR\synnax-server.exe" service install --data "$DataDir" --insecure'
        ${EndIf}
        Pop $0

        ${If} $0 == 0
            DetailPrint "Service installed successfully"

            ; Start the service
            DetailPrint "Starting Synnax service..."
            nsExec::ExecToLog '"$INSTDIR\synnax-server.exe" service start'
            Pop $0
        ${Else}
            DetailPrint "Warning: Service installation returned: $0"
        ${EndIf}
    ${EndIf}

    WriteUninstaller "$INSTDIR\uninstall.exe"

    # Add to Add/Remove Programs
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Synnax" \
                     "DisplayName" "Synnax Server"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Synnax" \
                     "UninstallString" "$INSTDIR\uninstall.exe"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Synnax" \
                     "InstallLocation" "$INSTDIR"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Synnax" \
                     "Publisher" "Synnax Labs"
    WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Synnax" \
                     "DisplayVersion" "${VERSION}"
SectionEnd

Section "Uninstall"
    ; Stop and uninstall service first
    DetailPrint "Stopping Synnax service..."
    nsExec::ExecToStack '"$INSTDIR\synnax-server.exe" service stop'
    Pop $0
    ; Wait for graceful shutdown
    Sleep 6000

    DetailPrint "Uninstalling Synnax service..."
    nsExec::ExecToStack '"$INSTDIR\synnax-server.exe" service uninstall'
    Pop $0

    DetailPrint "Removing $INSTDIR from system PATH..."
    EnVar::SetHKLM
    EnVar::DeleteValue "PATH" "$INSTDIR"
    Pop $0
    StrCmp $0 5 +2 0
    DetailPrint "Note: $INSTDIR was not in system PATH, nothing to remove."
    DetailPrint "EnVar::DeleteValue (system PATH) returned: $0"

    Delete "$INSTDIR\synnax-server.exe"
    Delete "$INSTDIR\synnax.bat"
    Delete "$INSTDIR\uninstall.exe"
    Delete "$DESKTOP\Synnax.lnk"
    Delete "$SMPROGRAMS\Synnax\Synnax.lnk"

    ; Remove registry entries
    DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\Synnax"

    RMDir "$SMPROGRAMS\Synnax"
    RMDir "$INSTDIR"
SectionEnd