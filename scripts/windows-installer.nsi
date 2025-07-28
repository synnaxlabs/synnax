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
RequestExecutionLevel admin
InstallDir "$PROGRAMFILES64\Synnax\Core"
ShowInstDetails show

Section "MainSection" SEC01
    DetailPrint "Resolved INSTALL DIR: $INSTDIR"

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


    # Add to system PATH using EnVar (requires EnVar.dll in x86-unicode folder)
    DetailPrint "Adding $INSTDIR to system PATH..."
    EnVar::SetHKLM
    EnVar::AddValue "PATH" "$INSTDIR"
    Pop $0
    DetailPrint "EnVar::AddValue (system PATH) returned: $0"

    WriteUninstaller "$INSTDIR\uninstall.exe"
SectionEnd

Section "Uninstall"
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
    RMDir "$SMPROGRAMS\Synnax"
    RMDir "$INSTDIR"
SectionEnd