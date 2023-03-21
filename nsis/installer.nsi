Name "Vencord Patcher"
ShowInstDetails nevershow
ShowUninstDetails nevershow
SpaceTexts none

OutFile "nsis-installer.exe"

RequestExecutionLevel user

!include "MUI2.nsh"

!define MUI_ICON "../winres/icon.ico"
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_UNPAGE_INSTFILES

Function .onInit
    StrCpy $INSTDIR $LOCALAPPDATA\VencordPatcher
FunctionEnd

Section "install"
    SetDetailsPrint none

    SetOutPath $INSTDIR

    File ..\VencordInstaller.exe

    WriteUninstaller $INSTDIR\uninstall.exe

    SetErrorLevel 0
    Quit
SectionEnd

Section "uninstall"
    RMDir /R /REBOOTOK $INSTDIR

    SetErrorLevel 0
    Quit
SectionEnd