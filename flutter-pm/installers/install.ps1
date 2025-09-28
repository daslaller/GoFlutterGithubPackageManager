# Bootstrap installer for flutter-pm (pre-release)
# Downloads the latest Windows artifact and installs to %USERPROFILE%\bin, then runs it.

$ErrorActionPreference = 'Stop'

$BinDir = Join-Path $env:USERPROFILE 'bin'
if (!(Test-Path $BinDir)) { New-Item -ItemType Directory -Path $BinDir | Out-Null }

# Download from GitHub Releases (latest)
$Arch = if ($env:PROCESSOR_ARCHITECTURE -match 'ARM64') { 'arm64' } else { 'amd64' }
$Url = "https://github.com/daslaller/GoFlutterGithubPackageManager/releases/latest/download/flutter-pm_windows_${Arch}.zip"

Write-Host "Downloading flutter-pm from $Url ..."
$Tmp = New-Item -ItemType Directory -Path ([IO.Path]::GetTempPath() + [IO.Path]::GetRandomFileName())
$Zip = Join-Path $Tmp 'fpm.zip'

Invoke-WebRequest -Uri $Url -OutFile $Zip -UseBasicParsing
Expand-Archive -Path $Zip -DestinationPath $Tmp -Force

Copy-Item -Force (Join-Path $Tmp 'flutter-pm.exe') (Join-Path $BinDir 'flutter-pm.exe')

# Ensure PATH contains %USERPROFILE%\bin
if (-not ($env:PATH -split ';' | Where-Object { $_ -ieq $BinDir })) {
  Write-Host "Adding $BinDir to PATH for current user"
  $current = [Environment]::GetEnvironmentVariable('PATH', 'User')
  if ($null -eq $current) { $current = '' }
  $new = if ($current) { "$current;$BinDir" } else { $BinDir }
  [Environment]::SetEnvironmentVariable('PATH', $new, 'User')
}

Write-Host "Installed flutter-pm to $BinDir"
& (Join-Path $BinDir 'flutter-pm.exe') @args
