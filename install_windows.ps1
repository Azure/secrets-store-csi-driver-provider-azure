# File to install the provider binary 

$global:TargetDir = $env:TARGET_DIR

if ([string]::IsNullOrEmpty($TargetDir))
{
    throw 'target dir is not set. please set TARGET_DIR env var'
}

$azureProviderDir = Join-Path -Path $TargetDir -ChildPath "\azure"

if (!(Test-Path $azureProviderDir))
{
    New-Item -path $azureProviderDir -type Directory
}

Write-Output "Copying file to $azureProviderDir/provider-azure.exe"
Copy-Item "C:\\secrets-store-csi-driver-provider-azure.exe" -Destination $azureProviderDir/provider-azure.exe
