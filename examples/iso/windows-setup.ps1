param([Parameter(Mandatory=$true)][string]$ClientAddress)
$ErrorActionPreference='Stop'
New-Item -ItemType Directory -Force C:\ProgramData\Packer | Out-Null
Copy-Item (Join-Path $PSScriptRoot 'verify-install.ps1') C:\ProgramData\Packer\verify-install.ps1 -Force
Enable-PSRemoting -Force -SkipNetworkProfileCheck
Get-ChildItem WSMan:\localhost\Listener | Remove-Item -Recurse -Force
$cert=New-SelfSignedCertificate -DnsName $env:COMPUTERNAME -CertStoreLocation Cert:\LocalMachine\My
New-Item -Path WSMan:\localhost\Listener -Transport HTTPS -Address * -CertificateThumbPrint $cert.Thumbprint -Force | Out-Null
Set-Item WSMan:\localhost\Service\AllowUnencrypted $false
Set-Service WinRM -StartupType Automatic
Get-NetFirewallRule -Name 'WINRM-HTTP-In-TCP*' -ErrorAction SilentlyContinue | Disable-NetFirewallRule
New-NetFirewallRule -DisplayName 'Packer acceptance HTTPS WinRM' -Direction Inbound -Action Allow -Protocol TCP -LocalPort 5986 -RemoteAddress $ClientAddress -Profile Any | Out-Null
New-Item -ItemType File -Force C:\ProgramData\Packer\installed | Out-Null
& C:\ProgramData\Packer\verify-install.ps1
