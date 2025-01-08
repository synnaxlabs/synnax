!include "MUI2.nsh"

# Define pages first
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

# Then language
!insertmacro MUI_LANGUAGE "English"

Name "Synnax Server"
OutFile "synnax-server-v${VERSION}-windows.exe"
RequestExecutionLevel user

# Set installation directory explicitly
InstallDir "$APPDATA\synnax"

Section "MainSection" SEC01
    # Create and clean install directory
    CreateDirectory "$INSTDIR"
    DetailPrint "Installing to: $INSTDIR"

    Delete "$INSTDIR\synnax-server.exe"
    Delete "$INSTDIR\python311.dll"

    # Install files
    SetOutPath "$INSTDIR"
    File /oname=synnax-server.exe "synnax.exe"
    File "python311.dll"

    # Create shortcuts
    CreateDirectory "$SMPROGRAMS\Synnax Server"
    CreateShortcut "$SMPROGRAMS\Synnax Server\Synnax Server.lnk" "$INSTDIR\synnax-server.exe"
    CreateShortcut "$DESKTOP\Synnax Server.lnk" "$INSTDIR\synnax-server.exe"

    # Add to PATH using EnVar plugin
    EnVar::SetHKCU
    EnVar::AddValue "Path" "$INSTDIR"
    Pop $0
    DetailPrint "EnVar::AddValue returned=$0"

    WriteUninstaller "$INSTDIR\uninstall.exe"
SectionEnd

Section "Uninstall"
    # Remove from PATH using EnVar plugin
    EnVar::SetHKCU
    EnVar::DeleteValue "Path" "$INSTDIR"

    # Remove files and directories
    Delete "$INSTDIR\synnax-server.exe"
    Delete "$INSTDIR\python311.dll"
    Delete "$INSTDIR\uninstall.exe"
    Delete "$DESKTOP\Synnax Server.lnk"
    Delete "$SMPROGRAMS\Synnax Server\Synnax Server.lnk"
    RMDir "$SMPROGRAMS\Synnax Server"
    RMDir "$INSTDIR"
SectionEnd
