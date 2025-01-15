!include "MUI2.nsh"

!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "English"

Name "Synnax"
OutFile "synnax-v${VERSION}-windows.exe"
RequestExecutionLevel user
InstallDir "$APPDATA\synnax"

Section "MainSection" SEC01
    CreateDirectory "$INSTDIR"
    DetailPrint "Installing to: $INSTDIR"
    Delete "$INSTDIR\synnax.exe"
    Delete "$INSTDIR\synnax-server.bat"
    Delete "$INSTDIR\python311.dll"
    
    SetOutPath "$INSTDIR"
    File /oname=synnax.exe "synnax.exe"
    
    # Create the batch file alias
    FileOpen $0 "$INSTDIR\synnax-server.bat" w
    FileWrite $0 "@echo off$\r$\n"
    FileWrite $0 'synnax.exe %*'
    FileClose $0
    
    File "python311.dll"
    
    CreateDirectory "$SMPROGRAMS\Synnax"
    CreateShortcut "$SMPROGRAMS\Synnax\Synnax.lnk" "$INSTDIR\synnax.exe"
    CreateShortcut "$DESKTOP\Synnax.lnk" "$INSTDIR\synnax.exe"
    
    # Add to PATH
    DetailPrint "Adding to PATH..."
    FileOpen $0 "$INSTDIR\add-path.ps1" w
    FileWrite $0 "[Environment]::SetEnvironmentVariable('Path', [Environment]::GetEnvironmentVariable('Path', 'User') + ';$INSTDIR', 'User')"
    FileClose $0
    nsExec::ExecToLog 'powershell -NoProfile -File "$INSTDIR\add-path.ps1"'
    Delete "$INSTDIR\add-path.ps1"
    
    WriteUninstaller "$INSTDIR\uninstall.exe"
SectionEnd

Section "Uninstall"
    DetailPrint "Removing from PATH..."
    FileOpen $0 "$INSTDIR\remove-path.ps1" w
    FileWrite $0 '[Environment]::SetEnvironmentVariable("Path", ($([Environment]::GetEnvironmentVariable("Path", "User")) -replace [regex]::Escape(";$INSTDIR")), "User")'
    FileClose $0
    nsExec::ExecToLog 'powershell -NoProfile -File "$INSTDIR\remove-path.ps1"'
    Delete "$INSTDIR\remove-path.ps1"
    
    Delete "$INSTDIR\synnax.exe"
    Delete "$INSTDIR\synnax-server.bat"
    Delete "$INSTDIR\python311.dll"
    Delete "$INSTDIR\uninstall.exe"
    Delete "$DESKTOP\Synnax.lnk"
    Delete "$SMPROGRAMS\Synnax\Synnax.lnk"
    RMDir "$SMPROGRAMS\Synnax"
    RMDir "$INSTDIR"
SectionEnd