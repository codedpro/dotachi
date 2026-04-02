; ---------------------------------------------------------------------------
; Dotachi LAN Gaming - NSIS Installer Script
; ---------------------------------------------------------------------------
; Builds a Windows installer that:
;   1. Installs the Dotachi Wails app
;   2. Checks for SoftEther VPN Client — downloads and installs if missing
;   3. Creates Start Menu and Desktop shortcuts
;   4. Registers an uninstaller
;   5. Sets up .dotachi file association (future click-to-join)
; ---------------------------------------------------------------------------

!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "nsDialogs.nsh"
!include "FileFunc.nsh"

; ---------------------------------------------------------------------------
; General configuration
; ---------------------------------------------------------------------------

!define PRODUCT_NAME        "Dotachi LAN Gaming"
!define PRODUCT_EXE         "dotachi.exe"
!define PRODUCT_VERSION     "1.0.0"
!define PRODUCT_PUBLISHER   "Dotachi"
!define PRODUCT_WEB_SITE    "https://dotachi.com"

!define INSTALL_DIR         "$PROGRAMFILES\Dotachi"
!define UNINSTALLER_NAME    "Uninstall.exe"

; SoftEther VPN Client installer details
!define SE_INSTALLER_URL    "https://github.com/SoftEtherVPN/SoftEtherVPN_Stable/releases/download/v4.42-9798-beta/softether-vpnclient-v4.42-9798-beta-2023.06.30-windows-x64_x86.exe"
!define SE_INSTALLER_FILE   "softether-vpnclient-installer.exe"
!define SE_REGISTRY_KEY     "SOFTWARE\SoftEther VPN Client"
!define SE_SERVICE_NAME     "SEVPNCLIENT"

; Registry keys for uninstaller
!define UNINST_REG_ROOT     HKLM
!define UNINST_REG_KEY      "Software\Microsoft\Windows\CurrentVersion\Uninstall\${PRODUCT_NAME}"

; ---------------------------------------------------------------------------
; Installer attributes
; ---------------------------------------------------------------------------

Name        "${PRODUCT_NAME}"
OutFile     "DotachiSetup.exe"
InstallDir  "${INSTALL_DIR}"
RequestExecutionLevel admin

; Version info embedded in the EXE
VIProductVersion                 "${PRODUCT_VERSION}.0"
VIAddVersionKey "ProductName"     "${PRODUCT_NAME}"
VIAddVersionKey "ProductVersion"  "${PRODUCT_VERSION}"
VIAddVersionKey "CompanyName"     "${PRODUCT_PUBLISHER}"
VIAddVersionKey "FileDescription" "${PRODUCT_NAME} Installer"
VIAddVersionKey "FileVersion"     "${PRODUCT_VERSION}"

; ---------------------------------------------------------------------------
; Modern UI configuration
; ---------------------------------------------------------------------------

!define MUI_ABORTWARNING
!define MUI_ICON "dotachi.ico"
!define MUI_UNICON "dotachi.ico"

; Installer pages
!insertmacro MUI_PAGE_WELCOME
!insertmacro MUI_PAGE_LICENSE "..\LICENSE"
!insertmacro MUI_PAGE_DIRECTORY
!insertmacro MUI_PAGE_COMPONENTS
!insertmacro MUI_PAGE_INSTFILES
!insertmacro MUI_PAGE_FINISH

; Uninstaller pages
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES

; Language
!insertmacro MUI_LANGUAGE "English"

; ---------------------------------------------------------------------------
; Installer sections
; ---------------------------------------------------------------------------

Section "Dotachi Application" SEC_APP
    SectionIn RO  ; Required — cannot be deselected

    SetOutPath "$INSTDIR"

    ; Copy main executable
    File "dotachi.exe"

    ; Create uninstaller
    WriteUninstaller "$INSTDIR\${UNINSTALLER_NAME}"

    ; Start Menu shortcuts
    CreateDirectory "$SMPROGRAMS\${PRODUCT_NAME}"
    CreateShortCut  "$SMPROGRAMS\${PRODUCT_NAME}\Dotachi.lnk" "$INSTDIR\${PRODUCT_EXE}" "" "$INSTDIR\${PRODUCT_EXE}" 0
    CreateShortCut  "$SMPROGRAMS\${PRODUCT_NAME}\Uninstall Dotachi.lnk" "$INSTDIR\${UNINSTALLER_NAME}" "" "$INSTDIR\${UNINSTALLER_NAME}" 0

    ; Register uninstaller in Add/Remove Programs
    WriteRegStr   ${UNINST_REG_ROOT} "${UNINST_REG_KEY}" "DisplayName"     "${PRODUCT_NAME}"
    WriteRegStr   ${UNINST_REG_ROOT} "${UNINST_REG_KEY}" "UninstallString" '"$INSTDIR\${UNINSTALLER_NAME}"'
    WriteRegStr   ${UNINST_REG_ROOT} "${UNINST_REG_KEY}" "InstallLocation" "$INSTDIR"
    WriteRegStr   ${UNINST_REG_ROOT} "${UNINST_REG_KEY}" "DisplayIcon"     "$INSTDIR\${PRODUCT_EXE}"
    WriteRegStr   ${UNINST_REG_ROOT} "${UNINST_REG_KEY}" "Publisher"       "${PRODUCT_PUBLISHER}"
    WriteRegStr   ${UNINST_REG_ROOT} "${UNINST_REG_KEY}" "URLInfoAbout"    "${PRODUCT_WEB_SITE}"
    WriteRegStr   ${UNINST_REG_ROOT} "${UNINST_REG_KEY}" "DisplayVersion"  "${PRODUCT_VERSION}"
    WriteRegDWORD ${UNINST_REG_ROOT} "${UNINST_REG_KEY}" "NoModify" 1
    WriteRegDWORD ${UNINST_REG_ROOT} "${UNINST_REG_KEY}" "NoRepair" 1

    ; Calculate installed size
    ${GetSize} "$INSTDIR" "/S=0K" $0 $1 $2
    IntFmt $0 "0x%08X" $0
    WriteRegDWORD ${UNINST_REG_ROOT} "${UNINST_REG_KEY}" "EstimatedSize" "$0"

    ; File association: .dotachi files (future click-to-join)
    WriteRegStr HKCR ".dotachi" "" "Dotachi.JoinLink"
    WriteRegStr HKCR "Dotachi.JoinLink" "" "Dotachi Join Link"
    WriteRegStr HKCR "Dotachi.JoinLink\DefaultIcon" "" "$INSTDIR\${PRODUCT_EXE},0"
    WriteRegStr HKCR "Dotachi.JoinLink\shell\open\command" "" '"$INSTDIR\${PRODUCT_EXE}" "--join" "%1"'

SectionEnd

Section "Desktop Shortcut" SEC_DESKTOP
    CreateShortCut "$DESKTOP\Dotachi.lnk" "$INSTDIR\${PRODUCT_EXE}" "" "$INSTDIR\${PRODUCT_EXE}" 0
SectionEnd

Section "SoftEther VPN Client" SEC_SOFTETHER
    SectionIn RO  ; Required — cannot be deselected

    ; Check if SoftEther VPN Client is already installed
    Call CheckSoftEtherInstalled
    Pop $0

    ${If} $0 == "1"
        DetailPrint "SoftEther VPN Client is already installed. Skipping."
    ${Else}
        DetailPrint "SoftEther VPN Client not found. Downloading..."

        ; Download the SoftEther VPN Client installer
        NSISdl::download "${SE_INSTALLER_URL}" "$TEMP\${SE_INSTALLER_FILE}"
        Pop $R0
        ${If} $R0 != "success"
            MessageBox MB_OK|MB_ICONEXCLAMATION "Failed to download SoftEther VPN Client.$\n$\nError: $R0$\n$\nPlease install it manually from:$\nhttps://www.softether.org/5-download"
            Goto softether_done
        ${EndIf}

        DetailPrint "Installing SoftEther VPN Client silently..."

        ; Run the SoftEther installer silently
        ExecWait '"$TEMP\${SE_INSTALLER_FILE}" /S' $R1

        ${If} $R1 != 0
            MessageBox MB_OK|MB_ICONEXCLAMATION "SoftEther VPN Client installation may have failed (exit code: $R1).$\n$\nPlease install it manually from:$\nhttps://www.softether.org/5-download"
        ${Else}
            DetailPrint "SoftEther VPN Client installed successfully."
        ${EndIf}

        ; Clean up downloaded installer
        Delete "$TEMP\${SE_INSTALLER_FILE}"

        ; Start the SoftEther VPN Client service
        DetailPrint "Starting SoftEther VPN Client service..."
        nsExec::ExecToLog 'net start ${SE_SERVICE_NAME}'

    softether_done:
    ${EndIf}

SectionEnd

; ---------------------------------------------------------------------------
; Section descriptions
; ---------------------------------------------------------------------------

!insertmacro MUI_FUNCTION_DESCRIPTION_BEGIN
    !insertmacro MUI_DESCRIPTION_TEXT ${SEC_APP}       "Install the Dotachi LAN Gaming application. (Required)"
    !insertmacro MUI_DESCRIPTION_TEXT ${SEC_DESKTOP}   "Create a shortcut on your Desktop."
    !insertmacro MUI_DESCRIPTION_TEXT ${SEC_SOFTETHER} "Install SoftEther VPN Client if not already present. (Required)"
!insertmacro MUI_FUNCTION_DESCRIPTION_END

; ---------------------------------------------------------------------------
; Helper functions
; ---------------------------------------------------------------------------

; CheckSoftEtherInstalled — pushes "1" on stack if installed, "0" otherwise
Function CheckSoftEtherInstalled
    ; Method 1: Check the registry
    ReadRegStr $0 HKLM "SOFTWARE\SoftEther VPN Client" ""
    ${If} $0 != ""
        Push "1"
        Return
    ${EndIf}

    ; Method 2: Check 64-bit registry view
    SetRegView 64
    ReadRegStr $0 HKLM "SOFTWARE\SoftEther VPN Client" ""
    SetRegView 32
    ${If} $0 != ""
        Push "1"
        Return
    ${EndIf}

    ; Method 3: Check if the service exists
    nsExec::ExecToStack 'sc query ${SE_SERVICE_NAME}'
    Pop $0
    ${If} $0 == 0
        Push "1"
        Return
    ${EndIf}

    ; Method 4: Check common install paths
    ${If} ${FileExists} "$PROGRAMFILES\SoftEther VPN Client\vpncmd.exe"
        Push "1"
        Return
    ${EndIf}
    ${If} ${FileExists} "$PROGRAMFILES64\SoftEther VPN Client\vpncmd.exe"
        Push "1"
        Return
    ${EndIf}

    Push "0"
FunctionEnd

; ---------------------------------------------------------------------------
; Uninstaller
; ---------------------------------------------------------------------------

Section "Uninstall"

    ; Remove application files
    Delete "$INSTDIR\${PRODUCT_EXE}"
    Delete "$INSTDIR\${UNINSTALLER_NAME}"
    RMDir  "$INSTDIR"

    ; Remove Start Menu shortcuts
    Delete "$SMPROGRAMS\${PRODUCT_NAME}\Dotachi.lnk"
    Delete "$SMPROGRAMS\${PRODUCT_NAME}\Uninstall Dotachi.lnk"
    RMDir  "$SMPROGRAMS\${PRODUCT_NAME}"

    ; Remove Desktop shortcut
    Delete "$DESKTOP\Dotachi.lnk"

    ; Remove file association
    DeleteRegKey HKCR ".dotachi"
    DeleteRegKey HKCR "Dotachi.JoinLink"

    ; Remove uninstaller registry key
    DeleteRegKey ${UNINST_REG_ROOT} "${UNINST_REG_KEY}"

    ; Note: We do NOT uninstall SoftEther VPN Client, as the user may
    ; want to keep it for other purposes.

SectionEnd
