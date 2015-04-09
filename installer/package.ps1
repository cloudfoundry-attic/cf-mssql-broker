<#
.SYNOPSIS
    Packaging and installation script for the MsSql Broker
.DESCRIPTION
    This script packages all the binaries into an self-extracting file.
    Upon self-extraction this script is run to unpack and install the MsSql Broker services.

.PARAMETER action
    This is the parameter that specifies what the script should do: package the binaries and create the installer, or install the services.

.PARAMETER binDir
    When the action is 'package', this parameter specifies where the binaries are located. Not used otherwise.

.NOTES
    Author: Florin Dragos
    Date:   March 31, 2015
#>
param (
    [Parameter(Mandatory=$true)]
    [ValidateSet('package','install')]
    [string] $action,
    [string] $binDir
)

if (($pshome -like "*syswow64*") -and ((Get-WmiObject Win32_OperatingSystem).OSArchitecture -like "64*")) {
    Write-Warning "Restarting script under 64 bit powershell"
    
    $powershellLocation = join-path ($pshome -replace "syswow64", "sysnative") "powershell.exe"
    $scriptPath = $SCRIPT:MyInvocation.MyCommand.Path
    
    # relaunch this script under 64 bit shell
    $process = Start-Process -Wait -PassThru -NoNewWindow $powershellLocation "-nologo -file ${scriptPath} -action ${action} -binDir ${binDir}"
    
    # This will exit the original powershell process. This will only be done in case of an x86 process on a x64 OS.
    exit $process.ExitCode
}

function DoAction-Package($binDir)
{
    Write-Output "Packaging files from the ${binDir} dir ..."
    [Reflection.Assembly]::LoadWithPartialName( "System.IO.Compression.FileSystem" ) | out-null

    $destFile = Join-Path $(Get-Location) "binaries.zip"
    $compressionLevel = [System.IO.Compression.CompressionLevel]::Optimal
    $includeBaseDir = $false
    Remove-Item -Force -Path $destFile -ErrorAction SilentlyContinue

    Write-Output 'Creating zip ...'

    [System.IO.Compression.ZipFile]::CreateFromDirectory($binDir, $destFile, $compressionLevel, $includeBaseDir)

    Write-Output 'Creating the self extracting exe ...'

    $installerProcess = Start-Process -Wait -PassThru -NoNewWindow 'iexpress' "/N /Q mssql-broker-installer.sed"

    if ($installerProcess.ExitCode -ne 0)
    {
        Write-Output $installerProcess.StandardOutput.ReadToEnd()
        Write-Error "There was an error building the installer."
        Write-Error $installerProcess.StandardError.ReadToEnd()
        exit 1
    }
    
    Write-Output 'Removing artifacts ...'
    Remove-Item -Force -Path $destfile -ErrorAction SilentlyContinue
    
    Write-Output 'Done.'
}

function DoAction-Install()
{
    Write-Output 'Installing MsSql Broker ...'
    
    if ($env:MSSQL_SERVER -eq $null)
    {
        Write-Error 'Could not find environment variable MSSQL_SERVER. Please set it and run the setup again.'
        exit 1        
    }
    $mssqlServer = $env:MSSQL_SERVER

    if ($env:BROKER_USERNAME -eq $null)
    {
        Write-Error 'Could not find environment variable BROKER_USERNAME. Please set it and run the setup again.'
        exit 1        
    }
    $brokerUsername = $env:BROKER_USERNAME

    if ($env:BROKER_PASSWORD -eq $null)
    {
        Write-Error 'Could not find environment variable BROKER_PASSWORD. Please set it and run the setup again.'
        exit 1        
    }
    $brokerPassword = $env:BROKER_PASSWORD

    
    if ($env:MSSQL_USER -eq $null)
    {
        $trustedConnection = $true
    }
    else
    {
        $mssqlUser = $env:MSSQL_USER
        if ($env:MSSQL_PASSWORD -eq $null)
        {
            $mssqlPassword = ""
        }
        
    }
    
    if ($env:BROKER_DESTFOLDER -eq $null)
    {
        $destFolder = 'c:\mssql-broker'
    }
    else
    {
        $destFolder = $env:BROKER_DESTFOLDER
    }

    if ($env:BROKER_LOGFOLDER -eq $null)
    {
        $logFolder = (Join-Path $destFolder 'log')
    }
    else
    {
        $logFolder = $env:BROKER_LOGFOLDER
    }

    if ($env:MSSQL_BINDING_HOST -eq $null)
    {
        $bindingHost = (get-netadapter | get-netipaddress | ? addressfamily -eq 'IPv4').ipaddress
    }
    else
    {
        $bindingHost = $env:MSSQL_BINDING_HOST
    }

    if ($env:MSSQL_SERVICE_NAME -eq $null)
    {
        $providedServiceName = "mssql"
    }

    Write-Output "Using server ${mssqlServer}"
    Write-Output "Using installation folder ${destFolder}"
    Write-Output "Using log folder ${logFolder}"

    foreach ($dir in @($destFolder, $logFolder))
    {
        Write-Output "Cleaning up directory ${dir}"
        Remove-Item -Force -Recurse -Path $dir -ErrorVariable errors -ErrorAction SilentlyContinue

        if ($errs.Count -eq 0)
        {
            Write-Output "Successfully cleaned the directory ${dir}"
        }
        else
        {
            Write-Error "There was an error cleaning up the directory '${dir}'.`r`nPlease make sure the folder and any of its child items are not in use, then run the installer again."
            exit 1;
        }

        Write-Output "Setting up directory ${dir}"
        New-Item -path $dir -type directory -Force -ErrorAction SilentlyContinue
    }

    [Reflection.Assembly]::LoadWithPartialName( "System.IO.Compression.FileSystem" ) | out-null
    $srcFile = ".\binaries.zip"

    Write-Output 'Unpacking files ...'
    try
    {
        [System.IO.Compression.ZipFile]::ExtractToDirectory($srcFile, $destFolder)
    }
    catch
    {
        Write-Error "There was an error writing to the installation directory '${destFolder}'.`r`nPlease make sure the folder and any of its child items are not in use, then run the installer again."
        exit 1;
    }

    $configFile = (Join-Path $destfolder 'cf_mssql_broker_config.json')

    $config = Get-Content -Raw -Path $configFile | ConvertFrom-Json
    $config.servedMssqlBindingHostname = $bindingHost
    $config.brokerMssqlConnection.server = $mssqlServer
    if($trustedConnection -eq $true)
    {
        $config.brokerMssqlConnection | Add-Member -Name "trusted_connection" -Value "yes" -MemberType NoteProperty -Force
    }
    else
    {
        $config.brokerMssqlConnection | Add-Member -Name "uid" -Value $mssqlUser -MemberType NoteProperty -Force
        $config.brokerMssqlConnection | Add-Member -Name "pwd" -Value $mssqlPassword -MemberType NoteProperty -Force
    }
    
    $config.brokerCredentials | Add-Member -Name "username" -Value $brokerUsername -MemberType NoteProperty -Force
    $config.brokerCredentials | Add-Member -Name "password" -Value $brokerPassword -MemberType NoteProperty -Force

    #set service catalog
    $serviceCatalog = $config.serviceCatalog | Select-Object -Index 0
    $serviceCatalog.name = $providedServiceName
    $serviceCatalog.id = [guid]::NewGuid()

    #generate plan ID
    $newPlanID = [guid]::NewGuid()
    $serviceCatalog.plans | Select-Object -Index 0 | Add-Member -Name "id" -Value $newPlanID -MemberType NoteProperty -Force

    $config  | ConvertTo-Json -depth 999 | Out-File $configFile -Encoding ascii

    InstallBroker $destfolder $logFolder $configFile
}

function InstallBroker($destfolder, $logFolder, $configFile)
{
    Write-Output "Installing service"

    $binary = (Join-Path $destfolder 'cf-mssql-broker.exe')
    $serviceName = "MsSqlBroker"

    $service = Get-WmiObject -Class Win32_Service -Filter "Name='${serviceName}'"
    if ($service -ne $null)
    {
        Write-Output "Stopping service ${serviceName}"
        Stop-Service -DisplayName $serviceName
        Write-Output "Removing service ${serviceName}"
        $service.delete()            
    }
    
    New-Service -Name $serviceName -BinaryPathName "${binary} -logDir ${logFolder} -config ${configFile}" -DisplayName $serviceName -StartupType Automatic
    Start-Service -DisplayName $serviceName
    
    # Setup a firewall rule
    New-NetFirewallRule -DisplayName “Allow MsSql Broker TCP/IP Communication” -Direction Inbound -Program $binary -RemoteAddress LocalSubnet -Action Allow
}

if ($action -eq 'package')
{
    if ([string]::IsNullOrWhiteSpace($binDir))
    {
        Write-Error 'The binDir parameter is mandatory when packaging.'
        exit 1
    }
    
    $binDir = Resolve-Path $binDir
    
    if ((Test-Path $binDir) -eq $false)
    {
        Write-Error "Could not find directory ${binDir}."
        exit 1        
    }
    
    Write-Output "Using binary dir ${binDir}"
    
    DoAction-Package $binDir
}
elseif ($action -eq 'install')
{
    DoAction-Install
}
