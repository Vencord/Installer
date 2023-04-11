Name "Vencord Installer"
ShowInstDetails nevershow
ShowUninstDetails nevershow
SpaceTexts none

OutFile "nsis-installer.exe"

RequestExecutionLevel user

!include "MUI2.nsh"

!define MUI_ICON "../winres/icon.ico"
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_UNPAGE_INSTFILES

!define APP_GUID "{5d91adf8-9657-4f7f-823c-14b073576daa}"

!define INSTALL_REGISTRY_KEY "Software\${APP_GUID}"
!define UNINSTALL_REGISTRY_KEY "Software\Microsoft\Windows\CurrentVersion\Uninstall\${APP_GUID}"

Function .onInit
    StrCpy $INSTDIR $LOCALAPPDATA\VencordInstaller
FunctionEnd

Section "install"
    SetDetailsPrint none

    SetOutPath $INSTDIR

    File ..\VencordInstaller.exe

    WriteUninstaller $INSTDIR\uninstall.exe

    CreateDirectory "$SMPROGRAMS\Vencord Installer"
    ClearErrors

    CreateShortCut "$SMPROGRAMS\Vencord Installer\Vencord Installer.lnk" "$INSTDIR\VencordInstaller.exe" "" "$INSTDIR\VencordInstaller.exe" 0 "" "" "Vencord patching utility."
    ClearErrors

    WriteRegStr SHELL_CONTEXT "${INSTALL_REGISTRY_KEY}" InstallLocation "$INSTDIR"
    WriteRegStr SHELL_CONTEXT "${INSTALL_REGISTRY_KEY}" KeepShortcuts "true"
    WriteRegStr SHELL_CONTEXT "${INSTALL_REGISTRY_KEY}" ShortcutName "Vencord Installer"
    WriteRegStr SHELL_CONTEXT "${INSTALL_REGISTRY_KEY}" MenuDirectory "Vencord Installer"

    WriteRegStr SHELL_CONTEXT "${UNINSTALL_REGISTRY_KEY}" DisplayName "Vencord Installer"
    WriteRegStr SHELL_CONTEXT "${UNINSTALL_REGISTRY_KEY}" UninstallString '"$INSTDIR\uninstall.exe"'
    WriteRegStr SHELL_CONTEXT "${UNINSTALL_REGISTRY_KEY}" QuietUninstallString '"$INSTDIR\uninstall.exe" /S'
    WriteRegStr SHELL_CONTEXT "${UNINSTALL_REGISTRY_KEY}" DisplayIcon '"$INSTDIR\VencordInstaller.exe",0'

    # disable modify/repair options
    WriteRegStr SHELL_CONTEXT "${UNINSTALL_REGISTRY_KEY}" NoModify 1
    WriteRegStr SHELL_CONTEXT "${UNINSTALL_REGISTRY_KEY}" NoRepair 1

    SetErrorLevel 0
    Quit
SectionEnd

Section "uninstall"
    RMDir /R /REBOOTOK $INSTDIR
    RMDir /R /REBOOTOK "$SMPROGRAMS\Vencord Installer"

    DeleteRegKey SHELL_CONTEXT "${UNINSTALL_REGISTRY_KEY}"
    DeleteRegKey SHELL_CONTEXT "${INSTALL_REGISTRY_KEY}"

    SetErrorLevel 0
    Quit
SectionEnd