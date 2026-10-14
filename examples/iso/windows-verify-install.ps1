$ErrorActionPreference='Stop'
if (!(Test-Path C:\ProgramData\Packer\installed)) { throw 'missing installed marker' }
Add-Type -TypeDefinition 'using System; using System.Runtime.InteropServices; public class PackerFirmware { [DllImport("kernel32.dll")] public static extern bool GetFirmwareType(out uint type); }'
[uint32]$firmware=0
if (![PackerFirmware]::GetFirmwareType([ref]$firmware) -or $firmware -ne 2) { throw 'not UEFI' }
if ((Get-CimInstance Win32_OperatingSystem).Caption -notmatch 'Windows Server 2022') { throw 'unexpected OS' }
if ($env:PROCESSOR_ARCHITECTURE -ne 'AMD64') { throw 'unexpected architecture' }
