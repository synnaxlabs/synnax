Unicode True

!include "MUI2.nsh"
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Name "Synnax"
OutFile "synnax-setup-v${VERSION}.exe"
RequestExecutionLevel user
InstallDir "$APPDATA\synnax"
ShowInstDetails show

Section "MainSection" SEC01
    CreateDirectory "$INSTDIR"
    DetailPrint "Installing to: $INSTDIR"
    Delete "$INSTDIR\synnax-server.exe"
    Delete "$INSTDIR\synnax.bat"

    SetOutPath "$INSTDIR"
    File /oname=synnax-server.exe "synnax-server.exe"

    # Create batch file alias
    FileOpen $0 "$INSTDIR\synnax.bat" w
    FileWrite $0 "@echo off$\r$\n"
    FileWrite $0 "synnax-server.exe %*$\r$\n"
    FileClose $0

    # Create shortcuts
    CreateDirectory "$SMPROGRAMS\Synnax"
    CreateShortcut "$SMPROGRAMS\Synnax\Synnax.lnk" "$INSTDIR\synnax-server.exe"
    CreateShortcut "$DESKTOP\Synnax.lnk" "$INSTDIR\synnax-server.exe"

    # Add to user PATH using EnVar
    DetailPrint "Adding $INSTDIR to user PATH..."
    EnVar::AddValue "PATH" "$INSTDIR"
    Pop $0
    DetailPrint "EnVar::AddValue returned: $0"

    # Broadcast environment change to notify Explorer and new processes
    System::Call 'user32::SendMessageTimeoutA(i 0xffff, i ${WM_SETTINGCHANGE}, i 0, t "Environment", i 0, i 5000, *i .r0)'

    WriteUninstaller "$INSTDIR\uninstall.exe"
SectionEnd

Section "Uninstall"
    DetailPrint "Removing $INSTDIR from user PATH..."
    EnVar::DeleteValue "PATH" "$INSTDIR"
    Pop $0
    DetailPrint "EnVar::DeleteValue returned: $0"

    # Broadcast environment change
    System::Call 'user32::SendMessageTimeoutA(i 0xffff, i ${WM_SETTINGCHANGE}, i 0, t "Environment", i 0, i 5000, *i .r0)'

    Delete "$INSTDIR\synnax-server.exe"
    Delete "$INSTDIR\synnax.bat"
    Delete "$INSTDIR\uninstall.exe"
    Delete "$DESKTOP\Synnax.lnk"
    Delete "$SMPROGRAMS\Synnax\Synnax.lnk"
    RMDir "$SMPROGRAMS\Synnax"
    RMDir "$INSTDIR"
SectionEnd