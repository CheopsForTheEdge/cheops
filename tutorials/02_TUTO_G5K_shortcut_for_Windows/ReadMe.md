# Setup your .ssh/config for windows

## Check OpenSSH

First make sure you have OpenSSH Installed (run a powershell terminal as Admin) :

``` Get-WindowsCapability -Online | ? Name -like 'OpenSSH*' ```

If the client is "Installed" then proceed, otherwise : 

``` Add-WindowsCapability -Online -Name OpenSSH.Client~~~~0.0.1.0 ```

And run the first command again to check if the installation was indeed successful.

## Create your .ssh config file for g5k

work in progress





