# Dotachi Windows Installer

Builds a Windows installer (DotachiSetup.exe) that installs the Dotachi LAN Gaming client and its SoftEther VPN Client dependency.

## Prerequisites

1. **NSIS** (Nullsoft Scriptable Install System) v3.x  
   Download from https://nsis.sourceforge.io/Download  
   Make sure `makensis` is in your PATH.

2. **NSIS Plugins** (included with standard NSIS install):
   - `NSISdl` (for downloading SoftEther installer)
   - `nsExec` (for running service commands)

3. **Wails** v2  
   Install: `go install github.com/wailsapp/wails/v2/cmd/wails@latest`

4. **Go** 1.21+  
   Download from https://go.dev/dl/

5. **Application icon**  
   Place a `dotachi.ico` file in this directory. See `ICON_NOTE.txt` for details.

6. **License file**  
   The installer references `../LICENSE`. Create this file in the client root directory if it does not exist.

## Building the Installer

### Quick build (recommended)

```bat
cd client\installer
build.bat
```

### Manual build

```bat
:: 1. Build the Wails app
cd client
wails build

:: 2. Copy binary to installer directory
copy build\bin\dotachi.exe installer\dotachi.exe

:: 3. Build the installer
cd installer
makensis dotachi.nsi
```

The output is `DotachiSetup.exe` in the `installer` directory.

## What the installer does

1. Copies `dotachi.exe` to `C:\Program Files\Dotachi\`
2. Checks if SoftEther VPN Client is installed (registry, service, and file checks)
3. If SoftEther is missing, downloads and installs it silently
4. Creates Start Menu shortcuts (Dotachi + Uninstall)
5. Optionally creates a Desktop shortcut
6. Registers in Windows Add/Remove Programs
7. Sets up `.dotachi` file association for future click-to-join support

## Testing

1. Build the installer on a Windows machine
2. Run `DotachiSetup.exe` on a clean Windows VM
3. Verify:
   - Dotachi appears in `C:\Program Files\Dotachi\`
   - SoftEther VPN Client is installed (if it was not already)
   - Start Menu shortcuts work
   - Desktop shortcut appears (if selected)
   - "Dotachi LAN Gaming" appears in Add/Remove Programs
   - Uninstaller removes all Dotachi files and shortcuts
   - Uninstaller does NOT remove SoftEther VPN Client

## Customization

- **Version**: Edit `PRODUCT_VERSION` in `dotachi.nsi`
- **SoftEther version**: Update `SE_INSTALLER_URL` in `dotachi.nsi`
- **Icon**: Replace `dotachi.ico` with your own (256x256 recommended)
